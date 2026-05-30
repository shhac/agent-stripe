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
