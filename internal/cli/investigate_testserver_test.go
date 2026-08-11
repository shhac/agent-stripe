package cli

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// route is what a stub path returns. Modelled as a struct because the previous
// string encoding smuggled three things through one type — a JSON body, ""
// meaning "empty list", and a sentinel-prefixed error — so a fixture body could
// in principle be misread as a fault.
type route struct {
	body   string
	status int
	code   string
}

func jsonRoute(body string) route { return route{body: body} }

func failingRoute(status int, code string) route { return route{status: status, code: code} }

// routeServer builds a stub Stripe from a path table. An unexpected path still
// fails loudly, which is the property the hand-rolled switches had.
func routeServer(t *testing.T, routes map[string]route) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		spec, ok := routes[r.URL.Path]
		if !ok {
			t.Errorf("unexpected request path %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"error":{"type":"invalid_request_error","code":"resource_missing"}}`)
			return
		}
		if spec.status != 0 {
			w.WriteHeader(spec.status)
			fmt.Fprintf(w, `{"error":{"type":"invalid_request_error","code":%q,"message":"injected"}}`, spec.code)
			return
		}
		fmt.Fprint(w, spec.body)
	}))
	t.Cleanup(server.Close)
	return server
}
