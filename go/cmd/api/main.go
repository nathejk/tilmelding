package main

import (
	"context"
	"database/sql"
	"expvar"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/nathejk/shared-go/types"
	"nathejk.dk/cmd/api/app"
	"nathejk.dk/internal/data"
	"nathejk.dk/internal/jsonlog"
	"nathejk.dk/internal/mailer"
	"nathejk.dk/internal/payment/mobilepay"
	"nathejk.dk/internal/sms"
	"nathejk.dk/internal/vcs"
	"nathejk.dk/nathejk/table"
	"nathejk.dk/nathejk/table/crewmember"
	"nathejk.dk/nathejk/table/klan"
	"nathejk.dk/nathejk/table/order"
	"nathejk.dk/nathejk/table/patrulje"
	payments "nathejk.dk/nathejk/table/payment"
	"nathejk.dk/nathejk/table/personnel"
	"nathejk.dk/nathejk/table/product"
	"nathejk.dk/nathejk/table/section"
	"nathejk.dk/nathejk/table/senior"
	"nathejk.dk/nathejk/table/signup"
	"nathejk.dk/nathejk/table/spejder"
	"nathejk.dk/pkg/sqlpersister"
	"nathejk.dk/superfluids/jetstream"
	"nathejk.dk/superfluids/streaminterface"
	"nathejk.dk/superfluids/xstream"
)

var (
	version = vcs.Version()
)

// Define a config struct to hold all the configuration settings for our application.
type config struct {
	port      int
	webroot   string
	baseurl   string
	year      types.YearSlug
	countdown struct {
		time   string
		videos []string
	}
	payment struct {
		dsn string
	}
	db struct {
		dsn          string
		maxOpenConns int
		maxIdleConns int
		maxIdleTime  string
	}
	jetstream struct {
		dsn string
	}
	sms struct {
		dsn string
	}
	smtp mailer.Config
}

type application struct {
	app.JsonApi

	config    config
	models    data.Models
	db        *sql.DB
	jetstream streaminterface.Stream
	commands  commands
	mailer    mailer.Mailer
	sms       sms.Sender
	payment   mobilepay.Client
	logger    *jsonlog.Logger
}

// commands wires the entity-local write-side APIs together for handlers
// to consume. Each field is satisfied by the table package responsible
// for that entity (commands.go alongside table.go).
type commands struct {
	Signup     signup.Commands
	Klan       klan.Commands
	Patrulje   patrulje.Commands
	Personnel  personnel.Commands
	Payment    payments.Commands
	Order      order.Commands
	Section    section.Commands
	Crewmember crewmember.Commands
}

func main() {
	var cfg config

	flag.IntVar(&cfg.port, "port", 80, "API server port")
	flag.StringVar(&cfg.webroot, "webroot", getEnv("WEBROOT", "/www"), "Static web root")
	flag.StringVar(&cfg.baseurl, "baseurl", getEnv("BASEURL", "https://tilmelding.nathejk.dk"), "Base url of website")
	var year string
	flag.StringVar(&year, "year", getEnv("YEAR", fmt.Sprintf("%d", time.Now().Year())), "active year slug")

	flag.StringVar(&cfg.sms.dsn, "sms-dsn", os.Getenv("SMS_DSN"), "SMS DSN")
	flag.StringVar(&cfg.jetstream.dsn, "jetstream-dsn", os.Getenv("JETSTREAM_DSN"), "NATS Streaming DSN")

	flag.StringVar(&cfg.db.dsn, "db-dsn", os.Getenv("DB_DSN"), "Database DSN")
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "Database max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", 25, "Database max idle connections")
	flag.StringVar(&cfg.db.maxIdleTime, "db-max-idle-time", "15m", "Database max connection idle time")

	flag.StringVar(&cfg.smtp.Host, "smtp-host", os.Getenv("SMTP_HOST"), "SMTP host")
	flag.IntVar(&cfg.smtp.Port, "smtp-port", getEnvAsInt("SMTP_PORT", 25), "SMTP port")
	flag.StringVar(&cfg.smtp.Username, "smtp-username", os.Getenv("SMTP_USERNAME"), "SMTP username")
	flag.StringVar(&cfg.smtp.Password, "smtp-password", os.Getenv("SMTP_PASSWORD"), "SMTP password")
	flag.StringVar(&cfg.smtp.Sender, "smtp-sender", "Nathejk <kontakt@nathejk.dk>", "SMTP sender")

	flag.StringVar(&cfg.countdown.time, "countdown", getEnv("COUNTDOWN", ""), "Time for countdown")
	flag.StringVar(&cfg.payment.dsn, "payment-dsn", getEnv("PAYMENT_DSN", ""), "DSN specifing a valid payment provider")
	cfg.countdown.videos = getEnvAsSlice("COUNTDOWN_VIDEOS", []string{}, "\n")

	flag.Parse()
	cfg.year = types.YearSlug(year)

	//logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)
	logger := jsonlog.New(os.Stdout, jsonlog.LevelInfo)
	logger.PrintInfo("Starting API...", nil)

	js, err := jetstream.New(cfg.jetstream.dsn)
	if err != nil {
		log.Printf("Error connecting %q", err)
	}
	logger.PrintInfo("Jetstream connected", nil)
	/*msg, err := js.LastMessage(streaminterface.SubjectFromStr("NATHEJK.>"))
	if err != nil {
		log.Fatalf("Last message: %q", err)
	}
	log.Printf("Last message (%d) %v", msg.Sequence(), msg)
	*/

	db := NewDatabase(cfg.db)
	if err := db.Open(); err != nil {
		logger.PrintFatal(err, nil)
	}
	defer db.Close()
	logger.PrintInfo("Database connected", nil)

	smsclient, err := sms.NewClient(cfg.sms.dsn)
	if err != nil {
		logger.PrintFatal(err, nil)
	}

	mailclient := mailer.NewFromConfig(cfg.smtp).AddOptions(mailer.WithGlobalVar("baseurl", cfg.baseurl))

	reader := db.DB()
	writer := sqlpersister.New(db.DB())

	// Product catalogue is seeded at startup. Idempotent: re-running this with
	// the same seed list updates names / prices / stock in place.
	tableProduct := product.New(writer, reader)
	if err := tableProduct.Seed(product.Seeds2026()); err != nil {
		logger.PrintFatal(err, nil)
	}

	tablePayment := table.NewPayment(writer, reader)
	tableStaff := personnel.New(js, writer, reader)
	tablePatrulje := patrulje.New(js, writer, reader)
	tableSpejder := spejder.New(writer, reader)
	tableSignup := signup.New(js, writer, reader, signup.WithSms(smsclient), signup.WithMailer(mailclient))
	// The 115-seat klan cap now lives on participation.klan.stock in the
	// product catalogue (see product.Seeds2026). klan.WithProductQueries
	// wires the catalogue in so RequestMemberCount can read it.
	tableKlan := klan.New(js, writer, reader, klan.WithProductQueries(tableProduct), klan.WithTeamMaxMemberCount(4))
	tableSenior := senior.New(writer, reader)

	// Order projection + commander. Subscribes to NATHEJK:*.order.*.{created,
	// lines.changed, cancelled, paid} via the mux below.
	tableOrder := order.New(js, writer, reader, cfg.year, tableProduct)

	// Crew organisation: sections (function/role/unit hierarchy) and the
	// crew members assigned to them. Both are event-sourced projections
	// registered on the mux below.
	tableSection := section.New(js, writer, reader)
	tableCrewmember := crewmember.New(js, writer, reader)

	// Saga that bridges payment events into NathejkOrderPaid, which the
	// order projector then projects into status=paid on the orders table.
	orderSaga := order.NewSaga(js, tableOrder, tablePayment, 0)

	mux := xstream.NewMux(js)
	mux.AddConsumer(table.NewConfirm(writer), tableKlan, tableSenior /*table.NewPatrulje(sqlw),*/, table.NewPatruljeStatus(writer) /*table.NewPatruljeMerged(sqlw),*/, tableSpejder, table.NewSpejderStatus(writer), tablePayment, tableStaff, tablePatrulje, tableSignup, tableOrder, orderSaga, tableSection, tableCrewmember)
	//mux.AddConsumer(table.NewSpejder(sqlw), table.NewSpejderStatus(sqlw))
	if err := mux.Run(context.Background()); err != nil {
		logger.PrintFatal(err, nil)
	}

	models := data.NewModels(reader, tablePayment, tableStaff, tablePatrulje, tableSignup, tableKlan, tableOrder, tableProduct, tableSection, tableCrewmember)

	expvar.NewString("version").Set(version)
	expvar.NewInt("timestamp").Set(time.Now().Unix())
	expvar.NewInt("goroutines").Set(int64(runtime.NumGoroutine()))

	paymentClient, err := mobilepay.New(cfg.payment.dsn)
	if err != nil {
		logger.PrintFatal(err, nil)
	}
	app := &application{
		JsonApi: app.JsonApi{
			Logger: logger,
		},
		config:    cfg,
		payment:   paymentClient,
		models:    models,
		db:        reader,
		jetstream: js,
		commands: commands{
			Signup:     tableSignup,
			Klan:       tableKlan,
			Patrulje:   tablePatrulje,
			Personnel:  tableStaff,
			Payment:    payments.NewCommands(js, paymentClient),
			Order:      tableOrder,
			Section:    tableSection,
			Crewmember: tableCrewmember,
		},
		mailer: mailclient,
		sms:    smsclient,
		logger: logger,
	}

	logger.PrintInfo("Application initialized", nil)

	logger.PrintFatal(app.Serve(fmt.Sprintf(":%d", cfg.port), app.routes()), nil)
}
