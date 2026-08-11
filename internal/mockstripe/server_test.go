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

func TestServerListPaginationCursors(t *testing.T) {
	server := httptest.NewServer(NewServer())
	defer server.Close()

	tests := []struct {
		name    string
		query   string
		wantIDs []string
		hasMore bool
	}{
		{name: "limit has more", query: "limit=1", wantIDs: []string{"cus_mock_123"}, hasMore: true},
		{name: "starting after", query: "starting_after=cus_mock_123&limit=1", wantIDs: []string{"cus_mock_456"}, hasMore: true},
		{name: "ending before", query: "ending_before=cus_mock_456", wantIDs: []string{"cus_mock_123"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, server.URL+"/v1/customers?"+tt.query, nil)
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
				HasMore bool             `json:"has_more"`
				Data    []map[string]any `json:"data"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
				t.Fatalf("Decode() error = %v", err)
			}
			if body.HasMore != tt.hasMore {
				t.Fatalf("has_more = %t, want %t", body.HasMore, tt.hasMore)
			}
			if len(body.Data) != len(tt.wantIDs) {
				t.Fatalf("len(data) = %d, want %d: %#v", len(body.Data), len(tt.wantIDs), body.Data)
			}
			for idx, want := range tt.wantIDs {
				if body.Data[idx]["id"] != want {
					t.Fatalf("data[%d].id = %v, want %s", idx, body.Data[idx]["id"], want)
				}
			}
		})
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

func TestFaultInjectionFailsMatchingPath(t *testing.T) {
	server := httptest.NewServer(NewServer())
	defer server.Close()

	req, _ := http.NewRequest(http.MethodGet, server.URL+"/v1/charges", nil)
	req.SetBasicAuth("sk_test_mock", "")
	req.Header.Set("X-Mock-Fault", "/v1/charges=500")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", resp.StatusCode)
	}

	// A path the rule does not name is unaffected.
	other, _ := http.NewRequest(http.MethodGet, server.URL+"/v1/refunds", nil)
	other.SetBasicAuth("sk_test_mock", "")
	other.Header.Set("X-Mock-Fault", "/v1/charges=500")
	resp2, err := http.DefaultClient.Do(other)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("unrelated path status = %d, want 200", resp2.StatusCode)
	}
}

func TestFaultInjectionCountsRepeats(t *testing.T) {
	server := httptest.NewServer(NewServer())
	defer server.Close()

	statuses := []int{}
	for range 3 {
		req, _ := http.NewRequest(http.MethodGet, server.URL+"/v1/charges", nil)
		req.SetBasicAuth("sk_test_mock", "")
		req.Header.Set("X-Mock-Fault", "/v1/charges=429x2")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request error: %v", err)
		}
		statuses = append(statuses, resp.StatusCode)
		resp.Body.Close()
	}
	want := []int{429, 429, 200}
	for idx, status := range statuses {
		if status != want[idx] {
			t.Fatalf("statuses = %v, want %v", statuses, want)
		}
	}
}

func TestInvoiceListScopesByCustomer(t *testing.T) {
	// The hand-written invoices handler ignored `customer`, so a test asserting
	// per-customer scoping would have passed against every fixture invoice.
	server := httptest.NewServer(NewServer())
	defer server.Close()

	req, _ := http.NewRequest(http.MethodGet, server.URL+"/v1/invoices?customer=cus_nobody", nil)
	req.SetBasicAuth("sk_test_mock", "")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	defer resp.Body.Close()

	var list struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(list.Data) != 0 {
		t.Fatalf("invoices for an unknown customer = %d, want none", len(list.Data))
	}
}
