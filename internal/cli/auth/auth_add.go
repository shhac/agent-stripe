package auth

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
	"github.com/shhac/agent-stripe/internal/config"
	"github.com/shhac/agent-stripe/internal/credential"
)

// auth add, in the same request/apply/output shape as auth update — the two
// mutating commands are now symmetric, and the one that writes a credential is
// as easy to find as the one that replaces it.

func registerAdd(parent *cobra.Command) {
	var opts authAddOptions

	cmd := &cobra.Command{
		Use:   "add <profile>",
		Short: "Add a Stripe profile with a Keychain-stored API key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := applyAuthAdd(cmd.Context(), args[0], opts)
			if err != nil {
				return err
			}
			shared.WriteItem(result.output(), "")
			return nil
		},
	}
	cmd.Flags().StringVar(&opts.apiKey, "api-key", "", "Stripe restricted, secret, or organization API key (required)")
	cmd.Flags().StringVar(&opts.contextValue, "context", "", "Default Stripe-Context for this profile")
	cmd.Flags().StringVar(&opts.apiVersion, "api-version", config.DefaultAPIVersion, "Default Stripe API version for /v1 requests")
	cmd.Flags().StringVar(&opts.v2APIVersion, "v2-api-version", config.DefaultV2APIVersion, "Default Stripe API version for /v2 requests (Accounts v2, v2 core events)")
	cmd.Flags().BoolVar(&opts.form, "form", false, "Prompt for the API key via a native OS dialog (LLM never sees the input)")
	parent.AddCommand(cmd)
}

type authAddOptions struct {
	apiKey       string
	contextValue string
	apiVersion   string
	v2APIVersion string
	form         bool
}

type authAddResult struct {
	alias   string
	profile config.Profile
	storage string
}

// applyAuthAdd separates the effects — prompt, keychain write, config write —
// from the command wiring, matching the shape `auth update` already uses. This
// is the higher-risk of the two commands, since it is the one that writes a
// credential, and it had the weaker seam.
func applyAuthAdd(ctx context.Context, alias string, opts authAddOptions) (authAddResult, error) {
	apiKey := opts.apiKey
	if opts.form {
		filledKey, err := promptAPIKeyViaDialog(ctx, alias, apiKey)
		if err != nil {
			return authAddResult{}, err
		}
		apiKey = filledKey
	}
	if err := shared.RequireFlag("api-key", apiKey, "Provide --api-key <rk_live...|rk_test...|sk_live...|sk_test...|sk_org...>"); err != nil {
		return authAddResult{}, err
	}

	storage, err := credentialStore(alias, apiKey)
	if err != nil {
		return authAddResult{}, err
	}

	// StoreProfile fills the version defaults, so the receipt reads the stored
	// profile back rather than re-deriving them here.
	if err := config.StoreProfile(alias, config.Profile{
		Context:        opts.contextValue,
		APIVersion:     opts.apiVersion,
		V2APIVersion:   opts.v2APIVersion,
		CredentialType: credential.Type(apiKey),
	}); err != nil {
		return authAddResult{}, err
	}
	return authAddResult{alias: alias, profile: config.Read().Profiles[alias], storage: storage}, nil
}

func (r authAddResult) output() map[string]any {
	fields := map[string]any{
		"status":         "added",
		"profile":        r.alias,
		"storage":        r.storage,
		"context":        r.profile.Context,
		"api_version":    r.profile.APIVersion,
		"v2_api_version": r.profile.V2APIVersion,
	}
	addCredentialType(fields, r.profile)
	return fields
}
