package mockstripe

import (
	"encoding/json"
	"net/http"
)

// The mock's response vocabulary. resource_routes.go, v1_routes.go and
// v2_routes.go are all written in terms of these, so they live under a name
// that says so rather than at the bottom of the transport file.

func writeOneByID(w http.ResponseWriter, items []map[string]any, id string, objectName string) {
	for _, item := range items {
		if item["id"] == id {
			writeJSON(w, http.StatusOK, item)
			return
		}
	}
	writeStripeError(w, http.StatusNotFound, "invalid_request_error", "resource_missing", "No such "+objectName+": '"+id+"'")
}

func writeList(w http.ResponseWriter, path string, items []map[string]any, r *http.Request) {
	page := listPage(items, r)
	writeJSON(w, http.StatusOK, map[string]any{
		"object":   "list",
		"url":      path,
		"has_more": page.hasMore,
		"data":     page.items,
	})
}

func writeSearchList(w http.ResponseWriter, path string, items []map[string]any, r *http.Request) {
	page := listPage(items, r)
	var nextPage any
	if page.hasMore && len(page.items) > 0 {
		nextPage = "mock_next_" + stringValue(page.items[len(page.items)-1], "id")
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"object":    "search_result",
		"url":       path,
		"has_more":  page.hasMore,
		"next_page": nextPage,
		"data":      page.items,
	})
}

func writeStripeError(w http.ResponseWriter, status int, errType, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]any{
			"type":            errType,
			"code":            code,
			"message":         message,
			"request_log_url": "https://dashboard.stripe.com/test/logs/req_mock_123",
		},
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(payload)
}

func requireGet(w http.ResponseWriter, r *http.Request) bool {
	return requireMethod(w, r, http.MethodGet)
}

func requireMethod(w http.ResponseWriter, r *http.Request, methods ...string) bool {
	for _, method := range methods {
		if r.Method == method {
			return true
		}
	}
	writeStripeError(w, http.StatusMethodNotAllowed, "invalid_request_error", "method_not_allowed", "Method not supported by mockstripe")
	return false
}
