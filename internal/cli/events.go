package cli

import (
	"context"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/api"
	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func registerEvents(root *cobra.Command, globals shared.GlobalsFunc) {
	var limit int
	var eventType string
	var eventTypes []string
	var createdGTE string
	var createdLTE string
	var cursor cursorFlags
	var deliverySuccess string
	var full bool

	events := &cobra.Command{
		Use:   "events",
		Short: "Stripe events and webhook-relevant activity",
	}
	events.AddCommand(&cobra.Command{
		Use:   "get <event-id>...",
		Short: "Retrieve events by ID",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := globals()
			return shared.GetEntities(flags, args, func(ctx context.Context, client *api.Client, id string) (any, error) {
				if err := validateExpectedStripeID(id, "event"); err != nil {
					return nil, err
				}
				return shared.FetchItem(ctx, client, flags, "/v1/events/"+url.PathEscape(id), url.Values{})
			})
		},
	})

	list := &cobra.Command{
		Use:   "list",
		Short: "List recent events; defaults to NDJSON",
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := globals()
			params := url.Values{}
			shared.AddLimit(params, limit)
			shared.AddCreatedRange(params, createdGTE, createdLTE)
			shared.AddString(params, "type", eventType)
			shared.AddMulti(params, "types[]", eventTypes)
			cursor.AddTo(params)
			shared.AddString(params, "delivery_success", deliverySuccess)
			if full {
				return shared.GetRawList(flags, "/v1/events", params)
			}
			return getSummarizedList(flags, "/v1/events", params, eventListSummary)
		},
	}
	list.Flags().IntVar(&limit, "limit", 10, "Maximum events to return (1-100)")
	list.Flags().StringVar(&eventType, "type", "", "Single event type, for example charge.failed")
	list.Flags().StringArrayVar(&eventTypes, "types", nil, "Event type filter; repeatable, up to Stripe's limit")
	list.Flags().StringVar(&createdGTE, "created-gte", "", "Created at or after Unix timestamp")
	list.Flags().StringVar(&createdLTE, "created-lte", "", "Created at or before Unix timestamp")
	cursor.AddFlags(list)
	list.Flags().StringVar(&deliverySuccess, "delivery-success", "", "Filter true/false for webhook delivery success")
	list.Flags().BoolVar(&full, "full", false, "Return full Stripe event objects instead of compact summaries")
	events.AddCommand(list)
	root.AddCommand(events)
}

func eventListSummary(event map[string]any) map[string]any {
	summary := map[string]any{}
	copyString(summary, event, "id")
	copyString(summary, event, "object")
	copyString(summary, event, "type")
	copyNumber(summary, event, "created")
	copyBool(summary, event, "livemode")
	copyString(summary, event, "api_version")
	copyString(summary, event, "account")
	copySubset(summary, event, "request", "id", "idempotency_key")
	if data, ok := event["data"].(map[string]any); ok {
		if object, ok := data["object"].(map[string]any); ok {
			out := map[string]any{}
			copyString(out, object, "id")
			copyString(out, object, "object")
			copyString(out, object, "status")
			copyString(out, object, "customer")
			if len(out) > 0 {
				summary["data_object"] = out
			}
		}
	}
	return summary
}
