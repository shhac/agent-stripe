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
