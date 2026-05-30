package auth

import (
	"context"
	"encoding/json"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/api"
	"github.com/shhac/agent-stripe/internal/cli/shared"
	"github.com/shhac/agent-stripe/internal/config"
	"github.com/shhac/agent-stripe/internal/credential"
	agenterrors "github.com/shhac/agent-stripe/internal/errors"
	"github.com/shhac/agent-stripe/internal/output"
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

func registerUpdate(parent *cobra.Command) {
	var contextValue string
	var apiVersion string
	var clearContext bool
	var setDefault bool

	cmd := &cobra.Command{
		Use:   "update <profile>",
		Short: "Update non-secret profile metadata",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cmd.Flags().Changed("context") && !cmd.Flags().Changed("api-version") && !clearContext && !setDefault {
				output.WriteError(output.Stderr(), agenterrors.New("no profile updates requested", agenterrors.FixableByAgent).
					WithHint("Use --context, --clear-context, --api-version, or --default"))
				return nil
			}
			if clearContext && cmd.Flags().Changed("context") {
				output.WriteError(output.Stderr(), agenterrors.New("--context and --clear-context cannot be used together", agenterrors.FixableByAgent))
				return nil
			}

			alias := args[0]
			if err := config.UpdateProfile(alias, func(profile config.Profile) config.Profile {
				if cmd.Flags().Changed("context") {
					profile.Context = contextValue
				}
				if clearContext {
					profile.Context = ""
				}
				if cmd.Flags().Changed("api-version") {
					profile.APIVersion = apiVersion
				}
				return profile
			}); err != nil {
				output.WriteError(output.Stderr(), agenterrors.Wrap(err, agenterrors.FixableByHuman).
					WithHint("Run 'agent-stripe auth list' to see configured profiles"))
				return nil
			}
			if setDefault {
				if err := config.SetDefault(alias); err != nil {
					output.WriteError(output.Stderr(), agenterrors.Wrap(err, agenterrors.FixableByHuman))
					return nil
				}
			}
			cfg := config.Read()
			profile := cfg.Profiles[alias]
			shared.WriteItem(map[string]any{
				"status":      "updated",
				"profile":     alias,
				"default":     cfg.DefaultProfile == alias,
				"context":     profile.Context,
				"api_version": profile.APIVersion,
			}, "")
			return nil
		},
	}
	cmd.Flags().StringVar(&contextValue, "context", "", "Default Stripe-Context for this profile")
	cmd.Flags().BoolVar(&clearContext, "clear-context", false, "Clear this profile's default Stripe-Context")
	cmd.Flags().StringVar(&apiVersion, "api-version", "", "Default Stripe API version")
	cmd.Flags().BoolVar(&setDefault, "default", false, "Make this the default profile")
	parent.AddCommand(cmd)
}

func registerAdd(parent *cobra.Command) {
	var apiKey string
	var contextValue string
	var apiVersion string
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
					output.WriteError(output.Stderr(), err)
					return nil
				}
				apiKey = filledKey
			}
			if !shared.RequireFlag("api-key", apiKey, "Provide --api-key <rk_live...|rk_test...|sk_live...|sk_test...|sk_org...>") {
				return nil
			}

			storage, err := credentialStore(alias, apiKey)
			if err != nil {
				output.WriteError(output.Stderr(), err)
				return nil
			}

			if apiVersion == "" {
				apiVersion = config.DefaultAPIVersion
			}
			if err := config.StoreProfile(alias, config.Profile{Context: contextValue, APIVersion: apiVersion}); err != nil {
				output.WriteError(output.Stderr(), err)
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
				output.WriteError(output.Stderr(), err)
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
			if err := credentialRemove(alias); err != nil {
				output.WriteError(output.Stderr(), err)
				return nil
			}
			if err := config.RemoveProfile(alias); err != nil {
				output.WriteError(output.Stderr(), err)
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
