package cli

import (
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/api"
	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func newInvestigateWebhookEvent(globals shared.GlobalsFunc, outputOpts *evidenceOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "webhook-event <event-id>",
		Short: "Explain a Stripe event and emit the underlying object",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithInvestigator(globals(), outputOpts, func(inv investigator) error {
				return inv.webhookEvent(args[0])
			})
		},
	}
}

func (i investigator) webhookEvent(eventID string) error {
	if err := validateExpectedStripeID(eventID, "event"); err != nil {
		return err
	}
	if isV2EventID(eventID) {
		return i.webhookEventV2(eventID)
	}
	event, err := i.get("/v1/events/"+url.PathEscape(eventID), url.Values{})
	if err != nil {
		if api.ErrorStatus(err) != 404 {
			return err
		}
		// v1 and v2 events share the evt_ prefix. A miss in v1 is worth one
		// look in the v2 stream before reporting the event as unknown.
		if v2Err := i.webhookEventV2(eventID); v2Err == nil {
			return nil
		}
		return err
	}
	i.add(entityRecord("event", event))
	data := mapAnyMap(event, "data")
	if underlying, ok := data["object"].(map[string]any); ok {
		i.add(evidenceRecord{
			Type:     "finding",
			Severity: eventSeverity(mapString(event, "type")),
			Summary:  "Event " + eventID + " is " + mapString(event, "type") + " for " + mapString(underlying, "object") + " " + mapString(underlying, "id") + ".",
		})
		return nil
	}
	i.add(evidenceRecord{Type: "finding", Severity: eventSeverity(mapString(event, "type")), Summary: "Event " + eventID + " is " + mapString(event, "type") + "."})
	return nil
}

// isV2EventID matches Stripe's sandbox v2 event IDs. Live-mode v2 IDs are not
// distinguishable from v1 by prefix, which is why webhookEvent also falls back
// to v2 on a v1 404.
func isV2EventID(eventID string) bool {
	return strings.HasPrefix(eventID, "evt_test_")
}

// webhookEventV2 handles thin events: there is no snapshot in the payload, so
// the related object is fetched and labelled as current state.
func (i investigator) webhookEventV2(eventID string) error {
	event, err := i.get(v2EventPath(eventID), url.Values{})
	if err != nil {
		return err
	}
	i.add(entityRecord(objectV2Event, event))
	eventType := mapString(event, "type")
	related := mapAnyMap(event, "related_object")
	relatedID := mapString(related, "id")
	relatedType := mapString(related, "type")

	summary := "Event " + eventID + " is " + eventType + "."
	if relatedID != "" {
		summary = "Event " + eventID + " is " + eventType + " for " + relatedType + " " + relatedID + "."
	}
	summary += " v2 events are thin: they carry no snapshot, so any object shown here is current state, not state at event time."

	if object, ok := i.v2EventRelatedObject(event); ok {
		i.add(entityRecord(mapString(object, "object"), object))
	}
	i.add(evidenceRecord{
		Type:     "finding",
		Severity: eventSeverity(eventType),
		Summary:  summary,
		Command:  v2EventFollowUpCommand(relatedType, relatedID),
		Data: map[string]any{
			"namespace":    namespaceV2,
			"event_type":   eventType,
			"related_id":   relatedID,
			"related_type": relatedType,
		},
	})
	return nil
}

// v2EventRelatedObject follows a thin event's related_object.url so the caller
// sees the object the event is about. The URL is namespace-qualified and
// already carries the include set Stripe considers relevant.
func (i investigator) v2EventRelatedObject(event map[string]any) (map[string]any, bool) {
	related := mapAnyMap(event, "related_object")
	target := mapString(related, "url")
	if target == "" {
		return nil, false
	}
	path, rawQuery, _ := strings.Cut(target, "?")
	params, err := url.ParseQuery(rawQuery)
	if err != nil {
		params = url.Values{}
	}
	object, err := i.get(path, params)
	if err != nil {
		return nil, false
	}
	return object, true
}

func v2EventFollowUpCommand(relatedType, relatedID string) string {
	if relatedType == objectV2Account && relatedID != "" {
		return "agent-stripe investigate account-health " + relatedID
	}
	return ""
}

func eventSeverity(eventType string) string {
	switch eventType {
	case "charge.failed", "payment_intent.payment_failed", "invoice.payment_failed", "payout.failed", "customer.subscription.deleted":
		return "warning"
	default:
		return "info"
	}
}
