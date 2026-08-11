package cli

import (
	"net/url"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func registerCheckoutSessions(root *cobra.Command, globals shared.GlobalsFunc) {
	registerResource(root, globals, resourceOptions{
		use:         "checkout-sessions",
		aliases:     []string{"checkout_sessions"},
		short:       "Checkout Session lookup and line items",
		idName:      "checkout-session-id",
		idKind:      "checkout_session",
		getShort:    "Retrieve a Checkout Session",
		listShort:   "List Checkout Sessions",
		expandGet:   true,
		listSummary: checkoutSessionListSummary,
		listFlags: []listFlag{
			{name: "customer", param: "customer", help: "Customer ID"},
			{name: "payment-intent", param: "payment_intent", help: "PaymentIntent ID"},
			{name: "subscription", param: "subscription", help: "Subscription ID"},
			{name: "payment-link", param: "payment_link", help: "Payment Link ID"},
		},
		extraCommands: []func(shared.GlobalsFunc) *cobra.Command{
			newCheckoutSessionLineItemsCommand,
		},
	})
}

func newCheckoutSessionLineItemsCommand(globals shared.GlobalsFunc) *cobra.Command {
	var limit int
	var cursor cursorFlags
	lineItems := &cobra.Command{
		Use:   "line-items <checkout-session-id>",
		Short: "List Checkout Session line items",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateExpectedStripeID(args[0], "checkout_session"); err != nil {
				return err
			}
			params := url.Values{}
			shared.AddLimit(params, limit)
			cursor.AddTo(params)
			return shared.GetRawList(globals(), "/v1/checkout/sessions/"+url.PathEscape(args[0])+"/line_items", params)
		},
	}
	lineItems.Flags().IntVar(&limit, "limit", 10, "Maximum results to return (1-100)")
	cursor.AddFlags(lineItems)
	return lineItems
}

func checkoutSessionListSummary(session map[string]any) map[string]any {
	summary := map[string]any{}
	copyString(summary, session, "id")
	copyString(summary, session, "object")
	copyNumber(summary, session, "created")
	copyString(summary, session, "status")
	copyString(summary, session, "mode")
	copyString(summary, session, "payment_status")
	copyString(summary, session, "customer")
	copyExpandableID(summary, session, "payment_intent")
	copyExpandableID(summary, session, "subscription")
	copyExpandableID(summary, session, "payment_link")
	copyNumber(summary, session, "amount_total")
	copyString(summary, session, "currency")
	copyNumber(summary, session, "expires_at")
	return summary
}
