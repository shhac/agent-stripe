package mockstripe

import "fmt"

func Routes() []string {
	routes := []string{
		"GET  /                         (route map; no auth required)",
		"GET  /healthz                  (no auth required)",
		"GET  /v1/account",
		"GET  /v1/balance",
		"GET  /v1/checkout/sessions",
		"GET  /v1/checkout/sessions/<id>",
		"GET  /v1/checkout/sessions/<id>/line_items",
		"GET  /v1/invoices",
		"GET  /v1/invoices/search",
		"POST /v1/invoices/create_preview",
		"GET  /v1/invoices/<id>",
		"GET  /v1/invoices/<id>/lines",
		"GET  /v1/transfers",
		"GET  /v1/transfers/<id>",
		"GET  /v1/transfers/<id>/reversals/<reversal_id>",
	}
	for _, resource := range mockResources() {
		routes = append(routes, fmt.Sprintf("GET  %s", resource.path))
		if resource.searchable {
			routes = append(routes, fmt.Sprintf("GET  %s/search", resource.path))
		}
		routes = append(routes, fmt.Sprintf("GET  %s/<id>", resource.path))
	}
	return routes
}
