package mockstripe

import (
	"net/http"
	"strings"
)

type mockResource struct {
	path       string
	objectName string
	items      func() []map[string]any
	searchable bool
	filters    []mockFilter
	// subLists maps a sub-path suffix to its fixtures, e.g. "line_items" for
	// /v1/checkout/sessions/<id>/line_items. Declaring them here keeps the
	// resource in the table instead of needing a hand-written handler.
	subLists map[string]func(id string) []map[string]any
	// nested serves <path>/<id>/<segment>/<sub-id>, which is how Stripe models
	// transfer reversals.
	nested       string
	nestedItems  func(parentID string) []map[string]any
	nestedObject string
	// subPaths marks a resource whose <path>/ space is served elsewhere because
	// it has sub-resources (accounts has persons, capabilities, external
	// accounts), so the table does not claim the pattern.
	subPaths bool
}

type mockFilter struct {
	query       string
	field       string
	kind        filterKind
	ignoreValue string
}

type filterKind string

const (
	filterString filterKind = "string"
	filterBool   filterKind = "bool"
)

func (s *Server) registerMockResource(resource mockResource) {
	if resource.searchable {
		s.mux.HandleFunc(resource.path+"/search", resource.handleSearch)
	}
	s.mux.HandleFunc(resource.path, resource.handleList)
	if !resource.subPaths {
		s.mux.HandleFunc(resource.path+"/", resource.handleGet)
	}
}

func (r mockResource) handleList(w http.ResponseWriter, req *http.Request) {
	if !requireGet(w, req) {
		return
	}
	writeList(w, r.path, r.filteredItems(req), req)
}

func (r mockResource) handleSearch(w http.ResponseWriter, req *http.Request) {
	if !requireGet(w, req) {
		return
	}
	if req.URL.Query().Get("query") == "" {
		writeStripeError(w, http.StatusBadRequest, "invalid_request_error", "parameter_missing", "Missing required param: query")
		return
	}
	writeSearchList(w, r.path+"/search", r.items(), req)
}

func (r mockResource) handleGet(w http.ResponseWriter, req *http.Request) {
	if !requireGet(w, req) {
		return
	}
	rest := strings.TrimPrefix(req.URL.Path, r.path+"/")
	if r.nested != "" && strings.Contains(rest, "/"+r.nested+"/") {
		parent, child, _ := strings.Cut(rest, "/"+r.nested+"/")
		writeOneByID(w, r.nestedItems(parent), child, r.nestedObject)
		return
	}
	for suffix, items := range r.subLists {
		if !strings.HasSuffix(rest, "/"+suffix) {
			continue
		}
		id := strings.TrimSuffix(rest, "/"+suffix)
		writeList(w, r.path+"/"+id+"/"+suffix, items(id), req)
		return
	}
	writeOneByID(w, r.items(), rest, r.objectName)
}

func (r mockResource) filteredItems(req *http.Request) []map[string]any {
	items := r.items()
	for _, filter := range r.filters {
		value := req.URL.Query().Get(filter.query)
		if value == "" || value == filter.ignoreValue {
			continue
		}
		switch filter.kind {
		case filterBool:
			items = filterByBoolString(items, filter.field, value)
		default:
			items = filterByString(items, filter.field, value)
		}
	}
	return items
}

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

func stringFilter(query, field string) mockFilter {
	return mockFilter{query: query, field: field, kind: filterString}
}

func boolFilter(query, field string) mockFilter {
	return mockFilter{query: query, field: field, kind: filterBool}
}
