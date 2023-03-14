/*
 * Genarate rsa keys.
 */

package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/go-sql-driver/mysql"
	"github.com/rs/cors"

	//bridge "nathejk.dk/bridge"
	"nathejk.dk/cmd/api/command"
	"nathejk.dk/cmd/api/handlers"
	"nathejk.dk/cmd/api/query"
	"nathejk.dk/pkg/cpsms"
	"nathejk.dk/pkg/memorystream"
	"nathejk.dk/pkg/memstat"
	"nathejk.dk/pkg/nats"
	"nathejk.dk/pkg/sqlpersister"
	"nathejk.dk/pkg/stream"
	"nathejk.dk/pkg/streaminterface"
	"nathejk.dk/table"
)

func main() {
	fmt.Println("Starting API service")

	natsstream := nats.NewNATSStreamUnique(os.Getenv("STAN_DSN"), "tilmelding-api")
	defer natsstream.Close()

	db, err := sql.Open("mysql", os.Getenv("DB_DSN_RW"))
	if err != nil {
		log.Fatal(err)
	}
	/*
		mdb, err := sql.Open("mysql", os.Getenv("MONOLITH_DB_DSN_RW"))
		if err != nil {
			log.Fatal(err)
		}
	*/
	sms := cpsms.New(os.Getenv("CPSMS_API_URL"), os.Getenv("CPSMS_API_KEY"))

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	go func() {
		<-sig
		log.Printf("\nReceived an interrupt, unsubscribing and closing connection...\n\n")
		natsstream.Close()
		os.Exit(99)
	}()

	//sqlw := sqldump.New(nil)
	//monolithsql := sqlpersister.New(mdb)
	sqlw := sqlpersister.New(db)
	//_, _ = sqlw, monolithsql

	memstream := memorystream.New()
	//cmd := command.NewCommand(natsstream)

	consumers := []streaminterface.Consumer{
		table.NewRegistrant(sqlw),
		table.NewPatrulje(sqlw),
		table.NewSpejder(sqlw),
		table.NewKlan(sqlw),
		table.NewSenior(sqlw),
		table.NewSignup(sqlw),
		table.NewPincode(sqlw),
		table.NewMailTemplate(sqlw),

		//bridge.NewPatrulje(monolithsql),
		//bridge.NewKlan(monolithsql),
		//	cmd,
	}

	streammux := stream.NewStreamMux(memstream)
	streammux.Handles(natsstream, natsstream.Channels()...)
	swtch, err := stream.NewSwitch(streammux, consumers)
	if err != nil {
		log.Fatal(err)
	}

	live := make(chan struct{})
	ctx := context.Background()
	go func() {
		err = swtch.Run(ctx, func() {
			memstat.PrintMemoryStats()
			fmt.Println(swtch.Stats().Format())
			live <- struct{}{}
		})
		if err != nil {
			log.Fatal(err)
		}
	}()

	// Waiting for live
	select {
	case <-ctx.Done():
		log.Fatal(ctx.Err())
	case <-live:
	}

	/*
		start := time.Now()
		log.Printf("Running models...")
		model.Run(natsstream, models, func() {
			//logHandler.Prefix = "live"
			//logHandler.Mod = 10000

			elapsed := time.Now().Sub(start)
			log.Printf("\n---------------------------------------------------\nAll caught up in: %s\n---------------------------------------------------\n", elapsed.String())
		}) //, logHandler)

		/*
		   r := chi.NewRouter()
		   	r.Use(middleware.Logger)
		   	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		   		w.Write([]byte("welcome"))
		   	})
		   	r.
		   	http.ListenAndServe(":3000", r)
	*/
	//api := handlers.NewApiController(natsstream, cmd, redisclient, sms)

	q := query.New(db)
	c := command.New(q, natsstream, sms)

	mux := chi.NewRouter()
	mux.Use(middleware.Logger)
	/*
		mux.Get("/chi", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("welcome"))
		})
		mux.Get("/chi/{id}", func(w http.ResponseWriter, r *http.Request) {
			log.Printf("r.URL.Path: %q", r.URL.Path[5:])
			log.Printf("chi.URLParam(id): %q", chi.URLParam(r, "id"))
			log.Printf("r.Query().Get(id)(id): %q", r.URL.Query().Get("id"))
			w.Write([]byte("welcome id"))
		})
		//mux := http.NewServeMux()
	*/
	mux.HandleFunc("/basicauth", basicauthHandler)
	/*
		mux.HandleFunc("/api/countdown", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]*time.Time{"countdown": q.SignupStart(types.TeamTypeKlan)})
		})
	*/
	mux.Get("/api/frontpage", handlers.Frontpage(q))
	mux.Get("/api/settings", handlers.ViewLimits(q))
	mux.Put("/api/settings", handlers.SaveLimits(c))
	mux.Get("/api/settings/mail/{slug}", handlers.MailTemplate(q))
	mux.Put("/api/settings/mail", handlers.SaveMailTemplate(c))
	mux.Post("/api/signup", handlers.SignUp(c))
	mux.Post("/api/confirm", handlers.Confirm(c))

	//mux.HandleFunc("/api/gogler", apiHandler(natsstreamw
	//mux.HandleFunc("/api/signup", api.ApiSignupHandler)
	//mux.HandleFunc("/api/patrulje/new", api.ApiPatruljeCreateHandler)
	mux.Get("/api/patrulje/{id}", handlers.ViewPatrulje(q))
	mux.Put("/api/patrulje", handlers.UpdatePatrulje(c))
	mux.Get("/api/klan/{id}", handlers.Klan(q))
	mux.Put("/api/klan", handlers.UpdateKlan(c))
	mux.Get("/api/checkout/{id}", handlers.Checkout(q, c))
	mux.Put("/api/mobilepay", handlers.Mobilepay(c))

	//mux.Get("/api/bridge/{id}", handlers.BridgeTeam(query.NewBridge(mdb)))

	//mux.HandleFunc("/api/team", api.ApiTeamUpdateHandler)
	//mux.HandleFunc("/api/team/", api.ApiTeamReadHandler)
	/*
		mux.HandleFunc("/api/senior/", func(w http.ResponseWriter, r *http.Request) {
			data, err := redisclient.Get("senior:" + r.URL.Path[12:]).Result()
			if err != nil && err != redis.Nil {
				log.Println(err)
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(data))
		})
	*/
	mux.Handle("/*", http.FileServer(http.Dir("/www")))
	//mux.Handle("/gøgler", http.FileServer(http.Dir("/www")))
	//mux.HandleFunc("/gøgler", IndexHandler("/www/index.html"))
	//	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
	//		w.Header().Set("Content-Type", "application/json")
	//		w.Write([]byte("{\"hello\": \"world\"}"))
	//	})

	fmt.Println("Running webserver")
	log.Fatal(http.ListenAndServe(":80", cors.Default().Handler(mux)))
}

//func publishToEventstream(signup Signup) {

func basicauthHandler(w http.ResponseWriter, r *http.Request) {

}

func IndexHandler(entrypoint string) func(w http.ResponseWriter, r *http.Request) {
	fn := func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, entrypoint)
	}
	return http.HandlerFunc(fn)
}
