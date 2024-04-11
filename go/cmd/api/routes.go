package main

import (
	"expvar"
	"net/http"
	"os"

	"github.com/julienschmidt/httprouter"
)

func (app *application) routes() http.Handler {
	router := httprouter.New()

	router.NotFound = http.HandlerFunc(app.NotFoundResponse)
	router.MethodNotAllowed = http.HandlerFunc(app.MethodNotAllowedResponse)
	/*
		router.HandlerFunc(http.MethodPut, "/api/*filepath", app.cleo.ProxyHandler)
		router.HandlerFunc(http.MethodGet, "/api/*filepath", app.cleo.ProxyHandler)
		router.HandlerFunc(http.MethodPost, "/api/*filepath", app.cleo.ProxyHandler)
		router.HandlerFunc(http.MethodDelete, "/api/*filepath", app.cleo.ProxyHandler)
		router.HandlerFunc(http.MethodPatch, "/api/*filepath", app.cleo.ProxyHandler)
	*/
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(SpaFileSystem(http.Dir(app.config.webroot))))
	mux.HandleFunc("/api/v1/healthcheck", app.HealthcheckHandler)
	mux.Handle("/api/", app.Metrics(router))
	mux.Handle("/debug/vars", expvar.Handler())

	return mux
}

type spaFileSystem struct {
	root http.FileSystem
}

func (fs *spaFileSystem) Open(name string) (http.File, error) {
	f, err := fs.root.Open(name)
	if os.IsNotExist(err) {
		return fs.root.Open("index.html")
	}
	return f, err
}
func SpaFileSystem(fs http.FileSystem) *spaFileSystem {
	return &spaFileSystem{root: fs}
}
