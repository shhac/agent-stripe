package auth

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
	"github.com/shhac/agent-stripe/internal/config"
	"github.com/shhac/agent-stripe/internal/credential"
	agenterrors "github.com/shhac/agent-stripe/internal/errors"
)

func registerUpdate(parent *cobra.Command) {
	var apiKey string
	var contextValue string
	var apiVersion string
	var clearContext bool
	var setDefault bool
	var form bool

	cmd := &cobra.Command{
		Use:   "update <profile>",
		Short: "Update a profile key or non-secret metadata",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			req, err := newAuthUpdateRequest(cmd, args[0], authUpdateOptions{
				apiKey:       apiKey,
				contextValue: contextValue,
				apiVersion:   apiVersion,
				clearContext: clearContext,
				setDefault:   setDefault,
				form:         form,
			})
			if err != nil {
				return writeAuthError(err)
			}
			result, err := applyAuthUpdate(cmd.Context(), req)
			if err != nil {
				return writeAuthError(err)
			}
			shared.WriteItem(result.output(), "")
			return nil
		},
	}
	cmd.Flags().StringVar(&apiKey, "api-key", "", "Replacement Stripe restricted or secret API key")
	cmd.Flags().StringVar(&contextValue, "context", "", "Default Stripe-Context for this profile")
	cmd.Flags().BoolVar(&clearContext, "clear-context", false, "Clear this profile's default Stripe-Context")
	cmd.Flags().StringVar(&apiVersion, "api-version", "", "Default Stripe API version")
	cmd.Flags().BoolVar(&setDefault, "default", false, "Make this the default profile")
	cmd.Flags().BoolVar(&form, "form", false, "Prompt for the replacement API key via a native OS dialog")
	parent.AddCommand(cmd)
}

type authUpdateOptions struct {
	apiKey       string
	contextValue string
	apiVersion   string
	clearContext bool
	setDefault   bool
	form         bool
}

type authUpdateRequest struct {
	alias          string
	apiKey         string
	contextValue   string
	apiVersion     string
	contextChanged bool
	versionChanged bool
	clearContext   bool
	setDefault     bool
	keyRequested   bool
	form           bool
}

type authUpdateResult struct {
	alias        string
	profile      config.Profile
	isDefault    bool
	keyRequested bool
	storage      string
}

func newAuthUpdateRequest(cmd *cobra.Command, alias string, opts authUpdateOptions) (authUpdateRequest, error) {
	req := authUpdateRequest{
		alias:          alias,
		apiKey:         opts.apiKey,
		contextValue:   opts.contextValue,
		apiVersion:     opts.apiVersion,
		contextChanged: cmd.Flags().Changed("context"),
		versionChanged: cmd.Flags().Changed("api-version"),
		clearContext:   opts.clearContext,
		setDefault:     opts.setDefault,
		keyRequested:   cmd.Flags().Changed("api-key") || opts.form,
		form:           opts.form,
	}
	if !req.contextChanged && !req.versionChanged && !req.clearContext && !req.setDefault && !req.keyRequested {
		return authUpdateRequest{}, agenterrors.New("no profile updates requested", agenterrors.FixableByAgent).
			WithHint("Use --api-key, --form, --context, --clear-context, --api-version, or --default")
	}
	if req.clearContext && req.contextChanged {
		return authUpdateRequest{}, agenterrors.New("--context and --clear-context cannot be used together", agenterrors.FixableByAgent)
	}
	if _, ok := config.Read().Profiles[alias]; !ok {
		return authUpdateRequest{}, agenterrors.Newf(agenterrors.FixableByHuman, "Profile %q is not configured", alias).
			WithHint("Run 'agent-stripe auth list' to see configured profiles")
	}
	return req, nil
}

func applyAuthUpdate(ctx context.Context, req authUpdateRequest) (authUpdateResult, error) {
	storage := ""
	credentialType := ""
	if req.keyRequested {
		apiKey, err := replacementAPIKey(ctx, req)
		if err != nil {
			return authUpdateResult{}, err
		}
		var storeErr error
		storage, storeErr = credentialStore(req.alias, apiKey)
		if storeErr != nil {
			return authUpdateResult{}, storeErr
		}
		credentialType = credential.Type(apiKey)
	}
	if err := config.UpdateProfile(req.alias, func(profile config.Profile) config.Profile {
		if req.contextChanged {
			profile.Context = req.contextValue
		}
		if req.clearContext {
			profile.Context = ""
		}
		if req.versionChanged {
			profile.APIVersion = req.apiVersion
		}
		if req.keyRequested {
			profile.CredentialType = credentialType
		}
		return profile
	}); err != nil {
		return authUpdateResult{}, agenterrors.Wrap(err, agenterrors.FixableByHuman).
			WithHint("Run 'agent-stripe auth list' to see configured profiles")
	}
	if req.setDefault {
		if err := config.SetDefault(req.alias); err != nil {
			return authUpdateResult{}, agenterrors.Wrap(err, agenterrors.FixableByHuman)
		}
	}
	cfg := config.Read()
	return authUpdateResult{
		alias:        req.alias,
		profile:      cfg.Profiles[req.alias],
		isDefault:    cfg.DefaultProfile == req.alias,
		keyRequested: req.keyRequested,
		storage:      storage,
	}, nil
}

func replacementAPIKey(ctx context.Context, req authUpdateRequest) (string, error) {
	apiKey := req.apiKey
	if req.form {
		filledKey, err := promptAPIKeyViaDialog(ctx, req.alias, apiKey)
		if err != nil {
			return "", err
		}
		apiKey = filledKey
	}
	if apiKey == "" {
		return "", agenterrors.Newf(agenterrors.FixableByAgent, "--%s is required", "api-key").
			WithHint("Provide --api-key <rk_live...|rk_test...|sk_live...|sk_test...> or use --form")
	}
	return apiKey, nil
}

func (r authUpdateResult) output() map[string]any {
	fields := map[string]any{
		"status":      "updated",
		"profile":     r.alias,
		"default":     r.isDefault,
		"context":     r.profile.Context,
		"api_version": r.profile.APIVersion,
		"credential":  "keychain",
	}
	addCredentialType(fields, r.profile)
	if r.keyRequested {
		fields["storage"] = r.storage
	}
	return fields
}
