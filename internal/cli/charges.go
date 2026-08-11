package cli

import (
	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func registerCharges(root *cobra.Command, globals shared.GlobalsFunc) {
	registerResource(root, globals, resourceOptions{
		use:         "charges",
		short:       "Charge lookup and search",
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

func chargeListSummary(charge map[string]any) map[string]any {
	summary := map[string]any{}
	copyString(summary, charge, "id")
	copyString(summary, charge, "object")
	copyNumber(summary, charge, "created")
	copyNumber(summary, charge, "amount")
	copyNumber(summary, charge, "amount_captured")
	copyNumber(summary, charge, "amount_refunded")
	copyString(summary, charge, "currency")
	copyString(summary, charge, "status")
	copyBool(summary, charge, "paid")
	copyBool(summary, charge, "refunded")
	copyBool(summary, charge, "disputed")
	copyString(summary, charge, "customer")
	copyString(summary, charge, "payment_method")
	copyExpandableID(summary, charge, "payment_intent")
	copyExpandableID(summary, charge, "balance_transaction")
	copyString(summary, charge, "failure_code")
	copyString(summary, charge, "failure_message")
	if details, ok := charge["payment_method_details"].(map[string]any); ok {
		out := map[string]any{}
		copyString(out, details, "type")
		if card, ok := details["card"].(map[string]any); ok {
			out["card"] = cardSummary(card)
		}
		if len(out) > 0 {
			summary["payment_method_details"] = out
		}
	}
	copySubset(summary, charge, "outcome", "type", "network_status", "risk_level", "seller_message")
	return summary
}
