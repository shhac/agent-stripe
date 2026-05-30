package mockstripe

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServerListsFilteredEvents(t *testing.T) {
	server := httptest.NewServer(NewServer())
	defer server.Close()

	req, err := http.NewRequest(http.MethodGet, server.URL+"/v1/events?type=charge.failed", nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("sk_test_mock:")))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}

	var body struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if len(body.Data) != 1 {
		t.Fatalf("len(data) = %d, want 1", len(body.Data))
	}
	if body.Data[0]["type"] != "charge.failed" {
		t.Fatalf("event type = %v", body.Data[0]["type"])
	}
}

func TestServerRejectsMissingAuth(t *testing.T) {
	server := httptest.NewServer(NewServer())
	defer server.Close()

	resp, err := http.Get(server.URL + "/v1/balance")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestServerIndexExposesRouteMapWithoutAuth(t *testing.T) {
	server := httptest.NewServer(NewServer())
	defer server.Close()

	resp, err := http.Get(server.URL + "/")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	var body struct {
		Service string   `json:"service"`
		Routes  []string `json:"routes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if body.Service != "mockstripe" {
		t.Fatalf("service = %q, want mockstripe", body.Service)
	}
	if len(body.Routes) == 0 {
		t.Fatal("expected route map")
	}
}

func TestRoutesIncludesDescriptorResources(t *testing.T) {
	routes := Routes()
	for _, want := range []string{
		"GET  /v1/payment_intents",
		"GET  /v1/payment_intents/search",
		"GET  /v1/payment_intents/<id>",
		"GET  /v1/radar/early_fraud_warnings/<id>",
	} {
		if !hasRoute(routes, want) {
			t.Fatalf("Routes() missing %q in %#v", want, routes)
		}
	}
}

func hasRoute(routes []string, want string) bool {
	for _, route := range routes {
		if route == want {
			return true
		}
	}
	return false
}
