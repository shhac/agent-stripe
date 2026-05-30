package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/auth"
	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func newRootCmd(version string) *cobra.Command {
	globals := &shared.GlobalFlags{}
	globalsFunc := func() *shared.GlobalFlags {
		return globals
	}
	root := &cobra.Command{
		Use:           "agent-stripe",
		Short:         "Stripe incident triage CLI for AI agents",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().StringVarP(&globals.Profile, "profile", "p", "", "Stripe profile alias (or AGENT_STRIPE_PROFILE)")
	root.PersistentFlags().StringVar(&globals.Context, "context", "", "Stripe-Context value for organization or related-account requests")
	root.PersistentFlags().StringVar(&globals.APIKey, "api-key", "", "API key override; never printed or persisted")
	root.PersistentFlags().StringVar(&globals.BaseURL, "base-url", "", "Stripe API base URL override for tests")
	root.PersistentFlags().StringVarP(&globals.Format, "format", "f", "", "Output format: json, yaml, jsonl")
	root.PersistentFlags().IntVarP(&globals.Timeout, "timeout", "t", 0, "Request timeout in milliseconds")
	root.PersistentFlags().StringVar(&globals.APIVersion, "api-version", "", "Stripe API version header override")
	root.PersistentFlags().BoolVarP(&globals.Debug, "debug", "d", false, "Log HTTP requests and responses to stderr")
	_ = root.PersistentFlags().MarkHidden("base-url")

	registerUsageCommand(root)
	registerPaymentsDomain(root)
	registerConnectDomain(root)
	auth.Register(root, globalsFunc)
	registerBalance(root, globalsFunc)
	registerCheckoutSessions(root, globalsFunc)
	registerCustomers(root, globalsFunc)
	registerEvents(root, globalsFunc)
	registerProducts(root, globalsFunc)
	registerPrices(root, globalsFunc)
	registerInvoices(root, globalsFunc)
	registerPaymentIntents(root, globalsFunc)
	registerSetupIntents(root, globalsFunc)
	registerCharges(root, globalsFunc)
	registerDisputes(root, globalsFunc)
	registerPaymentMethods(root, globalsFunc)
	registerRefunds(root, globalsFunc)
	registerSubscriptions(root, globalsFunc)
	registerTransfers(root, globalsFunc)
	registerPayouts(root, globalsFunc)
	registerBalanceTransactions(root, globalsFunc)
	registerApplicationFees(root, globalsFunc)
	registerPaymentLinks(root, globalsFunc)
	registerEarlyFraudWarnings(root, globalsFunc)
	registerAccounts(root, globalsFunc)
	registerInvestigate(root, globalsFunc)
	registerRawAPI(root, globalsFunc)

	return root
}

func Execute(version string) error {
	err := newRootCmd(version).Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	return err
}
