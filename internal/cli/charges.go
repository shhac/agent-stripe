package cli

import (
	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func registerCharges(root *cobra.Command, globals shared.GlobalsFunc) {
	registerResource(root, globals, resourceOptions{
		use:         "charges",
		short:       "Charge lookup and search",
		path:        "/v1/charges",
		idName:      "charge-id",
		idKind:      "charge",
		getShort:    "Retrieve a charge by ID",
		listShort:   "List charges",
		searchShort: "Search charges with Stripe Search Query Language",
		searchable:  true,
		searchHint:  "Use a Stripe search query, for example metadata['order_id']:'123'",
		usageText:   paymentsUsageText,
		expandGet:   true,
		listSummary: chargeListSummary,
		listFlags: []listFlag{
			{name: "customer", param: "customer", help: "Customer ID"},
			{name: "payment-intent", param: "payment_intent", help: "PaymentIntent ID"},
		},
	})
}
