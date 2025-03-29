package main

import (
	"context"
	"expvar"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"time"

	"nathejk.dk/cmd/api/app"
	"nathejk.dk/internal/data"
	"nathejk.dk/internal/jsonlog"
	"nathejk.dk/internal/mailer"
	"nathejk.dk/internal/sms"
	"nathejk.dk/internal/vcs"
	"nathejk.dk/nathejk/commands"
	"nathejk.dk/nathejk/table"
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
	countdown struct {
		time   string
		videos []string
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
	jetstream streaminterface.Stream
	commands  commands.Commands
	mailer    mailer.Mailer
	sms       sms.Sender
	logger    *jsonlog.Logger
}

func main() {
	var cfg config

	flag.IntVar(&cfg.port, "port", 80, "API server port")
	flag.StringVar(&cfg.webroot, "webroot", getEnv("WEBROOT", "/www"), "Static web root")

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
	flag.StringVar(&cfg.smtp.Sender, "smtp-sender", "Nathejk <hej@nathejk.dk>", "SMTP sender")

	flag.StringVar(&cfg.countdown.time, "countdown", getEnv("COUNTDOWN", ""), "Time for countdown")
	cfg.countdown.videos = getEnvAsSlice("COUNTDOWN_VIDEOS", []string{}, "\n")

	flag.Parse()

	//logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)
	logger := jsonlog.New(os.Stdout, jsonlog.LevelInfo)
	logger.PrintInfo("Starting API...", nil)

	js, err := jetstream.New(cfg.jetstream.dsn)
	if err != nil {
		log.Printf("Error connecting %q", err)
	}
	logger.PrintInfo("Jetstream connected", nil)
	/*msg, err := js.LastMessage(streaminterface.SubjectFromStr("NATHEJK.2024.>"))
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

	sqlw := sqlpersister.New(db.DB())

	mux := xstream.NewMux(js)
	mux.AddConsumer(table.NewSignup(sqlw), table.NewConfirm(sqlw), table.NewKlan(sqlw), table.NewSenior(sqlw), table.NewPatrulje(sqlw), table.NewPatruljeStatus(sqlw) /*table.NewPatruljeMerged(sqlw),*/, table.NewSpejder(sqlw), table.NewSpejderStatus(sqlw))
	//mux.AddConsumer(table.NewSpejder(sqlw), table.NewSpejderStatus(sqlw))
	if err := mux.Run(context.Background()); err != nil {
		logger.PrintFatal(err, nil)
	}

	models := data.NewModels(db.DB())

	expvar.NewString("version").Set(version)
	expvar.NewInt("timestamp").Set(time.Now().Unix())
	expvar.NewInt("goroutines").Set(int64(runtime.NumGoroutine()))

	smsclient, err := sms.NewClient(cfg.sms.dsn)
	if err != nil {
		logger.PrintFatal(err, nil)
	}

	app := &application{
		JsonApi: app.JsonApi{
			Logger: logger,
		},
		config:    cfg,
		models:    models,
		jetstream: js,
		commands:  commands.New(js, models),
		mailer:    mailer.NewFromConfig(cfg.smtp),
		sms:       smsclient,
		logger:    logger,
	}
	logger.PrintInfo("Application initialized", nil)

	logger.PrintFatal(app.Serve(fmt.Sprintf(":%d", cfg.port), app.routes()), nil)
}
