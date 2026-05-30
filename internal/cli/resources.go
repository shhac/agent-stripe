package cli

import (
	"net/url"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/api"
	"github.com/shhac/agent-stripe/internal/cli/shared"
)

type listFlag struct {
	name  string
	param string
	help  string
}

type resourceOptions struct {
	use           string
	aliases       []string
	short         string
	path          string
	idName        string
	getShort      string
	listShort     string
	searchShort   string
	searchHint    string
	searchable    bool
	listFlags     []listFlag
	expandGet     bool
	expandList    bool
	lineItems     bool
	previewPath   string
	previewFlags  []listFlag
	extraCommands []func(shared.GlobalsFunc) *cobra.Command
}

func registerResource(root *cobra.Command, globals shared.GlobalsFunc, opts resourceOptions) {
	resource := &cobra.Command{
		Use:     opts.use,
		Aliases: opts.aliases,
		Short:   opts.short,
	}
	resource.AddCommand(newResourceGetCommand(globals, opts))
	resource.AddCommand(newResourceListCommand(globals, opts))
	if opts.searchable {
		resource.AddCommand(newResourceSearchCommand(globals, opts))
	}
	if opts.lineItems {
		resource.AddCommand(newInvoiceLineItemsCommand(globals))
	}
	if opts.previewPath != "" {
		resource.AddCommand(newInvoicePreviewCommand(globals, opts))
	}
	for _, newCommand := range opts.extraCommands {
		resource.AddCommand(newCommand(globals))
	}
	root.AddCommand(resource)
}

func newResourceGetCommand(globals shared.GlobalsFunc, opts resourceOptions) *cobra.Command {
	var expand []string
	cmd := &cobra.Command{
		Use:   "get <" + opts.idName + ">",
		Short: resourceText(opts.getShort, "Retrieve a "+opts.idName),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			params := url.Values{}
			api.AddExpand(params, expand)
			return shared.GetRawItem(globals(), opts.path+"/"+url.PathEscape(args[0]), params)
		},
	}
	if opts.expandGet {
		cmd.Flags().StringArrayVar(&expand, "expand", nil, "Expand response property; repeatable")
	}
	return cmd
}

func newResourceListCommand(globals shared.GlobalsFunc, opts resourceOptions) *cobra.Command {
	var limit int
	var createdGTE string
	var createdLTE string
	var startingAfter string
	var expand []string
	values := make(map[string]*string, len(opts.listFlags))
	cmd := &cobra.Command{
		Use:   "list",
		Short: resourceText(opts.listShort, "List "+opts.use),
		RunE: func(cmd *cobra.Command, args []string) error {
			params := url.Values{}
			api.AddLimit(params, limit)
			api.AddCreatedRange(params, createdGTE, createdLTE)
			shared.AddString(params, "starting_after", startingAfter)
			api.AddExpand(params, expand)
			for _, flag := range opts.listFlags {
				shared.AddString(params, flag.param, *values[flag.name])
			}
			return shared.GetRawList(globals(), opts.path, params)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 10, "Maximum results to return (1-100)")
	cmd.Flags().StringVar(&createdGTE, "created-gte", "", "Created at or after Unix timestamp")
	cmd.Flags().StringVar(&createdLTE, "created-lte", "", "Created at or before Unix timestamp")
	cmd.Flags().StringVar(&startingAfter, "starting-after", "", "Stripe cursor")
	if opts.expandList {
		cmd.Flags().StringArrayVar(&expand, "expand", nil, "Expand response property; repeatable")
	}
	for _, flag := range opts.listFlags {
		var value string
		values[flag.name] = &value
		cmd.Flags().StringVar(&value, flag.name, "", flag.help)
	}
	return cmd
}

func newResourceSearchCommand(globals shared.GlobalsFunc, opts resourceOptions) *cobra.Command {
	var limit int
	var query string
	var page string
	cmd := &cobra.Command{
		Use:   "search",
		Short: resourceText(opts.searchShort, "Search "+opts.use+" with Stripe Search Query Language"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !shared.RequireFlag("query", query, opts.searchHint) {
				return nil
			}
			params := url.Values{"query": []string{query}}
			api.AddLimit(params, limit)
			shared.AddString(params, "page", page)
			return shared.GetRawList(globals(), opts.path+"/search", params)
		},
	}
	cmd.Flags().StringVar(&query, "query", "", "Stripe search query")
	cmd.Flags().IntVar(&limit, "limit", 10, "Maximum results to return (1-100)")
	cmd.Flags().StringVar(&page, "page", "", "Search pagination cursor")
	return cmd
}

func resourceText(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func newInvoiceLineItemsCommand(globals shared.GlobalsFunc) *cobra.Command {
	var limit int
	var startingAfter string
	cmd := &cobra.Command{
		Use:   "line-items <invoice-id>",
		Short: "List line items on an invoice",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			params := url.Values{}
			api.AddLimit(params, limit)
			shared.AddString(params, "starting_after", startingAfter)
			return shared.GetRawList(globals(), "/v1/invoices/"+url.PathEscape(args[0])+"/lines", params)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 10, "Maximum results to return (1-100)")
	cmd.Flags().StringVar(&startingAfter, "starting-after", "", "Stripe cursor")
	return cmd
}

func newInvoicePreviewCommand(globals shared.GlobalsFunc, opts resourceOptions) *cobra.Command {
	values := make(map[string]*string, len(opts.previewFlags))
	cmd := &cobra.Command{
		Use:   "preview",
		Short: "Create a preview invoice for a customer or subscription",
		RunE: func(cmd *cobra.Command, args []string) error {
			params := url.Values{}
			for _, flag := range opts.previewFlags {
				shared.AddString(params, flag.param, *values[flag.name])
			}
			return shared.PostFormRawItem(globals(), opts.previewPath, params)
		},
	}
	for _, flag := range opts.previewFlags {
		var value string
		values[flag.name] = &value
		cmd.Flags().StringVar(&value, flag.name, "", flag.help)
	}
	return cmd
}

func registerCustomers(root *cobra.Command, globals shared.GlobalsFunc) {
	registerResource(root, globals, resourceOptions{
		use:        "customers",
		short:      "Customer lookup and search",
		path:       "/v1/customers",
		idName:     "customer-id",
		searchable: true,
		searchHint: "Use a Stripe search query, for example email:'person@example.com' or metadata['tenant_id']:'acme'",
		listFlags: []listFlag{
			{name: "email", param: "email", help: "Customer email address"},
		},
	})
}

func registerInvoices(root *cobra.Command, globals shared.GlobalsFunc) {
	registerResource(root, globals, resourceOptions{
		use:        "invoices",
		short:      "Invoice lookup, line items, and payment bridge investigation",
		path:       "/v1/invoices",
		idName:     "invoice-id",
		searchable: true,
		searchHint: "Use a Stripe search query, for example number:'ABC-0001' or metadata['order_id']:'123'",
		expandGet:  true,
		expandList: true,
		lineItems:  true,
		listFlags: []listFlag{
			{name: "customer", param: "customer", help: "Customer ID"},
			{name: "subscription", param: "subscription", help: "Subscription ID"},
			{name: "status", param: "status", help: "Invoice status: draft, open, paid, uncollectible, void"},
		},
		previewPath: "/v1/invoices/create_preview",
		previewFlags: []listFlag{
			{name: "customer", param: "customer", help: "Customer ID"},
			{name: "subscription", param: "subscription", help: "Subscription ID"},
			{name: "preview-mode", param: "preview_mode", help: "Preview mode: next or recurring"},
		},
	})
}

func registerPaymentMethods(root *cobra.Command, globals shared.GlobalsFunc) {
	registerResource(root, globals, resourceOptions{
		use:    "payment-methods",
		short:  "PaymentMethod lookup by customer and type",
		path:   "/v1/payment_methods",
		idName: "payment-method-id",
		listFlags: []listFlag{
			{name: "customer", param: "customer", help: "Customer ID"},
			{name: "type", param: "type", help: "PaymentMethod type, for example card or us_bank_account"},
		},
	})
}

func registerRefunds(root *cobra.Command, globals shared.GlobalsFunc) {
	registerResource(root, globals, resourceOptions{
		use:       "refunds",
		short:     "Refund lookup for failed or reversed customer payments",
		path:      "/v1/refunds",
		idName:    "refund-id",
		expandGet: true,
		listFlags: []listFlag{
			{name: "charge", param: "charge", help: "Charge ID"},
			{name: "payment-intent", param: "payment_intent", help: "PaymentIntent ID"},
		},
	})
}

func registerTransfers(root *cobra.Command, globals shared.GlobalsFunc) {
	registerResource(root, globals, resourceOptions{
		use:       "transfers",
		short:     "Connect transfer lookup and reversals",
		path:      "/v1/transfers",
		idName:    "transfer-id",
		expandGet: true,
		listFlags: []listFlag{
			{name: "destination", param: "destination", help: "Connected account ID"},
			{name: "transfer-group", param: "transfer_group", help: "Transfer group"},
		},
	})
}

func registerPayouts(root *cobra.Command, globals shared.GlobalsFunc) {
	registerResource(root, globals, resourceOptions{
		use:       "payouts",
		short:     "Payout lookup for Stripe balance to bank movement",
		path:      "/v1/payouts",
		idName:    "payout-id",
		expandGet: true,
		listFlags: []listFlag{
			{name: "status", param: "status", help: "Payout status"},
			{name: "destination", param: "destination", help: "External account ID"},
		},
	})
}

func registerBalanceTransactions(root *cobra.Command, globals shared.GlobalsFunc) {
	registerResource(root, globals, resourceOptions{
		use:     "balance-transactions",
		aliases: []string{"balance_transactions", "txns"},
		short:   "Balance transaction ledger lookup",
		path:    "/v1/balance_transactions",
		idName:  "balance-transaction-id",
		listFlags: []listFlag{
			{name: "type", param: "type", help: "Balance transaction type"},
			{name: "payout", param: "payout", help: "Payout ID"},
		},
	})
}

func registerApplicationFees(root *cobra.Command, globals shared.GlobalsFunc) {
	registerResource(root, globals, resourceOptions{
		use:       "application-fees",
		aliases:   []string{"application_fees"},
		short:     "Connect application fee lookup",
		path:      "/v1/application_fees",
		idName:    "application-fee-id",
		expandGet: true,
		listFlags: []listFlag{
			{name: "charge", param: "charge", help: "Charge ID"},
		},
	})
}

func registerProducts(root *cobra.Command, globals shared.GlobalsFunc) {
	registerResource(root, globals, resourceOptions{
		use:        "products",
		short:      "Product catalog lookup and search",
		path:       "/v1/products",
		idName:     "product-id",
		searchable: true,
		searchHint: "Use a Stripe search query, for example metadata['internal_id']:'prod_123' or name:'Pro'",
		listFlags: []listFlag{
			{name: "active", param: "active", help: "Filter by active=true|false"},
		},
	})
}

func registerPrices(root *cobra.Command, globals shared.GlobalsFunc) {
	registerResource(root, globals, resourceOptions{
		use:        "prices",
		short:      "Price lookup and search",
		path:       "/v1/prices",
		idName:     "price-id",
		searchable: true,
		searchHint: "Use a Stripe search query, for example product:'prod_...' or metadata['internal_id']:'price_123'",
		expandGet:  true,
		listFlags: []listFlag{
			{name: "active", param: "active", help: "Filter by active=true|false"},
			{name: "product", param: "product", help: "Product ID"},
			{name: "type", param: "type", help: "Price type: one_time or recurring"},
		},
	})
}

func registerSetupIntents(root *cobra.Command, globals shared.GlobalsFunc) {
	registerResource(root, globals, resourceOptions{
		use:       "setup-intents",
		aliases:   []string{"setup_intents"},
		short:     "SetupIntent lookup for saved payment method setup",
		path:      "/v1/setup_intents",
		idName:    "setup-intent-id",
		expandGet: true,
		listFlags: []listFlag{
			{name: "customer", param: "customer", help: "Customer ID"},
			{name: "payment-method", param: "payment_method", help: "PaymentMethod ID"},
		},
	})
}

func registerPaymentLinks(root *cobra.Command, globals shared.GlobalsFunc) {
	registerResource(root, globals, resourceOptions{
		use:       "payment-links",
		aliases:   []string{"payment_links"},
		short:     "Payment Link lookup",
		path:      "/v1/payment_links",
		idName:    "payment-link-id",
		expandGet: true,
		listFlags: []listFlag{
			{name: "active", param: "active", help: "Filter by active=true|false"},
		},
	})
}

func registerEarlyFraudWarnings(root *cobra.Command, globals shared.GlobalsFunc) {
	registerResource(root, globals, resourceOptions{
		use:       "early-fraud-warnings",
		aliases:   []string{"early_fraud_warnings", "efw"},
		short:     "Radar early fraud warning lookup",
		path:      "/v1/radar/early_fraud_warnings",
		idName:    "early-fraud-warning-id",
		expandGet: true,
		listFlags: []listFlag{
			{name: "charge", param: "charge", help: "Charge ID"},
			{name: "payment-intent", param: "payment_intent", help: "PaymentIntent ID"},
		},
	})
}

func registerCheckoutSessions(root *cobra.Command, globals shared.GlobalsFunc) {
	var limit int
	var customer string
	var paymentIntent string
	var subscription string
	var paymentLink string
	var createdGTE string
	var createdLTE string
	var startingAfter string
	var expand []string

	sessions := &cobra.Command{
		Use:     "checkout-sessions",
		Aliases: []string{"checkout_sessions"},
		Short:   "Checkout Session lookup and line items",
	}
	get := &cobra.Command{
		Use:   "get <checkout-session-id>",
		Short: "Retrieve a Checkout Session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			params := url.Values{}
			api.AddExpand(params, expand)
			return shared.GetRawItem(globals(), "/v1/checkout/sessions/"+url.PathEscape(args[0]), params)
		},
	}
	get.Flags().StringArrayVar(&expand, "expand", nil, "Expand response property; repeatable")
	sessions.AddCommand(get)

	list := &cobra.Command{
		Use:   "list",
		Short: "List Checkout Sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			params := url.Values{}
			api.AddLimit(params, limit)
			api.AddCreatedRange(params, createdGTE, createdLTE)
			shared.AddString(params, "customer", customer)
			shared.AddString(params, "payment_intent", paymentIntent)
			shared.AddString(params, "subscription", subscription)
			shared.AddString(params, "payment_link", paymentLink)
			shared.AddString(params, "starting_after", startingAfter)
			return shared.GetRawList(globals(), "/v1/checkout/sessions", params)
		},
	}
	list.Flags().IntVar(&limit, "limit", 10, "Maximum results to return (1-100)")
	list.Flags().StringVar(&customer, "customer", "", "Customer ID")
	list.Flags().StringVar(&paymentIntent, "payment-intent", "", "PaymentIntent ID")
	list.Flags().StringVar(&subscription, "subscription", "", "Subscription ID")
	list.Flags().StringVar(&paymentLink, "payment-link", "", "Payment Link ID")
	list.Flags().StringVar(&createdGTE, "created-gte", "", "Created at or after Unix timestamp")
	list.Flags().StringVar(&createdLTE, "created-lte", "", "Created at or before Unix timestamp")
	list.Flags().StringVar(&startingAfter, "starting-after", "", "Stripe cursor")
	sessions.AddCommand(list)

	lineItems := &cobra.Command{
		Use:   "line-items <checkout-session-id>",
		Short: "List Checkout Session line items",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			params := url.Values{}
			api.AddLimit(params, limit)
			return shared.GetRawList(globals(), "/v1/checkout/sessions/"+url.PathEscape(args[0])+"/line_items", params)
		},
	}
	lineItems.Flags().IntVar(&limit, "limit", 10, "Maximum results to return (1-100)")
	sessions.AddCommand(lineItems)
	root.AddCommand(sessions)
}
