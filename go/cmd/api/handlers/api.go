package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/xeipuuv/gojsonschema"

	//"net/http/httputil"

	"nathejk.dk/pkg/streaminterface"
)

type PostGogler struct {
	Name       string `json:"name"`
	Phone      string `json:"phone"`
	Email      string `json:"email"`
	Address    string `json:"address"`
	PostalCode string `json:"postalCode"`
	Group      string `json:"group"`
	Friday     bool   `json:"friday"`
	Saturday   bool   `json:"saturday"`
	Sunday     bool   `json:"sunday"`
	Photo      bool   `json:"photo"`
	Comment    string `json:"comment"`
}

func apiHandler(publisher streaminterface.Publisher) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.Method + ":" + r.URL.Path {
		case "POST:/api/gogler":
			schema := gojsonschema.NewReferenceLoader("file:///app/schema/gogler.json")

			body, _ := ioutil.ReadAll(r.Body)
			document := gojsonschema.NewStringLoader(string(body))
			result, err := gojsonschema.Validate(schema, document)
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}

			if result.Valid() {
				var u PostGogler
				json.Unmarshal(body, &u)
				msg := publisher.MessageFunc()(streaminterface.SubjectFromStr("nathejk:gglr.signedup"))
				//msg.Msg().Type = "gglr.signedup"
				msg.SetBody(u)
				publisher.Publish(msg)

				url := "https://api.cpsms.dk/v2/send"
				fmt.Println("URL:>", url)

				text := "https://www.mobilepay.dk/erhverv/betalingslink/betalingslink-svar?phone=775771&amount=50&comment=" + u.Phone + "&lock=1"
				values := map[string]string{"to": "45" + u.Phone, "from": "Nathejk", "message": text}

				jsonStr, _ := json.Marshal(values)
				req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
				req.Header.Set("Authorization", "Basic "+os.Getenv("CPSMS_API_KEY"))
				req.Header.Set("Content-Type", "application/json")

				client := &http.Client{}
				resp, err := client.Do(req)
				if err != nil {
					panic(err)
				}
				defer resp.Body.Close()

				fmt.Fprint(w, "\"OK\"")
			} else {
				log.Printf("Malformed gøgler:\n")
				for _, desc := range result.Errors() {
					log.Printf("- %s\n", desc)
				}
				http.Error(w, "malformed gglr", 400)
				return
			}
		default:
			fmt.Fprintf(w, "API route [%s]", r.Method+":"+r.URL.Path)
		}
	}
}
