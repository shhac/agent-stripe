package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	agenterrors "github.com/shhac/agent-stripe/internal/errors"
)

func TestDecodeList(t *testing.T) {
	raw := json.RawMessage(`{"object":"list","has_more":true,"data":[{"id":"evt_1"}]}`)

	list, err := DecodeList(raw)
	if err != nil {
		t.Fatalf("DecodeList() error = %v", err)
	}
	if !list.HasMore {
		t.Fatalf("HasMore = false, want true")
	}
	if len(list.Data) != 1 {
		t.Fatalf("len(Data) = %d, want 1", len(list.Data))
	}
}

func TestClassifyHTTPErrorIncludesStripeHints(t *testing.T) {
	body := []byte(`{"error":{"message":"No such payment_intent","code":"resource_missing","request_log_url":"https://dashboard.stripe.com/test/logs/req_123"}}`)

	err := classifyHTTPError(404, "req_123", body)
	if err.FixableBy != agenterrors.FixableByAgent {
		t.Fatalf("FixableBy = %q", err.FixableBy)
	}
	if err.Hint == "" {
		t.Fatalf("Hint should be populated")
	}
	if err.Message == "" || err.Message == "HTTP 404" {
		t.Fatalf("Message = %q", err.Message)
	}
}

func TestAddHelpers(t *testing.T) {
	values := url.Values{}
	AddLimit(values, 12)
	AddCreatedRange(values, "100", "200")
	AddExpand(values, []string{"latest_charge", "customer"})

	if values.Get("limit") != "12" {
		t.Fatalf("limit = %q", values.Get("limit"))
	}
	if values.Get("created[gte]") != "100" || values.Get("created[lte]") != "200" {
		t.Fatalf("created range not set: %v", values)
	}
	if got := values["expand[]"]; len(got) != 2 {
		t.Fatalf("expand[] = %v", got)
	}
}

func TestPostFormSendsFormBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if got := r.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
			t.Fatalf("Content-Type = %q", got)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm() error = %v", err)
		}
		if got := r.Form.Get("subscription"); got != "sub_123" {
			t.Fatalf("subscription = %q", got)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"upcoming_in_123","object":"invoice"}`))
	}))
	defer server.Close()

	client := NewClient(Options{APIKey: "sk_test_123", BaseURL: server.URL})
	raw, err := client.PostForm(t.Context(), "/v1/invoices/create_preview", url.Values{"subscription": []string{"sub_123"}})
	if err != nil {
		t.Fatalf("PostForm() error = %v", err)
	}
	if string(raw) == "" {
		t.Fatalf("PostForm() returned empty body")
	}
}
