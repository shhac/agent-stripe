package cli

import (
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func newInvestigateWebhookDelivery(globals shared.GlobalsFunc, outputOpts *investigationOutputOptions) *cobra.Command {
	var endpoint string
	cmd := &cobra.Command{
		Use:   "webhook-delivery <event-id|webhook-endpoint-id>",
		Short: "Explain webhook delivery health from event pending_webhooks and endpoint config",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithInvestigator(globals(), outputOpts, func(inv investigator) ([]evidenceRecord, error) {
				return inv.webhookDelivery(args[0], endpoint)
			})
		},
	}
	cmd.Flags().StringVar(&endpoint, "endpoint", "", "Webhook endpoint ID to inspect alongside an event")
	return cmd
}

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

func (i investigator) webhookDelivery(id, endpointID string) ([]evidenceRecord, error) {
	if err := validateAllowedStripeID(id, "event", "webhook_endpoint"); err != nil {
		return nil, err
	}
	records := []evidenceRecord{}
	if strings.HasPrefix(id, "evt_") {
		event, err := i.get("/v1/events/"+url.PathEscape(id), url.Values{})
		if err != nil {
			return nil, err
		}
		records = append(records, entityRecord("event", event), webhookEventDeliveryFinding(event))
		if endpointID != "" {
			endpoint, err := i.get("/v1/webhook_endpoints/"+url.PathEscape(endpointID), url.Values{})
			if err != nil {
				return nil, err
			}
			records = append(records, entityRecord("webhook_endpoint", endpoint), webhookEndpointFinding(endpoint, mapString(event, "type")))
		} else if endpoints, err := i.list("/v1/webhook_endpoints", url.Values{"limit": []string{"100"}}); err == nil {
			for _, endpoint := range endpoints {
				if endpointHandlesEvent(endpoint, mapString(event, "type")) {
					records = append(records, entityRecord("webhook_endpoint", endpoint))
				}
			}
		}
		return records, nil
	}
	endpoint, err := i.get("/v1/webhook_endpoints/"+url.PathEscape(id), url.Values{})
	if err != nil {
		return nil, err
	}
	records = append(records, entityRecord("webhook_endpoint", endpoint), webhookEndpointFinding(endpoint, ""))
	return records, nil
}

func webhookEventDeliveryFinding(event map[string]any) evidenceRecord {
	pending, _ := mapInt64(event, "pending_webhooks")
	severity := "info"
	if pending > 0 {
		severity = "warning"
	}
	summary := fmt.Sprintf("Event %s type=%s has pending_webhooks=%d.", mapString(event, "id"), mapString(event, "type"), pending)
	if pending > 0 {
		summary += " Some configured endpoints may not have successfully acknowledged it yet."
	}
	return evidenceRecord{Type: "finding", Severity: severity, Summary: summary, Data: map[string]any{
		"event":            mapString(event, "id"),
		"type":             mapString(event, "type"),
		"pending_webhooks": pending,
		"request":          mapAnyMap(event, "request"),
	}}
}

func webhookEndpointFinding(endpoint map[string]any, eventType string) evidenceRecord {
	severity := "info"
	status := mapString(endpoint, "status")
	if status != "" && status != "enabled" {
		severity = "warning"
	}
	handles := true
	if eventType != "" {
		handles = endpointHandlesEvent(endpoint, eventType)
		if !handles {
			severity = "warning"
		}
	}
	summary := fmt.Sprintf("Webhook endpoint %s status=%s.", mapString(endpoint, "id"), status)
	if eventType != "" {
		summary += fmt.Sprintf(" Handles %s=%t.", eventType, handles)
	}
	return evidenceRecord{Type: "finding", Severity: severity, Summary: summary, Data: map[string]any{
		"webhook_endpoint": mapString(endpoint, "id"),
		"status":           status,
		"event_type":       eventType,
		"handles_event":    handles,
	}}
}

func endpointHandlesEvent(endpoint map[string]any, eventType string) bool {
	enabled, _ := endpoint["enabled_events"].([]any)
	for _, value := range enabled {
		if got, _ := value.(string); got == "*" || got == eventType {
			return true
		}
	}
	return false
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
		records = append(records, entityRecord("payment_method", pm))
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
		records = append(records, entityRecord("customer", customer))
		found, err := i.list("/v1/setup_intents", valuesWithLimit(limit, "customer", id))
		if err != nil {
			return nil, err
		}
		setupIntents = found
	}
	for _, seti := range setupIntents {
		records = append(records, entityRecord("setup_intent", seti))
		if pmID := idFromValue(seti["payment_method"]); pmID != "" {
			if pm, err := i.get("/v1/payment_methods/"+url.PathEscape(pmID), url.Values{}); err == nil {
				records = append(records, entityRecord("payment_method", pm))
			}
		}
		records = append(records, setupFinding(seti))
	}
	if len(setupIntents) == 0 {
		records = append(records, evidenceRecord{Type: "finding", Severity: "warning", Summary: "No SetupIntents found for " + id + "."})
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
		records = append(records, event.record())
	}
	records = append(records, evidenceRecord{Type: "finding", Severity: "info", Summary: fmt.Sprintf("Timeline gathered %d timestamped customer objects for %s.", len(events), customerID)})
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
