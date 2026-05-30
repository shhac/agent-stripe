package auth

import (
	"context"
	"encoding/json"
	"net/url"
	"os"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/api"
	"github.com/shhac/agent-stripe/internal/cli/shared"
	"github.com/shhac/agent-stripe/internal/config"
	"github.com/shhac/agent-stripe/internal/credential"
	"github.com/shhac/agent-stripe/internal/output"
)

func Register(root *cobra.Command) {
	auth := &cobra.Command{
		Use:   "auth",
		Short: "Manage Stripe API credentials and profiles",
	}

	registerAdd(auth)
	registerCheck(auth)
	registerDefault(auth)
	registerList(auth)
	registerRemove(auth)

	root.AddCommand(auth)
}

func registerAdd(parent *cobra.Command) {
	var apiKey string
	var contextValue string
	var apiVersion string

	cmd := &cobra.Command{
		Use:   "add <profile>",
		Short: "Add a Stripe profile with a Keychain-stored API key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			alias := args[0]
			if !shared.RequireFlag("api-key", apiKey, "Provide --api-key <rk_live...|rk_test...|sk_live...|sk_test...|sk_org...>") {
				return nil
			}

			storage, err := credential.Store(alias, apiKey)
			if err != nil {
				output.WriteError(os.Stderr, err)
				return nil
			}

			if apiVersion == "" {
				apiVersion = config.DefaultAPIVersion
			}
			if err := config.StoreProfile(alias, config.Profile{Context: contextValue, APIVersion: apiVersion}); err != nil {
				output.WriteError(os.Stderr, err)
				return nil
			}

			shared.WriteItem(map[string]any{
				"status":      "added",
				"profile":     alias,
				"storage":     storage,
				"context":     contextValue,
				"api_version": apiVersion,
			}, "")
			return nil
		},
	}
	cmd.Flags().StringVar(&apiKey, "api-key", "", "Stripe restricted, secret, or organization API key (required)")
	cmd.Flags().StringVar(&contextValue, "context", "", "Default Stripe-Context for this profile")
	cmd.Flags().StringVar(&apiVersion, "api-version", config.DefaultAPIVersion, "Default Stripe API version")
	parent.AddCommand(cmd)
}

func registerCheck(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "check [profile]",
		Short: "Verify stored credentials by retrieving account details",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := &shared.GlobalFlags{}
			if len(args) > 0 {
				flags.Profile = args[0]
			}

			return shared.WithClient(flags, func(ctx context.Context, client *api.Client) error {
				raw, err := client.Get(ctx, "/v1/account", url.Values{})
				if err != nil {
					return err
				}
				var account map[string]any
				if err := json.Unmarshal(raw, &account); err != nil {
					return err
				}
				shared.WriteItem(map[string]any{
					"status":  "ok",
					"account": account,
				}, "")
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
				output.WriteError(os.Stderr, err)
				return nil
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
				profiles = append(profiles, map[string]any{
					"profile":     alias,
					"default":     alias == cfg.DefaultProfile,
					"context":     profile.Context,
					"api_version": profile.APIVersion,
					"credential":  "keychain",
				})
			}
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
			if err := credential.Remove(alias); err != nil {
				output.WriteError(os.Stderr, err)
				return nil
			}
			if err := config.RemoveProfile(alias); err != nil {
				output.WriteError(os.Stderr, err)
				return nil
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
