package cli

import (
	"net/url"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/api"
	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func registerDisputes(root *cobra.Command, globals shared.GlobalsFunc) {
	var limit int
	var charge string
	var paymentIntent string
	var createdGTE string
	var createdLTE string
	var startingAfter string

	disputes := &cobra.Command{
		Use:   "disputes",
		Short: "Dispute lookup and evidence-status triage",
	}
	disputes.AddCommand(&cobra.Command{
		Use:   "get <dispute-id>",
		Short: "Retrieve a dispute by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return shared.GetRawItem(globals(), "/v1/disputes/"+url.PathEscape(args[0]), url.Values{})
		},
	})
	list := &cobra.Command{
		Use:   "list",
		Short: "List disputes, optionally by charge or PaymentIntent",
		RunE: func(cmd *cobra.Command, args []string) error {
			params := url.Values{}
			api.AddLimit(params, limit)
			api.AddCreatedRange(params, createdGTE, createdLTE)
			shared.AddString(params, "charge", charge)
			shared.AddString(params, "payment_intent", paymentIntent)
			shared.AddString(params, "starting_after", startingAfter)
			return shared.GetRawList(globals(), "/v1/disputes", params)
		},
	}
	list.Flags().IntVar(&limit, "limit", 10, "Maximum results to return (1-100)")
	list.Flags().StringVar(&charge, "charge", "", "Charge ID")
	list.Flags().StringVar(&paymentIntent, "payment-intent", "", "PaymentIntent ID")
	list.Flags().StringVar(&createdGTE, "created-gte", "", "Created at or after Unix timestamp")
	list.Flags().StringVar(&createdLTE, "created-lte", "", "Created at or before Unix timestamp")
	list.Flags().StringVar(&startingAfter, "starting-after", "", "Stripe cursor")
	disputes.AddCommand(list)
	root.AddCommand(disputes)
}
