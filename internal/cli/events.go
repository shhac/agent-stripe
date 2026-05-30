package cli

import (
	"net/url"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func registerEvents(root *cobra.Command, globals shared.GlobalsFunc) {
	var limit int
	var eventType string
	var eventTypes []string
	var createdGTE string
	var createdLTE string
	var startingAfter string
	var endingBefore string
	var deliverySuccess string

	events := &cobra.Command{
		Use:   "events",
		Short: "Stripe events and webhook-relevant activity",
	}
	events.AddCommand(&cobra.Command{
		Use:   "get <event-id>",
		Short: "Retrieve an event by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return shared.GetRawItem(globals(), "/v1/events/"+url.PathEscape(args[0]), url.Values{})
		},
	})

	list := &cobra.Command{
		Use:   "list",
		Short: "List recent events; defaults to NDJSON",
		RunE: func(cmd *cobra.Command, args []string) error {
			params := url.Values{}
			shared.AddLimit(params, limit)
			shared.AddCreatedRange(params, createdGTE, createdLTE)
			shared.AddString(params, "type", eventType)
			shared.AddMulti(params, "types[]", eventTypes)
			shared.AddString(params, "starting_after", startingAfter)
			shared.AddString(params, "ending_before", endingBefore)
			shared.AddString(params, "delivery_success", deliverySuccess)
			return shared.GetRawList(globals(), "/v1/events", params)
		},
	}
	list.Flags().IntVar(&limit, "limit", 10, "Maximum events to return (1-100)")
	list.Flags().StringVar(&eventType, "type", "", "Single event type, for example charge.failed")
	list.Flags().StringArrayVar(&eventTypes, "types", nil, "Event type filter; repeatable, up to Stripe's limit")
	list.Flags().StringVar(&createdGTE, "created-gte", "", "Created at or after Unix timestamp")
	list.Flags().StringVar(&createdLTE, "created-lte", "", "Created at or before Unix timestamp")
	list.Flags().StringVar(&startingAfter, "starting-after", "", "Stripe cursor")
	list.Flags().StringVar(&endingBefore, "ending-before", "", "Stripe cursor")
	list.Flags().StringVar(&deliverySuccess, "delivery-success", "", "Filter true/false for webhook delivery success")
	events.AddCommand(list)
	root.AddCommand(events)
}
