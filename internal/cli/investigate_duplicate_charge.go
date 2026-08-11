package cli

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func newInvestigateDuplicateCharge(globals shared.GlobalsFunc, outputOpts *evidenceOptions) *cobra.Command {
	var customer string
	var last4 string
	var windowHours int
	var limit int
	cmd := &cobra.Command{
		Use:   "duplicate-charge",
		Short: "Find repeated charges to a customer for the same amount in a short window",
		Long: "Answers \"we think this customer was charged twice\". Groups the customer's recent\n" +
			"charges by amount, currency, and card last4, and reports groups with more than one\n" +
			"charge inside the window. Card last4 is not unique, so --customer is required.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := shared.RequireFlag("customer", customer, "Duplicate detection is scoped to one customer; last4 and amount are not unique on their own"); err != nil {
				return err
			}
			return runWithInvestigator(globals(), outputOpts, func(inv investigator) error {
				return inv.duplicateCharge(customer, last4, windowHours, limit)
			})
		},
	}
	cmd.Flags().StringVar(&customer, "customer", "", "Customer ID (required)")
	cmd.Flags().StringVar(&last4, "last4", "", "Only consider charges on a card ending in these digits")
	cmd.Flags().IntVar(&windowHours, "window-hours", 24, "How close together charges must be to count as duplicates")
	cmd.Flags().IntVar(&limit, "limit", 25, "Maximum recent charges to inspect")
	return cmd
}

func (i investigator) duplicateCharge(customer, last4 string, windowHours, limit int) error {
	if err := validateExpectedStripeID(customer, "customer"); err != nil {
		return err
	}
	charges, err := i.list("/v1/charges", valuesWithLimit(limit, "customer", customer))
	if err != nil {
		return err
	}

	groups := map[string][]map[string]any{}
	for _, charge := range charges {
		if last4 != "" && cardLast4(charge) != last4 {
			continue
		}
		if mapString(charge, "status") == "failed" {
			continue
		}
		groups[duplicateChargeKey(charge)] = append(groups[duplicateChargeKey(charge)], charge)
	}

	clusters := 0
	for _, group := range sortedGroupKeys(groups) {
		charges := groups[group]
		if len(charges) < 2 {
			continue
		}
		for _, cluster := range clusterByWindow(charges, int64(windowHours)*3600) {
			if len(cluster) < 2 {
				continue
			}
			clusters++
			for _, charge := range cluster {
				i.add(entityRecord("charge", charge))
			}
			i.add(duplicateChargeFinding(customer, cluster))
		}
	}
	if clusters == 0 {
		i.add(evidenceRecord{
			Type:     "finding",
			Severity: "info",
			Summary: fmt.Sprintf("No duplicate charges found for customer %s across %d recent charges within %dh.",
				customer, len(charges), windowHours),
			Data: map[string]any{"customer": customer, "inspected": len(charges), "window_hours": windowHours},
		})
	}
	return nil
}

// duplicateChargeKey is what "the same charge twice" means here: same money, on
// the same card. Two different cards for the same amount is a customer
// retrying, not a double charge.
func duplicateChargeKey(charge map[string]any) string {
	amount, _ := mapInt64(charge, "amount")
	return fmt.Sprintf("%d|%s|%s", amount, mapString(charge, "currency"), cardLast4(charge))
}

// clusterByWindow splits charges that share a key into runs that are actually
// close together, so two legitimate monthly charges are not called duplicates.
func clusterByWindow(charges []map[string]any, windowSeconds int64) [][]map[string]any {
	sorted := append([]map[string]any(nil), charges...)
	sort.SliceStable(sorted, func(a, b int) bool {
		left, _ := mapInt64(sorted[a], "created")
		right, _ := mapInt64(sorted[b], "created")
		return left < right
	})

	var clusters [][]map[string]any
	current := []map[string]any{}
	var previous int64
	for _, charge := range sorted {
		created, _ := mapInt64(charge, "created")
		if len(current) > 0 && created-previous > windowSeconds {
			clusters = append(clusters, current)
			current = nil
		}
		current = append(current, charge)
		previous = created
	}
	if len(current) > 0 {
		clusters = append(clusters, current)
	}
	return clusters
}

func duplicateChargeFinding(customer string, cluster []map[string]any) evidenceRecord {
	ids := make([]string, 0, len(cluster))
	for _, charge := range cluster {
		ids = append(ids, mapString(charge, "id"))
	}
	first, _ := mapInt64(cluster[0], "created")
	last, _ := mapInt64(cluster[len(cluster)-1], "created")
	amount, _ := mapInt64(cluster[0], "amount")

	refunded := 0
	for _, charge := range cluster {
		if mapBool(charge, "refunded") {
			refunded++
		}
	}
	summary := fmt.Sprintf("Customer %s has %d charges of %d %s on card ending %s within %d seconds: %s.",
		customer, len(cluster), amount, mapString(cluster[0], "currency"), cardLast4(cluster[0]), last-first, joinAndTruncate(ids, 6))
	if refunded > 0 {
		summary += fmt.Sprintf(" %d of them are already refunded.", refunded)
	}
	return evidenceRecord{
		Type:     "finding",
		Severity: "warning",
		Summary:  summary,
		Command:  "agent-stripe investigate refund " + ids[len(ids)-1],
		Data: map[string]any{
			"customer":       customer,
			"charges":        ids,
			"amount":         amount,
			"currency":       mapString(cluster[0], "currency"),
			"last4":          cardLast4(cluster[0]),
			"seconds_apart":  last - first,
			"refunded_count": refunded,
		},
	}
}

func sortedGroupKeys(groups map[string][]map[string]any) []string {
	keys := make([]string, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
