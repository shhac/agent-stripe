package mockstripe

// Which resources the mock serves. The engine that drives them lives in
// resource_routes.go; this file is data and grows with every resource added.

func mockResources() []mockResource {
	return []mockResource{
		{
			path:       "/v1/customers",
			objectName: "customer",
			items:      customers,
			searchable: true,
			filters: []mockFilter{
				stringFilter("email", "email"),
			},
		},
		{
			path:       "/v1/events",
			objectName: "event",
			items:      events,
			filters: []mockFilter{
				stringFilter("type", "type"),
			},
		},
		{
			path:       "/v1/webhook_endpoints",
			objectName: "webhook_endpoint",
			items:      webhookEndpoints,
		},
		{
			path:       "/v1/products",
			objectName: "product",
			items:      products,
			searchable: true,
			filters: []mockFilter{
				boolFilter("active", "active"),
			},
		},
		{
			path:       "/v1/prices",
			objectName: "price",
			items:      prices,
			searchable: true,
			filters: []mockFilter{
				boolFilter("active", "active"),
				stringFilter("product", "product"),
				stringFilter("type", "type"),
			},
		},
		{
			path:       "/v1/payment_intents",
			objectName: "payment_intent",
			items:      paymentIntents,
			searchable: true,
		},
		{
			path:       "/v1/setup_intents",
			objectName: "setup_intent",
			items:      setupIntents,
			filters: []mockFilter{
				stringFilter("customer", "customer"),
				stringFilter("payment_method", "payment_method"),
			},
		},
		{
			path:       "/v1/payment_methods",
			objectName: "payment_method",
			items:      paymentMethods,
			filters: []mockFilter{
				stringFilter("customer", "customer"),
				stringFilter("type", "type"),
			},
		},
		{
			path:       "/v1/charges",
			objectName: "charge",
			items:      charges,
			searchable: true,
			filters: []mockFilter{
				stringFilter("payment_intent", "payment_intent"),
				stringFilter("customer", "customer"),
			},
		},
		{
			path:       "/v1/disputes",
			objectName: "dispute",
			items:      disputes,
			filters: []mockFilter{
				stringFilter("charge", "charge"),
				stringFilter("payment_intent", "payment_intent"),
			},
		},
		{
			path:       "/v1/refunds",
			objectName: "refund",
			items:      refunds,
			filters: []mockFilter{
				stringFilter("charge", "charge"),
				stringFilter("payment_intent", "payment_intent"),
			},
		},
		{
			path:       "/v1/subscriptions",
			objectName: "subscription",
			items:      subscriptions,
			searchable: true,
			filters: []mockFilter{
				stringFilter("customer", "customer"),
				{query: "status", field: "status", kind: filterString, ignoreValue: "all"},
			},
		},
		{
			path:       "/v1/subscription_items",
			objectName: "subscription_item",
			items:      subscriptionItems,
			filters: []mockFilter{
				stringFilter("subscription", "subscription"),
			},
		},
		{
			path:       "/v1/payouts",
			objectName: "payout",
			items:      payouts,
			filters: []mockFilter{
				stringFilter("status", "status"),
			},
		},
		{
			path:       "/v1/balance_transactions",
			objectName: "balance_transaction",
			items:      balanceTransactions,
			filters: []mockFilter{
				stringFilter("type", "type"),
				stringFilter("payout", "payout"),
			},
		},
		{
			path:       "/v1/application_fees",
			objectName: "application_fee",
			items:      applicationFees,
			filters: []mockFilter{
				stringFilter("charge", "charge"),
			},
		},
		{
			path:       "/v1/payment_links",
			objectName: "payment_link",
			items:      paymentLinks,
			filters: []mockFilter{
				boolFilter("active", "active"),
			},
		},
		{
			path:       "/v1/radar/early_fraud_warnings",
			objectName: "early_fraud_warning",
			items:      earlyFraudWarnings,
			filters: []mockFilter{
				stringFilter("charge", "charge"),
				stringFilter("payment_intent", "payment_intent"),
			},
		},
		{
			path:       "/v1/invoices",
			objectName: "invoice",
			items:      invoices,
			searchable: true,
			filters: []mockFilter{
				stringFilter("subscription", "subscription"),
				stringFilter("status", "status"),
				// The hand-written handler ignored customer, so any test
				// asserting per-customer invoice scoping passed vacuously.
				stringFilter("customer", "customer"),
			},
			subLists: map[string]func(string) []map[string]any{"lines": invoiceLines},
		},
		{
			path:       "/v1/checkout/sessions",
			objectName: "checkout.session",
			items:      checkoutSessions,
			filters: []mockFilter{
				stringFilter("customer", "customer"),
				stringFilter("payment_intent", "payment_intent"),
				stringFilter("subscription", "subscription"),
				stringFilter("payment_link", "payment_link"),
			},
			subLists: map[string]func(string) []map[string]any{"line_items": checkoutLineItems},
		},
		{
			path:       "/v1/transfers",
			objectName: "transfer",
			items:      transfers,
			filters: []mockFilter{
				stringFilter("destination", "destination"),
				stringFilter("transfer_group", "transfer_group"),
			},
			nested:       "reversals",
			nestedItems:  transferReversals,
			nestedObject: "transfer_reversal",
		},
		{
			path:       "/v1/accounts",
			objectName: "account",
			items:      accounts,
			subPaths:   true,
		},
	}
}
