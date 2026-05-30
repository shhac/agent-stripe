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
	s.mux.HandleFunc(resource.path+"/", resource.handleGet)
}

func (r mockResource) handleList(w http.ResponseWriter, req *http.Request) {
	if !requireGet(w, req) {
		return
	}
	writeList(w, r.path, limit(r.filteredItems(req), req))
}

func (r mockResource) handleSearch(w http.ResponseWriter, req *http.Request) {
	if !requireGet(w, req) {
		return
	}
	if req.URL.Query().Get("query") == "" {
		writeStripeError(w, http.StatusBadRequest, "invalid_request_error", "parameter_missing", "Missing required param: query")
		return
	}
	writeSearchList(w, r.path+"/search", limit(r.items(), req))
}

func (r mockResource) handleGet(w http.ResponseWriter, req *http.Request) {
	if !requireGet(w, req) {
		return
	}
	id := strings.TrimPrefix(req.URL.Path, r.path+"/")
	writeOneByID(w, r.items(), id, r.objectName)
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
			path:       "/v1/accounts",
			objectName: "account",
			items:      accounts,
		},
	}
}

func stringFilter(query, field string) mockFilter {
	return mockFilter{query: query, field: field, kind: filterString}
}

func boolFilter(query, field string) mockFilter {
	return mockFilter{query: query, field: field, kind: filterBool}
}
