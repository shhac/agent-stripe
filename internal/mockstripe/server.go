package mockstripe

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

type Server struct {
	mux *http.ServeMux
}

func NewServer() http.Handler {
	s := &Server{mux: http.NewServeMux()}
	s.routes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Request-Id", "req_mock_123")
	w.Header().Set("Stripe-Mock", "true")
	if !hasBasicKey(r) {
		writeStripeError(w, http.StatusUnauthorized, "authentication_error", "api_key_missing", "No API key provided")
		return
	}
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	s.mux.HandleFunc("/v1/account", s.handleSelfAccount)
	s.mux.HandleFunc("/v1/balance", s.handleBalance)
	s.mux.HandleFunc("/v1/events", s.handleEventsList)
	s.mux.HandleFunc("/v1/events/", s.handleEventGet)
	s.mux.HandleFunc("/v1/payment_intents/search", s.handlePaymentIntentSearch)
	s.mux.HandleFunc("/v1/payment_intents", s.handlePaymentIntentsList)
	s.mux.HandleFunc("/v1/payment_intents/", s.handlePaymentIntentGet)
	s.mux.HandleFunc("/v1/charges/search", s.handleChargeSearch)
	s.mux.HandleFunc("/v1/charges", s.handleChargesList)
	s.mux.HandleFunc("/v1/charges/", s.handleChargeGet)
	s.mux.HandleFunc("/v1/disputes", s.handleDisputesList)
	s.mux.HandleFunc("/v1/disputes/", s.handleDisputeGet)
	s.mux.HandleFunc("/v1/subscriptions/search", s.handleSubscriptionSearch)
	s.mux.HandleFunc("/v1/subscriptions", s.handleSubscriptionsList)
	s.mux.HandleFunc("/v1/subscriptions/", s.handleSubscriptionGet)
	s.mux.HandleFunc("/v1/subscription_items", s.handleSubscriptionItemsList)
	s.mux.HandleFunc("/v1/invoices", s.handleInvoicesList)
	s.mux.HandleFunc("/v1/accounts", s.handleAccountsList)
	s.mux.HandleFunc("/v1/accounts/", s.handleAccountGet)
}

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

func (s *Server) handleEventsList(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	items := filterByEventType(events(), r.URL.Query().Get("type"))
	writeList(w, "/v1/events", limit(items, r))
}

func (s *Server) handleEventGet(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/v1/events/")
	writeOneByID(w, events(), id, "event")
}

func (s *Server) handlePaymentIntentsList(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	writeList(w, "/v1/payment_intents", limit(paymentIntents(), r))
}

func (s *Server) handlePaymentIntentSearch(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	if r.URL.Query().Get("query") == "" {
		writeStripeError(w, http.StatusBadRequest, "invalid_request_error", "parameter_missing", "Missing required param: query")
		return
	}
	writeSearchList(w, "/v1/payment_intents/search", limit(paymentIntents(), r))
}

func (s *Server) handlePaymentIntentGet(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/v1/payment_intents/")
	writeOneByID(w, paymentIntents(), id, "payment_intent")
}

func (s *Server) handleChargesList(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	items := charges()
	if pi := r.URL.Query().Get("payment_intent"); pi != "" {
		items = filterByString(items, "payment_intent", pi)
	}
	writeList(w, "/v1/charges", limit(items, r))
}

func (s *Server) handleChargeSearch(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	if r.URL.Query().Get("query") == "" {
		writeStripeError(w, http.StatusBadRequest, "invalid_request_error", "parameter_missing", "Missing required param: query")
		return
	}
	writeSearchList(w, "/v1/charges/search", limit(charges(), r))
}

func (s *Server) handleChargeGet(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/v1/charges/")
	writeOneByID(w, charges(), id, "charge")
}

func (s *Server) handleDisputesList(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	items := disputes()
	if charge := r.URL.Query().Get("charge"); charge != "" {
		items = filterByString(items, "charge", charge)
	}
	if pi := r.URL.Query().Get("payment_intent"); pi != "" {
		items = filterByString(items, "payment_intent", pi)
	}
	writeList(w, "/v1/disputes", limit(items, r))
}

func (s *Server) handleDisputeGet(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/v1/disputes/")
	writeOneByID(w, disputes(), id, "dispute")
}

func (s *Server) handleSubscriptionsList(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	items := subscriptions()
	if customer := r.URL.Query().Get("customer"); customer != "" {
		items = filterByString(items, "customer", customer)
	}
	if status := r.URL.Query().Get("status"); status != "" && status != "all" {
		items = filterByString(items, "status", status)
	}
	writeList(w, "/v1/subscriptions", limit(items, r))
}

func (s *Server) handleSubscriptionSearch(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	if r.URL.Query().Get("query") == "" {
		writeStripeError(w, http.StatusBadRequest, "invalid_request_error", "parameter_missing", "Missing required param: query")
		return
	}
	writeSearchList(w, "/v1/subscriptions/search", limit(subscriptions(), r))
}

func (s *Server) handleSubscriptionGet(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/v1/subscriptions/")
	writeOneByID(w, subscriptions(), id, "subscription")
}

func (s *Server) handleSubscriptionItemsList(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	items := subscriptionItems()
	if subscription := r.URL.Query().Get("subscription"); subscription != "" {
		items = filterByString(items, "subscription", subscription)
	}
	writeList(w, "/v1/subscription_items", limit(items, r))
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
	writeList(w, "/v1/invoices", limit(items, r))
}

func (s *Server) handleAccountsList(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	writeList(w, "/v1/accounts", limit(accounts(), r))
}

func (s *Server) handleAccountGet(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/v1/accounts/")
	writeOneByID(w, accounts(), id, "account")
}

func hasBasicKey(r *http.Request) bool {
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Basic ") {
		return false
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(header, "Basic "))
	if err != nil {
		return false
	}
	key := strings.TrimSuffix(string(raw), ":")
	return strings.HasPrefix(key, "sk_") || strings.HasPrefix(key, "rk_")
}

func requireGet(w http.ResponseWriter, r *http.Request) bool {
	if r.Method == http.MethodGet {
		return true
	}
	writeStripeError(w, http.StatusMethodNotAllowed, "invalid_request_error", "method_not_allowed", "Only GET is supported by mockstripe")
	return false
}

func writeOneByID(w http.ResponseWriter, items []map[string]any, id string, objectName string) {
	for _, item := range items {
		if item["id"] == id {
			writeJSON(w, http.StatusOK, item)
			return
		}
	}
	writeStripeError(w, http.StatusNotFound, "invalid_request_error", "resource_missing", "No such "+objectName+": '"+id+"'")
}

func writeList(w http.ResponseWriter, path string, items []map[string]any) {
	writeJSON(w, http.StatusOK, map[string]any{
		"object":   "list",
		"url":      path,
		"has_more": false,
		"data":     items,
	})
}

func writeSearchList(w http.ResponseWriter, path string, items []map[string]any) {
	writeJSON(w, http.StatusOK, map[string]any{
		"object":    "search_result",
		"url":       path,
		"has_more":  false,
		"next_page": nil,
		"data":      items,
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

func limit(items []map[string]any, r *http.Request) []map[string]any {
	n := 10
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed >= 0 {
			n = parsed
		}
	}
	if n > len(items) {
		n = len(items)
	}
	return items[:n]
}

func filterByEventType(items []map[string]any, eventType string) []map[string]any {
	if eventType == "" {
		return items
	}
	return filterByString(items, "type", eventType)
}

func filterByString(items []map[string]any, key, want string) []map[string]any {
	filtered := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if got, _ := item[key].(string); got == want {
			filtered = append(filtered, item)
		}
	}
	return filtered
}
