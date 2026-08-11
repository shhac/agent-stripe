package cli

import (
	"net/url"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func registerSubscriptions(root *cobra.Command, globals shared.GlobalsFunc) {
	registerResource(root, globals, resourceOptions{
		use:         "subscriptions",
		aliases:     []string{"subs"},
		short:       "Subscription lifecycle, invoice, and item investigation",
		idName:      "subscription-id",
		idKind:      "subscription",
		getShort:    "Retrieve a subscription by ID",
		listShort:   "List subscriptions; defaults to NDJSON",
		searchShort: "Search subscriptions with Stripe Search Query Language",
		searchable:  true,
		searchHint:  "Use a Stripe search query, for example metadata['tenant_id']:'acme'",
		usageText:   subscriptionsUsageText,
		expandGet:   true,
		listSummary: subscriptionListSummary,
		listFlags: []listFlag{
			{name: "customer", param: "customer", help: "Customer ID"},
			{name: "status", param: "status", help: "Subscription status: incomplete, trialing, active, past_due, canceled, unpaid, paused, all, ended"},
			{name: "price", param: "price", help: "Only subscriptions containing this Price ID"},
			{name: "current-period-end-gte", param: "current_period_end[gte]", help: "Minimum item current period end at or after Unix timestamp"},
			{name: "current-period-end-lte", param: "current_period_end[lte]", help: "Minimum item current period end at or before Unix timestamp"},
		},
		extraCommands: []func(shared.GlobalsFunc) *cobra.Command{
			newSubscriptionItemsCommand,
			newSubscriptionInvoicesCommand,
		},
	})
}

func newSubscriptionItemsCommand(globals shared.GlobalsFunc) *cobra.Command {
	var limit int
	var cursor cursorFlags
	var expand []string
	cmd := &cobra.Command{
		Use:   "items <subscription-id>",
		Short: "List subscription items for a subscription",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateExpectedStripeID(args[0], "subscription"); err != nil {
				return err
			}
			params := url.Values{"subscription": []string{args[0]}}
			shared.AddLimit(params, limit)
			cursor.AddTo(params)
			shared.AddExpand(params, expand)
			return shared.GetRawList(globals(), "/v1/subscription_items", params)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 10, "Maximum results to return (1-100)")
	cursor.AddFlags(cmd)
	cmd.Flags().StringArrayVar(&expand, "expand", nil, "Expand response property; repeatable")
	return cmd
}

func newSubscriptionInvoicesCommand(globals shared.GlobalsFunc) *cobra.Command {
	var limit int
	var invoiceStatus string
	var cursor cursorFlags
	cmd := &cobra.Command{
		Use:   "invoices <subscription-id>",
		Short: "List invoices for a subscription",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateExpectedStripeID(args[0], "subscription"); err != nil {
				return err
			}
			params := url.Values{"subscription": []string{args[0]}}
			shared.AddLimit(params, limit)
			shared.AddString(params, "status", invoiceStatus)
			cursor.AddTo(params)
			return shared.GetRawList(globals(), "/v1/invoices", params)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 10, "Maximum results to return (1-100)")
	cmd.Flags().StringVar(&invoiceStatus, "status", "", "Invoice status: draft, open, paid, uncollectible, void")
	cursor.AddFlags(cmd)
	return cmd
}

const subscriptionsUsageText = `subscriptions — renewal, collection, and item triage

COMMON STARTS
  agent-stripe subscriptions get sub_... --expand latest_invoice --expand latest_invoice.payment_intent
  agent-stripe subscriptions list --customer cus_... --status active|past_due|unpaid|all
  agent-stripe subscriptions invoices sub_... --status open
  agent-stripe subscriptions items sub_... --expand data.price.product
  agent-stripe investigate subscription-renewal --subscription sub_...
  agent-stripe investigate subscription-renewal --customer cus_...
  agent-stripe investigate subscription-renewal --metadata tenant_id=acme
  agent-stripe investigate subscription-items --subscription sub_...
  agent-stripe investigate subscription-amount-change --subscription sub_...
  agent-stripe investigate collection-risk --days 30
  agent-stripe investigate subscription-cancel-risk --days 30

QUESTIONS THIS ANSWERS
  Last paid invoice, latest PaymentIntent/Charge, next renewal time, next preview amount.
  Which Price/Product/metadata drove a subscription charge.
  Whether the customer needs outreach for missing, expiring, declined, or action-required payment details.

OUTPUT NOTES
  Investigation output emits subscription, invoice, payment, item, price, and product evidence records.
  Subscription lists are compact by default; use subscriptions list --full or subscriptions get sub_... for full objects.
  Use --full or --expand-field for verbose item/product metadata.
  Redaction is independent from truncation; use --expose for redacted fields.
`
