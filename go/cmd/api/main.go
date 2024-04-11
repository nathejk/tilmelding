package main

import (
	"expvar"
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"

	"nathejk.dk/cmd/api/app"
	"nathejk.dk/internal/jsonlog"
	"nathejk.dk/internal/vcs"
)

var (
	version = vcs.Version()
)

// Define a config struct to hold all the configuration settings for our application.
type config struct {
	port    int
	webroot string
}

type application struct {
	app.JsonApi

	config config
	logger *jsonlog.Logger
}

func main() {
	var cfg config

	flag.IntVar(&cfg.port, "port", 80, "API server port")

	flag.StringVar(&cfg.webroot, "webroot", getEnv("WEBROOT", "/www"), "Static web root")

	flag.Parse()

	//logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)
	logger := jsonlog.New(os.Stdout, jsonlog.LevelInfo)

	expvar.NewString("version").Set(version)
	expvar.NewInt("timestamp").Set(time.Now().Unix())
	expvar.NewInt("goroutines").Set(int64(runtime.NumGoroutine()))

	app := &application{
		JsonApi: app.JsonApi{
			Logger: logger,
		},
		config: cfg,
		logger: logger,
	}

	logger.PrintFatal(app.Serve(fmt.Sprintf(":%d", cfg.port), app.routes()), nil)
}
