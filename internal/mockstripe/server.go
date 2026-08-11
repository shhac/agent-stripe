package mockstripe

import (
	"encoding/base64"
	"net/http"
	"strings"
)

type Server struct {
	mux    *http.ServeMux
	faults *faultRules
}

func NewServer() http.Handler {
	s := &Server{mux: http.NewServeMux(), faults: newFaultRules()}
	s.routes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Request-Id", "req_mock_123")
	w.Header().Set("Stripe-Mock", "true")
	if r.URL.Path == "/" || r.URL.Path == "/healthz" || r.URL.Path == "/_mock/faults" {
		s.mux.ServeHTTP(w, r)
		return
	}
	// Faults are applied after the route map but before auth, so a test can
	// drive the retry and degraded-evidence paths without a real Stripe.
	if s.faults.apply(w, r) {
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
	s.mux.HandleFunc("/_mock/faults", s.handleFaults)
	s.mux.HandleFunc("/v1/account", s.handleSelfAccount)
	s.mux.HandleFunc("/v1/balance", s.handleBalance)
	s.mux.HandleFunc("/v1/invoices/create_preview", s.handleInvoicePreview)
	s.mux.HandleFunc("/v1/accounts/", s.handleAccountSubresource)
	s.mux.HandleFunc("/v2/money_management/payout_methods", s.handleV2PayoutMethods)
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
