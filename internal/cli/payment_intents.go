package cli

import (
	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func registerPaymentIntents(root *cobra.Command, globals shared.GlobalsFunc) {
	registerResource(root, globals, resourceOptions{
		use:         "payment-intents",
		aliases:     []string{"payment_intents", "pis"},
		short:       "PaymentIntent lookup and search",
		path:        "/v1/payment_intents",
		idName:      "payment-intent-id",
		getShort:    "Retrieve a PaymentIntent by ID",
		listShort:   "List PaymentIntents; use search for richer filters",
		searchShort: "Search PaymentIntents with Stripe Search Query Language",
		searchable:  true,
		searchHint:  "Use a Stripe search query, for example metadata['order_id']:'123'",
		expandGet:   true,
		listFlags: []listFlag{
			{name: "customer", param: "customer", help: "Customer ID"},
		},
	})
}
