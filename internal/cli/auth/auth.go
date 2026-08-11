package auth

import (
	"context"
	"encoding/json"
	"net/url"
	"sort"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/api"
	"github.com/shhac/agent-stripe/internal/cli/shared"
	"github.com/shhac/agent-stripe/internal/config"
	"github.com/shhac/agent-stripe/internal/credential"
)

var (
	credentialStore  = credential.Store
	credentialRemove = credential.Remove
)

func Register(root *cobra.Command, globals shared.GlobalsFunc) {
	auth := &cobra.Command{
		Use:   "auth",
		Short: "Manage Stripe API credentials and profiles",
	}

	registerAdd(auth)
	registerCheck(auth, globals)
	registerDefault(auth)
	registerList(auth)
	registerRemove(auth)
	registerUpdate(auth)

	root.AddCommand(auth)
}

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

func registerCheck(parent *cobra.Command, globals shared.GlobalsFunc) {
	cmd := &cobra.Command{
		Use:   "check [profile]",
		Short: "Verify stored credentials by retrieving account details",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := globals()
			if len(args) > 0 {
				flags.Profile = args[0]
			}

			resolved, err := shared.ResolveProfile(flags)
			if err != nil {
				return err
			}
			credentialType := credential.Type(resolved.APIKey)
			metadataStatus := refreshStoredCredentialType(resolved, credentialType)
			return shared.WithResolvedClient(flags, resolved, func(ctx context.Context, client *api.Client) error {
				raw, err := client.Get(ctx, "/v1/account", url.Values{})
				if err != nil {
					return err
				}
				var account map[string]any
				if err := json.Unmarshal(raw, &account); err != nil {
					return err
				}
				item := map[string]any{
					"status":                     "ok",
					"profile":                    resolved.Alias,
					"credential_type":            credentialType,
					"credential_metadata_status": metadataStatus,
					"account":                    account,
				}
				addCredentialTypeHint(item, credentialType, false, resolved.Alias)
				shared.WriteItem(item, "")
				return nil
			})
		},
	}
	parent.AddCommand(cmd)
}

func registerDefault(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "default <profile>",
		Short: "Set the default profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			alias := args[0]
			if err := config.SetDefault(alias); err != nil {
				return err
			}
			shared.WriteItem(map[string]any{
				"status":  "default_set",
				"profile": alias,
			}, "")
			return nil
		},
	}
	parent.AddCommand(cmd)
}

// credentialBackend reports where a profile's key actually lives. It is looked
// up rather than assumed: on a host without a usable keychain the key is in the
// 0600 index file, and printing "keychain" there understates the exposure.
func credentialBackend(alias string) string {
	backend, err := credential.Backend(alias)
	if err != nil {
		return "unknown"
	}
	return backend
}

func registerList(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List configured profiles without exposing secrets",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Read()
			profiles := make([]map[string]any, 0, len(cfg.Profiles))
			for alias, profile := range cfg.Profiles {
				item := map[string]any{
					"profile":        alias,
					"default":        alias == cfg.DefaultProfile,
					"context":        profile.Context,
					"api_version":    profile.APIVersion,
					"v2_api_version": profile.V2APIVersion,
					"credential":     credentialBackend(alias),
				}
				addCredentialType(item, profile)
				addCredentialTypeHint(item, item["credential_type"].(string), profile.CredentialType == "", alias)
				profiles = append(profiles, item)
			}
			sort.Slice(profiles, func(i, j int) bool {
				return profiles[i]["profile"].(string) < profiles[j]["profile"].(string)
			})
			shared.WritePaginatedList(shared.ToAnySlice(profiles), nil, "")
			return nil
		},
	}
	parent.AddCommand(cmd)
}

func registerRemove(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "remove <profile>",
		Short: "Remove a profile and its Keychain credential",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			alias := args[0]
			if err := credentialRemove(alias); err != nil {
				return err
			}
			if err := config.RemoveProfile(alias); err != nil {
				return err
			}
			shared.WriteItem(map[string]any{
				"status":  "removed",
				"profile": alias,
			}, "")
			return nil
		},
	}
	parent.AddCommand(cmd)
}
