package mobilepay

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"time"

	"github.com/moul/http2curl"
	"github.com/tdewolff/minify/v2"
	mjson "github.com/tdewolff/minify/v2/json"
)

var minifier *minify.M

func init() {
	minifier = minify.New()
	minifier.AddFuncRegexp(regexp.MustCompile("[/+]json$"), mjson.Minify)
}

// TransportLogger is an http.RoundTripper that logs requests and responses
type TransportLogger struct {
	Transport http.RoundTripper
	//logger    *slog.Logger
	printBody bool
}

// RoundTrip implements http.RoundTripper, and allows us to intercept
// all outbound requests using it
func (t *TransportLogger) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	logProps := make(map[string]any)
	start := time.Now()

	defer func() {
		end := time.Now()

		logProps["duration"] = slog.DurationValue(end.Sub(start))

		attrs := []slog.Attr{
			slog.String("method", req.Method),
			slog.String("uri", req.URL.String()),
		}

		for key, value := range logProps {
			attrs = append(attrs, slog.Any(key, value))
		}

		slog.Default().LogAttrs(ctx, slog.LevelInfo, "Outgoing request", attrs...)
	}()

	if req.Body != nil {
		logProps["hasBody"] = slog.BoolValue(true)
		logProps["content-type"] = slog.StringValue(req.Header.Get("Content-Type"))
	} else {
		logProps["hasBody"] = slog.BoolValue(false)
	}
	logProps["hasAuthHeader"] = slog.BoolValue(false)
	if req.Header.Get("Authorization") != "" {
		logProps["hasAuthHeader"] = slog.BoolValue(true)
	}

	resp, err := t.Transport.RoundTrip(req)
	if err != nil {
		logProps["error"] = slog.StringValue(err.Error())
		return resp, err
	}
	if resp == nil {
		return resp, err
	}

	logProps["responseCode"] = slog.IntValue(resp.StatusCode)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Add cURL command to log for debugging purposes
		cmd, err := http2curl.GetCurlCommand(req)
		if err == nil {
			logProps["curl"] = slog.StringValue(cmd.String())
		} else {
			logProps["curl"] = slog.StringValue("N/A")
		}
	}

	// If the body debugging is turned on, or this is an outgoing bad request, log the body so that we are able to fix
	// the issue.
	if t.printBody || resp.StatusCode == http.StatusBadRequest {
		body, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		resp.Body = io.NopCloser(bytes.NewReader(body))
		if readErr == nil && len(body) > 0 {
			if len(body) > 1000 {
				body = body[0:1000]
			}
			logProps["responseBody"] = slog.StringValue(pretty(req.Header.Get("Content-Type"), string(body)))
		}
	}

	return resp, err
}

// pretty exists to avoid logging sensitive data, and to make the output
// prettier in some cases
func pretty(mime, content string) string {
	var prettified string
	var err error

	switch mime {
	case "application/json":
		prettified, err = minifier.String(mime, content)
		if err != nil {
			prettified = content
		}
	case "application/html":
		prettified = fmt.Sprintf("<html response (%d bytes)>", len(content))
	default:
		prettified = content
	}
	return prettified
}
