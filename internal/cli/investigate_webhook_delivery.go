package cli

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func newInvestigateWebhookDelivery(globals shared.GlobalsFunc, outputOpts *evidenceOptions) *cobra.Command {
	var endpoint string
	cmd := &cobra.Command{
		Use:   "webhook-delivery <event-id|webhook-endpoint-id>",
		Short: "Explain webhook delivery health from event pending_webhooks and endpoint config",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithInvestigator(globals(), outputOpts, func(inv investigator) error {
				return inv.webhookDelivery(args[0], endpoint)
			})
		},
	}
	cmd.Flags().StringVar(&endpoint, "endpoint", "", "Webhook endpoint ID to inspect alongside an event")
	return cmd
}

func (i investigator) webhookDelivery(id, endpointID string) error {
	if err := validateAllowedStripeID(id, "event", "webhook_endpoint"); err != nil {
		return err
	}
	if strings.HasPrefix(id, "evt_") {
		event, err := i.get("/v1/events/"+url.PathEscape(id), url.Values{})
		if err != nil {
			return err
		}
		i.add(entityRecord("event", event), webhookEventDeliveryFinding(event))
		if endpointID != "" {
			endpoint, err := i.get("/v1/webhook_endpoints/"+url.PathEscape(endpointID), url.Values{})
			if err != nil {
				return err
			}
			i.add(entityRecord("webhook_endpoint", endpoint), webhookEndpointFinding(endpoint, mapString(event, "type")))
		} else if endpoints, err := i.list("/v1/webhook_endpoints", url.Values{"limit": []string{"100"}}); err == nil {
			for _, endpoint := range endpoints {
				if endpointHandlesEvent(endpoint, mapString(event, "type")) {
					i.add(entityRecord("webhook_endpoint", endpoint))
				}
			}
		}
		return nil
	}
	endpoint, err := i.get("/v1/webhook_endpoints/"+url.PathEscape(id), url.Values{})
	if err != nil {
		return err
	}
	i.add(entityRecord("webhook_endpoint", endpoint), webhookEndpointFinding(endpoint, ""))
	return nil
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
