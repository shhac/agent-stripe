package shared

import (
	"os"

	"github.com/shhac/agent-stripe/internal/config"
	"github.com/shhac/agent-stripe/internal/credential"
	agenterrors "github.com/shhac/agent-stripe/internal/errors"
	"github.com/shhac/lib-agent-cli/creds"
)

// Profile resolution: the precedence chain from flag to environment to stored
// profile to built-in default, for the API key, the Stripe context, and both
// API versions. It is the only code combining config, credential, and the
// environment, which is why it sits apart from client construction.

type ResolvedProfile struct {
	Alias            string
	Profile          config.Profile
	APIKey           string
	CredentialSource string
	BaseURL          string
}

var credentialGet = credential.GetWithBackend

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
	apiKey, backend, err := credentialGet(alias)
	if err != nil {
		return nil, agenterrors.Wrap(err, agenterrors.FixableByHuman).
			WithHint("Re-add the profile with 'agent-stripe auth add " + alias + " --api-key <key>'")
	}

	profile = applyProfileOverrides(profile, flags)
	return &ResolvedProfile{
		Alias:            alias,
		Profile:          profile,
		APIKey:           apiKey,
		CredentialSource: backend,
		BaseURL:          resolveBaseURL(flags),
	}, nil
}

func resolveAlias(flags *GlobalFlags, cfg *config.Config) string {
	return creds.FirstNonEmpty(flags.Profile, os.Getenv("AGENT_STRIPE_PROFILE"), cfg.DefaultProfile)
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
			Context:      creds.FirstNonEmpty(flags.Context, os.Getenv("STRIPE_CONTEXT")),
			APIVersion:   creds.FirstNonEmpty(flags.APIVersion, os.Getenv("STRIPE_API_VERSION"), config.DefaultAPIVersion),
			V2APIVersion: creds.FirstNonEmpty(flags.V2APIVersion, os.Getenv("STRIPE_V2_API_VERSION"), config.DefaultV2APIVersion),
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
	if flags.V2APIVersion != "" {
		profile.V2APIVersion = flags.V2APIVersion
	}
	if profile.APIVersion == "" {
		profile.APIVersion = config.DefaultAPIVersion
	}
	if profile.V2APIVersion == "" {
		profile.V2APIVersion = config.DefaultV2APIVersion
	}
	return profile
}

func resolveBaseURL(flags *GlobalFlags) string {
	return creds.FirstNonEmpty(flags.BaseURL, os.Getenv("AGENT_STRIPE_BASE_URL"))
}
