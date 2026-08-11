package mockstripe

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
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
	if r.URL.Path == "/" || r.URL.Path == "/healthz" {
		s.mux.ServeHTTP(w, r)
		return
	}
	// Stripe's namespaces authenticate differently: /v1 accepts the Basic form
	// the SDKs send, /v2 is documented as Bearer. The mock enforces the split so
	// a client that gets it wrong fails here rather than in production.
	if strings.HasPrefix(r.URL.Path, "/v2/") {
		if !hasBearerKey(r) {
			writeV2Error(w, http.StatusUnauthorized, "authentication_error", "api_key_missing", "No Bearer API key provided")
			return
		}
		s.mux.ServeHTTP(w, r)
		return
	}
	if !hasBasicKey(r) {
		writeStripeError(w, http.StatusUnauthorized, "authentication_error", "api_key_missing", "No API key provided")
		return
	}
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	s.mux.HandleFunc("/", s.handleIndex)
	s.mux.HandleFunc("/healthz", s.handleHealthz)
	s.mux.HandleFunc("/v1/account", s.handleSelfAccount)
	s.mux.HandleFunc("/v1/balance", s.handleBalance)
	s.mux.HandleFunc("/v1/checkout/sessions", s.handleCheckoutSessionsList)
	s.mux.HandleFunc("/v1/checkout/sessions/", s.handleCheckoutSessionGetOrLines)
	s.mux.HandleFunc("/v1/invoices/search", s.handleInvoiceSearch)
	s.mux.HandleFunc("/v1/invoices/create_preview", s.handleInvoicePreview)
	s.mux.HandleFunc("/v1/invoices/", s.handleInvoiceGetOrLines)
	s.mux.HandleFunc("/v1/invoices", s.handleInvoicesList)
	s.mux.HandleFunc("/v1/transfers", s.handleTransfersList)
	s.mux.HandleFunc("/v1/transfers/", s.handleTransferGetOrReversal)
	s.mux.HandleFunc("/v2/core/accounts", s.handleV2Accounts)
	s.mux.HandleFunc("/v2/core/accounts/", s.handleV2AccountPath)
	s.mux.HandleFunc("/v2/core/events", s.handleV2Events)
	s.mux.HandleFunc("/v2/core/events/", s.handleV2Event)
	for _, resource := range mockResources() {
		s.registerMockResource(resource)
	}
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		writeStripeError(w, http.StatusNotFound, "invalid_request_error", "resource_missing", "No such mockstripe route: "+r.URL.Path)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"service": "mockstripe",
		"routes":  Routes(),
	})
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
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

func (s *Server) handleCheckoutSessionsList(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	items := checkoutSessions()
	if customer := r.URL.Query().Get("customer"); customer != "" {
		items = filterByString(items, "customer", customer)
	}
	if pi := r.URL.Query().Get("payment_intent"); pi != "" {
		items = filterByString(items, "payment_intent", pi)
	}
	if subscription := r.URL.Query().Get("subscription"); subscription != "" {
		items = filterByString(items, "subscription", subscription)
	}
	if paymentLink := r.URL.Query().Get("payment_link"); paymentLink != "" {
		items = filterByString(items, "payment_link", paymentLink)
	}
	writeList(w, "/v1/checkout/sessions", items, r)
}

func (s *Server) handleCheckoutSessionGetOrLines(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	rest := strings.TrimPrefix(r.URL.Path, "/v1/checkout/sessions/")
	if strings.HasSuffix(rest, "/line_items") {
		id := strings.TrimSuffix(rest, "/line_items")
		writeList(w, "/v1/checkout/sessions/"+id+"/line_items", checkoutLineItems(id), r)
		return
	}
	writeOneByID(w, checkoutSessions(), rest, "checkout.session")
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

func (s *Server) handleTransfersList(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	items := transfers()
	if destination := r.URL.Query().Get("destination"); destination != "" {
		items = filterByString(items, "destination", destination)
	}
	if group := r.URL.Query().Get("transfer_group"); group != "" {
		items = filterByString(items, "transfer_group", group)
	}
	writeList(w, "/v1/transfers", items, r)
}

func (s *Server) handleTransferGetOrReversal(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	rest := strings.TrimPrefix(r.URL.Path, "/v1/transfers/")
	if strings.Contains(rest, "/reversals/") {
		parts := strings.Split(rest, "/reversals/")
		writeOneByID(w, transferReversals(parts[0]), parts[1], "transfer_reversal")
		return
	}
	writeOneByID(w, transfers(), rest, "transfer")
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
	return isStripeKey(strings.TrimSuffix(string(raw), ":"))
}

func hasBearerKey(r *http.Request) bool {
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return false
	}
	return isStripeKey(strings.TrimPrefix(header, "Bearer "))
}

func isStripeKey(key string) bool {
	return strings.HasPrefix(key, "sk_") || strings.HasPrefix(key, "rk_")
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

type pagedList struct {
	items   []map[string]any
	hasMore bool
}

func listPage(items []map[string]any, r *http.Request) pagedList {
	items = applyCursor(items, r.URL.Query())
	n := 10
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed >= 0 {
			n = parsed
		}
	}
	if n >= len(items) {
		return pagedList{items: items}
	}
	return pagedList{items: items[:n], hasMore: true}
}

func applyCursor(items []map[string]any, query url.Values) []map[string]any {
	if startingAfter := query.Get("starting_after"); startingAfter != "" {
		if idx := indexByID(items, startingAfter); idx >= 0 && idx+1 < len(items) {
			items = items[idx+1:]
		} else if idx >= 0 {
			items = nil
		}
	}
	if endingBefore := query.Get("ending_before"); endingBefore != "" {
		if idx := indexByID(items, endingBefore); idx >= 0 {
			items = items[:idx]
		}
	}
	return items
}

func indexByID(items []map[string]any, id string) int {
	for idx, item := range items {
		if stringValue(item, "id") == id {
			return idx
		}
	}
	return -1
}

func stringValue(item map[string]any, key string) string {
	value, _ := item[key].(string)
	return value
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

func filterByBoolString(items []map[string]any, key, want string) []map[string]any {
	filtered := make([]map[string]any, 0, len(items))
	wantBool := want == "true"
	for _, item := range items {
		if got, _ := item[key].(bool); got == wantBool {
			filtered = append(filtered, item)
		}
	}
	return filtered
}
