package cli

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	agenterrors "github.com/shhac/agent-stripe/internal/errors"
)

func TestRefundRecoveryRequiresParentTransferForReversal(t *testing.T) {
	_, err := (investigator{}).refundRecovery("trr_123", "")
	var apiErr *agenterrors.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("refundRecovery() error = %#v, want APIError", err)
	}
	if apiErr.FixableBy != agenterrors.FixableByAgent {
		t.Fatalf("FixableBy = %q, want agent", apiErr.FixableBy)
	}
	if !strings.Contains(apiErr.Hint, "/v1/transfers/{transfer}/reversals/{reversal}") {
		t.Fatalf("Hint = %q, want nested reversal path", apiErr.Hint)
	}
}

func TestOutgoingPaymentFlagsDisabledConnectedAccount(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/accounts/acct_disabled":
			fmt.Fprint(w, `{"id":"acct_disabled","object":"account","charges_enabled":false,"payouts_enabled":false,"requirements":{"currently_due":["external_account"]}}`)
		case "/v1/transfers":
			fmt.Fprint(w, `{"object":"list","data":[],"has_more":false}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	records, err := testInvestigator(server).outgoingPayment("acct_disabled")
	if err != nil {
		t.Fatalf("outgoingPayment() error = %v", err)
	}
	assertRecordObject(t, records, "account", "acct_disabled")
	finding := findFinding(records, "Connected account acct_disabled is not fully enabled")
	if finding == nil || finding.Severity != "warning" {
		t.Fatalf("finding = %#v, want disabled account warning", finding)
	}
}
