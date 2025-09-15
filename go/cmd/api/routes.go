package main

import (
	"expvar"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	jsonapi "nathejk.dk/cmd/api/app"
	"nathejk.dk/internal/payment/mobilepay"
)

func (app *application) routes() http.Handler {
	router := httprouter.New()

	router.NotFound = http.HandlerFunc(app.NotFoundResponse)
	router.MethodNotAllowed = http.HandlerFunc(app.MethodNotAllowedResponse)

	router.HandlerFunc(http.MethodGet, "/api/home", app.homeHandler)
	router.HandlerFunc(http.MethodPost, "/api/signup", app.signupHandler)
	router.HandlerFunc(http.MethodPost, "/api/signup/pincode", app.signupPincodeHandler)
	router.HandlerFunc(http.MethodGet, "/api/signup/:id", app.showSignupHandler)
	router.HandlerFunc(http.MethodGet, "/api/patrulje/:id", app.showPatruljeHandler)
	router.HandlerFunc(http.MethodPut, "/api/patrulje/:id", app.updatePatruljeHandler)
	router.HandlerFunc(http.MethodGet, "/api/klan/:id", app.showKlanHandler)
	router.HandlerFunc(http.MethodPut, "/api/klan/:id", app.updateKlanHandler)
	router.HandlerFunc(http.MethodGet, "/api/personnel/:id", app.showPersonnelHandler)
	router.HandlerFunc(http.MethodPut, "/api/personnel/:id", app.updatePersonnelHandler)
	router.HandlerFunc(http.MethodPut, "/api/pay/:id", app.sendMobilepaySmsHandler)
	router.HandlerFunc(http.MethodGet, "/api/payment/:ref", app.showPaymentHandler)
	router.HandlerFunc(http.MethodGet, "/api/assignnumbers", app.assignNumberHandler)

	callback := httprouter.New()
	callback.NotFound = http.HandlerFunc(app.NotFoundResponse)
	callback.MethodNotAllowed = http.HandlerFunc(app.MethodNotAllowedResponse)
	callback.HandlerFunc(http.MethodGet, "/callback/mobilepay/:ref", app.mobilepayCallbackHandler)
	callback.HandlerFunc(http.MethodGet, "/callback/email/:id", app.confirmSignupHandler)

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
	mux.Handle("/callback/", callback)
	mux.Handle("/debug/vars", expvar.Handler())

	mux.HandleFunc("/mobilepay", func(w http.ResponseWriter, r *http.Request) {
		key := uuid.New().String()
		payment, err := app.payment.CreatePayment(key, mobilepay.Payment{
			Customer:           mobilepay.Customer{PhoneNumber: "4540733886"},
			Amount:             mobilepay.Amount{Currency: mobilepay.CurrencyDKK, Value: 1000},
			PaymentMethod:      mobilepay.PaymentMethod{Type: "WALLET"},
			PaymentDescription: "Nathejk tilmelding",
			Reference:          mobilepay.PaymentReference(key),
			UserFlow:           "WEB_REDIRECT",
			ReturnUrl:          "https://nathejk.dk",
			Receipt: mobilepay.Receipt{
				OrderLines: []mobilepay.OrderLine{},
				BottomLine: mobilepay.BottomLine{
					Currency: mobilepay.CurrencyDKK,
				},
			},
		})
		if err != nil {
			app.ServerErrorResponse(w, r, err)
		}
		env := jsonapi.Envelope{
			"payment": payment,
		}
		err = app.WriteJSON(w, http.StatusOK, env, nil)
		if err != nil {
			app.ServerErrorResponse(w, r, err)
		}
	})
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
