package cli

import (
	"net/url"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

// newAccountsV2PayoutMethodsCommand reads the payout methods belonging to a
// recipient-configured v2 account. These answer "the recipient has an active
// payout capability but hasn't been paid — where would the money go?", which
// neither the account object nor /v1/payouts can tell you for a UA2 recipient.
//
// The endpoint is addressed by Stripe-Context, not by a path segment, so the
// account ID is taken as an argument and applied to the request's context.
func newAccountsV2PayoutMethodsCommand(globals shared.GlobalsFunc) *cobra.Command {
	var limit int
	var page string
	cmd := &cobra.Command{
		Use:   "payout-methods <account-id>",
		Short: "List a v2 recipient's payout methods (preview API)",
		Long: "Lists /v2/money_management/payout_methods scoped to the recipient account.\n" +
			"This surface is preview-only, so it needs a preview train:\n" +
			"  agent-stripe accounts-v2 payout-methods acct_... --v2-api-version 2026-06-24.preview",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateExpectedStripeID(args[0], "account"); err != nil {
				return err
			}
			params := url.Values{}
			shared.AddLimit(params, limit)
			shared.AddString(params, "page", page)
			return shared.GetRawListInContext(globals(), args[0], "/v2/money_management/payout_methods", params)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 10, "Maximum results to return")
	cmd.Flags().StringVar(&page, "page", "", "Page token from a previous @pagination next_page")
	return cmd
}
