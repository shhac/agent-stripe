package auth

import (
	"bytes"
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

func (h *authCommandHarness) run(args ...string) (string, string) {
	h.t.Helper()
	h.stdout.Reset()
	h.stderr.Reset()
	root := &cobra.Command{Use: "agent-stripe"}
	Register(root, func() *shared.GlobalFlags { return &shared.GlobalFlags{} })
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		h.t.Fatalf("agent-stripe %v failed: %v\nstdout:\n%s\nstderr:\n%s", args, err, h.stdout.String(), h.stderr.String())
	}
	return h.stdout.String(), h.stderr.String()
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
}

func TestAuthAddStorageFailureDoesNotWriteProfile(t *testing.T) {
	h := newAuthCommandHarness(t)
	credentialStore = func(name, apiKey string) (string, error) {
		return "", errors.New("keychain unavailable")
	}

	stdout, stderr := h.run("auth", "add", "prod", "--api-key", "sk_test_secret")

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

	stdout, stderr := h.run("auth", "remove", "prod")

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
