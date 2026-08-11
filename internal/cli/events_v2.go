package cli

import (
	"context"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/api"
	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func registerEventsV2(root *cobra.Command, globals shared.GlobalsFunc) {
	events := &cobra.Command{
		Use:     "events-v2",
		Aliases: []string{"events_v2", "ev2"},
		Short:   "Stripe v2 core events (thin events for Accounts v2 and other /v2 resources)",
	}
	events.AddCommand(newEventsV2GetCommand(globals))
	events.AddCommand(newEventsV2ListCommand(globals))
	events.AddCommand(newUsageCommand(eventsV2UsageText))
	root.AddCommand(events)
}

func newEventsV2GetCommand(globals shared.GlobalsFunc) *cobra.Command {
	return &cobra.Command{
		Use:   "get <event-id>...",
		Short: "Retrieve v2 core events by ID",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := globals()
			return shared.GetEntities(flags, args, func(ctx context.Context, client *api.Client, id string) (any, error) {
				if err := validateExpectedStripeID(id, "event"); err != nil {
					return nil, err
				}
				return shared.FetchItem(ctx, client, flags, v2EventPath(id), url.Values{})
			})
		},
	}
}

func newEventsV2ListCommand(globals shared.GlobalsFunc) *cobra.Command {
	var limit int
	var page string
	var objectID string
	var types []string
	var createdGTE string
	var createdLTE string
	var full bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List v2 core events from the last 30 days",
		RunE: func(cmd *cobra.Command, args []string) error {
			params := v2EventListParams(limit, page, objectID, types, createdGTE, createdLTE)
			if full {
				return shared.GetRawV2List(globals(), "/v2/core/events", params)
			}
			return getSummarizedV2List(globals(), "/v2/core/events", params, v2EventListSummary)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 10, "Maximum results to return")
	cmd.Flags().StringVar(&page, "page", "", "Page token from a previous @pagination next_page")
	cmd.Flags().StringVar(&objectID, "object-id", "", "Only events related to this object, for example acct_...")
	cmd.Flags().StringArrayVar(&types, "type", nil, "Event type filter; repeatable, up to 20")
	cmd.Flags().StringVar(&createdGTE, "created-gte", "", "Created at or after an RFC3339 timestamp")
	cmd.Flags().StringVar(&createdLTE, "created-lte", "", "Created at or before an RFC3339 timestamp")
	cmd.Flags().BoolVar(&full, "full", false, "Return full Stripe objects instead of compact summaries")
	return cmd
}

func v2EventListParams(limit int, page, objectID string, types []string, createdGTE, createdLTE string) url.Values {
	params := url.Values{}
	shared.AddLimit(params, limit)
	shared.AddString(params, "page", page)
	shared.AddString(params, "object_id", objectID)
	shared.AddIndexed(params, "types", types)
	shared.AddString(params, "created[gte]", createdGTE)
	shared.AddString(params, "created[lte]", createdLTE)
	return params
}

func v2EventPath(id string) string {
	return "/v2/core/events/" + url.PathEscape(id)
}

const eventsV2UsageText = `agent-stripe events-v2 — Stripe v2 core events

WHEN TO USE THIS INSTEAD OF 'events'
  'events' reads /v1/events, which carries a full snapshot of the changed object.
  'events-v2' reads /v2/core/events, which carries thin events: no snapshot, just
  related_object {id, type, url}. Accounts v2 capability and requirement changes are
  emitted ONLY here, so /v1/events will never show them.

COMMANDS
  agent-stripe events-v2 list --object-id acct_...
  agent-stripe events-v2 list --type "v2.core.account[requirements].updated"
  agent-stripe events-v2 get evt_test_...

ACCOUNT EVENT TYPES
  v2.core.account.created|updated|closed
  v2.core.account[configuration.merchant|configuration.customer|configuration.recipient].updated
  v2.core.account[configuration.merchant|...].capability_status_updated
  v2.core.account[requirements].updated
  v2.core.account[future_requirements].updated
  v2.core.account[identity].updated
  v2.core.account[defaults].updated

NOTES
  Events are retained for 30 days. Timestamps are RFC3339, not Unix seconds.
  For an account-shaped narrative rather than raw events, use
  'agent-stripe investigate account-events acct_...'.
`
