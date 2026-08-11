package main

import (
	"os/exec"
	"strings"
	"testing"
)

// The rest of the e2e suite runs in-process. This case keeps one real process
// boundary under test, because exit codes and the stdout/stderr split are part
// of the CLI's contract with a calling agent and cannot be observed in-process.
func TestBinaryExitCodeAndStreamSplit(t *testing.T) {
	cmd := exec.Command("go", "run", "./cmd/agent-stripe", "--api-key", "sk_test_mock",
		"--base-url", "http://127.0.0.1:1", "balance", "get")
	cmd.Dir = "../.."
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected a non-zero exit for an unreachable API")
	}
	if stdout.String() != "" {
		t.Fatalf("stdout should be empty on a command-level failure, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), `"fixable_by"`) {
		t.Fatalf("stderr should carry the structured error, got %q", stderr.String())
	}
}
