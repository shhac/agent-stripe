package shared

import (
	"errors"
	"testing"

	"github.com/shhac/agent-stripe/internal/config"
	"github.com/shhac/agent-stripe/internal/credential"
)

func withResolveProfileTest(t *testing.T) {
	t.Helper()
	config.SetConfigDir(t.TempDir())
	t.Cleanup(func() { config.SetConfigDir("") })
	t.Setenv("AGENT_STRIPE_PROFILE", "")
	t.Setenv("STRIPE_API_KEY", "")
	t.Setenv("STRIPE_CONTEXT", "")
	t.Setenv("STRIPE_API_VERSION", "")
	t.Setenv("AGENT_STRIPE_BASE_URL", "")
	previousGet := credentialGet
	credentialGet = func(name string) (string, string, error) {
		return "", "", errors.New("unexpected credential lookup for " + name)
	}
	t.Cleanup(func() { credentialGet = previousGet })
}

func TestResolveProfileDirectAPIKeyFromFlags(t *testing.T) {
	withResolveProfileTest(t)
	t.Setenv("STRIPE_CONTEXT", "acct_env")
	t.Setenv("STRIPE_API_VERSION", "2024-01-01")
	t.Setenv("AGENT_STRIPE_BASE_URL", "http://mock")

	got, err := ResolveProfile(&GlobalFlags{
		Profile:    "flag-profile",
		APIKey:     "sk_flag",
		Context:    "acct_flag",
		APIVersion: "2025-01-01",
	})
	if err != nil {
		t.Fatalf("ResolveProfile() error = %v", err)
	}
	if got.Alias != "flag-profile" || got.APIKey != "sk_flag" || got.CredentialSource != "flag" {
		t.Fatalf("resolved = %+v", got)
	}
	if got.Profile.Context != "acct_flag" || got.Profile.APIVersion != "2025-01-01" {
		t.Fatalf("profile = %+v", got.Profile)
	}
	if got.BaseURL != "http://mock" {
		t.Fatalf("BaseURL = %q", got.BaseURL)
	}
}

func TestResolveProfileDirectAPIKeyFromEnv(t *testing.T) {
	withResolveProfileTest(t)
	t.Setenv("AGENT_STRIPE_PROFILE", "env-profile")
	t.Setenv("STRIPE_API_KEY", "sk_env")
	t.Setenv("STRIPE_CONTEXT", "acct_env")
	t.Setenv("STRIPE_API_VERSION", "2024-01-01")

	got, err := ResolveProfile(&GlobalFlags{})
	if err != nil {
		t.Fatalf("ResolveProfile() error = %v", err)
	}
	if got.Alias != "env-profile" || got.APIKey != "sk_env" || got.CredentialSource != "env" {
		t.Fatalf("resolved = %+v", got)
	}
	if got.Profile.Context != "acct_env" || got.Profile.APIVersion != "2024-01-01" {
		t.Fatalf("profile = %+v", got.Profile)
	}
}

func TestResolveProfileStoredProfile(t *testing.T) {
	withResolveProfileTest(t)
	if err := config.StoreProfile("prod", config.Profile{Context: "acct_saved"}); err != nil {
		t.Fatalf("StoreProfile() error = %v", err)
	}
	credentialGet = func(name string) (string, string, error) {
		if name != "prod" {
			t.Fatalf("credential lookup name = %q", name)
		}
		return "sk_keychain", credential.BackendKeychain, nil
	}

	got, err := ResolveProfile(&GlobalFlags{Context: "acct_override"})
	if err != nil {
		t.Fatalf("ResolveProfile() error = %v", err)
	}
	if got.Alias != "prod" || got.APIKey != "sk_keychain" || got.CredentialSource != "keychain" {
		t.Fatalf("resolved = %+v", got)
	}
	if got.Profile.Context != "acct_override" || got.Profile.APIVersion != config.DefaultAPIVersion {
		t.Fatalf("profile = %+v", got.Profile)
	}
}

func TestResolveProfileMissingProfile(t *testing.T) {
	withResolveProfileTest(t)

	_, err := ResolveProfile(&GlobalFlags{})
	if err == nil {
		t.Fatalf("expected missing profile error")
	}
	if !errors.Is(err, nil) && err.Error() != "No Stripe profile configured" {
		t.Fatalf("error = %v", err)
	}
}

func TestResolveProfileMissingCredential(t *testing.T) {
	withResolveProfileTest(t)
	if err := config.StoreProfile("prod", config.Profile{}); err != nil {
		t.Fatalf("StoreProfile() error = %v", err)
	}
	credentialGet = func(name string) (string, string, error) {
		return "", credential.BackendKeychain, errors.New("missing key")
	}

	_, err := ResolveProfile(&GlobalFlags{})
	if err == nil {
		t.Fatalf("expected credential error")
	}
	if err.Error() != "missing key" {
		t.Fatalf("error = %v", err)
	}
}
