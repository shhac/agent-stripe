package cli

import (
	"net/url"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func registerBalance(root *cobra.Command, globals shared.GlobalsFunc) {
	balance := &cobra.Command{
		Use:   "balance",
		Short: "Balance and funds availability",
	}
	balance.AddCommand(&cobra.Command{
		Use:   "get",
		Short: "Retrieve the current Stripe balance",
		RunE: func(cmd *cobra.Command, args []string) error {
			return shared.GetRawItem(globals(), "/v1/balance", url.Values{})
		},
	})
	root.AddCommand(balance)
}
