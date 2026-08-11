package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os/exec"
	"strings"
	"testing"

	"github.com/shhac/agent-stripe/internal/mockstripe"
)

type mockCLIRunner struct {
	t      *testing.T
	server *httptest.Server
}

func newMockCLIRunner(t *testing.T) *mockCLIRunner {
	t.Helper()
	server := httptest.NewServer(mockstripe.NewServer())
	t.Cleanup(server.Close)
	return &mockCLIRunner{t: t, server: server}
}

// ArmFault makes the mock fail matching paths for the rest of the test, so an
// e2e case can check what the CLI reports when a related lookup fails rather
// than only ever seeing a healthy Stripe. Spec format is documented on
// mockstripe's fault rules (e.g. "/v1/disputes=500", "/v1/charges=429x2").
func (r *mockCLIRunner) ArmFault(spec string) {
	r.t.Helper()
	resp, err := http.Post(r.server.URL+"/_mock/faults?rules="+url.QueryEscape(spec), "", nil)
	if err != nil {
		r.t.Fatalf("arm fault %q: %v", spec, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		r.t.Fatalf("arm fault %q: status %d", spec, resp.StatusCode)
	}
}

func (r *mockCLIRunner) Run(args ...string) string {
	r.t.Helper()
	allArgs := []string{"run", "./cmd/agent-stripe", "--api-key", "sk_test_mock", "--base-url", r.server.URL}
	allArgs = append(allArgs, args...)
	cmd := exec.Command("go", allArgs...)
	cmd.Dir = "../.."

	out, err := cmd.CombinedOutput()
	if err != nil {
		r.t.Fatalf("agent-stripe %v failed: %v\n%s", args, err, out)
	}
	return string(out)
}

// RunExpectingError runs the CLI for a case that must fail — a structured
// error on stderr and a non-zero exit (the single-sink contract). It returns
// the combined output and fatals only if the command unexpectedly succeeded.
func (r *mockCLIRunner) RunExpectingError(args ...string) string {
	r.t.Helper()
	allArgs := []string{"run", "./cmd/agent-stripe", "--api-key", "sk_test_mock", "--base-url", r.server.URL}
	allArgs = append(allArgs, args...)
	cmd := exec.Command("go", allArgs...)
	cmd.Dir = "../.."

	out, err := cmd.CombinedOutput()
	if err == nil {
		r.t.Fatalf("agent-stripe %v unexpectedly succeeded; expected a non-zero exit\n%s", args, out)
	}
	return string(out)
}

func runMockCLIErr(t *testing.T, args ...string) string {
	t.Helper()
	return newMockCLIRunner(t).RunExpectingError(args...)
}

func (r *mockCLIRunner) AssertContains(out string, wants ...string) {
	r.t.Helper()
	assertContains(r.t, out, wants...)
}

func assertContains(t *testing.T, out string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %s:\n%s", want, out)
		}
	}
}

func assertNotContains(t *testing.T, out string, blocked ...string) {
	t.Helper()
	for _, value := range blocked {
		if strings.Contains(out, value) {
			t.Fatalf("output unexpectedly contained %s:\n%s", value, out)
		}
	}
}

func runMockCLI(t *testing.T, args ...string) string {
	t.Helper()
	return newMockCLIRunner(t).Run(args...)
}
