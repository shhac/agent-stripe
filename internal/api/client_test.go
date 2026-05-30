package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

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

	err := classifyHTTPError(404, "req_123", "", 0, body)
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

func TestGetRetriesStripeRateLimitThenSucceeds(t *testing.T) {
	withFastRetry(t)
	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt := atomic.AddInt32(&requests, 1)
		if attempt == 1 {
			w.Header().Set("Stripe-Rate-Limited-Reason", "endpoint-rate")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":{"message":"too many requests","code":"rate_limit"}}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"pi_retry","object":"payment_intent"}`))
	}))
	defer server.Close()

	client := NewClient(Options{APIKey: "sk_test_123", BaseURL: server.URL, MaxRetries: 2})
	raw, err := client.Get(t.Context(), "/v1/payment_intents/pi_retry", nil)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !strings.Contains(string(raw), "pi_retry") {
		t.Fatalf("raw = %s, want pi_retry", raw)
	}
	if got := atomic.LoadInt32(&requests); got != 2 {
		t.Fatalf("requests = %d, want 2", got)
	}
}

func TestGetStopsAfterRateLimitRetries(t *testing.T) {
	withFastRetry(t)
	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		w.Header().Set("Request-Id", "req_rate_limit")
		w.Header().Set("Stripe-Rate-Limited-Reason", "global-rate")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"too many requests","code":"rate_limit"}}`))
	}))
	defer server.Close()

	client := NewClient(Options{APIKey: "sk_test_123", BaseURL: server.URL, MaxRetries: 2})
	_, err := client.Get(t.Context(), "/v1/events", nil)
	var apiErr *agenterrors.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %#v, want APIError", err)
	}
	if apiErr.FixableBy != agenterrors.FixableByRetry {
		t.Fatalf("FixableBy = %q, want retry", apiErr.FixableBy)
	}
	if !strings.Contains(apiErr.Hint, "rate limit reason: global-rate") {
		t.Fatalf("Hint = %q, want rate limit reason", apiErr.Hint)
	}
	if !strings.Contains(apiErr.Hint, "Retried 2 time(s)") {
		t.Fatalf("Hint = %q, want retry count", apiErr.Hint)
	}
	if got := atomic.LoadInt32(&requests); got != 3 {
		t.Fatalf("requests = %d, want initial + 2 retries", got)
	}
}

func TestPostFormRetriesPreviewBody(t *testing.T) {
	withFastRetry(t)
	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm() error = %v", err)
		}
		if got := r.Form.Get("subscription"); got != "sub_123" {
			t.Fatalf("subscription = %q, want sub_123", got)
		}
		attempt := atomic.AddInt32(&requests, 1)
		if attempt == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":{"message":"lock timeout","code":"lock_timeout"}}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"upcoming_in_123","object":"invoice"}`))
	}))
	defer server.Close()

	client := NewClient(Options{APIKey: "sk_test_123", BaseURL: server.URL, MaxRetries: 1})
	raw, err := client.PostForm(t.Context(), "/v1/invoices/create_preview", url.Values{"subscription": []string{"sub_123"}})
	if err != nil {
		t.Fatalf("PostForm() error = %v", err)
	}
	if !strings.Contains(string(raw), "upcoming_in_123") {
		t.Fatalf("raw = %s, want upcoming invoice", raw)
	}
	if got := atomic.LoadInt32(&requests); got != 2 {
		t.Fatalf("requests = %d, want 2", got)
	}
}

func withFastRetry(t *testing.T) {
	t.Helper()
	previous := retryBaseDelay
	retryBaseDelay = time.Millisecond
	t.Cleanup(func() { retryBaseDelay = previous })
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

func TestGetSendsAuthContextVersionAndQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if got := r.URL.Query().Get("limit"); got != "3" {
			t.Fatalf("limit query = %q, want 3", got)
		}
		if got := r.Header.Get("Stripe-Context"); got != "acct_123" {
			t.Fatalf("Stripe-Context = %q, want acct_123", got)
		}
		if got := r.Header.Get("Stripe-Version"); got != "2025-10-29.clover" {
			t.Fatalf("Stripe-Version = %q, want 2025-10-29.clover", got)
		}
		if got := r.Header.Get("Authorization"); got != "Basic c2tfdGVzdF8xMjM6" {
			t.Fatalf("Authorization = %q, want encoded basic auth", got)
		}
		if got := r.Header.Get("Content-Type"); got != "" {
			t.Fatalf("Content-Type = %q, want empty for GET", got)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"pi_123","object":"payment_intent"}`))
	}))
	defer server.Close()

	client := NewClient(Options{
		APIKey:     "sk_test_123",
		Context:    "acct_123",
		APIVersion: "2025-10-29.clover",
		BaseURL:    server.URL,
	})
	raw, err := client.Get(t.Context(), "/v1/payment_intents/pi_123", url.Values{"limit": []string{"3"}})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if string(raw) == "" {
		t.Fatalf("Get() returned empty body")
	}
}
