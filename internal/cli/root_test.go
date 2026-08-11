package cli

import (
	"bytes"
	"encoding/json"
	agentmcp "github.com/shhac/lib-agent-mcp"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/shhac/agent-stripe/internal/config"
	"github.com/shhac/agent-stripe/internal/output"
)

type cliTestHarness struct {
	t      *testing.T
	stdout bytes.Buffer
	stderr bytes.Buffer
}

func newCLITestHarness(t *testing.T) *cliTestHarness {
	t.Helper()
	config.SetConfigDir(t.TempDir())
	t.Cleanup(func() { config.SetConfigDir("") })
	t.Setenv("AGENT_STRIPE_PROFILE", "")
	t.Setenv("STRIPE_API_KEY", "")
	t.Setenv("STRIPE_CONTEXT", "")
	t.Setenv("STRIPE_API_VERSION", "")
	t.Setenv("AGENT_STRIPE_BASE_URL", "")
	h := &cliTestHarness{t: t}
	restore := output.SetWritersForTest(&h.stdout, &h.stderr)
	t.Cleanup(restore)
	return h
}

// run executes a command expected to succeed, failing the test if it bubbles an
// error to the single sink.
func (h *cliTestHarness) run(args ...string) (string, string) {
	h.t.Helper()
	stdout, stderr, err := h.execute(args...)
	if err != nil {
		h.t.Fatalf("agent-stripe %v failed: %v\nstdout:\n%s\nstderr:\n%s", args, err, stdout, stderr)
	}
	return stdout, stderr
}

// runErr executes a command expected to fail, rendering the bubbled error to the
// captured stderr exactly as libcli.Run does in production (which is the only
// place errors are rendered now). It fails the test if the command unexpectedly
// succeeds.
func (h *cliTestHarness) runErr(args ...string) (string, string) {
	h.t.Helper()
	stdout, _, err := h.execute(args...)
	if err == nil {
		h.t.Fatalf("agent-stripe %v succeeded, want error", args)
	}
	output.WriteError(&h.stderr, err)
	return stdout, h.stderr.String()
}

func (h *cliTestHarness) execute(args ...string) (string, string, error) {
	h.t.Helper()
	h.stdout.Reset()
	h.stderr.Reset()
	cmd := newRootCmd("test")
	cmd.SetArgs(args)
	err := cmd.Execute()
	return h.stdout.String(), h.stderr.String(), err
}

func TestConfigCommandsEditNonSecretDefaults(t *testing.T) {
	h := newCLITestHarness(t)

	stdout, _ := h.run("config", "path")
	if !strings.Contains(stdout, "credentials.json") {
		t.Fatalf("config path missing credentials index path: %s", stdout)
	}

	stdout, _ = h.run("config", "set", "max-retries", "4")
	assertJSONField(t, stdout, "status", "set")
	cfg := config.Read()
	if cfg.Defaults.MaxRetries == nil || *cfg.Defaults.MaxRetries != 4 {
		t.Fatalf("MaxRetries = %#v, want 4", cfg.Defaults.MaxRetries)
	}

	stdout, _ = h.run("config", "get", "max_retries")
	assertJSONField(t, stdout, "value", float64(4))

	stdout, _ = h.run("config", "show")
	if strings.Contains(stdout, "sk_test") {
		t.Fatalf("config show leaked secret-looking value: %s", stdout)
	}
	if !strings.Contains(stdout, `"max_retries": 4`) {
		t.Fatalf("config show missing max_retries: %s", stdout)
	}

	stdout, _ = h.run("config", "unset", "max-retries")
	assertJSONField(t, stdout, "status", "unset")
	cfg = config.Read()
	if cfg.Defaults.MaxRetries != nil {
		t.Fatalf("MaxRetries = %#v, want nil after unset", *cfg.Defaults.MaxRetries)
	}
}

func TestAuthUpdateEditsProfileMetadata(t *testing.T) {
	h := newCLITestHarness(t)
	if err := config.StoreProfile("prod", config.Profile{Context: "acct_old", APIVersion: "2024-01-01"}); err != nil {
		t.Fatalf("StoreProfile() error = %v", err)
	}
	if err := config.StoreProfile("sandbox", config.Profile{}); err != nil {
		t.Fatalf("StoreProfile() error = %v", err)
	}

	stdout, _ := h.run("auth", "update", "sandbox", "--context", "acct_new", "--api-version", "2025-06-30.basil", "--default")
	assertJSONField(t, stdout, "status", "updated")

	cfg := config.Read()
	if cfg.DefaultProfile != "sandbox" {
		t.Fatalf("DefaultProfile = %q, want sandbox", cfg.DefaultProfile)
	}
	profile := cfg.Profiles["sandbox"]
	if profile.Context != "acct_new" || profile.APIVersion != "2025-06-30.basil" {
		t.Fatalf("profile = %+v", profile)
	}

	stdout, _ = h.run("auth", "update", "sandbox", "--clear-context")
	assertJSONField(t, stdout, "context", "")
	if got := config.Read().Profiles["sandbox"].Context; got != "" {
		t.Fatalf("Context = %q, want cleared", got)
	}
}

func TestConfigDefaultsApplyToClientWhenFlagsOmitted(t *testing.T) {
	h := newCLITestHarness(t)
	if err := config.SetDefaultValue("max_retries", 5); err != nil {
		t.Fatalf("SetDefaultValue(max_retries) error = %v", err)
	}
	if err := config.SetDefaultValue("timeout_ms", 1234); err != nil {
		t.Fatalf("SetDefaultValue(timeout_ms) error = %v", err)
	}
	server := newBalanceServer(t)
	defer server.Close()

	_, stderr := h.run("--api-key", "sk_test_123", "--base-url", server.URL, "--debug", "balance", "get")
	if !strings.Contains(stderr, `"max_retries":5`) {
		t.Fatalf("stderr missing configured max_retries: %s", stderr)
	}
	if !strings.Contains(stderr, `"timeout_ms":1234`) {
		t.Fatalf("stderr missing configured timeout_ms: %s", stderr)
	}
}

func TestGlobalFlagsOverrideConfigDefaults(t *testing.T) {
	h := newCLITestHarness(t)
	if err := config.SetDefaultValue("max_retries", 5); err != nil {
		t.Fatalf("SetDefaultValue(max_retries) error = %v", err)
	}
	if err := config.SetDefaultValue("timeout_ms", 1234); err != nil {
		t.Fatalf("SetDefaultValue(timeout_ms) error = %v", err)
	}
	server := newBalanceServer(t)
	defer server.Close()

	_, stderr := h.run("--api-key", "sk_test_123", "--base-url", server.URL, "--debug", "--max-retries", "0", "--timeout", "88", "balance", "get")
	if !strings.Contains(stderr, `"max_retries":0`) {
		t.Fatalf("stderr missing flag max_retries: %s", stderr)
	}
	if !strings.Contains(stderr, `"timeout_ms":88`) {
		t.Fatalf("stderr missing flag timeout_ms: %s", stderr)
	}
}

func TestAuthCheckIncludesCredentialType(t *testing.T) {
	h := newCLITestHarness(t)
	server := newAccountServer(t)
	defer server.Close()

	stdout, stderr := h.run("--api-key", "rk_test_123", "--base-url", server.URL, "auth", "check")

	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	assertJSONField(t, stdout, "status", "ok")
	assertJSONField(t, stdout, "credential_type", "rk_test")
	assertJSONField(t, stdout, "credential_metadata_status", "not_stored")
}

func newBalanceServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/balance" {
			t.Fatalf("path = %s, want /v1/balance", r.URL.Path)
		}
		w.Header().Set("Request-Id", "req_test")
		_, _ = w.Write([]byte(`{"object":"balance","available":[],"pending":[]}`))
	}))
}

func newAccountServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/account" {
			t.Fatalf("path = %s, want /v1/account", r.URL.Path)
		}
		w.Header().Set("Request-Id", "req_test")
		_, _ = w.Write([]byte(`{"object":"account","id":"acct_test"}`))
	}))
}

func assertJSONField(t *testing.T, raw, key string, want any) {
	t.Helper()
	var got map[string]any
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, raw)
	}
	if got[key] != want {
		t.Fatalf("%s = %#v, want %#v in %s", key, got[key], want, raw)
	}
}

func TestMCPExposesEveryGroupExceptTheDenyList(t *testing.T) {
	root := newRootCmd("test")

	// Stated as a deny-list so a new resource is agent-facing by default; the
	// invariant worth pinning is that credential and config surfaces never are.
	denied := map[string]bool{"auth": true, "config": true, "usage": true, "completion": true, "help": true}
	exposed := 0
	for _, cmd := range root.Commands() {
		isExposed := cmd.Annotations[agentmcp.AnnotationExpose] == "true"
		switch {
		case denied[cmd.Name()] && isExposed:
			t.Errorf("%s must not be exposed as an MCP tool", cmd.Name())
		case !denied[cmd.Name()] && cmd.Name() != "mcp" && !cmd.Hidden && !isExposed:
			t.Errorf("%s should be exposed as an MCP tool", cmd.Name())
		case isExposed:
			exposed++
		}
	}
	if exposed < 20 {
		t.Fatalf("only %d groups exposed; the deny-list is probably too broad", exposed)
	}
}
