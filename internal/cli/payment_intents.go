package cli

import (
	"net/url"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/api"
	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func registerPaymentIntents(root *cobra.Command, globals shared.GlobalsFunc) {
	var limit int
	var query string
	var page string
	var expand []string
	var customer string
	var createdGTE string
	var createdLTE string
	var startingAfter string

	pi := &cobra.Command{
		Use:     "payment-intents",
		Aliases: []string{"payment_intents", "pis"},
		Short:   "PaymentIntent lookup and search",
	}
	pi.AddCommand(&cobra.Command{
		Use:   "get <payment-intent-id>",
		Short: "Retrieve a PaymentIntent by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			params := url.Values{}
			api.AddExpand(params, expand)
			return shared.GetRawItem(globals(), "/v1/payment_intents/"+url.PathEscape(args[0]), params)
		},
	})

	search := &cobra.Command{
		Use:   "search",
		Short: "Search PaymentIntents with Stripe Search Query Language",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !shared.RequireFlag("query", query, "Use a Stripe search query, for example metadata['order_id']:'123'") {
				return nil
			}
			params := url.Values{"query": []string{query}}
			api.AddLimit(params, limit)
			shared.AddString(params, "page", page)
			return shared.GetRawList(globals(), "/v1/payment_intents/search", params)
		},
	}
	search.Flags().StringVar(&query, "query", "", "Stripe search query")
	search.Flags().IntVar(&limit, "limit", 10, "Maximum results to return (1-100)")
	search.Flags().StringVar(&page, "page", "", "Search pagination cursor")
	pi.AddCommand(search)

	list := &cobra.Command{
		Use:   "list",
		Short: "List PaymentIntents; use search for richer filters",
		RunE: func(cmd *cobra.Command, args []string) error {
			params := url.Values{}
			api.AddLimit(params, limit)
			api.AddCreatedRange(params, createdGTE, createdLTE)
			shared.AddString(params, "customer", customer)
			shared.AddString(params, "starting_after", startingAfter)
			return shared.GetRawList(globals(), "/v1/payment_intents", params)
		},
	}
	list.Flags().IntVar(&limit, "limit", 10, "Maximum results to return (1-100)")
	list.Flags().StringVar(&customer, "customer", "", "Customer ID")
	list.Flags().StringVar(&createdGTE, "created-gte", "", "Created at or after Unix timestamp")
	list.Flags().StringVar(&createdLTE, "created-lte", "", "Created at or before Unix timestamp")
	list.Flags().StringVar(&startingAfter, "starting-after", "", "Stripe cursor")
	pi.AddCommand(list)

	for _, cmd := range pi.Commands() {
		if cmd.Name() == "get" {
			cmd.Flags().StringArrayVar(&expand, "expand", nil, "Expand response property; repeatable")
		}
	}
	root.AddCommand(pi)
}
