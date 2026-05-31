package cli

import (
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func newInvestigateIncomingPayment(globals shared.GlobalsFunc, outputOpts *investigationOutputOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "incoming-payment <payment-intent-id|charge-id|invoice-id>",
		Short: "Explain what happened to a customer payment to you",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithInvestigator(globals(), outputOpts, func(inv investigator) ([]evidenceRecord, error) {
				return inv.incomingPayment(args[0])
			})
		},
	}
}

func (i investigator) incomingPayment(id string) ([]evidenceRecord, error) {
	if err := validateAllowedStripeID(id, "invoice", "charge", "payment_intent"); err != nil {
		return nil, err
	}
	switch {
	case strings.HasPrefix(id, "in_"):
		return i.invoicePayment(id)
	case strings.HasPrefix(id, "ch_"):
		charge, err := i.get("/v1/charges/"+url.PathEscape(id), url.Values{})
		if err != nil {
			return nil, err
		}
		return i.paymentIncidentFromCharge(charge)
	default:
		pi, err := i.get("/v1/payment_intents/"+url.PathEscape(id), url.Values{})
		if err != nil {
			return nil, err
		}
		return i.paymentIncidentFromPI(pi)
	}
}

func (i investigator) paymentIncidentFromPI(pi map[string]any) ([]evidenceRecord, error) {
	records := i.appendEvidence(nil, entityRecord("payment_intent", pi))
	charge, err := i.latestChargeForPaymentIntent(pi)
	if err != nil {
		return nil, err
	}
	if charge != nil {
		records = i.appendEvidence(records, entityRecord("charge", charge))
	}
	records = i.appendEvidenceAll(records, i.relatedDisputesAndRefunds(pi, charge))
	records = i.appendEvidence(records, evidenceRecord{
		Type:     "finding",
		Severity: severityForPayment(pi, charge),
		Summary:  paymentFailureSummary(pi, charge),
	})
	return records, nil
}

func (i investigator) paymentIncidentFromCharge(charge map[string]any) ([]evidenceRecord, error) {
	records := i.appendEvidence(nil, entityRecord("charge", charge))
	if piID := idFromValue(charge["payment_intent"]); piID != "" {
		pi, err := i.get("/v1/payment_intents/"+url.PathEscape(piID), url.Values{})
		if err == nil {
			records = i.appendEvidence(records, entityRecord("payment_intent", pi))
		}
	}
	records = i.appendEvidenceAll(records, i.relatedDisputesAndRefunds(nil, charge))
	records = i.appendEvidence(records, evidenceRecord{
		Type:     "finding",
		Severity: severityForPayment(nil, charge),
		Summary:  paymentFailureSummary(nil, charge),
	})
	return records, nil
}

func (i investigator) relatedDisputesAndRefunds(pi, charge map[string]any) []evidenceRecord {
	records := []evidenceRecord{}
	params := url.Values{}
	if charge != nil {
		shared.AddString(params, "charge", mapString(charge, "id"))
	}
	if pi != nil {
		shared.AddString(params, "payment_intent", mapString(pi, "id"))
	}
	if len(params) == 0 {
		return records
	}
	if disputes, err := i.list("/v1/disputes", params); err == nil {
		for _, dispute := range disputes {
			records = i.appendEvidence(records, entityRecord("dispute", dispute))
		}
	}
	if refunds, err := i.list("/v1/refunds", params); err == nil {
		for _, refund := range refunds {
			records = i.appendEvidence(records, entityRecord("refund", refund))
		}
	}
	return records
}

func (i investigator) latestChargeForPaymentIntent(pi map[string]any) (map[string]any, error) {
	if pi == nil {
		return nil, nil
	}
	if charge, ok := pi["latest_charge"].(map[string]any); ok {
		return charge, nil
	}
	chargeID := idFromValue(pi["latest_charge"])
	if chargeID != "" {
		return i.get("/v1/charges/"+url.PathEscape(chargeID), url.Values{})
	}
	charges, err := i.list("/v1/charges", url.Values{"payment_intent": []string{mapString(pi, "id")}, "limit": []string{"1"}})
	if err != nil || len(charges) == 0 {
		return nil, err
	}
	return charges[0], nil
}
