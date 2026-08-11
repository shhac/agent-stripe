package cli

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// routeServer builds a stub Stripe from a path→body table. Thirteen tests were
// hand-rolling the same switch-with-a-fatal-default; a table reads as the
// fixture it is, and an unexpected path still fails loudly.
//
// A body of "" means the route exists and returns an empty list; use
// failingRoute to make a path return a Stripe-shaped error.
func routeServer(t *testing.T, routes map[string]string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := routes[r.URL.Path]
		if !ok {
			t.Errorf("unexpected request path %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"error":{"type":"invalid_request_error","code":"resource_missing"}}`)
			return
		}
		if status, errBody, isFailure := parseFailingRoute(body); isFailure {
			w.WriteHeader(status)
			fmt.Fprint(w, errBody)
			return
		}
		if body == "" {
			fmt.Fprint(w, emptyList)
			return
		}
		fmt.Fprint(w, body)
	}))
	t.Cleanup(server.Close)
	return server
}

const emptyList = `{"object":"list","data":[],"has_more":false}`

const failPrefix = "\x00fail\x00"

// failingRoute marks a route as returning a Stripe error with the given status
// and code.
func failingRoute(status int, code string) string {
	return fmt.Sprintf("%s%d %s", failPrefix, status, code)
}

func parseFailingRoute(body string) (int, string, bool) {
	if len(body) < len(failPrefix) || body[:len(failPrefix)] != failPrefix {
		return 0, "", false
	}
	var status int
	var code string
	_, _ = fmt.Sscanf(body[len(failPrefix):], "%d %s", &status, &code)
	return status, fmt.Sprintf(`{"error":{"type":"invalid_request_error","code":%q,"message":"injected"}}`, code), true
}
