package cli

import (
	"fmt"
	"sort"
)

var timelineInvestigation = investigationSpec{
	use:          "timeline <customer-id>",
	short:        "Build a chronological customer activity timeline from recent Stripe objects",
	runWithLimit: investigator.timeline,
	limitDefault: 5,
	limitHelp:    "Maximum recent objects per collection",
}

func (i investigator) timeline(customerID string, limit int) error {
	if err := validateExpectedStripeID(customerID, "customer"); err != nil {
		return err
	}
	mark := i.count()
	if err := i.customerContext(customerID, limit); err != nil {
		return err
	}
	events := timelineEvents(i.since(mark))
	sort.SliceStable(events, func(a, b int) bool { return events[a].created < events[b].created })
	for _, event := range events {
		i.add(event.record())
	}
	i.add(evidenceRecord{Type: "finding", Severity: "info", Summary: fmt.Sprintf("Timeline gathered %d timestamped customer objects for %s.", len(events), customerID)})
	return nil
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
