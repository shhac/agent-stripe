package mockstripe

import (
	"net/http"
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
