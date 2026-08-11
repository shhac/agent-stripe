package mockstripe

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
)

// Fault injection. Without it every mock response is healthy, so the code paths
// that matter most in triage — a retried 429, a related lookup that fails, a
// gateway returning HTML — are unreachable from the e2e suite and an
// investigation can claim "no disputes" when the dispute call actually failed.
//
// A fault is requested per path with the X-Mock-Fault header:
//
//	X-Mock-Fault: /v1/disputes=500          fail that path with a Stripe error
//	X-Mock-Fault: /v1/charges=429x2         429 twice, then succeed (retry path)
//	X-Mock-Fault: /v1/events=garbage        non-JSON body
//	X-Mock-Fault: *=500                     fail everything
//
// Several rules are comma-separated. Counting for the "x2" form is per server
// instance, so one httptest server per test keeps them independent.
// A client that cannot set headers — the e2e suite drives the CLI as a
// subprocess — arms the same rules out of band with
// POST /_mock/faults?rules=<spec>, which applies them to every later request.
type faultRules struct {
	mu        sync.Mutex
	remaining map[string]int
	standing  string
}

func newFaultRules() *faultRules {
	return &faultRules{remaining: map[string]int{}}
}

func (f *faultRules) arm(rules string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.standing = rules
	f.remaining = map[string]int{}
}

func (f *faultRules) standingRules() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.standing
}

type faultRule struct {
	status int
	body   string
	times  int
}

// apply writes a fault response and reports whether it handled the request.
func (f *faultRules) apply(w http.ResponseWriter, r *http.Request) bool {
	rule, key, ok := f.match(r)
	if !ok {
		return false
	}
	if rule.times > 0 && !f.consume(key, rule.times) {
		return false
	}
	if rule.body == "garbage" {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(rule.status)
		_, _ = w.Write([]byte("<html><body>upstream gateway error</body></html>"))
		return true
	}
	if rule.status == http.StatusTooManyRequests {
		w.Header().Set("Stripe-Rate-Limited-Reason", "endpoint-rate")
	}
	if strings.HasPrefix(r.URL.Path, "/v2/") {
		writeV2Error(w, rule.status, "invalid_request_error", "mock_injected_fault", "Injected fault for testing")
		return true
	}
	writeStripeError(w, rule.status, "invalid_request_error", "mock_injected_fault", "Injected fault for testing")
	return true
}

func (f *faultRules) match(r *http.Request) (faultRule, string, bool) {
	header := r.Header.Get("X-Mock-Fault")
	if header == "" {
		header = f.standingRules()
	}
	if header == "" {
		return faultRule{}, "", false
	}
	for _, entry := range strings.Split(header, ",") {
		pattern, spec, found := strings.Cut(strings.TrimSpace(entry), "=")
		if !found {
			continue
		}
		if pattern != "*" && !strings.HasPrefix(r.URL.Path, pattern) {
			continue
		}
		rule, ok := parseFaultSpec(spec)
		if !ok {
			continue
		}
		return rule, pattern + "|" + spec, true
	}
	return faultRule{}, "", false
}

func parseFaultSpec(spec string) (faultRule, bool) {
	if spec == "garbage" {
		return faultRule{status: http.StatusBadGateway, body: "garbage"}, true
	}
	statusPart, timesPart, repeated := strings.Cut(spec, "x")
	status, err := strconv.Atoi(statusPart)
	if err != nil || status < 400 || status > 599 {
		return faultRule{}, false
	}
	rule := faultRule{status: status}
	if repeated {
		times, err := strconv.Atoi(timesPart)
		if err != nil || times < 1 {
			return faultRule{}, false
		}
		rule.times = times
	}
	return rule, true
}

// consume reports whether this request still falls inside the rule's count.
func (f *faultRules) consume(key string, times int) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	used := f.remaining[key]
	if used >= times {
		return false
	}
	f.remaining[key] = used + 1
	return true
}

func (s *Server) handleFaults(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost, http.MethodDelete) {
		return
	}
	rules := ""
	if r.Method == http.MethodPost {
		rules = r.URL.Query().Get("rules")
	}
	s.faults.arm(rules)
	writeJSON(w, http.StatusOK, map[string]any{"faults": rules})
}
