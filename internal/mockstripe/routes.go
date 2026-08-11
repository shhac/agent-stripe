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
		"POST /v1/invoices/create_preview",
		"GET  /v1/transfers",
		"GET  /v1/transfers/<id>",
		"GET  /v1/transfers/<id>/reversals/<reversal_id>",
		"GET  /v2/core/accounts                        (Bearer auth; indexed applied_configurations[i], closed, limit, page)",
		"GET  /v2/core/accounts/<id>                   (Bearer auth; indexed include[i]; omitted fields are null)",
		"GET  /v2/core/accounts/<id>/persons",
		"GET  /v2/core/accounts/<id>/persons/<person_id>",
		"GET  /v2/core/events                          (Bearer auth; object_id, indexed types[i], limit, page)",
		"GET  /v2/core/events/<id>",
		"",
		"Fault injection (header, any route):",
		"  X-Mock-Fault: /v1/disputes=500      fail that path",
		"  X-Mock-Fault: /v1/charges=429x2     429 twice then succeed",
		"  X-Mock-Fault: /v1/events=garbage    non-JSON body",
		"  X-Mock-Fault: *=500                 fail everything",
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
