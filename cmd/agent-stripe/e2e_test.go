package main

import (
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"

	"github.com/shhac/agent-stripe/internal/mockstripe"
)

func TestCLIAgainstMockStripe(t *testing.T) {
	server := httptest.NewServer(mockstripe.NewServer())
	defer server.Close()

	cmd := exec.Command("go", "run", "./cmd/agent-stripe",
		"--api-key", "sk_test_mock",
		"--base-url", server.URL,
		"events", "list",
		"--type", "charge.failed",
		"--limit", "1",
	)
	cmd.Dir = "../.."

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("agent-stripe failed: %v\n%s", err, out)
	}
	text := string(out)
	if !strings.Contains(text, `"evt_mock_charge_failed"`) {
		t.Fatalf("output did not contain mock event: %s", text)
	}
	if !strings.Contains(text, `"charge.failed"`) {
		t.Fatalf("output did not contain event type: %s", text)
	}
}

func TestCLIDebugAgainstMockStripe(t *testing.T) {
	server := httptest.NewServer(mockstripe.NewServer())
	defer server.Close()

	cmd := exec.Command("go", "run", "./cmd/agent-stripe",
		"--debug",
		"--api-key", "sk_test_mock",
		"--base-url", server.URL,
		"balance", "get",
	)
	cmd.Dir = "../.."

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("agent-stripe failed: %v\n%s", err, out)
	}
	text := string(out)
	for _, want := range []string{`"@debug":"client"`, `"credential_source":"flag"`, `"@debug":"http"`, `"request_id":"req_mock_123"`} {
		if !strings.Contains(text, want) {
			t.Fatalf("debug output missing %s:\n%s", want, text)
		}
	}
	if strings.Contains(text, "sk_test_mock") {
		t.Fatalf("debug output leaked API key:\n%s", text)
	}
}

func TestCLISubscriptionsAgainstMockStripe(t *testing.T) {
	server := httptest.NewServer(mockstripe.NewServer())
	defer server.Close()

	cmd := exec.Command("go", "run", "./cmd/agent-stripe",
		"--api-key", "sk_test_mock",
		"--base-url", server.URL,
		"subscriptions", "invoices", "sub_mock_past_due",
		"--status", "open",
	)
	cmd.Dir = "../.."

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("agent-stripe failed: %v\n%s", err, out)
	}
	text := string(out)
	if !strings.Contains(text, `"in_mock_open_failed"`) {
		t.Fatalf("output did not contain mock invoice: %s", text)
	}
	if !strings.Contains(text, `"payment_intent":"pi_mock_failed"`) {
		t.Fatalf("output did not contain failed PaymentIntent: %s", text)
	}
}
