package cli

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func newInvestigateAccountHealth(globals shared.GlobalsFunc, outputOpts *investigationOutputOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "account-health <account-id>",
		Short: "Explain connected account requirements, capabilities, and money movement blockers",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithInvestigator(globals(), outputOpts, func(inv investigator) ([]evidenceRecord, error) {
				return inv.accountHealth(args[0])
			})
		},
	}
}

func newInvestigateDisputeImpact(globals shared.GlobalsFunc, outputOpts *investigationOutputOptions) *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:   "dispute-impact <dispute-id|charge-id|customer-id>",
		Short: "Summarize dispute exposure and related payment/refund evidence",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithInvestigator(globals(), outputOpts, func(inv investigator) ([]evidenceRecord, error) {
				return inv.disputeImpact(args[0], limit)
			})
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 5, "Maximum charges to inspect for customer input")
	return cmd
}

func newInvestigateFraudReview(globals shared.GlobalsFunc, outputOpts *investigationOutputOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "fraud-review <early-fraud-warning-id|charge-id|payment-intent-id>",
		Short: "Gather Radar early fraud warnings, disputes, refunds, and risk outcome",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithInvestigator(globals(), outputOpts, func(inv investigator) ([]evidenceRecord, error) {
				return inv.fraudReview(args[0])
			})
		},
	}
}

func (i investigator) accountHealth(accountID string) ([]evidenceRecord, error) {
	if err := validateExpectedStripeID(accountID, "account"); err != nil {
		return nil, err
	}
	account, err := i.get("/v1/accounts/"+url.PathEscape(accountID), url.Values{})
	if err != nil {
		return nil, err
	}
	records := []evidenceRecord{entityRecord("account", account), accountHealthFinding(account)}
	if transfers, err := i.list("/v1/transfers", valuesWithLimit(5, "destination", accountID)); err == nil {
		records = appendListRecords(records, "transfer", transfers)
	}
	return records, nil
}

func accountHealthFinding(account map[string]any) evidenceRecord {
	severity := "info"
	blockers := []string{}
	if !mapBool(account, "charges_enabled") {
		blockers = append(blockers, "charges disabled")
	}
	if !mapBool(account, "payouts_enabled") {
		blockers = append(blockers, "payouts disabled")
	}
	requirements := mapAnyMap(account, "requirements")
	if len(requirements) > 0 {
		blockers = append(blockers, "requirements present")
	}
	if len(blockers) > 0 {
		severity = "warning"
	}
	summary := fmt.Sprintf("Account %s charges_enabled=%t payouts_enabled=%t.", mapString(account, "id"), mapBool(account, "charges_enabled"), mapBool(account, "payouts_enabled"))
	if len(blockers) > 0 {
		summary += " Blockers: " + strings.Join(blockers, ", ") + "."
	}
	return evidenceRecord{
		Type:     "finding",
		Severity: severity,
		Summary:  summary,
		Data: map[string]any{
			"account":             mapString(account, "id"),
			"charges_enabled":     mapBool(account, "charges_enabled"),
			"payouts_enabled":     mapBool(account, "payouts_enabled"),
			"requirements":        requirements,
			"capabilities":        mapAnyMap(account, "capabilities"),
			"future_requirements": mapAnyMap(account, "future_requirements"),
		},
	}
}

func (i investigator) disputeImpact(id string, limit int) ([]evidenceRecord, error) {
	if err := validateAllowedStripeID(id, "dispute", "charge", "customer"); err != nil {
		return nil, err
	}
	disputes := []map[string]any{}
	switch {
	case strings.HasPrefix(id, "dp_"):
		dispute, err := i.get("/v1/disputes/"+url.PathEscape(id), url.Values{})
		if err != nil {
			return nil, err
		}
		disputes = append(disputes, dispute)
	case strings.HasPrefix(id, "ch_"):
		found, err := i.list("/v1/disputes", valuesWithLimit(10, "charge", id))
		if err != nil {
			return nil, err
		}
		disputes = found
	default:
		charges, err := i.list("/v1/charges", valuesWithLimit(limit, "customer", id))
		if err != nil {
			return nil, err
		}
		for _, charge := range charges {
			found, _ := i.list("/v1/disputes", valuesWithLimit(10, "charge", mapString(charge, "id")))
			disputes = append(disputes, found...)
		}
	}
	records := []evidenceRecord{}
	for _, dispute := range disputes {
		records = append(records, i.disputeImpactRecords(dispute)...)
	}
	if len(records) == 0 {
		records = append(records, evidenceRecord{Type: "finding", Severity: "info", Summary: "No dispute exposure found for " + id + "."})
	}
	return records, nil
}

func (i investigator) disputeImpactRecords(dispute map[string]any) []evidenceRecord {
	records := []evidenceRecord{entityRecord("dispute", dispute), disputeImpactFinding(dispute)}
	if chargeID := idFromValue(dispute["charge"]); chargeID != "" {
		if charge, err := i.get("/v1/charges/"+url.PathEscape(chargeID), url.Values{}); err == nil {
			records = append(records, entityRecord("charge", charge))
			if refunds, err := i.list("/v1/refunds", valuesWithLimit(5, "charge", chargeID)); err == nil {
				records = appendListRecords(records, "refund", refunds)
			}
		}
	}
	if piID := idFromValue(dispute["payment_intent"]); piID != "" {
		if pi, err := i.get("/v1/payment_intents/"+url.PathEscape(piID), url.Values{}); err == nil {
			records = append(records, entityRecord("payment_intent", pi))
		}
	}
	return records
}

func disputeImpactFinding(dispute map[string]any) evidenceRecord {
	severity := "info"
	status := mapString(dispute, "status")
	if status == "needs_response" || status == "under_review" || status == "warning_needs_response" {
		severity = "warning"
	}
	details := mapAnyMap(dispute, "evidence_details")
	summary := fmt.Sprintf("Dispute %s status=%s reason=%s amount=%s.", mapString(dispute, "id"), status, mapString(dispute, "reason"), formatAmount(dispute))
	if dueBy, ok := mapInt64(details, "due_by"); ok && dueBy > 0 {
		summary += fmt.Sprintf(" Evidence is due by Unix time %d.", dueBy)
	}
	return evidenceRecord{Type: "finding", Severity: severity, Summary: summary, Data: map[string]any{
		"dispute":          mapString(dispute, "id"),
		"charge":           idFromValue(dispute["charge"]),
		"payment_intent":   idFromValue(dispute["payment_intent"]),
		"amount":           mapValue(dispute, "amount"),
		"currency":         mapString(dispute, "currency"),
		"reason":           mapString(dispute, "reason"),
		"status":           status,
		"evidence_details": details,
	}}
}

func (i investigator) fraudReview(id string) ([]evidenceRecord, error) {
	if err := validateAllowedStripeID(id, "early_fraud_warning", "charge", "payment_intent"); err != nil {
		return nil, err
	}
	records := []evidenceRecord{}
	if strings.HasPrefix(id, "issfr_") {
		efw, err := i.get("/v1/radar/early_fraud_warnings/"+url.PathEscape(id), url.Values{})
		if err != nil {
			return nil, err
		}
		records = append(records, entityRecord("early_fraud_warning", efw))
		id = firstNonEmpty(idFromValue(efw["charge"]), idFromValue(efw["payment_intent"]))
	}
	if strings.HasPrefix(id, "pi_") {
		pi, err := i.get("/v1/payment_intents/"+url.PathEscape(id), url.Values{})
		if err != nil {
			return nil, err
		}
		records = append(records, entityRecord("payment_intent", pi))
		if charge, err := i.latestChargeForPaymentIntent(pi); err == nil && charge != nil {
			records = append(records, i.fraudReviewForCharge(charge)...)
		}
	} else if strings.HasPrefix(id, "ch_") {
		charge, err := i.get("/v1/charges/"+url.PathEscape(id), url.Values{})
		if err != nil {
			return nil, err
		}
		records = append(records, i.fraudReviewForCharge(charge)...)
	}
	if len(records) == 0 {
		records = append(records, evidenceRecord{Type: "finding", Severity: "warning", Summary: "No fraud-review evidence found for " + id + "."})
	}
	records = append(records, evidenceRecord{Type: "finding", Severity: "warning", Summary: "Fraud review evidence gathered. Check charge outcome risk fields, early fraud warnings, disputes, and refunds before deciding customer action."})
	return records, nil
}

func (i investigator) fraudReviewForCharge(charge map[string]any) []evidenceRecord {
	records := []evidenceRecord{entityRecord("charge", charge)}
	if warnings, err := i.list("/v1/radar/early_fraud_warnings", valuesWithLimit(10, "charge", mapString(charge, "id"))); err == nil {
		records = appendListRecords(records, "early_fraud_warning", warnings)
	}
	records = append(records, i.relatedDisputesAndRefunds(nil, charge)...)
	return records
}
