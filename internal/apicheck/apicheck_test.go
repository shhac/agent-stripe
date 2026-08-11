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
	"fmt"
	"net/http"
	"net/http/httptest"
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

// notNetworked are the leaf commands that never reach Stripe, so they have no
// requests to validate.
var notNetworked = map[string]bool{
	"usage": true, "version": true, "mcp": true,
	"auth add": true, "auth update": true, "auth remove": true, "auth default": true, "auth list": true,
	"config show": true, "config path": true, "config get": true, "config set": true, "config unset": true,
	"payments usage": true, "connect usage": true, "invoices usage": true, "subscriptions usage": true,
	"accounts-v2 usage": true, "events-v2 usage": true, "investigate usage": true,
	"charges usage": true, "payment-intents usage": true, "mcp usage": true,
	"mcp pair reset": true, "mcp pair rotate": true,
	// api get is the raw escape hatch: its path comes from the caller, so there
	// is no fixed request to validate.
	"api get": true,
	// auth check reaches Stripe, but through the credential store rather than a
	// profile the harness can supply.
	"auth check": true,
}

// TestCommandTableCoversEveryCommand is what makes the spec check mean
// something. The table used to carry a comment asking contributors to keep it
// exhaustive, with nothing enforcing it — so a new command's requests would go
// unvalidated silently, which is exactly the failure the package exists to
// prevent.
func TestCommandTableCoversEveryCommand(t *testing.T) {
	covered := map[string]bool{}
	for _, args := range commands() {
		for end := len(args); end > 0; end-- {
			covered[strings.Join(args[:end], " ")] = true
		}
	}

	var missing []string
	for _, leaf := range cli.LeafCommandsForTest("apicheck") {
		if covered[leaf] || notNetworked[leaf] {
			continue
		}
		missing = append(missing, leaf)
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("commands reaching Stripe with no entry in commands() (add them, or to notNetworked):\n  %s",
			strings.Join(missing, "\n  "))
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

	var unknownPaths, unknownParams []string
	v2Skipped := map[string]bool{}
	seen := map[string]bool{}
	v1Checked := 0
	for _, req := range rec.requests {
		key := req.method + " " + req.path + " " + strings.Join(req.params, ",")
		if seen[key] {
			continue
		}
		seen[key] = true

		if strings.HasPrefix(req.path, "/v2/") {
			v2Skipped[req.method+" "+req.path] = true
			continue
		}
		v1Checked++
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

	skipped := make([]string, 0, len(v2Skipped))
	for path := range v2Skipped {
		skipped = append(skipped, path)
	}
	sort.Strings(skipped)
	t.Logf("checked %d distinct /v1 requests against spec version %s", v1Checked, spec.version)
	t.Logf("/v2 requests not covered by Stripe's published spec (%d), verified against docs only:\n  %s",
		len(skipped), strings.Join(skipped, "\n  "))
}
