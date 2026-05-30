package cli

import (
	"net/url"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/api"
	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func registerSubscriptions(root *cobra.Command, globals shared.GlobalsFunc) {
	subscriptions := &cobra.Command{
		Use:     "subscriptions",
		Aliases: []string{"subs"},
		Short:   "Subscription lifecycle, invoice, and item investigation",
	}
	subscriptions.AddCommand(newSubscriptionGetCommand(globals))
	subscriptions.AddCommand(newSubscriptionListCommand(globals))
	subscriptions.AddCommand(newSubscriptionSearchCommand(globals))
	subscriptions.AddCommand(newSubscriptionItemsCommand(globals))
	subscriptions.AddCommand(newSubscriptionInvoicesCommand(globals))
	root.AddCommand(subscriptions)
}

func newSubscriptionGetCommand(globals shared.GlobalsFunc) *cobra.Command {
	var expand []string
	cmd := &cobra.Command{
		Use:   "get <subscription-id>",
		Short: "Retrieve a subscription by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			params := url.Values{}
			api.AddExpand(params, expand)
			return shared.GetRawItem(globals(), "/v1/subscriptions/"+url.PathEscape(args[0]), params)
		},
	}
	cmd.Flags().StringArrayVar(&expand, "expand", nil, "Expand response property; repeatable")
	return cmd
}

func newSubscriptionListCommand(globals shared.GlobalsFunc) *cobra.Command {
	var limit int
	var customer string
	var status string
	var price string
	var createdGTE string
	var createdLTE string
	var currentPeriodEndGTE string
	var currentPeriodEndLTE string
	var startingAfter string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List subscriptions; defaults to NDJSON",
		RunE: func(cmd *cobra.Command, args []string) error {
			params := url.Values{}
			api.AddLimit(params, limit)
			api.AddCreatedRange(params, createdGTE, createdLTE)
			shared.AddString(params, "customer", customer)
			shared.AddString(params, "status", status)
			shared.AddString(params, "price", price)
			shared.AddString(params, "current_period_end[gte]", currentPeriodEndGTE)
			shared.AddString(params, "current_period_end[lte]", currentPeriodEndLTE)
			shared.AddString(params, "starting_after", startingAfter)
			return shared.GetRawList(globals(), "/v1/subscriptions", params)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 10, "Maximum results to return (1-100)")
	cmd.Flags().StringVar(&customer, "customer", "", "Customer ID")
	cmd.Flags().StringVar(&status, "status", "", "Subscription status: incomplete, trialing, active, past_due, canceled, unpaid, paused, all, ended")
	cmd.Flags().StringVar(&price, "price", "", "Only subscriptions containing this Price ID")
	cmd.Flags().StringVar(&createdGTE, "created-gte", "", "Created at or after Unix timestamp")
	cmd.Flags().StringVar(&createdLTE, "created-lte", "", "Created at or before Unix timestamp")
	cmd.Flags().StringVar(&currentPeriodEndGTE, "current-period-end-gte", "", "Minimum item current period end at or after Unix timestamp")
	cmd.Flags().StringVar(&currentPeriodEndLTE, "current-period-end-lte", "", "Minimum item current period end at or before Unix timestamp")
	cmd.Flags().StringVar(&startingAfter, "starting-after", "", "Stripe cursor")
	return cmd
}

func newSubscriptionSearchCommand(globals shared.GlobalsFunc) *cobra.Command {
	var limit int
	var query string
	var page string
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search subscriptions with Stripe Search Query Language",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !shared.RequireFlag("query", query, "Use a Stripe search query, for example metadata['tenant_id']:'acme'") {
				return nil
			}
			params := url.Values{"query": []string{query}}
			api.AddLimit(params, limit)
			shared.AddString(params, "page", page)
			return shared.GetRawList(globals(), "/v1/subscriptions/search", params)
		},
	}
	cmd.Flags().StringVar(&query, "query", "", "Stripe search query")
	cmd.Flags().IntVar(&limit, "limit", 10, "Maximum results to return (1-100)")
	cmd.Flags().StringVar(&page, "page", "", "Search pagination cursor")
	return cmd
}

func newSubscriptionItemsCommand(globals shared.GlobalsFunc) *cobra.Command {
	var limit int
	var expand []string
	cmd := &cobra.Command{
		Use:   "items <subscription-id>",
		Short: "List subscription items for a subscription",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			params := url.Values{"subscription": []string{args[0]}}
			api.AddLimit(params, limit)
			api.AddExpand(params, expand)
			return shared.GetRawList(globals(), "/v1/subscription_items", params)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 10, "Maximum results to return (1-100)")
	cmd.Flags().StringArrayVar(&expand, "expand", nil, "Expand response property; repeatable")
	return cmd
}

func newSubscriptionInvoicesCommand(globals shared.GlobalsFunc) *cobra.Command {
	var limit int
	var invoiceStatus string
	var startingAfter string
	cmd := &cobra.Command{
		Use:   "invoices <subscription-id>",
		Short: "List invoices for a subscription",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			params := url.Values{"subscription": []string{args[0]}}
			api.AddLimit(params, limit)
			shared.AddString(params, "status", invoiceStatus)
			shared.AddString(params, "starting_after", startingAfter)
			return shared.GetRawList(globals(), "/v1/invoices", params)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 10, "Maximum results to return (1-100)")
	cmd.Flags().StringVar(&invoiceStatus, "status", "", "Invoice status: draft, open, paid, uncollectible, void")
	cmd.Flags().StringVar(&startingAfter, "starting-after", "", "Stripe cursor")
	return cmd
}
