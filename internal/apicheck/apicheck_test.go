//go:build apicheck

// Package apicheck validates the requests the CLI actually emits against
// Stripe's published OpenAPI spec: every path must exist, and every query
// parameter must be one the endpoint declares. It catches a mistyped filter or
// a wrong path without a live API call, which unit tests against our own mock
// cannot — the mock was written from the same reading of the docs as the code.
//
// Guarded by a build tag because it needs the ~8MB spec:
//
//	make apicheck
//
// The /v2 namespace is deliberately not in Stripe's published spec files, so
// those requests are reported for review rather than validated. That gap is the
// reason the v2 defaults are documented as doc-derived.
package apicheck

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/shhac/agent-stripe/internal/cli"
	"github.com/shhac/agent-stripe/internal/mockstripe"
	"github.com/shhac/agent-stripe/internal/output"
)

type recordedRequest struct {
	method string
	path   string
	params []string
	from   string
}

// recorder wraps the mock so the CLI runs its real code paths — including the
// chained lookups inside investigations — while every request is captured.
type recorder struct {
	inner    http.Handler
	requests []recordedRequest
	command  string
}

func (r *recorder) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	params := make([]string, 0, len(req.URL.Query()))
	for key := range req.URL.Query() {
		params = append(params, key)
	}
	sort.Strings(params)
	r.requests = append(r.requests, recordedRequest{
		method: req.Method,
		path:   req.URL.Path,
		params: params,
		from:   r.command,
	})
	r.inner.ServeHTTP(w, req)
}

// commands covers every command that reaches Stripe. Adding a command without
// adding it here means its requests go unvalidated, so keep this exhaustive.
func commands() [][]string {
	return [][]string{
		{"balance", "get"},
		{"accounts", "self"},
		{"accounts", "list"},
		{"accounts", "get", "acct_mock_connected"},
		{"accounts", "persons", "list", "acct_mock_connected", "--relationship", "owner"},
		{"accounts", "persons", "get", "acct_mock_connected", "person_mock_v1_rep"},
		{"accounts", "capabilities", "acct_mock_connected"},
		{"accounts", "external-accounts", "acct_mock_connected", "--object", "bank_account"},
		{"customers", "list", "--email", "buyer@example.com"},
		{"customers", "get", "cus_mock_123"},
		{"customers", "search", "--query", "email:'buyer@example.com'"},
		{"events", "list", "--type", "charge.failed", "--created-gte", "1700000000"},
		{"events", "get", "evt_mock_charge_failed"},
		{"charges", "list", "--customer", "cus_mock_123"},
		{"charges", "get", "ch_mock_succeeded", "--expand", "payment_intent"},
		{"charges", "search", "--query", "amount>100"},
		{"payment-intents", "list", "--customer", "cus_mock_123"},
		{"payment-intents", "get", "pi_mock_succeeded"},
		{"payment-methods", "list", "--customer", "cus_mock_123", "--type", "card"},
		{"payment-methods", "get", "pm_mock_card"},
		{"setup-intents", "list", "--customer", "cus_mock_123"},
		{"invoices", "list", "--customer", "cus_mock_123", "--status", "open"},
		{"invoices", "get", "in_mock_paid"},
		{"invoices", "line-items", "in_mock_paid"},
		{"invoices", "search", "--query", "number:'MOCK-0001'"},
		{"invoices", "preview", "--subscription", "sub_mock_active"},
		{"subscriptions", "list", "--customer", "cus_mock_123", "--status", "all"},
		{"subscriptions", "get", "sub_mock_active"},
		{"subscriptions", "items", "sub_mock_active"},
		{"subscriptions", "invoices", "sub_mock_active", "--status", "open"},
		{"disputes", "list", "--charge", "ch_mock_succeeded"},
		{"refunds", "list", "--payment-intent", "pi_mock_succeeded"},
		{"transfers", "list", "--destination", "acct_mock_connected", "--transfer-group", "order_123"},
		{"payouts", "list", "--status", "failed"},
		{"balance-transactions", "list", "--type", "charge", "--payout", "po_mock_failed"},
		{"application-fees", "list", "--charge", "ch_mock_succeeded"},
		{"products", "list", "--active", "true"},
		{"prices", "list", "--product", "prod_mock_basic", "--type", "recurring"},
		{"payment-links", "list", "--active", "true"},
		{"early-fraud-warnings", "list", "--charge", "ch_mock_succeeded"},
		{"checkout-sessions", "list", "--customer", "cus_mock_123"},
		{"checkout-sessions", "line-items", "cs_mock_paid"},
		{"investigate", "resolve", "ch_mock_succeeded"},
		{"investigate", "customer-context", "--customer", "cus_mock_123"},
		{"investigate", "customer-card-payment", "--customer", "cus_mock_123", "--last4", "4242"},
		{"investigate", "incoming-payment", "pi_mock_failed"},
		{"investigate", "invoice-payment", "in_mock_paid"},
		{"investigate", "invoice-collection", "cus_mock_123"},
		{"investigate", "invoice-metadata", "in_mock_paid"},
		{"investigate", "subscription-renewal", "--subscription", "sub_mock_active"},
		{"investigate", "subscription-items", "--subscription", "sub_mock_active"},
		{"investigate", "subscription-amount-change", "--subscription", "sub_mock_active"},
		{"investigate", "entitlement", "--subscription", "sub_mock_active"},
		{"investigate", "collection-risk", "--days", "30"},
		{"investigate", "subscription-cancel-risk", "--days", "30"},
		{"investigate", "checkout-session", "cs_mock_paid"},
		{"investigate", "payment-method-readiness", "cus_mock_123"},
		{"investigate", "setup", "cus_mock_123"},
		{"investigate", "timeline", "cus_mock_123"},
		{"investigate", "dispute-response", "dp_mock_open"},
		{"investigate", "dispute-impact", "ch_mock_succeeded"},
		{"investigate", "fraud-review", "ch_mock_succeeded"},
		{"investigate", "refund", "ch_mock_succeeded"},
		{"investigate", "refund-recovery", "ch_mock_succeeded"},
		{"investigate", "ledger", "ch_mock_succeeded"},
		{"investigate", "payout-failure", "po_mock_failed"},
		{"investigate", "outgoing-payment", "tr_mock_failed"},
		{"investigate", "webhook-event", "evt_mock_charge_failed"},
		{"investigate", "webhook-delivery", "evt_mock_charge_failed"},
		{"investigate", "account-health", "acct_mock_connected"},
		{"investigate", "account-health", "acct_mock_v2_restricted"},
		{"investigate", "account-events", "acct_mock_v2_restricted"},
		{"investigate", "connect-readiness", "--limit", "3"},
		{"accounts-v2", "list"},
		{"accounts-v2", "get", "acct_mock_v2_restricted"},
		{"accounts-v2", "persons", "list", "acct_mock_v2_restricted"},
		{"accounts-v2", "payout-methods", "acct_mock_v2_recipient"},
		{"events-v2", "list", "--object-id", "acct_mock_v2_restricted"},
		{"events-v2", "get", "evt_test_mock_requirements_updated"},
	}
}

func TestRequestsMatchStripeOpenAPISpec(t *testing.T) {
	spec := loadSpec(t)
	rec := &recorder{inner: mockstripe.NewServer()}
	server := httptest.NewServer(rec)
	defer server.Close()

	for _, args := range commands() {
		rec.command = strings.Join(args, " ")
		var sink bytes.Buffer
		restore := output.SetWritersForTest(&sink, &sink)
		full := append([]string{"--api-key", "sk_test_mock", "--base-url", server.URL}, args...)
		cli.RunForTest("apicheck", full)
		restore()
	}

	var unknownPaths, unknownParams, v2Skipped []string
	seen := map[string]bool{}
	for _, req := range rec.requests {
		key := req.method + " " + req.path + " " + strings.Join(req.params, ",")
		if seen[key] {
			continue
		}
		seen[key] = true

		if strings.HasPrefix(req.path, "/v2/") {
			v2Skipped = append(v2Skipped, req.method+" "+req.path)
			continue
		}
		operation, template, ok := spec.lookup(req.method, req.path)
		if !ok {
			unknownPaths = append(unknownPaths, fmt.Sprintf("%s %s (from: %s)", req.method, req.path, req.from))
			continue
		}
		for _, param := range req.params {
			if operation.declares(param) {
				continue
			}
			unknownParams = append(unknownParams,
				fmt.Sprintf("%s %s: %q not declared (from: %s)", req.method, template, param, req.from))
		}
	}

	sort.Strings(unknownPaths)
	sort.Strings(unknownParams)
	if len(unknownPaths) > 0 {
		t.Errorf("paths not in Stripe's spec:\n  %s", strings.Join(unknownPaths, "\n  "))
	}
	if len(unknownParams) > 0 {
		t.Errorf("query parameters Stripe does not declare:\n  %s", strings.Join(unknownParams, "\n  "))
	}

	sort.Strings(v2Skipped)
	v2Skipped = dedupe(v2Skipped)
	t.Logf("checked %d distinct /v1 requests against spec version %s", len(seen)-len(v2Skipped), spec.version)
	t.Logf("/v2 requests not covered by Stripe's published spec (%d), verified against docs only:\n  %s",
		len(v2Skipped), strings.Join(v2Skipped, "\n  "))
}

func dedupe(values []string) []string {
	seen := map[string]bool{}
	out := values[:0]
	for _, value := range values {
		if seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

type openAPISpec struct {
	version string
	paths   map[string]map[string]operation
}

type operation struct {
	params map[string]bool
}

// declares reports whether the endpoint accepts a query parameter. Stripe
// declares nested and indexed parameters by their base name (created, expand),
// so created[gte] and expand[] match created and expand.
func (o operation) declares(param string) bool {
	if o.params[param] {
		return true
	}
	base := param
	if idx := strings.IndexByte(base, '['); idx > 0 {
		base = base[:idx]
	}
	return o.params[base]
}

func (s openAPISpec) lookup(method, path string) (operation, string, bool) {
	methods, ok := s.paths[path]
	if ok {
		op, ok := methods[strings.ToLower(method)]
		return op, path, ok
	}
	// Try templated paths: /v1/charges/{charge} against /v1/charges/ch_123.
	requestSegments := strings.Split(strings.Trim(path, "/"), "/")
	for template, methods := range s.paths {
		templateSegments := strings.Split(strings.Trim(template, "/"), "/")
		if len(templateSegments) != len(requestSegments) {
			continue
		}
		match := true
		for idx, segment := range templateSegments {
			if strings.HasPrefix(segment, "{") {
				continue
			}
			if segment != requestSegments[idx] {
				match = false
				break
			}
		}
		if !match {
			continue
		}
		if op, ok := methods[strings.ToLower(method)]; ok {
			return op, template, true
		}
	}
	return operation{}, "", false
}

func loadSpec(t *testing.T) openAPISpec {
	t.Helper()
	path := os.Getenv("STRIPE_OPENAPI_SPEC")
	if path == "" {
		t.Skip("set STRIPE_OPENAPI_SPEC to Stripe's spec3.json (see `make apicheck`)")
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open spec: %v", err)
	}
	defer file.Close()

	var raw struct {
		Info  struct{ Version string } `json:"info"`
		Paths map[string]map[string]struct {
			Parameters []struct {
				Name string `json:"name"`
				In   string `json:"in"`
			} `json:"parameters"`
		} `json:"paths"`
	}
	if err := json.NewDecoder(file).Decode(&raw); err != nil {
		t.Fatalf("decode spec: %v", err)
	}

	spec := openAPISpec{version: raw.Info.Version, paths: map[string]map[string]operation{}}
	for path, methods := range raw.Paths {
		spec.paths[path] = map[string]operation{}
		for method, op := range methods {
			params := map[string]bool{}
			for _, param := range op.Parameters {
				if param.In == "query" {
					params[param.Name] = true
				}
			}
			spec.paths[path][strings.ToLower(method)] = operation{params: params}
		}
	}
	return spec
}
