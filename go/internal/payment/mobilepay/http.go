package mobilepay

import (
	"net/http"
	"time"
)

func httpClient() *http.Client {
	return &http.Client{
		Transport: &TransportLogger{
			Transport: &http.Transport{
				MaxIdleConns:        0,
				MaxIdleConnsPerHost: 0,
				MaxConnsPerHost:     50,
				// DefaultClient behaviour of reading proxy configuration from the environment.
				Proxy: http.ProxyFromEnvironment,
			},
			//logger:    c.logger,
			//printBody: c.logBody,
		},
		Timeout: 60 * time.Second,
	}
}

/*
func defaultHeaders() http.Header {
	return http.Header{
		"User-Agent":   []string{"Nathejk"},
		"Content-Type": []string{"application/json"},
		"Accept":       []string{"application/json"},
	}
}

/*
type transport struct {
    headers map[string]string
    base    http.RoundTripper
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
    for k, v := range t.headers {
        req.Header.Add(k, v)
    }
    base := t.base
    if base == nil {
        base = http.DefaultTransport
    }
    return base.RoundTrip(req)
}

func main() {
    cli := &http.Client{
        Transport: &transport{
            headers: map[string]string{
                "X-Test": "true",
            },
        },
    }
    rsp, err := cli.Get("http://localhost:8080")
    defer rsp.Body.Close()
    if err != nil {
        panic(err)
    }
}
*/
