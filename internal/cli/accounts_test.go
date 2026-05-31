package cli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAccountsListDefaultsToCompactSummaries(t *testing.T) {
	h := newCLITestHarness(t)
	server := newAccountsListServer(t)
	defer server.Close()

	stdout, stderr := h.run("--api-key", "sk_test_123", "--base-url", server.URL, "accounts", "list", "--limit", "1")

	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	item := firstNDJSONItem(t, stdout)
	if item["id"] != "acct_123" || item["object"] != "account" {
		t.Fatalf("unexpected account identity in %s", stdout)
	}
	for _, key := range []string{"email", "business_profile", "company", "individual", "external_accounts", "settings"} {
		if _, ok := item[key]; ok {
			t.Fatalf("compact account summary exposed %q in %s", key, stdout)
		}
	}
	requirements, ok := item["requirements"].(map[string]any)
	if !ok {
		t.Fatalf("requirements summary missing or wrong type: %#v", item["requirements"])
	}
	if requirements["currently_due_count"] != float64(2) || requirements["past_due_count"] != float64(1) {
		t.Fatalf("requirements summary = %#v", requirements)
	}
	capabilities, ok := item["capabilities"].(map[string]any)
	if !ok {
		t.Fatalf("capabilities summary missing or wrong type: %#v", item["capabilities"])
	}
	if capabilities["active_count"] != float64(1) || capabilities["pending_count"] != float64(1) {
		t.Fatalf("capabilities summary = %#v", capabilities)
	}
}

func TestAccountsListFullReturnsRawStripeObjects(t *testing.T) {
	h := newCLITestHarness(t)
	server := newAccountsListServer(t)
	defer server.Close()

	stdout, stderr := h.run("--api-key", "sk_test_123", "--base-url", server.URL, "accounts", "list", "--limit", "1", "--full")

	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	item := firstNDJSONItem(t, stdout)
	if _, ok := item["business_profile"]; !ok {
		t.Fatalf("--full output omitted business_profile: %s", stdout)
	}
	if _, ok := item["external_accounts"]; !ok {
		t.Fatalf("--full output omitted external_accounts: %s", stdout)
	}
	if !strings.Contains(stdout, "@redacted") {
		t.Fatalf("--full output should still use default redaction: %s", stdout)
	}
}

func TestAccountsListCompactHonorsJSONFormat(t *testing.T) {
	h := newCLITestHarness(t)
	server := newAccountsListServer(t)
	defer server.Close()

	stdout, stderr := h.run("--api-key", "sk_test_123", "--base-url", server.URL, "--format", "json", "accounts", "list", "--limit", "1")

	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	var out struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("output is not JSON list: %v\n%s", err, stdout)
	}
	if len(out.Data) != 1 || out.Data[0]["id"] != "acct_123" {
		t.Fatalf("unexpected JSON data: %#v", out.Data)
	}
}

func newAccountsListServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/accounts" {
			t.Fatalf("path = %s, want /v1/accounts", r.URL.Path)
		}
		if got := r.URL.Query().Get("limit"); got != "1" {
			t.Fatalf("limit = %q, want 1", got)
		}
		w.Header().Set("Request-Id", "req_test")
		_, _ = w.Write([]byte(`{
			"object": "list",
			"has_more": false,
			"data": [{
				"id": "acct_123",
				"object": "account",
				"type": "express",
				"business_type": "company",
				"country": "GB",
				"default_currency": "gbp",
				"created": 1700000000,
				"charges_enabled": true,
				"payouts_enabled": false,
				"details_submitted": true,
				"email": "owner@example.test",
				"business_profile": {"name": "Example Ltd", "url": "https://example.test"},
				"company": {"name": "Example Ltd"},
				"individual": {"first_name": "Ada", "last_name": "Lovelace"},
				"external_accounts": {"object": "list", "data": [{"id": "ba_123", "last4": "6789"}]},
				"settings": {"payouts": {"schedule": {"interval": "daily"}}},
				"controller": {
					"type": "application",
					"requirement_collection": "stripe",
					"stripe_dashboard": {"type": "express"}
				},
				"requirements": {
					"disabled_reason": "requirements.past_due",
					"currently_due": ["external_account", "company.owners_provided"],
					"past_due": ["external_account"],
					"pending_verification": []
				},
				"future_requirements": {
					"currently_due": ["company.verification.document"],
					"eventually_due": ["company.tax_id"]
				},
				"capabilities": {
					"card_payments": "active",
					"transfers": "pending",
					"tax_reporting_us_1099_k": "inactive"
				}
			}]
		}`))
	}))
}

func firstNDJSONItem(t *testing.T, raw string) map[string]any {
	t.Helper()
	line := strings.Split(strings.TrimSpace(raw), "\n")[0]
	var item map[string]any
	if err := json.Unmarshal([]byte(line), &item); err != nil {
		t.Fatalf("line is not JSON: %v\n%s", err, line)
	}
	return item
}
