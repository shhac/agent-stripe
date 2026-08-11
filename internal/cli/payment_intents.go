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
		idName:      "payment-intent-id",
		idKind:      "payment_intent",
		getShort:    "Retrieve a PaymentIntent by ID",
		listShort:   "List PaymentIntents; use search for richer filters",
		searchShort: "Search PaymentIntents with Stripe Search Query Language",
		searchable:  true,
		searchHint:  "Use a Stripe search query, for example metadata['order_id']:'123'",
		usageText:   paymentsUsageText,
		expandGet:   true,
		listSummary: paymentIntentListSummary,
		listFlags: []listFlag{
			{name: "customer", param: "customer", help: "Customer ID"},
		},
	})
}

func paymentIntentListSummary(pi map[string]any) map[string]any {
	summary := map[string]any{}
	copyString(summary, pi, "id")
	copyString(summary, pi, "object")
	copyNumber(summary, pi, "created")
	copyNumber(summary, pi, "amount")
	copyNumber(summary, pi, "amount_received")
	copyNumber(summary, pi, "amount_capturable")
	copyString(summary, pi, "currency")
	copyString(summary, pi, "status")
	copyString(summary, pi, "customer")
	copyString(summary, pi, "payment_method")
	copyExpandableID(summary, pi, "latest_charge")
	copyString(summary, pi, "invoice")
	copyString(summary, pi, "capture_method")
	copySubset(summary, pi, "last_payment_error", "code", "decline_code", "type", "payment_method_type", "message")
	return summary
}
