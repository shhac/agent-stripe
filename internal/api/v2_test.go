package api

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestV2RequestsUseBearerAuthAndV2Version(t *testing.T) {
	var seen []*http.Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Clone(r.Context()))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()

	client := NewClient(Options{
		APIKey:       "sk_test_123",
		Context:      "acct_platform/acct_connected",
		APIVersion:   "2025-06-30.basil",
		V2APIVersion: "2026-07-29.dahlia",
		BaseURL:      server.URL,
	})
	if _, err := client.Get(context.Background(), "/v2/core/accounts", url.Values{}); err != nil {
		t.Fatalf("Get(/v2) error = %v", err)
	}
	if _, err := client.Get(context.Background(), "/v1/accounts", url.Values{}); err != nil {
		t.Fatalf("Get(/v1) error = %v", err)
	}
	if len(seen) != 2 {
		t.Fatalf("len(requests) = %d, want 2", len(seen))
	}

	v2, v1 := seen[0], seen[1]
	if got := v2.Header.Get("Authorization"); got != "Bearer sk_test_123" {
		t.Fatalf("v2 Authorization = %q, want Bearer", got)
	}
	if got := v2.Header.Get("Stripe-Version"); got != "2026-07-29.dahlia" {
		t.Fatalf("v2 Stripe-Version = %q, want the v2 train", got)
	}
	if got := v2.Header.Get("Stripe-Context"); got != "acct_platform/acct_connected" {
		t.Fatalf("v2 Stripe-Context = %q, want the context to still apply", got)
	}
	if got := v1.Header.Get("Authorization"); got != "Basic c2tfdGVzdF8xMjM6" {
		t.Fatalf("v1 Authorization = %q, want Basic", got)
	}
	if got := v1.Header.Get("Stripe-Version"); got != "2025-06-30.basil" {
		t.Fatalf("v1 Stripe-Version = %q, want the v1 train", got)
	}
}

func TestIsV2Path(t *testing.T) {
	for path, want := range map[string]bool{
		"/v2/core/accounts":   true,
		"/v2/core/events?a=b": true,
		"/v1/accounts":        false,
		"/v2beta/thing":       false,
	} {
		if got := IsV2Path(path); got != want {
			t.Fatalf("IsV2Path(%q) = %t, want %t", path, got, want)
		}
	}
}

func TestDecodeV2ListExtractsPageToken(t *testing.T) {
	raw := json.RawMessage(`{"data":[{"id":"acct_1"},{"id":"acct_2"}],` +
		`"next_page_url":"/v2/core/accounts?page=page_abc123&limit=2","previous_page_url":null}`)
	list, err := DecodeV2List(raw)
	if err != nil {
		t.Fatalf("DecodeV2List() error = %v", err)
	}
	if len(list.Data) != 2 {
		t.Fatalf("len(Data) = %d, want 2", len(list.Data))
	}
	if !list.HasMore() {
		t.Fatalf("HasMore() = false, want true when next_page_url is set")
	}
	if got := list.NextPageToken(); got != "page_abc123" {
		t.Fatalf("NextPageToken() = %q, want page_abc123", got)
	}
	if list.PreviousPageURL != "" {
		t.Fatalf("PreviousPageURL = %q, want empty", list.PreviousPageURL)
	}
}

func TestDecodeV2ListWithoutNextPage(t *testing.T) {
	list, err := DecodeV2List(json.RawMessage(`{"data":[],"next_page_url":null}`))
	if err != nil {
		t.Fatalf("DecodeV2List() error = %v", err)
	}
	if list.HasMore() {
		t.Fatalf("HasMore() = true, want false")
	}
}

func TestClassifyHTTPErrorExposesStripeCodeForNamespaceFallback(t *testing.T) {
	body := []byte(`{"error":{"type":"invalid_request_error","code":"v1_account_instead_of_v2_account","message":"V1 Account ID cannot be used in V2 Account APIs."}}`)
	err := classifyHTTPError(httpErrorInput{status: 400, body: body, v2: true})

	if got := ErrorCode(err); got != "v1_account_instead_of_v2_account" {
		t.Fatalf("ErrorCode() = %q", got)
	}
	if got := ErrorStatus(err); got != 400 {
		t.Fatalf("ErrorStatus() = %d, want 400", got)
	}
	var stripeErr *StripeError
	if !stderrors.As(err, &stripeErr) || stripeErr.Type != "invalid_request_error" {
		t.Fatalf("error should carry a StripeError cause, got %#v", err)
	}
	if got := err.Hint; got == "" || !containsAll(got, "accounts get", "--namespace v1") {
		t.Fatalf("Hint = %q, want the v1 command suggestion", got)
	}
}

func TestClassifyHTTPErrorHintsPreviewVersionOnV2BadRequest(t *testing.T) {
	err := classifyHTTPError(httpErrorInput{
		status: 400,
		body:   []byte(`{"error":{"type":"invalid_request_error","message":"Unknown field"}}`),
		v2:     true,
	})
	if !containsAll(err.Hint, "--v2-api-version") {
		t.Fatalf("Hint = %q, want a version hint for a v2 400", err.Hint)
	}
}

func TestErrorCodeIgnoresNonStripeErrors(t *testing.T) {
	if got := ErrorCode(stderrors.New("boom")); got != "" {
		t.Fatalf("ErrorCode() = %q, want empty", got)
	}
	if got := ErrorStatus(nil); got != 0 {
		t.Fatalf("ErrorStatus(nil) = %d, want 0", got)
	}
}

func containsAll(value string, wants ...string) bool {
	for _, want := range wants {
		if !strings.Contains(value, want) {
			return false
		}
	}
	return true
}
