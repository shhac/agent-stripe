package cli

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func newInvestigateTimeline(globals shared.GlobalsFunc, outputOpts *investigationOutputOptions) *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:   "timeline <customer-id>",
		Short: "Build a chronological customer activity timeline from recent Stripe objects",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithInvestigator(globals(), outputOpts, func(inv investigator) ([]evidenceRecord, error) {
				return inv.timeline(args[0], limit)
			})
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 5, "Maximum recent objects per collection")
	return cmd
}

func (i investigator) timeline(customerID string, limit int) ([]evidenceRecord, error) {
	if err := validateExpectedStripeID(customerID, "customer"); err != nil {
		return nil, err
	}
	records, err := i.customerContext(customerID, limit)
	if err != nil {
		return nil, err
	}
	events := timelineEvents(records)
	sort.SliceStable(events, func(a, b int) bool { return events[a].created < events[b].created })
	for _, event := range events {
		records = i.appendEvidence(records, event.record())
	}
	records = i.appendEvidence(records, evidenceRecord{Type: "finding", Severity: "info", Summary: fmt.Sprintf("Timeline gathered %d timestamped customer objects for %s.", len(events), customerID)})
	return records, nil
}

type timelineEvent struct {
	created int64
	object  string
	id      string
	status  string
}

func timelineEvents(records []evidenceRecord) []timelineEvent {
	events := []timelineEvent{}
	for _, record := range records {
		if record.Type != "entity" || record.Data == nil {
			continue
		}
		created, ok := mapInt64(record.Data, "created")
		if !ok || created == 0 {
			continue
		}
		events = append(events, timelineEvent{
			created: created,
			object:  record.Object,
			id:      record.ID,
			status:  firstNonEmpty(mapString(record.Data, "status"), mapString(record.Data, "payment_status")),
		})
	}
	return events
}

func (e timelineEvent) record() evidenceRecord {
	return evidenceRecord{Type: "finding", Severity: "info", Summary: fmt.Sprintf("%d: %s %s status=%s.", e.created, e.object, e.id, e.status), Data: map[string]any{
		"created": e.created,
		"object":  e.object,
		"id":      e.id,
		"status":  e.status,
	}}
}
