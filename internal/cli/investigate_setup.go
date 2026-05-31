package cli

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func newInvestigateSetup(globals shared.GlobalsFunc, outputOpts *investigationOutputOptions) *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:   "setup <setup-intent-id|payment-method-id|customer-id>",
		Short: "Explain saved-payment setup status and reusable payment method readiness",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithInvestigator(globals(), outputOpts, func(inv investigator) ([]evidenceRecord, error) {
				return inv.setup(args[0], limit)
			})
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 5, "Maximum SetupIntents to inspect for customer or payment method input")
	return cmd
}

func (i investigator) setup(id string, limit int) ([]evidenceRecord, error) {
	if err := validateAllowedStripeID(id, "setup_intent", "payment_method", "customer"); err != nil {
		return nil, err
	}
	records := []evidenceRecord{}
	setupIntents := []map[string]any{}
	switch {
	case strings.HasPrefix(id, "seti_"):
		seti, err := i.get("/v1/setup_intents/"+url.PathEscape(id), url.Values{})
		if err != nil {
			return nil, err
		}
		setupIntents = append(setupIntents, seti)
	case strings.HasPrefix(id, "pm_"):
		pm, err := i.get("/v1/payment_methods/"+url.PathEscape(id), url.Values{})
		if err != nil {
			return nil, err
		}
		records = i.appendEvidence(records, entityRecord("payment_method", pm))
		found, err := i.list("/v1/setup_intents", valuesWithLimit(limit, "payment_method", id))
		if err != nil {
			return nil, err
		}
		setupIntents = found
	default:
		customer, err := i.get("/v1/customers/"+url.PathEscape(id), url.Values{})
		if err != nil {
			return nil, err
		}
		records = i.appendEvidence(records, entityRecord("customer", customer))
		found, err := i.list("/v1/setup_intents", valuesWithLimit(limit, "customer", id))
		if err != nil {
			return nil, err
		}
		setupIntents = found
	}
	for _, seti := range setupIntents {
		records = i.appendEvidence(records, entityRecord("setup_intent", seti))
		if pmID := idFromValue(seti["payment_method"]); pmID != "" {
			if pm, err := i.get("/v1/payment_methods/"+url.PathEscape(pmID), url.Values{}); err == nil {
				records = i.appendEvidence(records, entityRecord("payment_method", pm))
			}
		}
		records = i.appendEvidence(records, setupFinding(seti))
	}
	if len(setupIntents) == 0 {
		records = i.appendEvidence(records, evidenceRecord{Type: "finding", Severity: "warning", Summary: "No SetupIntents found for " + id + "."})
	}
	return records, nil
}

func setupFinding(seti map[string]any) evidenceRecord {
	status := mapString(seti, "status")
	severity := "info"
	if status != "succeeded" {
		severity = "warning"
	}
	return evidenceRecord{Type: "finding", Severity: severity, Summary: fmt.Sprintf("SetupIntent %s status=%s usage=%s.", mapString(seti, "id"), status, mapString(seti, "usage")), Data: map[string]any{
		"setup_intent":     mapString(seti, "id"),
		"customer":         idFromValue(seti["customer"]),
		"payment_method":   idFromValue(seti["payment_method"]),
		"status":           status,
		"usage":            mapString(seti, "usage"),
		"last_setup_error": mapAnyMap(seti, "last_setup_error"),
	}}
}
