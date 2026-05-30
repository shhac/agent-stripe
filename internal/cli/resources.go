package cli

import (
	"context"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/api"
	"github.com/shhac/agent-stripe/internal/cli/shared"
	agenterrors "github.com/shhac/agent-stripe/internal/errors"
)

func registerBalance(root *cobra.Command, globals shared.GlobalsFunc) {
	balance := &cobra.Command{
		Use:   "balance",
		Short: "Balance and funds availability",
	}
	balance.AddCommand(&cobra.Command{
		Use:   "get",
		Short: "Retrieve the current Stripe balance",
		RunE: func(cmd *cobra.Command, args []string) error {
			return shared.WithClient(globals(), func(ctx context.Context, client *api.Client) error {
				raw, err := client.Get(ctx, "/v1/balance", url.Values{})
				if err != nil {
					return err
				}
				shared.WriteRawItem(raw, globals().Format)
				return nil
			})
		},
	})
	root.AddCommand(balance)
}

func registerEvents(root *cobra.Command, globals shared.GlobalsFunc) {
	var limit int
	var eventType string
	var eventTypes []string
	var createdGTE string
	var createdLTE string
	var startingAfter string
	var endingBefore string
	var deliverySuccess string

	events := &cobra.Command{
		Use:   "events",
		Short: "Stripe events and webhook-relevant activity",
	}
	events.AddCommand(&cobra.Command{
		Use:   "get <event-id>",
		Short: "Retrieve an event by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return shared.WithClient(globals(), func(ctx context.Context, client *api.Client) error {
				raw, err := client.Get(ctx, "/v1/events/"+url.PathEscape(args[0]), url.Values{})
				if err != nil {
					return err
				}
				shared.WriteRawItem(raw, globals().Format)
				return nil
			})
		},
	})

	list := &cobra.Command{
		Use:   "list",
		Short: "List recent events; defaults to NDJSON",
		RunE: func(cmd *cobra.Command, args []string) error {
			params := url.Values{}
			api.AddLimit(params, limit)
			api.AddCreatedRange(params, createdGTE, createdLTE)
			shared.AddString(params, "type", eventType)
			shared.AddMulti(params, "types[]", eventTypes)
			shared.AddString(params, "starting_after", startingAfter)
			shared.AddString(params, "ending_before", endingBefore)
			shared.AddString(params, "delivery_success", deliverySuccess)
			return shared.WithClient(globals(), func(ctx context.Context, client *api.Client) error {
				raw, err := client.Get(ctx, "/v1/events", params)
				if err != nil {
					return err
				}
				return shared.WriteRawList(raw, globals().Format)
			})
		},
	}
	list.Flags().IntVar(&limit, "limit", 10, "Maximum events to return (1-100)")
	list.Flags().StringVar(&eventType, "type", "", "Single event type, for example charge.failed")
	list.Flags().StringArrayVar(&eventTypes, "types", nil, "Event type filter; repeatable, up to Stripe's limit")
	list.Flags().StringVar(&createdGTE, "created-gte", "", "Created at or after Unix timestamp")
	list.Flags().StringVar(&createdLTE, "created-lte", "", "Created at or before Unix timestamp")
	list.Flags().StringVar(&startingAfter, "starting-after", "", "Stripe cursor")
	list.Flags().StringVar(&endingBefore, "ending-before", "", "Stripe cursor")
	list.Flags().StringVar(&deliverySuccess, "delivery-success", "", "Filter true/false for webhook delivery success")
	events.AddCommand(list)
	root.AddCommand(events)
}

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
			return shared.WithClient(globals(), func(ctx context.Context, client *api.Client) error {
				raw, err := client.Get(ctx, "/v1/payment_intents/"+url.PathEscape(args[0]), params)
				if err != nil {
					return err
				}
				shared.WriteRawItem(raw, globals().Format)
				return nil
			})
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
			return shared.WithClient(globals(), func(ctx context.Context, client *api.Client) error {
				raw, err := client.Get(ctx, "/v1/payment_intents/search", params)
				if err != nil {
					return err
				}
				return shared.WriteRawList(raw, globals().Format)
			})
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
			return shared.WithClient(globals(), func(ctx context.Context, client *api.Client) error {
				raw, err := client.Get(ctx, "/v1/payment_intents", params)
				if err != nil {
					return err
				}
				return shared.WriteRawList(raw, globals().Format)
			})
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

func registerCharges(root *cobra.Command, globals shared.GlobalsFunc) {
	var limit int
	var query string
	var page string
	var expand []string
	var customer string
	var paymentIntent string
	var createdGTE string
	var createdLTE string
	var startingAfter string

	charges := &cobra.Command{
		Use:   "charges",
		Short: "Charge lookup and search",
	}
	get := &cobra.Command{
		Use:   "get <charge-id>",
		Short: "Retrieve a charge by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			params := url.Values{}
			api.AddExpand(params, expand)
			return shared.WithClient(globals(), func(ctx context.Context, client *api.Client) error {
				raw, err := client.Get(ctx, "/v1/charges/"+url.PathEscape(args[0]), params)
				if err != nil {
					return err
				}
				shared.WriteRawItem(raw, globals().Format)
				return nil
			})
		},
	}
	get.Flags().StringArrayVar(&expand, "expand", nil, "Expand response property; repeatable")
	charges.AddCommand(get)

	search := &cobra.Command{
		Use:   "search",
		Short: "Search charges with Stripe Search Query Language",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !shared.RequireFlag("query", query, "Use a Stripe search query, for example metadata['order_id']:'123'") {
				return nil
			}
			params := url.Values{"query": []string{query}}
			api.AddLimit(params, limit)
			shared.AddString(params, "page", page)
			return shared.WithClient(globals(), func(ctx context.Context, client *api.Client) error {
				raw, err := client.Get(ctx, "/v1/charges/search", params)
				if err != nil {
					return err
				}
				return shared.WriteRawList(raw, globals().Format)
			})
		},
	}
	search.Flags().StringVar(&query, "query", "", "Stripe search query")
	search.Flags().IntVar(&limit, "limit", 10, "Maximum results to return (1-100)")
	search.Flags().StringVar(&page, "page", "", "Search pagination cursor")
	charges.AddCommand(search)

	list := &cobra.Command{
		Use:   "list",
		Short: "List charges",
		RunE: func(cmd *cobra.Command, args []string) error {
			params := url.Values{}
			api.AddLimit(params, limit)
			api.AddCreatedRange(params, createdGTE, createdLTE)
			shared.AddString(params, "customer", customer)
			shared.AddString(params, "payment_intent", paymentIntent)
			shared.AddString(params, "starting_after", startingAfter)
			return shared.WithClient(globals(), func(ctx context.Context, client *api.Client) error {
				raw, err := client.Get(ctx, "/v1/charges", params)
				if err != nil {
					return err
				}
				return shared.WriteRawList(raw, globals().Format)
			})
		},
	}
	list.Flags().IntVar(&limit, "limit", 10, "Maximum results to return (1-100)")
	list.Flags().StringVar(&customer, "customer", "", "Customer ID")
	list.Flags().StringVar(&paymentIntent, "payment-intent", "", "PaymentIntent ID")
	list.Flags().StringVar(&createdGTE, "created-gte", "", "Created at or after Unix timestamp")
	list.Flags().StringVar(&createdLTE, "created-lte", "", "Created at or before Unix timestamp")
	list.Flags().StringVar(&startingAfter, "starting-after", "", "Stripe cursor")
	charges.AddCommand(list)
	root.AddCommand(charges)
}

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
			return shared.WithClient(globals(), func(ctx context.Context, client *api.Client) error {
				raw, err := client.Get(ctx, "/v1/disputes/"+url.PathEscape(args[0]), url.Values{})
				if err != nil {
					return err
				}
				shared.WriteRawItem(raw, globals().Format)
				return nil
			})
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
			return shared.WithClient(globals(), func(ctx context.Context, client *api.Client) error {
				raw, err := client.Get(ctx, "/v1/disputes", params)
				if err != nil {
					return err
				}
				return shared.WriteRawList(raw, globals().Format)
			})
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

func registerAccounts(root *cobra.Command, globals shared.GlobalsFunc) {
	var limit int
	var startingAfter string

	accounts := &cobra.Command{
		Use:   "accounts",
		Short: "Stripe account and connected-account lookup",
	}
	accounts.AddCommand(&cobra.Command{
		Use:   "self",
		Short: "Retrieve the account for the active key/context",
		RunE: func(cmd *cobra.Command, args []string) error {
			return shared.WithClient(globals(), func(ctx context.Context, client *api.Client) error {
				raw, err := client.Get(ctx, "/v1/account", url.Values{})
				if err != nil {
					return err
				}
				shared.WriteRawItem(raw, globals().Format)
				return nil
			})
		},
	})
	accounts.AddCommand(&cobra.Command{
		Use:   "get <account-id>",
		Short: "Retrieve a connected account by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return shared.WithClient(globals(), func(ctx context.Context, client *api.Client) error {
				raw, err := client.Get(ctx, "/v1/accounts/"+url.PathEscape(args[0]), url.Values{})
				if err != nil {
					return err
				}
				shared.WriteRawItem(raw, globals().Format)
				return nil
			})
		},
	})
	list := &cobra.Command{
		Use:   "list",
		Short: "List connected accounts",
		RunE: func(cmd *cobra.Command, args []string) error {
			params := url.Values{}
			api.AddLimit(params, limit)
			shared.AddString(params, "starting_after", startingAfter)
			return shared.WithClient(globals(), func(ctx context.Context, client *api.Client) error {
				raw, err := client.Get(ctx, "/v1/accounts", params)
				if err != nil {
					return err
				}
				return shared.WriteRawList(raw, globals().Format)
			})
		},
	}
	list.Flags().IntVar(&limit, "limit", 10, "Maximum results to return (1-100)")
	list.Flags().StringVar(&startingAfter, "starting-after", "", "Stripe cursor")
	accounts.AddCommand(list)
	root.AddCommand(accounts)
}

func registerRawAPI(root *cobra.Command, globals shared.GlobalsFunc) {
	var queryPairs []string

	apiCmd := &cobra.Command{
		Use:   "api",
		Short: "Read-only raw Stripe API escape hatch",
	}
	get := &cobra.Command{
		Use:   "get <path>",
		Short: "GET a Stripe API path with active credentials",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			params, err := parseQueryPairs(queryPairs)
			if err != nil {
				return err
			}
			return shared.WithClient(globals(), func(ctx context.Context, client *api.Client) error {
				raw, err := client.Get(ctx, args[0], params)
				if err != nil {
					return err
				}
				shared.WriteRawItem(raw, globals().Format)
				return nil
			})
		},
	}
	get.Flags().StringArrayVar(&queryPairs, "query", nil, "Query parameter as k=v; repeatable")
	apiCmd.AddCommand(get)
	root.AddCommand(apiCmd)
}

func parseQueryPairs(pairs []string) (url.Values, error) {
	values := url.Values{}
	for _, pair := range pairs {
		key, value, ok := splitQueryPair(pair)
		if !ok {
			return nil, sharedQueryError(pair)
		}
		values.Add(key, value)
	}
	return values, nil
}

func splitQueryPair(pair string) (string, string, bool) {
	for i, r := range pair {
		if r == '=' {
			return pair[:i], pair[i+1:], i > 0
		}
	}
	return "", "", false
}

func sharedQueryError(pair string) error {
	return agenterrors.Newf(agenterrors.FixableByAgent, "invalid --query value %q", pair).
		WithHint("Use --query key=value, for example --query expand[]=latest_charge")
}
