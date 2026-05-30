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
		path:        "/v1/subscriptions",
		idName:      "subscription-id",
		getShort:    "Retrieve a subscription by ID",
		listShort:   "List subscriptions; defaults to NDJSON",
		searchShort: "Search subscriptions with Stripe Search Query Language",
		searchable:  true,
		searchHint:  "Use a Stripe search query, for example metadata['tenant_id']:'acme'",
		usageText:   subscriptionsUsageText,
		expandGet:   true,
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
	var startingAfter string
	var endingBefore string
	var expand []string
	cmd := &cobra.Command{
		Use:   "items <subscription-id>",
		Short: "List subscription items for a subscription",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			params := url.Values{"subscription": []string{args[0]}}
			shared.AddLimit(params, limit)
			shared.AddString(params, "starting_after", startingAfter)
			shared.AddString(params, "ending_before", endingBefore)
			shared.AddExpand(params, expand)
			return shared.GetRawList(globals(), "/v1/subscription_items", params)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 10, "Maximum results to return (1-100)")
	cmd.Flags().StringVar(&startingAfter, "starting-after", "", "Stripe cursor")
	cmd.Flags().StringVar(&endingBefore, "ending-before", "", "Stripe cursor")
	markCursorFlagsMutuallyExclusive(cmd)
	cmd.Flags().StringArrayVar(&expand, "expand", nil, "Expand response property; repeatable")
	return cmd
}

func newSubscriptionInvoicesCommand(globals shared.GlobalsFunc) *cobra.Command {
	var limit int
	var invoiceStatus string
	var startingAfter string
	var endingBefore string
	cmd := &cobra.Command{
		Use:   "invoices <subscription-id>",
		Short: "List invoices for a subscription",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			params := url.Values{"subscription": []string{args[0]}}
			shared.AddLimit(params, limit)
			shared.AddString(params, "status", invoiceStatus)
			shared.AddString(params, "starting_after", startingAfter)
			shared.AddString(params, "ending_before", endingBefore)
			return shared.GetRawList(globals(), "/v1/invoices", params)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 10, "Maximum results to return (1-100)")
	cmd.Flags().StringVar(&invoiceStatus, "status", "", "Invoice status: draft, open, paid, uncollectible, void")
	cmd.Flags().StringVar(&startingAfter, "starting-after", "", "Stripe cursor")
	cmd.Flags().StringVar(&endingBefore, "ending-before", "", "Stripe cursor")
	markCursorFlagsMutuallyExclusive(cmd)
	return cmd
}
