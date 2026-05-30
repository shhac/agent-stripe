package main

import (
	"net/http/httptest"
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

func runMockCLI(t *testing.T, args ...string) string {
	t.Helper()
	return newMockCLIRunner(t).Run(args...)
}
