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
	s.mux.HandleFunc("/v1/checkout/sessions", s.handleCheckoutSessionsList)
	s.mux.HandleFunc("/v1/checkout/sessions/", s.handleCheckoutSessionGetOrLines)
	s.mux.HandleFunc("/v1/customers/search", s.handleCustomerSearch)
	s.mux.HandleFunc("/v1/customers", s.handleCustomersList)
	s.mux.HandleFunc("/v1/customers/", s.handleCustomerGet)
	s.mux.HandleFunc("/v1/events", s.handleEventsList)
	s.mux.HandleFunc("/v1/events/", s.handleEventGet)
	s.mux.HandleFunc("/v1/products/search", s.handleProductSearch)
	s.mux.HandleFunc("/v1/products", s.handleProductsList)
	s.mux.HandleFunc("/v1/products/", s.handleProductGet)
	s.mux.HandleFunc("/v1/prices/search", s.handlePriceSearch)
	s.mux.HandleFunc("/v1/prices", s.handlePricesList)
	s.mux.HandleFunc("/v1/prices/", s.handlePriceGet)
	s.mux.HandleFunc("/v1/invoices/search", s.handleInvoiceSearch)
	s.mux.HandleFunc("/v1/invoices/create_preview", s.handleInvoicePreview)
	s.mux.HandleFunc("/v1/invoices/", s.handleInvoiceGetOrLines)
	s.mux.HandleFunc("/v1/invoices", s.handleInvoicesList)
	s.mux.HandleFunc("/v1/payment_intents/search", s.handlePaymentIntentSearch)
	s.mux.HandleFunc("/v1/payment_intents", s.handlePaymentIntentsList)
	s.mux.HandleFunc("/v1/payment_intents/", s.handlePaymentIntentGet)
	s.mux.HandleFunc("/v1/setup_intents", s.handleSetupIntentsList)
	s.mux.HandleFunc("/v1/setup_intents/", s.handleSetupIntentGet)
	s.mux.HandleFunc("/v1/payment_methods", s.handlePaymentMethodsList)
	s.mux.HandleFunc("/v1/payment_methods/", s.handlePaymentMethodGet)
	s.mux.HandleFunc("/v1/charges/search", s.handleChargeSearch)
	s.mux.HandleFunc("/v1/charges", s.handleChargesList)
	s.mux.HandleFunc("/v1/charges/", s.handleChargeGet)
	s.mux.HandleFunc("/v1/disputes", s.handleDisputesList)
	s.mux.HandleFunc("/v1/disputes/", s.handleDisputeGet)
	s.mux.HandleFunc("/v1/refunds", s.handleRefundsList)
	s.mux.HandleFunc("/v1/refunds/", s.handleRefundGet)
	s.mux.HandleFunc("/v1/subscriptions/search", s.handleSubscriptionSearch)
	s.mux.HandleFunc("/v1/subscriptions", s.handleSubscriptionsList)
	s.mux.HandleFunc("/v1/subscriptions/", s.handleSubscriptionGet)
	s.mux.HandleFunc("/v1/subscription_items", s.handleSubscriptionItemsList)
	s.mux.HandleFunc("/v1/transfers", s.handleTransfersList)
	s.mux.HandleFunc("/v1/transfers/", s.handleTransferGetOrReversal)
	s.mux.HandleFunc("/v1/payouts", s.handlePayoutsList)
	s.mux.HandleFunc("/v1/payouts/", s.handlePayoutGet)
	s.mux.HandleFunc("/v1/balance_transactions", s.handleBalanceTransactionsList)
	s.mux.HandleFunc("/v1/balance_transactions/", s.handleBalanceTransactionGet)
	s.mux.HandleFunc("/v1/application_fees", s.handleApplicationFeesList)
	s.mux.HandleFunc("/v1/application_fees/", s.handleApplicationFeeGet)
	s.mux.HandleFunc("/v1/payment_links", s.handlePaymentLinksList)
	s.mux.HandleFunc("/v1/payment_links/", s.handlePaymentLinkGet)
	s.mux.HandleFunc("/v1/radar/early_fraud_warnings", s.handleEarlyFraudWarningsList)
	s.mux.HandleFunc("/v1/radar/early_fraud_warnings/", s.handleEarlyFraudWarningGet)
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
	writeList(w, "/v1/checkout/sessions", limit(items, r))
}

func (s *Server) handleCheckoutSessionGetOrLines(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	rest := strings.TrimPrefix(r.URL.Path, "/v1/checkout/sessions/")
	if strings.HasSuffix(rest, "/line_items") {
		id := strings.TrimSuffix(rest, "/line_items")
		writeList(w, "/v1/checkout/sessions/"+id+"/line_items", limit(checkoutLineItems(id), r))
		return
	}
	writeOneByID(w, checkoutSessions(), rest, "checkout.session")
}

func (s *Server) handleCustomersList(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	items := customers()
	if email := r.URL.Query().Get("email"); email != "" {
		items = filterByString(items, "email", email)
	}
	writeList(w, "/v1/customers", limit(items, r))
}

func (s *Server) handleCustomerSearch(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	if r.URL.Query().Get("query") == "" {
		writeStripeError(w, http.StatusBadRequest, "invalid_request_error", "parameter_missing", "Missing required param: query")
		return
	}
	writeSearchList(w, "/v1/customers/search", limit(customers(), r))
}

func (s *Server) handleCustomerGet(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/v1/customers/")
	writeOneByID(w, customers(), id, "customer")
}

func (s *Server) handleEventGet(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/v1/events/")
	writeOneByID(w, events(), id, "event")
}

func (s *Server) handleProductsList(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	items := products()
	if active := r.URL.Query().Get("active"); active != "" {
		items = filterByBoolString(items, "active", active)
	}
	writeList(w, "/v1/products", limit(items, r))
}

func (s *Server) handleProductSearch(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	if r.URL.Query().Get("query") == "" {
		writeStripeError(w, http.StatusBadRequest, "invalid_request_error", "parameter_missing", "Missing required param: query")
		return
	}
	writeSearchList(w, "/v1/products/search", limit(products(), r))
}

func (s *Server) handleProductGet(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/v1/products/")
	writeOneByID(w, products(), id, "product")
}

func (s *Server) handlePricesList(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	items := prices()
	if active := r.URL.Query().Get("active"); active != "" {
		items = filterByBoolString(items, "active", active)
	}
	if product := r.URL.Query().Get("product"); product != "" {
		items = filterByString(items, "product", product)
	}
	if typ := r.URL.Query().Get("type"); typ != "" {
		items = filterByString(items, "type", typ)
	}
	writeList(w, "/v1/prices", limit(items, r))
}

func (s *Server) handlePriceSearch(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	if r.URL.Query().Get("query") == "" {
		writeStripeError(w, http.StatusBadRequest, "invalid_request_error", "parameter_missing", "Missing required param: query")
		return
	}
	writeSearchList(w, "/v1/prices/search", limit(prices(), r))
}

func (s *Server) handlePriceGet(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/v1/prices/")
	writeOneByID(w, prices(), id, "price")
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

func (s *Server) handleSetupIntentsList(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	items := setupIntents()
	if customer := r.URL.Query().Get("customer"); customer != "" {
		items = filterByString(items, "customer", customer)
	}
	if pm := r.URL.Query().Get("payment_method"); pm != "" {
		items = filterByString(items, "payment_method", pm)
	}
	writeList(w, "/v1/setup_intents", limit(items, r))
}

func (s *Server) handleSetupIntentGet(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/v1/setup_intents/")
	writeOneByID(w, setupIntents(), id, "setup_intent")
}

func (s *Server) handlePaymentMethodsList(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	items := paymentMethods()
	if customer := r.URL.Query().Get("customer"); customer != "" {
		items = filterByString(items, "customer", customer)
	}
	if typ := r.URL.Query().Get("type"); typ != "" {
		items = filterByString(items, "type", typ)
	}
	writeList(w, "/v1/payment_methods", limit(items, r))
}

func (s *Server) handlePaymentMethodGet(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/v1/payment_methods/")
	writeOneByID(w, paymentMethods(), id, "payment_method")
}

func (s *Server) handleChargesList(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	items := charges()
	if pi := r.URL.Query().Get("payment_intent"); pi != "" {
		items = filterByString(items, "payment_intent", pi)
	}
	if customer := r.URL.Query().Get("customer"); customer != "" {
		items = filterByString(items, "customer", customer)
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

func (s *Server) handleRefundsList(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	items := refunds()
	if charge := r.URL.Query().Get("charge"); charge != "" {
		items = filterByString(items, "charge", charge)
	}
	if pi := r.URL.Query().Get("payment_intent"); pi != "" {
		items = filterByString(items, "payment_intent", pi)
	}
	writeList(w, "/v1/refunds", limit(items, r))
}

func (s *Server) handleRefundGet(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/v1/refunds/")
	writeOneByID(w, refunds(), id, "refund")
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

func (s *Server) handleInvoiceSearch(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	if r.URL.Query().Get("query") == "" {
		writeStripeError(w, http.StatusBadRequest, "invalid_request_error", "parameter_missing", "Missing required param: query")
		return
	}
	writeSearchList(w, "/v1/invoices/search", limit(invoices(), r))
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
	writeList(w, "/v1/invoices", limit(items, r))
}

func (s *Server) handleInvoiceGetOrLines(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	rest := strings.TrimPrefix(r.URL.Path, "/v1/invoices/")
	if strings.HasSuffix(rest, "/lines") {
		id := strings.TrimSuffix(rest, "/lines")
		writeList(w, "/v1/invoices/"+id+"/lines", limit(invoiceLines(id), r))
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
	writeList(w, "/v1/transfers", limit(items, r))
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

func (s *Server) handlePayoutsList(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	items := payouts()
	if status := r.URL.Query().Get("status"); status != "" {
		items = filterByString(items, "status", status)
	}
	writeList(w, "/v1/payouts", limit(items, r))
}

func (s *Server) handlePayoutGet(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/v1/payouts/")
	writeOneByID(w, payouts(), id, "payout")
}

func (s *Server) handleBalanceTransactionsList(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	items := balanceTransactions()
	if typ := r.URL.Query().Get("type"); typ != "" {
		items = filterByString(items, "type", typ)
	}
	if payout := r.URL.Query().Get("payout"); payout != "" {
		items = filterByString(items, "payout", payout)
	}
	writeList(w, "/v1/balance_transactions", limit(items, r))
}

func (s *Server) handleBalanceTransactionGet(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/v1/balance_transactions/")
	writeOneByID(w, balanceTransactions(), id, "balance_transaction")
}

func (s *Server) handleApplicationFeesList(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	items := applicationFees()
	if charge := r.URL.Query().Get("charge"); charge != "" {
		items = filterByString(items, "charge", charge)
	}
	writeList(w, "/v1/application_fees", limit(items, r))
}

func (s *Server) handleApplicationFeeGet(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/v1/application_fees/")
	writeOneByID(w, applicationFees(), id, "application_fee")
}

func (s *Server) handlePaymentLinksList(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	items := paymentLinks()
	if active := r.URL.Query().Get("active"); active != "" {
		items = filterByBoolString(items, "active", active)
	}
	writeList(w, "/v1/payment_links", limit(items, r))
}

func (s *Server) handlePaymentLinkGet(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/v1/payment_links/")
	writeOneByID(w, paymentLinks(), id, "payment_link")
}

func (s *Server) handleEarlyFraudWarningsList(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	items := earlyFraudWarnings()
	if charge := r.URL.Query().Get("charge"); charge != "" {
		items = filterByString(items, "charge", charge)
	}
	if pi := r.URL.Query().Get("payment_intent"); pi != "" {
		items = filterByString(items, "payment_intent", pi)
	}
	writeList(w, "/v1/radar/early_fraud_warnings", limit(items, r))
}

func (s *Server) handleEarlyFraudWarningGet(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/v1/radar/early_fraud_warnings/")
	writeOneByID(w, earlyFraudWarnings(), id, "early_fraud_warning")
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
