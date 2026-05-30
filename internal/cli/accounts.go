package cli

import (
	"net/url"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func registerAccounts(root *cobra.Command, globals shared.GlobalsFunc) {
	var limit int
	var startingAfter string
	var endingBefore string

	accounts := &cobra.Command{
		Use:   "accounts",
		Short: "Stripe account and connected-account lookup",
	}
	accounts.AddCommand(&cobra.Command{
		Use:   "self",
		Short: "Retrieve the account for the active key/context",
		RunE: func(cmd *cobra.Command, args []string) error {
			return shared.GetRawItem(globals(), "/v1/account", url.Values{})
		},
	})
	accounts.AddCommand(&cobra.Command{
		Use:   "get <account-id>",
		Short: "Retrieve a connected account by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return shared.GetRawItem(globals(), "/v1/accounts/"+url.PathEscape(args[0]), url.Values{})
		},
	})
	list := &cobra.Command{
		Use:   "list",
		Short: "List connected accounts",
		RunE: func(cmd *cobra.Command, args []string) error {
			params := url.Values{}
			shared.AddLimit(params, limit)
			shared.AddString(params, "starting_after", startingAfter)
			shared.AddString(params, "ending_before", endingBefore)
			return shared.GetRawList(globals(), "/v1/accounts", params)
		},
	}
	list.Flags().IntVar(&limit, "limit", 10, "Maximum results to return (1-100)")
	list.Flags().StringVar(&startingAfter, "starting-after", "", "Stripe cursor")
	list.Flags().StringVar(&endingBefore, "ending-before", "", "Stripe cursor")
	markCursorFlagsMutuallyExclusive(list)
	accounts.AddCommand(list)
	root.AddCommand(accounts)
}
