package mockstripe

import (
	"net/http"
	"strings"
)

// Hand-written /v1 handlers: the endpoints whose shape does not fit the
// mockResource table — a singleton, a sub-resource list, or a read-like POST.

func (s *Server) handleSelfAccount(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":                "acct_mock_platform",
		"object":            "account",
		"charges_enabled":   true,
		"payouts_enabled":   true,
		"details_submitted": true,
		"metadata": map[string]string{
			"environment": "mock",
		},
	})
}

func (s *Server) handleBalance(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"object": "balance",
		"available": []map[string]any{{
			"amount":   123450,
			"currency": "usd",
		}},
		"pending": []map[string]any{{
			"amount":   6789,
			"currency": "usd",
		}},
		"livemode": false,
	})
}

func (s *Server) handleInvoiceSearch(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	if r.URL.Query().Get("query") == "" {
		writeStripeError(w, http.StatusBadRequest, "invalid_request_error", "parameter_missing", "Missing required param: query")
		return
	}
	writeSearchList(w, "/v1/invoices/search", invoices(), r)
}

func (s *Server) handleInvoicePreview(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	if err := r.ParseForm(); err != nil {
		writeStripeError(w, http.StatusBadRequest, "invalid_request_error", "invalid_form", "Invalid form body")
		return
	}
	subscription := r.Form.Get("subscription")
	amount := 4200
	customer := r.Form.Get("customer")
	if subscription == "sub_mock_past_due" {
		amount = 29700
		customer = "cus_mock_456"
	}
	if customer == "" {
		customer = "cus_mock_123"
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":           "upcoming_in_mock",
		"object":       "invoice",
		"customer":     customer,
		"subscription": subscription,
		"amount_due":   amount,
		"currency":     "usd",
		"status":       "draft",
	})
}

func (s *Server) handleInvoicesList(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	items := invoices()
	if subscription := r.URL.Query().Get("subscription"); subscription != "" {
		items = filterByString(items, "subscription", subscription)
	}
	if status := r.URL.Query().Get("status"); status != "" {
		items = filterByString(items, "status", status)
	}
	writeList(w, "/v1/invoices", items, r)
}

func (s *Server) handleInvoiceGetOrLines(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	rest := strings.TrimPrefix(r.URL.Path, "/v1/invoices/")
	if strings.HasSuffix(rest, "/lines") {
		id := strings.TrimSuffix(rest, "/lines")
		writeList(w, "/v1/invoices/"+id+"/lines", invoiceLines(id), r)
		return
	}
	writeOneByID(w, invoices(), rest, "invoice")
}
