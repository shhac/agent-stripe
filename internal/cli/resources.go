package cli

import (
	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func registerCustomers(root *cobra.Command, globals shared.GlobalsFunc) {
	registerResource(root, globals, resourceOptions{
		use:         "customers",
		short:       "Customer lookup and search",
		path:        "/v1/customers",
		idName:      "customer-id",
		idKind:      "customer",
		searchable:  true,
		searchHint:  "Use a Stripe search query, for example email:'person@example.com' or metadata['tenant_id']:'acme'",
		listSummary: customerListSummary,
		listFlags: []listFlag{
			{name: "email", param: "email", help: "Customer email address"},
		},
	})
}

func registerPaymentMethods(root *cobra.Command, globals shared.GlobalsFunc) {
	registerResource(root, globals, resourceOptions{
		use:         "payment-methods",
		short:       "PaymentMethod lookup by customer and type",
		path:        "/v1/payment_methods",
		idName:      "payment-method-id",
		idKind:      "payment_method",
		listSummary: paymentMethodListSummary,
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
		idKind:    "refund",
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
		idKind:    "transfer",
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
		idKind:    "payout",
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
		idKind:  "balance_transaction",
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
		idKind:    "application_fee",
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
		idKind:     "product",
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
		idKind:     "price",
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
		use:         "setup-intents",
		aliases:     []string{"setup_intents"},
		short:       "SetupIntent lookup for saved payment method setup",
		path:        "/v1/setup_intents",
		idName:      "setup-intent-id",
		idKind:      "setup_intent",
		expandGet:   true,
		listSummary: setupIntentListSummary,
		listFlags: []listFlag{
			{name: "customer", param: "customer", help: "Customer ID"},
			{name: "payment-method", param: "payment_method", help: "PaymentMethod ID"},
		},
	})
}

func registerPaymentLinks(root *cobra.Command, globals shared.GlobalsFunc) {
	registerResource(root, globals, resourceOptions{
		use:         "payment-links",
		aliases:     []string{"payment_links"},
		short:       "Payment Link lookup",
		path:        "/v1/payment_links",
		idName:      "payment-link-id",
		idKind:      "payment_link",
		expandGet:   true,
		listSummary: paymentLinkListSummary,
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
		idKind:    "early_fraud_warning",
		expandGet: true,
		listFlags: []listFlag{
			{name: "charge", param: "charge", help: "Charge ID"},
			{name: "payment-intent", param: "payment_intent", help: "PaymentIntent ID"},
		},
	})
}
