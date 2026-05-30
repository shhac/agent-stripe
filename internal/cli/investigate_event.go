package cli

import (
	"net/url"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func newInvestigateWebhookEvent(globals shared.GlobalsFunc, outputOpts *investigationOutputOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "webhook-event <event-id>",
		Short: "Explain a Stripe event and emit the underlying object",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithInvestigator(globals(), outputOpts, func(inv investigator) ([]evidenceRecord, error) {
				return inv.webhookEvent(args[0])
			})
		},
	}
}

func (i investigator) webhookEvent(eventID string) ([]evidenceRecord, error) {
	event, err := i.get("/v1/events/"+url.PathEscape(eventID), url.Values{})
	if err != nil {
		return nil, err
	}
	records := []evidenceRecord{entityRecord("event", event)}
	data := mapAnyMap(event, "data")
	if underlying, ok := data["object"].(map[string]any); ok {
		records = append(records, evidenceRecord{
			Type:     "finding",
			Severity: eventSeverity(mapString(event, "type")),
			Summary:  "Event " + eventID + " is " + mapString(event, "type") + " for " + mapString(underlying, "object") + " " + mapString(underlying, "id") + ".",
		})
		return records, nil
	}
	records = append(records, evidenceRecord{Type: "finding", Severity: eventSeverity(mapString(event, "type")), Summary: "Event " + eventID + " is " + mapString(event, "type") + "."})
	return records, nil
}

func eventSeverity(eventType string) string {
	switch eventType {
	case "charge.failed", "payment_intent.payment_failed", "invoice.payment_failed", "payout.failed", "customer.subscription.deleted":
		return "warning"
	default:
		return "info"
	}
}
