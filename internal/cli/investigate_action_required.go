package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func newInvestigateActionRequired(globals shared.GlobalsFunc, outputOpts *evidenceOptions) *cobra.Command {
	var customer string
	var limit int
	cmd := &cobra.Command{
		Use:   "action-required",
		Short: "Find payments stalled waiting on the customer (SCA, 3DS, bank authorization)",
		Long: "Answers \"the payment isn't failing, it's just not finishing\". Finds PaymentIntents\n" +
			"in requires_action or requires_confirmation and the invoices behind them, and\n" +
			"reports where the customer can complete it.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithInvestigator(globals(), outputOpts, func(inv investigator) error {
				return inv.actionRequired(customer, limit)
			})
		},
	}
	cmd.Flags().StringVar(&customer, "customer", "", "Restrict to one customer")
	cmd.Flags().IntVar(&limit, "limit", 25, "Maximum PaymentIntents to inspect")
	return cmd
}

func (i investigator) actionRequired(customer string, limit int) error {
	if customer != "" {
		if err := validateExpectedStripeID(customer, "customer"); err != nil {
			return err
		}
	}
	params := valuesWithLimit(limit)
	if customer != "" {
		params = valuesWithLimit(limit, "customer", customer)
	}
	intents, more, err := i.listPage("/v1/payment_intents", params)
	if err != nil {
		return err
	}

	stalled := 0
	for _, pi := range intents {
		status := mapString(pi, "status")
		if status != "requires_action" && status != "requires_confirmation" && status != "requires_source_action" {
			continue
		}
		stalled++
		i.add(entityRecord("payment_intent", pi))
		invoice := i.followRef(pi, "invoice")
		i.add(actionRequiredFinding(pi, invoice))
	}
	if stalled == 0 {
		summary := fmt.Sprintf("No payments are waiting on customer action across %d inspected PaymentIntents.", len(intents))
		if more {
			summary += " Older PaymentIntents were not inspected; raise --limit to widen the scan."
		}
		i.add(evidenceRecord{
			Type:     "finding",
			Severity: "info",
			Summary:  summary,
			Data:     map[string]any{"inspected": len(intents), "customer": customer, "scan_truncated": more},
		})
	}
	return nil
}

func actionRequiredFinding(pi, invoice map[string]any) evidenceRecord {
	amount, _ := mapInt64(pi, "amount")
	data := map[string]any{
		"payment_intent": mapString(pi, "id"),
		"status":         mapString(pi, "status"),
		"customer":       mapString(pi, "customer"),
		"amount":         amount,
		"currency":       mapString(pi, "currency"),
	}
	if nextAction := mapAnyMap(pi, "next_action"); len(nextAction) > 0 {
		data["next_action_type"] = mapString(nextAction, "type")
	}

	summary := fmt.Sprintf("PaymentIntent %s for %d %s is %s and needs the customer to act",
		mapString(pi, "id"), amount, mapString(pi, "currency"), mapString(pi, "status"))
	if kind, ok := data["next_action_type"].(string); ok && kind != "" {
		summary += " (" + kind + ")"
	}
	summary += "."

	// The completion URLs are redacted by policy, so say they exist and how to
	// reveal them rather than leaking them into a summary.
	if invoice != nil {
		data["invoice"] = mapString(invoice, "id")
		if mapString(invoice, "hosted_invoice_url") != "" {
			data["has_hosted_invoice_url"] = true
			summary += fmt.Sprintf(" Invoice %s has a hosted payment page; reveal it with --expose hosted_invoice_url.", mapString(invoice, "id"))
		}
	}
	return evidenceRecord{
		Type:     "finding",
		Severity: "warning",
		Summary:  summary,
		Command:  "agent-stripe investigate incoming-payment " + mapString(pi, "id"),
		Data:     data,
	}
}
