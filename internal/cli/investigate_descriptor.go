package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func newInvestigateStatementDescriptor(globals shared.GlobalsFunc, outputOpts *evidenceOptions) *cobra.Command {
	var descriptor string
	var customer string
	var limit int
	cmd := &cobra.Command{
		Use:   "statement-descriptor",
		Short: "Identify which charge a bank statement line refers to",
		Long: "Answers \"what is this line on my statement?\". Stripe's charge search does not\n" +
			"index statement descriptors, so this scans recent charges and matches their\n" +
			"statement_descriptor and calculated_statement_descriptor. Narrow it with\n" +
			"--customer when you know who was charged; otherwise raise --limit deliberately,\n" +
			"since matching happens client-side over the page that was fetched.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := shared.RequireFlag("descriptor", descriptor, "Pass the text as it appears on the statement, for example --descriptor 'FUREVER'"); err != nil {
				return err
			}
			return runWithInvestigator(globals(), outputOpts, func(inv investigator) error {
				return inv.statementDescriptor(descriptor, customer, limit)
			})
		},
	}
	cmd.Flags().StringVar(&descriptor, "descriptor", "", "Statement text to match, case-insensitive (required)")
	cmd.Flags().StringVar(&customer, "customer", "", "Restrict the scan to one customer")
	cmd.Flags().IntVar(&limit, "limit", 50, "Maximum recent charges to scan")
	return cmd
}

func (i investigator) statementDescriptor(descriptor, customer string, limit int) error {
	if customer != "" {
		if err := validateExpectedStripeID(customer, "customer"); err != nil {
			return err
		}
	}
	// valuesWithLimit routes through shared.AddString, which skips empty values,
	// so the optional scope needs no branch.
	params := valuesWithLimit(limit, "customer", customer)
	charges, more, err := i.listPage("/v1/charges", params)
	if err != nil {
		return err
	}

	matches := []map[string]any{}
	for _, charge := range charges {
		if chargeMatchesDescriptor(charge, descriptor) {
			matches = append(matches, charge)
		}
	}
	if len(matches) == 0 {
		summary := fmt.Sprintf("No charge among the %d most recent matched descriptor %q. Statement text is often truncated or prefixed by the bank; try a shorter fragment, a --customer, or a larger --limit.",
			len(charges), descriptor)
		if more {
			summary += " Older charges exist beyond this page and were not scanned."
		}
		i.add(evidenceRecord{
			Type:     "finding",
			Severity: "warning",
			Summary:  summary,
			Data:     map[string]any{"descriptor": descriptor, "inspected": len(charges), "scan_truncated": more},
		})
		return nil
	}
	for _, charge := range matches {
		i.add(entityRecord("charge", charge), descriptorMatchFinding(descriptor, charge))
	}
	return nil
}

func chargeMatchesDescriptor(charge map[string]any, descriptor string) bool {
	want := strings.ToLower(strings.TrimSpace(descriptor))
	if want == "" {
		return false
	}
	for _, field := range []string{"statement_descriptor", "calculated_statement_descriptor", "statement_descriptor_suffix"} {
		value := strings.ToLower(mapString(charge, field))
		if value == "" {
			continue
		}
		// Banks truncate and prefix, so match either direction rather than
		// requiring the two strings to be equal.
		if strings.Contains(value, want) || strings.Contains(want, value) {
			return true
		}
	}
	return false
}

func descriptorMatchFinding(descriptor string, charge map[string]any) evidenceRecord {
	amount, _ := mapInt64(charge, "amount")
	shown := firstNonEmpty(
		mapString(charge, "calculated_statement_descriptor"),
		mapString(charge, "statement_descriptor"),
		mapString(charge, "statement_descriptor_suffix"),
	)
	return evidenceRecord{
		Type:     "finding",
		Severity: "info",
		Summary: fmt.Sprintf("Descriptor %q matches charge %s for %d %s to customer %s (statement text %q).",
			descriptor, mapString(charge, "id"), amount, mapString(charge, "currency"), mapString(charge, "customer"), shown),
		Command: "agent-stripe investigate incoming-payment " + mapString(charge, "id"),
		Data: map[string]any{
			"descriptor":      descriptor,
			"charge":          mapString(charge, "id"),
			"customer":        mapString(charge, "customer"),
			"amount":          amount,
			"currency":        mapString(charge, "currency"),
			"statement_text":  shown,
			"payment_intent":  idFromValue(charge["payment_intent"]),
			"card_last4":      cardLast4(charge),
			"charge_disputed": mapBool(charge, "disputed"),
		},
	}
}
