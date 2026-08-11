package cli

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestCustomerContextReportsRelatedListErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/customers/cus_123":
			fmt.Fprint(w, `{"id":"cus_123","object":"customer"}`)
		case "/v1/payment_methods":
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprint(w, `{"error":{"type":"permission_error","message":"missing payment method permission"}}`)
		default:
			fmt.Fprint(w, `{"object":"list","data":[],"has_more":false}`)
		}
	}))
	defer server.Close()

	inv := testInvestigator(server)

	err := inv.customerContext("cus_123", 5)
	if err != nil {
		t.Fatalf("customerContext returned error: %v", err)
	}
	records := inv.records()

	warning := findFinding(records, "Could not gather payment_method context")
	if warning == nil {
		t.Fatalf("expected warning for failed payment_methods list, got %#v", records)
		return
	}
	if warning.Severity != "warning" {
		t.Fatalf("warning severity = %q, want warning", warning.Severity)
	}
	if warning.Data["path"] != "/v1/payment_methods" {
		t.Fatalf("warning path = %#v, want /v1/payment_methods", warning.Data["path"])
	}
}

func TestAddRelatedListReturnsItemsOnSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/charges" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("customer"); got != "cus_123" {
			t.Fatalf("customer query = %q, want cus_123", got)
		}
		fmt.Fprint(w, `{"object":"list","data":[{"id":"ch_123","object":"charge"}],"has_more":false}`)
	}))
	defer server.Close()

	inv := testInvestigator(server)

	items := inv.addRelatedList("charge", "/v1/charges", url.Values{"customer": []string{"cus_123"}})
	if len(items) != 1 || mapString(items[0], "id") != "ch_123" {
		t.Fatalf("items = %#v, want ch_123", items)
	}
	records := inv.records()
	if len(records) != 1 || records[0].Object != "charge" || records[0].ID != "ch_123" {
		t.Fatalf("records = %#v, want charge entity", records)
	}
}

func findFinding(records []evidenceRecord, prefix string) *evidenceRecord {
	for idx := range records {
		record := &records[idx]
		if record.Type == "finding" && len(record.Summary) >= len(prefix) && record.Summary[:len(prefix)] == prefix {
			return record
		}
	}
	return nil
}
