package cli

import (
	"context"
	"encoding/json"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/api"
	"github.com/shhac/agent-stripe/internal/cli/shared"
	agenterrors "github.com/shhac/agent-stripe/internal/errors"
)

func registerAccounts(root *cobra.Command, globals shared.GlobalsFunc) {
	var limit int
	var cursor cursorFlags
	var full bool

	accounts := &cobra.Command{
		Use:   "accounts",
		Short: "Stripe account and connected-account lookup",
	}
	accounts.AddCommand(&cobra.Command{
		Use:   "self",
		Short: "Retrieve the account for the active key/context",
		RunE: func(cmd *cobra.Command, args []string) error {
			return shared.GetRawItem(globals(), "/v1/account", url.Values{})
		},
	})
	accounts.AddCommand(&cobra.Command{
		Use:   "get <account-id>",
		Short: "Retrieve a connected account by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateExpectedStripeID(args[0], "account"); err != nil {
				return err
			}
			return shared.GetRawItem(globals(), "/v1/accounts/"+url.PathEscape(args[0]), url.Values{})
		},
	})
	list := &cobra.Command{
		Use:   "list",
		Short: "List connected accounts",
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := globals()
			params := url.Values{}
			shared.AddLimit(params, limit)
			cursor.AddTo(params)
			if full {
				return shared.GetRawList(flags, "/v1/accounts", params)
			}
			return listAccountSummaries(flags, params)
		},
	}
	list.Flags().IntVar(&limit, "limit", 10, "Maximum results to return (1-100)")
	cursor.AddFlags(list)
	list.Flags().BoolVar(&full, "full", false, "Return full Stripe account objects instead of compact summaries")
	accounts.AddCommand(list)
	root.AddCommand(accounts)
}

func listAccountSummaries(flags *shared.GlobalFlags, params url.Values) error {
	return shared.WithClient(flags, func(ctx context.Context, client *api.Client) error {
		raw, err := client.Get(ctx, "/v1/accounts", params)
		if err != nil {
			return err
		}
		list, err := api.DecodeList(raw)
		if err != nil {
			return err
		}
		items := make([]any, 0, len(list.Data))
		for _, rawItem := range list.Data {
			var account map[string]any
			if err := json.Unmarshal(rawItem, &account); err != nil {
				return agenterrors.Wrap(err, agenterrors.FixableByAgent)
			}
			items = append(items, accountListSummary(account))
		}
		shared.WritePaginatedList(items, listPagination(list), flags.Format)
		return nil
	})
}

func accountListSummary(account map[string]any) map[string]any {
	summary := map[string]any{}
	copyString(summary, account, "id")
	copyString(summary, account, "object")
	copyString(summary, account, "type")
	copyString(summary, account, "business_type")
	copyString(summary, account, "country")
	copyString(summary, account, "default_currency")
	copyNumber(summary, account, "created")
	copyBool(summary, account, "charges_enabled")
	copyBool(summary, account, "payouts_enabled")
	copyBool(summary, account, "details_submitted")
	addControllerSummary(summary, account)
	addRequirementSummary(summary, account, "requirements")
	addRequirementSummary(summary, account, "future_requirements")
	addCapabilitiesSummary(summary, account)
	return summary
}

func addControllerSummary(summary, account map[string]any) {
	controller, ok := account["controller"].(map[string]any)
	if !ok {
		return
	}
	out := map[string]any{}
	copyString(out, controller, "type")
	copyString(out, controller, "requirement_collection")
	if dashboard, ok := controller["stripe_dashboard"].(map[string]any); ok {
		copyString(out, dashboard, "type")
		if dashboardType, ok := out["type"]; ok {
			out["stripe_dashboard_type"] = dashboardType
			delete(out, "type")
		}
	}
	if len(out) > 0 {
		summary["controller"] = out
	}
}

func addRequirementSummary(summary, account map[string]any, key string) {
	requirements, ok := account[key].(map[string]any)
	if !ok {
		return
	}
	out := map[string]any{}
	copyString(out, requirements, "disabled_reason")
	copyNumber(out, requirements, "current_deadline")
	addArrayCount(out, requirements, "currently_due")
	addArrayCount(out, requirements, "past_due")
	addArrayCount(out, requirements, "pending_verification")
	addArrayCount(out, requirements, "eventually_due")
	if len(out) > 0 {
		summary[key] = out
	}
}

func addCapabilitiesSummary(summary, account map[string]any) {
	capabilities, ok := account["capabilities"].(map[string]any)
	if !ok || len(capabilities) == 0 {
		return
	}
	counts := map[string]int{}
	for _, value := range capabilities {
		if status, ok := value.(string); ok && status != "" {
			counts[status]++
		}
	}
	if len(counts) == 0 {
		return
	}
	out := map[string]any{}
	for _, status := range []string{"active", "pending", "inactive"} {
		if counts[status] > 0 {
			out[status+"_count"] = counts[status]
		}
	}
	if len(out) > 0 {
		summary["capabilities"] = out
	}
}

func copyString(out, in map[string]any, key string) {
	value, ok := in[key].(string)
	if ok && value != "" {
		out[key] = value
	}
}

func copyBool(out, in map[string]any, key string) {
	value, ok := in[key].(bool)
	if ok {
		out[key] = value
	}
}

func copyNumber(out, in map[string]any, key string) {
	if value, ok := in[key].(float64); ok {
		out[key] = value
	}
}

func addArrayCount(out, in map[string]any, key string) {
	items, ok := in[key].([]any)
	if ok && len(items) > 0 {
		out[key+"_count"] = len(items)
	}
}
