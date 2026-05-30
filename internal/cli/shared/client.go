package shared

import (
	"context"
	"os"

	"github.com/shhac/agent-stripe/internal/api"
	"github.com/shhac/agent-stripe/internal/config"
	"github.com/shhac/agent-stripe/internal/credential"
	agenterrors "github.com/shhac/agent-stripe/internal/errors"
	"github.com/shhac/agent-stripe/internal/output"
)

func ResolveProfile(flags *GlobalFlags) (string, config.Profile, string, error) {
	cfg := config.Read()
	alias := flags.Profile
	if alias == "" {
		alias = os.Getenv("AGENT_STRIPE_PROFILE")
	}
	if alias == "" {
		alias = cfg.DefaultProfile
	}

	apiKey := flags.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("STRIPE_API_KEY")
	}
	if apiKey != "" {
		return alias, config.Profile{
			Context: firstNonEmpty(flags.Context, os.Getenv("STRIPE_CONTEXT")),
			APIVersion: firstNonEmpty(flags.APIVersion, os.Getenv("STRIPE_API_VERSION"),
				config.DefaultAPIVersion),
		}, apiKey, nil
	}

	if alias == "" {
		return "", config.Profile{}, "", agenterrors.New("No Stripe profile configured", agenterrors.FixableByHuman).
			WithHint("Run 'agent-stripe auth add <profile> --api-key <rk_or_sk>' or set STRIPE_API_KEY")
	}

	profile, ok := cfg.Profiles[alias]
	if !ok {
		return "", config.Profile{}, "", agenterrors.Newf(agenterrors.FixableByHuman, "Profile %q is not configured", alias).
			WithHint("Run 'agent-stripe auth list' to see configured profiles")
	}
	apiKey, err := credential.Get(alias)
	if err != nil {
		return "", config.Profile{}, "", agenterrors.Wrap(err, agenterrors.FixableByHuman).
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
	return alias, profile, apiKey, nil
}

func WithClient(flags *GlobalFlags, fn func(context.Context, *api.Client) error) error {
	_, profile, apiKey, err := ResolveProfile(flags)
	if err != nil {
		output.WriteError(os.Stderr, err)
		return nil
	}
	ctx, cancel := ContextWithTimeout(context.Background(), flags.Timeout)
	defer cancel()

	client := api.NewClient(api.Options{
		APIKey:     apiKey,
		Context:    profile.Context,
		APIVersion: profile.APIVersion,
	})
	client.SetDebug(flags.Debug)
	if err := fn(ctx, client); err != nil {
		output.WriteError(os.Stderr, err)
		return nil
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
