package shared

import (
	"context"
	"encoding/json"
	"net/url"
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

var credentialGet = credential.Get

func ResolveProfile(flags *GlobalFlags) (*ResolvedProfile, error) {
	cfg := config.Read()
	alias := resolveAlias(flags, cfg)

	if apiKey, source := resolveDirectAPIKey(flags); apiKey != "" {
		return resolvedDirectProfile(alias, flags, apiKey, source), nil
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
	apiKey, err := credentialGet(alias)
	if err != nil {
		return nil, agenterrors.Wrap(err, agenterrors.FixableByHuman).
			WithHint("Re-add the profile with 'agent-stripe auth add " + alias + " --api-key <key>'")
	}

	profile = applyProfileOverrides(profile, flags)
	return &ResolvedProfile{
		Alias:            alias,
		Profile:          profile,
		APIKey:           apiKey,
		CredentialSource: "keychain",
		BaseURL:          resolveBaseURL(flags),
	}, nil
}

func resolveAlias(flags *GlobalFlags, cfg *config.Config) string {
	return firstNonEmpty(flags.Profile, os.Getenv("AGENT_STRIPE_PROFILE"), cfg.DefaultProfile)
}

func resolveDirectAPIKey(flags *GlobalFlags) (string, string) {
	if flags.APIKey != "" {
		return flags.APIKey, "flag"
	}
	if apiKey := os.Getenv("STRIPE_API_KEY"); apiKey != "" {
		return apiKey, "env"
	}
	return "", ""
}

func resolvedDirectProfile(alias string, flags *GlobalFlags, apiKey, source string) *ResolvedProfile {
	return &ResolvedProfile{
		Alias: alias,
		Profile: config.Profile{
			Context:    firstNonEmpty(flags.Context, os.Getenv("STRIPE_CONTEXT")),
			APIVersion: firstNonEmpty(flags.APIVersion, os.Getenv("STRIPE_API_VERSION"), config.DefaultAPIVersion),
		},
		APIKey:           apiKey,
		CredentialSource: source,
		BaseURL:          resolveBaseURL(flags),
	}
}

func applyProfileOverrides(profile config.Profile, flags *GlobalFlags) config.Profile {
	if flags.Context != "" {
		profile.Context = flags.Context
	}
	if flags.APIVersion != "" {
		profile.APIVersion = flags.APIVersion
	}
	if profile.APIVersion == "" {
		profile.APIVersion = config.DefaultAPIVersion
	}
	return profile
}

func resolveBaseURL(flags *GlobalFlags) string {
	return firstNonEmpty(flags.BaseURL, os.Getenv("AGENT_STRIPE_BASE_URL"))
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

func GetRawItem(flags *GlobalFlags, path string, params url.Values) error {
	return WithClient(flags, func(ctx context.Context, client *api.Client) error {
		raw, err := client.Get(ctx, path, params)
		if err != nil {
			return err
		}
		WriteRawItem(raw, flags.Format)
		return nil
	})
}

func GetRawList(flags *GlobalFlags, path string, params url.Values) error {
	return WithClient(flags, func(ctx context.Context, client *api.Client) error {
		raw, err := client.Get(ctx, path, params)
		if err != nil {
			return err
		}
		return WriteRawList(raw, flags.Format)
	})
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
