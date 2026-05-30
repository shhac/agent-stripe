package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/auth"
	"github.com/shhac/agent-stripe/internal/cli/shared"
)

var (
	flagProfile    string
	flagContext    string
	flagAPIKey     string
	flagFormat     string
	flagTimeout    int
	flagDebug      bool
	flagAPIVersion string
)

func allGlobals() *shared.GlobalFlags {
	return &shared.GlobalFlags{
		Profile:    flagProfile,
		Context:    flagContext,
		APIKey:     flagAPIKey,
		Format:     flagFormat,
		Timeout:    flagTimeout,
		Debug:      flagDebug,
		APIVersion: flagAPIVersion,
	}
}

func newRootCmd(version string) *cobra.Command {
	root := &cobra.Command{
		Use:           "agent-stripe",
		Short:         "Stripe incident triage CLI for AI agents",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().StringVarP(&flagProfile, "profile", "p", "", "Stripe profile alias (or AGENT_STRIPE_PROFILE)")
	root.PersistentFlags().StringVar(&flagContext, "context", "", "Stripe-Context value for organization or related-account requests")
	root.PersistentFlags().StringVar(&flagAPIKey, "api-key", "", "API key override; never printed or persisted")
	root.PersistentFlags().StringVarP(&flagFormat, "format", "f", "", "Output format: json, yaml, jsonl")
	root.PersistentFlags().IntVarP(&flagTimeout, "timeout", "t", 0, "Request timeout in milliseconds")
	root.PersistentFlags().StringVar(&flagAPIVersion, "api-version", "", "Stripe API version header override")
	root.PersistentFlags().BoolVarP(&flagDebug, "debug", "d", false, "Log HTTP requests and responses to stderr")

	registerUsageCommand(root)
	auth.Register(root)
	registerBalance(root, allGlobals)
	registerEvents(root, allGlobals)
	registerPaymentIntents(root, allGlobals)
	registerCharges(root, allGlobals)
	registerDisputes(root, allGlobals)
	registerAccounts(root, allGlobals)
	registerRawAPI(root, allGlobals)

	return root
}

func Execute(version string) error {
	err := newRootCmd(version).Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	return err
}
