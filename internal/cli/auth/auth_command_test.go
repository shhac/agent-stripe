package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
	"github.com/shhac/agent-stripe/internal/config"
	"github.com/shhac/agent-stripe/internal/output"
)

type authCommandHarness struct {
	t      *testing.T
	stdout bytes.Buffer
	stderr bytes.Buffer
}

func newAuthCommandHarness(t *testing.T) *authCommandHarness {
	t.Helper()
	config.SetConfigDir(t.TempDir())
	t.Cleanup(func() { config.SetConfigDir("") })
	h := &authCommandHarness{t: t}
	restoreWriters := output.SetWritersForTest(&h.stdout, &h.stderr)
	t.Cleanup(restoreWriters)
	previousStore := credentialStore
	previousRemove := credentialRemove
	t.Cleanup(func() {
		credentialStore = previousStore
		credentialRemove = previousRemove
	})
	return h
}

// run executes a command expected to succeed, failing the test if it bubbles an
// error to the single sink.
func (h *authCommandHarness) run(args ...string) (string, string) {
	h.t.Helper()
	stdout, stderr, err := h.execute(args...)
	if err != nil {
		h.t.Fatalf("agent-stripe %v failed: %v\nstdout:\n%s\nstderr:\n%s", args, err, stdout, stderr)
	}
	return stdout, stderr
}

// runErr executes a command expected to fail, rendering the bubbled error to the
// captured stderr exactly as libcli.Run does in production. It fails the test if
// the command unexpectedly succeeds.
func (h *authCommandHarness) runErr(args ...string) (string, string) {
	h.t.Helper()
	stdout, _, err := h.execute(args...)
	if err == nil {
		h.t.Fatalf("agent-stripe %v succeeded, want error", args)
	}
	output.WriteError(&h.stderr, err)
	return stdout, h.stderr.String()
}

func (h *authCommandHarness) execute(args ...string) (string, string, error) {
	h.t.Helper()
	h.stdout.Reset()
	h.stderr.Reset()
	root := &cobra.Command{Use: "agent-stripe", SilenceUsage: true, SilenceErrors: true}
	Register(root, func() *shared.GlobalFlags { return &shared.GlobalFlags{} })
	root.SetArgs(args)
	err := root.Execute()
	return h.stdout.String(), h.stderr.String(), err
}

func TestAuthAddStoresSecretOutOfBandAndDoesNotPrintIt(t *testing.T) {
	h := newAuthCommandHarness(t)
	secret := "sk_test_secret"
	var storedAlias string
	var storedSecret string
	credentialStore = func(name, apiKey string) (string, error) {
		storedAlias = name
		storedSecret = apiKey
		return "keychain", nil
	}

	stdout, stderr := h.run("auth", "add", "prod", "--api-key", secret, "--context", "acct_123", "--api-version", "2025-06-30.basil")

	if storedAlias != "prod" || storedSecret != secret {
		t.Fatalf("credentialStore(%q, %q), want prod and secret", storedAlias, storedSecret)
	}
	if strings.Contains(stdout+stderr, secret) {
		t.Fatalf("auth add leaked API key\nstdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
	assertAuthJSONField(t, stdout, "status", "added")
	cfg := config.Read()
	profile := cfg.Profiles["prod"]
	if cfg.DefaultProfile != "prod" || profile.Context != "acct_123" || profile.APIVersion != "2025-06-30.basil" {
		t.Fatalf("config = %+v, profile = %+v", cfg, profile)
	}
	if profile.CredentialType != "sk_test" {
		t.Fatalf("CredentialType = %q, want sk_test", profile.CredentialType)
	}
}

func TestAuthUpdateCanReplaceKeyAndCredentialType(t *testing.T) {
	h := newAuthCommandHarness(t)
	if err := config.StoreProfile("prod", config.Profile{CredentialType: "sk_test"}); err != nil {
		t.Fatalf("StoreProfile() error = %v", err)
	}
	var storedAlias string
	var storedSecret string
	credentialStore = func(name, apiKey string) (string, error) {
		storedAlias = name
		storedSecret = apiKey
		return "keychain", nil
	}

	stdout, stderr := h.run("auth", "update", "prod", "--api-key", "rk_test_replacement")

	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if storedAlias != "prod" || storedSecret != "rk_test_replacement" {
		t.Fatalf("credentialStore(%q, %q), want prod and replacement key", storedAlias, storedSecret)
	}
	if strings.Contains(stdout, storedSecret) {
		t.Fatalf("auth update leaked replacement key: %s", stdout)
	}
	assertAuthJSONField(t, stdout, "status", "updated")
	assertAuthJSONField(t, stdout, "credential_type", "rk_test")
	if got := config.Read().Profiles["prod"].CredentialType; got != "rk_test" {
		t.Fatalf("CredentialType = %q, want rk_test", got)
	}
}

func TestAuthAddStorageFailureDoesNotWriteProfile(t *testing.T) {
	h := newAuthCommandHarness(t)
	credentialStore = func(name, apiKey string) (string, error) {
		return "", errors.New("keychain unavailable")
	}

	stdout, stderr := h.runErr("auth", "add", "prod", "--api-key", "sk_test_secret")

	if stdout != "" {
		t.Fatalf("stdout = %q, want empty on failure", stdout)
	}
	if !strings.Contains(stderr, "keychain unavailable") {
		t.Fatalf("stderr = %q, want keychain error", stderr)
	}
	if _, ok := config.Read().Profiles["prod"]; ok {
		t.Fatalf("profile was written after credential storage failure")
	}
}

func TestAuthRemoveDeletesCredentialThenProfile(t *testing.T) {
	h := newAuthCommandHarness(t)
	if err := config.StoreProfile("prod", config.Profile{Context: "acct_123"}); err != nil {
		t.Fatalf("StoreProfile() error = %v", err)
	}
	var removedAlias string
	credentialRemove = func(name string) error {
		removedAlias = name
		return nil
	}

	stdout, _ := h.run("auth", "remove", "prod")

	if removedAlias != "prod" {
		t.Fatalf("credentialRemove(%q), want prod", removedAlias)
	}
	assertAuthJSONField(t, stdout, "status", "removed")
	if _, ok := config.Read().Profiles["prod"]; ok {
		t.Fatalf("profile still exists after auth remove")
	}
}

func TestAuthRemoveCredentialFailureLeavesProfile(t *testing.T) {
	h := newAuthCommandHarness(t)
	if err := config.StoreProfile("prod", config.Profile{Context: "acct_123"}); err != nil {
		t.Fatalf("StoreProfile() error = %v", err)
	}
	credentialRemove = func(name string) error {
		return errors.New("keychain delete failed")
	}

	stdout, stderr := h.runErr("auth", "remove", "prod")

	if stdout != "" {
		t.Fatalf("stdout = %q, want empty on failure", stdout)
	}
	if !strings.Contains(stderr, "keychain delete failed") {
		t.Fatalf("stderr = %q, want keychain error", stderr)
	}
	if _, ok := config.Read().Profiles["prod"]; !ok {
		t.Fatalf("profile was removed after credential deletion failure")
	}
}

func TestAuthListCredentialTypeHints(t *testing.T) {
	h := newAuthCommandHarness(t)
	if err := config.StoreProfile("good", config.Profile{CredentialType: "rk_live"}); err != nil {
		t.Fatalf("StoreProfile(good) error = %v", err)
	}
	if err := config.StoreProfile("legacy", config.Profile{}); err != nil {
		t.Fatalf("StoreProfile(legacy) error = %v", err)
	}
	if err := config.StoreProfile("publishable", config.Profile{CredentialType: "pk_test"}); err != nil {
		t.Fatalf("StoreProfile(publishable) error = %v", err)
	}
	if err := config.StoreProfile("weird", config.Profile{CredentialType: "unknown"}); err != nil {
		t.Fatalf("StoreProfile(weird) error = %v", err)
	}

	stdout, _ := h.run("auth", "list")
	items := parseAuthNDJSON(t, stdout)

	if got := items["good"]["credential_type"]; got != "rk_live" {
		t.Fatalf("good credential_type = %#v, want rk_live", got)
	}
	if _, ok := items["good"]["hint"]; ok {
		t.Fatalf("good profile should not have a hint: %#v", items["good"])
	}
	if got := items["legacy"]["hint"]; got != "Credential type is not stored for this profile yet. Run 'agent-stripe auth check legacy' to refresh profile metadata." {
		t.Fatalf("legacy hint = %#v", got)
	}
	if got := items["weird"]["hint"]; got != "Stored credential format is not recognized by agent-stripe. It may still work; run 'agent-stripe auth check weird' to test it." {
		t.Fatalf("weird hint = %#v", got)
	}
	if got := items["publishable"]["hint"]; got != "Publishable keys cannot authenticate agent-stripe API requests. Run 'agent-stripe auth update publishable --form' with a restricted or secret key." {
		t.Fatalf("publishable hint = %#v", got)
	}
}

func TestRefreshStoredCredentialTypeUpdatesProfileMetadata(t *testing.T) {
	h := newAuthCommandHarness(t)
	_ = h
	if err := config.StoreProfile("legacy", config.Profile{}); err != nil {
		t.Fatalf("StoreProfile() error = %v", err)
	}

	status := refreshStoredCredentialType(&shared.ResolvedProfile{
		Alias:            "legacy",
		CredentialSource: "keychain",
	}, "rk_test")

	if status != "updated" {
		t.Fatalf("status = %q, want updated", status)
	}
	if got := config.Read().Profiles["legacy"].CredentialType; got != "rk_test" {
		t.Fatalf("CredentialType = %q, want rk_test", got)
	}
}

func assertAuthJSONField(t *testing.T, raw, key string, want any) {
	t.Helper()
	var got map[string]any
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, raw)
	}
	if got[key] != want {
		t.Fatalf("%s = %#v, want %#v in %s", key, got[key], want, raw)
	}
}

func parseAuthNDJSON(t *testing.T, raw string) map[string]map[string]any {
	t.Helper()
	items := map[string]map[string]any{}
	for _, line := range strings.Split(strings.TrimSpace(raw), "\n") {
		if line == "" {
			continue
		}
		var item map[string]any
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			t.Fatalf("output line is not JSON: %v\n%s", err, line)
		}
		profile, _ := item["profile"].(string)
		if profile != "" {
			items[profile] = item
		}
	}
	return items
}

func TestApplyAuthAddStoresProfileAndReportsBackend(t *testing.T) {
	newAuthCommandHarness(t)

	result, err := applyAuthAdd(context.Background(), "sandbox", authAddOptions{
		apiKey:       "sk_test_abc",
		contextValue: "acct_platform/acct_connected",
	})
	if err != nil {
		t.Fatalf("applyAuthAdd() error = %v", err)
	}

	fields := result.output()
	if fields["status"] != "added" || fields["profile"] != "sandbox" {
		t.Fatalf("output = %#v", fields)
	}
	// Versions come back from the stored profile, so the receipt cannot drift
	// from what was actually persisted.
	if fields["api_version"] != config.DefaultAPIVersion || fields["v2_api_version"] != config.DefaultV2APIVersion {
		t.Fatalf("versions = %#v", fields)
	}
	if fields["context"] != "acct_platform/acct_connected" {
		t.Fatalf("context = %#v", fields["context"])
	}
	if fields["credential_type"] != "sk_test" {
		t.Fatalf("credential_type = %#v", fields["credential_type"])
	}
	if _, leaked := fields["api_key"]; leaked {
		t.Fatalf("receipt must never carry the key: %#v", fields)
	}
	for _, value := range fields {
		if text, ok := value.(string); ok && strings.Contains(text, "sk_test_abc") {
			t.Fatalf("receipt leaked the key: %#v", fields)
		}
	}
}

func TestApplyAuthAddRequiresAKey(t *testing.T) {
	newAuthCommandHarness(t)
	if _, err := applyAuthAdd(context.Background(), "sandbox", authAddOptions{}); err == nil {
		t.Fatalf("applyAuthAdd() error = nil, want the missing-key error")
	}
}
