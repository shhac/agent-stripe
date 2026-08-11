package cli

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func v2AccountJSONWithID(id string) string {
	return strings.Replace(v2RestrictedAccountJSON, "acct_v2_restricted", id, 1)
}

func v2AccountServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/core/accounts/acct_v2":
			if got := r.Header.Get("Authorization"); !strings.HasPrefix(got, "Bearer ") {
				t.Fatalf("v2 Authorization = %q, want Bearer", got)
			}
			if got := r.URL.Query().Get("include[6]"); got != "requirements" {
				t.Fatalf("include[6] = %q, want every include requested by default", got)
			}
			fmt.Fprint(w, v2AccountJSONWithID("acct_v2"))
		case "/v2/core/accounts/acct_v2/persons":
			fmt.Fprint(w, `{"data":[{"id":"person_1","object":"v2.core.account_person","account":"acct_v2","relationship":{"representative":true}}],"next_page_url":null}`)
		case "/v2/core/events":
			if got := r.URL.Query().Get("object_id"); got != "acct_v2" {
				t.Fatalf("object_id = %q", got)
			}
			fmt.Fprint(w, `{"data":[
              {"id":"evt_test_1","object":"v2.core.event","type":"v2.core.account[configuration.merchant].capability_status_updated","created":"2026-07-30T09:14:02.000Z","related_object":{"id":"acct_v2","type":"v2.core.account","url":"/v2/core/accounts/acct_v2?include=configuration.merchant"}},
              {"id":"evt_test_2","object":"v2.core.event","type":"v2.core.account[requirements].updated","created":"2026-07-30T09:13:58.000Z","related_object":{"id":"acct_v2","type":"v2.core.account","url":"/v2/core/accounts/acct_v2?include=requirements"}}
            ],"next_page_url":null}`)
		case "/v1/transfers":
			fmt.Fprint(w, `{"object":"list","data":[],"has_more":false}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
}

func TestAccountHealthReadsV2AccountWhenAvailable(t *testing.T) {
	server := v2AccountServer(t)
	defer server.Close()

	inv := testInvestigator(server)
	err := inv.accountHealth("acct_v2", namespaceAuto)
	if err != nil {
		t.Fatalf("accountHealth() error = %v", err)
	}
	records := inv.records()
	assertRecordObject(t, records, objectV2Account, "acct_v2")
	assertRecordObject(t, records, objectV2Person, "person_1")
	finding := findFinding(records, "Accounts v2 account acct_v2")
	if finding == nil {
		t.Fatalf("records = %#v, want a v2 account finding", records)
	}
	if finding.Severity != "warning" {
		t.Fatalf("Severity = %q, want warning", finding.Severity)
	}
	if !strings.Contains(finding.Summary, "merchant.card_payments") {
		t.Fatalf("summary should name the restricted capability: %s", finding.Summary)
	}
}

func TestAccountHealthNamespaceV2DoesNotFallBack(t *testing.T) {
	var v1Reads int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/v1/") {
			v1Reads++
		}
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error":{"type":"invalid_request_error","code":"v1_account_instead_of_v2_account","message":"nope"}}`)
	}))
	defer server.Close()

	if err := testInvestigator(server).accountHealth("acct_v1_only", namespaceV2); err == nil {
		t.Fatalf("accountHealth(--namespace v2) error = nil, want the v2 error surfaced")
	}
	if v1Reads != 0 {
		t.Fatalf("v1 reads = %d, want 0 when the namespace is pinned to v2", v1Reads)
	}
}

func TestAccountHealthNamespaceV1SkipsTheV2Probe(t *testing.T) {
	var v2Reads int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/v2/"):
			v2Reads++
			w.WriteHeader(http.StatusInternalServerError)
		case r.URL.Path == "/v1/accounts/acct_v1":
			fmt.Fprint(w, `{"id":"acct_v1","object":"account","charges_enabled":true,"payouts_enabled":true}`)
		default:
			fmt.Fprint(w, `{"object":"list","data":[],"has_more":false}`)
		}
	}))
	defer server.Close()

	inv := testInvestigator(server)
	err := inv.accountHealth("acct_v1", namespaceV1)
	if err != nil {
		t.Fatalf("accountHealth() error = %v", err)
	}
	records := inv.records()
	if v2Reads != 0 {
		t.Fatalf("v2 reads = %d, want 0 when the namespace is pinned to v1", v2Reads)
	}
	if findFinding(records, "Connect v1 account acct_v1") == nil {
		t.Fatalf("records = %#v, want the v1 finding", records)
	}
}

func TestAccountHealthSurfacesNonNamespaceErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error":{"type":"authentication_error","code":"api_key_expired","message":"expired"}}`)
	}))
	defer server.Close()

	// A 401 is not "this is a v1 account": retrying against v1 would hide a
	// credential problem behind a second identical failure.
	err := testInvestigator(server).accountHealth("acct_any", namespaceAuto)
	if err == nil || !strings.Contains(err.Error(), "Authentication failed") {
		t.Fatalf("accountHealth() error = %v, want the auth error surfaced", err)
	}
}

func TestAccountEventsSummarizesV2ThinEvents(t *testing.T) {
	server := v2AccountServer(t)
	defer server.Close()

	inv := testInvestigator(server)
	err := inv.accountEvents("acct_v2", 20, nil)
	if err != nil {
		t.Fatalf("accountEvents() error = %v", err)
	}
	records := inv.records()
	assertRecordObject(t, records, objectV2Event, "evt_test_1")
	finding := findFinding(records, "Account acct_v2 has 2 v2 event(s)")
	if finding == nil {
		t.Fatalf("records = %#v, want an events finding", records)
	}
	if finding.Severity != "warning" {
		t.Fatalf("Severity = %q, want warning for capability/requirement changes", finding.Severity)
	}
	if !strings.Contains(finding.Summary, "current, not point-in-time") {
		t.Fatalf("summary should label thin-event state: %s", finding.Summary)
	}
}

func TestAccountEventsExplainsEmptyResultForV1Accounts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/core/accounts/acct_v1_only":
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, `{"error":{"type":"invalid_request_error","code":"v1_account_instead_of_v2_account","message":"nope"}}`)
		case "/v2/core/events":
			fmt.Fprint(w, `{"data":[],"next_page_url":null}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	inv := testInvestigator(server)
	err := inv.accountEvents("acct_v1_only", 20, nil)
	if err != nil {
		t.Fatalf("accountEvents() error = %v", err)
	}
	records := inv.records()
	if findFinding(records, "Account acct_v1_only is not an Accounts v2 account") == nil {
		t.Fatalf("records = %#v, want the namespace note", records)
	}
	if findFinding(records, "No v2 core events for account acct_v1_only") == nil {
		t.Fatalf("records = %#v, want the empty-result explanation", records)
	}
}

func TestResolveNamesTheAccountNamespace(t *testing.T) {
	server := v2AccountServer(t)
	defer server.Close()

	inv := testInvestigator(server)
	err := inv.resolve("acct_v2")
	if err != nil {
		t.Fatalf("resolve() error = %v", err)
	}
	records := inv.records()
	finding := findFinding(records, "Resolved acct_v2 as an Accounts v2 account")
	if finding == nil {
		t.Fatalf("records = %#v, want a namespace-aware resolution", records)
	}
	if finding.Data["namespace"] != namespaceV2 {
		t.Fatalf("namespace = %v", finding.Data["namespace"])
	}
}

func TestWebhookEventFollowsV2ThinEventRelatedObject(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/core/events/evt_test_1":
			fmt.Fprint(w, `{"id":"evt_test_1","object":"v2.core.event","type":"v2.core.account[requirements].updated","created":"2026-07-30T09:13:58.000Z","related_object":{"id":"acct_v2","type":"v2.core.account","url":"/v2/core/accounts/acct_v2?include=requirements"}}`)
		case "/v2/core/accounts/acct_v2":
			if got := r.URL.Query().Get("include"); got != "requirements" {
				t.Fatalf("include = %q, want the event's own include set", got)
			}
			fmt.Fprint(w, v2RestrictedAccountJSON)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	inv := testInvestigator(server)
	err := inv.webhookEvent("evt_test_1")
	if err != nil {
		t.Fatalf("webhookEvent() error = %v", err)
	}
	records := inv.records()
	assertRecordObject(t, records, objectV2Event, "evt_test_1")
	assertRecordObject(t, records, objectV2Account, "acct_v2_restricted")
	if findFinding(records, "Event evt_test_1 is v2.core.account[requirements].updated") == nil {
		t.Fatalf("records = %#v, want a thin-event finding", records)
	}
}
