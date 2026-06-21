package cli

import (
	"context"
	"net/url"

	libcli "github.com/shhac/lib-agent-cli/cli"
	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/api"
	"github.com/shhac/agent-stripe/internal/cli/shared"
	"github.com/shhac/agent-stripe/internal/output"
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
			flags := globals()
			return shared.WithClient(flags, func(ctx context.Context, client *api.Client) error {
				obj, err := shared.FetchItem(ctx, client, flags, "/v1/balance", url.Values{})
				if err != nil {
					return err
				}
				return libcli.EmitItem(output.Stdout(), flags.Format, obj)
			})
		},
	})
	root.AddCommand(balance)
}
