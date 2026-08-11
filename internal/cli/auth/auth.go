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
	var apiKey string
	var contextValue string
	var apiVersion string
	var v2APIVersion string
	var form bool

	cmd := &cobra.Command{
		Use:   "add <profile>",
		Short: "Add a Stripe profile with a Keychain-stored API key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			alias := args[0]
			if form {
				filledKey, err := promptAPIKeyViaDialog(cmd.Context(), alias, apiKey)
				if err != nil {
					return err
				}
				apiKey = filledKey
			}
			if err := shared.RequireFlag("api-key", apiKey, "Provide --api-key <rk_live...|rk_test...|sk_live...|sk_test...|sk_org...>"); err != nil {
				return err
			}

			storage, err := credentialStore(alias, apiKey)
			if err != nil {
				return err
			}

			if apiVersion == "" {
				apiVersion = config.DefaultAPIVersion
			}
			if v2APIVersion == "" {
				v2APIVersion = config.DefaultV2APIVersion
			}
			credentialType := credential.Type(apiKey)
			if err := config.StoreProfile(alias, config.Profile{Context: contextValue, APIVersion: apiVersion, V2APIVersion: v2APIVersion, CredentialType: credentialType}); err != nil {
				return err
			}

			shared.WriteItem(map[string]any{
				"status":          "added",
				"profile":         alias,
				"storage":         storage,
				"context":         contextValue,
				"api_version":     apiVersion,
				"v2_api_version":  v2APIVersion,
				"credential_type": credentialType,
			}, "")
			return nil
		},
	}
	cmd.Flags().StringVar(&apiKey, "api-key", "", "Stripe restricted, secret, or organization API key (required)")
	cmd.Flags().StringVar(&contextValue, "context", "", "Default Stripe-Context for this profile")
	cmd.Flags().StringVar(&apiVersion, "api-version", config.DefaultAPIVersion, "Default Stripe API version for /v1 requests")
	cmd.Flags().StringVar(&v2APIVersion, "v2-api-version", config.DefaultV2APIVersion, "Default Stripe API version for /v2 requests (Accounts v2, v2 core events)")
	cmd.Flags().BoolVar(&form, "form", false, "Prompt for the API key via a native OS dialog (LLM never sees the input)")
	parent.AddCommand(cmd)
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
					"credential":     "keychain",
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
