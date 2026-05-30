package shared

import (
	"context"
	"encoding/json"
	"os"

	"github.com/shhac/agent-stripe/internal/api"
	"github.com/shhac/agent-stripe/internal/config"
	"github.com/shhac/agent-stripe/internal/credential"
	agenterrors "github.com/shhac/agent-stripe/internal/errors"
	"github.com/shhac/agent-stripe/internal/output"
)

type ResolvedProfile struct {
	Alias            string
	Profile          config.Profile
	APIKey           string
	CredentialSource string
	BaseURL          string
}

func ResolveProfile(flags *GlobalFlags) (*ResolvedProfile, error) {
	cfg := config.Read()
	alias := flags.Profile
	if alias == "" {
		alias = os.Getenv("AGENT_STRIPE_PROFILE")
	}
	if alias == "" {
		alias = cfg.DefaultProfile
	}

	apiKey := flags.APIKey
	credentialSource := "flag"
	if apiKey == "" {
		apiKey = os.Getenv("STRIPE_API_KEY")
		credentialSource = "env"
	}
	if apiKey != "" {
		return &ResolvedProfile{
			Alias: alias,
			Profile: config.Profile{
				Context: firstNonEmpty(flags.Context, os.Getenv("STRIPE_CONTEXT")),
				APIVersion: firstNonEmpty(flags.APIVersion, os.Getenv("STRIPE_API_VERSION"),
					config.DefaultAPIVersion),
			},
			APIKey:           apiKey,
			CredentialSource: credentialSource,
			BaseURL:          firstNonEmpty(flags.BaseURL, os.Getenv("AGENT_STRIPE_BASE_URL")),
		}, nil
	}

	if alias == "" {
		return nil, agenterrors.New("No Stripe profile configured", agenterrors.FixableByHuman).
			WithHint("Run 'agent-stripe auth add <profile> --api-key <rk_or_sk>' or set STRIPE_API_KEY")
	}

	profile, ok := cfg.Profiles[alias]
	if !ok {
		return nil, agenterrors.Newf(agenterrors.FixableByHuman, "Profile %q is not configured", alias).
			WithHint("Run 'agent-stripe auth list' to see configured profiles")
	}
	apiKey, err := credential.Get(alias)
	if err != nil {
		return nil, agenterrors.Wrap(err, agenterrors.FixableByHuman).
			WithHint("Re-add the profile with 'agent-stripe auth add " + alias + " --api-key <key>'")
	}

	if flags.Context != "" {
		profile.Context = flags.Context
	}
	if flags.APIVersion != "" {
		profile.APIVersion = flags.APIVersion
	}
	if profile.APIVersion == "" {
		profile.APIVersion = config.DefaultAPIVersion
	}
	return &ResolvedProfile{
		Alias:            alias,
		Profile:          profile,
		APIKey:           apiKey,
		CredentialSource: "keychain",
		BaseURL:          firstNonEmpty(flags.BaseURL, os.Getenv("AGENT_STRIPE_BASE_URL")),
	}, nil
}

func WithClient(flags *GlobalFlags, fn func(context.Context, *api.Client) error) error {
	resolved, err := ResolveProfile(flags)
	if err != nil {
		output.WriteError(os.Stderr, err)
		return nil
	}
	if flags.Debug {
		WriteDebug(map[string]any{
			"@debug":            "client",
			"profile":           resolved.Alias,
			"credential_source": resolved.CredentialSource,
			"context":           resolved.Profile.Context,
			"api_version":       resolved.Profile.APIVersion,
			"base_url":          resolvedBaseURL(resolved.BaseURL),
			"timeout_ms":        flags.Timeout,
		})
	}
	ctx, cancel := ContextWithTimeout(context.Background(), flags.Timeout)
	defer cancel()

	client := api.NewClient(api.Options{
		APIKey:     resolved.APIKey,
		Context:    resolved.Profile.Context,
		APIVersion: resolved.Profile.APIVersion,
		BaseURL:    resolved.BaseURL,
	})
	client.SetDebug(flags.Debug)
	if err := fn(ctx, client); err != nil {
		output.WriteError(os.Stderr, err)
		return nil
	}
	return nil
}

func WriteDebug(fields map[string]any) {
	enc := json.NewEncoder(os.Stderr)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(fields)
}

func resolvedBaseURL(baseURL string) string {
	if baseURL == "" {
		return "https://api.stripe.com"
	}
	return baseURL
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
