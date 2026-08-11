package cli

import (
	"context"
	"net/url"

	"github.com/spf13/cobra"

	libcli "github.com/shhac/lib-agent-cli/cli"

	"github.com/shhac/agent-stripe/internal/api"
	"github.com/shhac/agent-stripe/internal/cli/shared"
	"github.com/shhac/agent-stripe/internal/output"
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
			flags := globals()
			return shared.WithClient(flags, func(ctx context.Context, client *api.Client) error {
				obj, err := shared.FetchItem(ctx, client, flags, "/v1/account", url.Values{})
				if err != nil {
					return err
				}
				return libcli.EmitItem(output.Stdout(), flags.Format, obj)
			})
		},
	})
	accounts.AddCommand(&cobra.Command{
		Use:   "get <account-id>...",
		Short: "Retrieve connected accounts by ID",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := globals()
			return shared.GetEntities(flags, args, func(ctx context.Context, client *api.Client, id string) (any, error) {
				if err := validateExpectedStripeID(id, "account"); err != nil {
					return nil, err
				}
				return shared.FetchItem(ctx, client, flags, "/v1/accounts/"+url.PathEscape(id), url.Values{})
			})
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
			return getSummarizedList(flags, "/v1/accounts", params, accountListSummary)
		},
	}
	list.Flags().IntVar(&limit, "limit", 10, "Maximum results to return (1-100)")
	cursor.AddFlags(list)
	list.Flags().BoolVar(&full, "full", false, "Return full Stripe account objects instead of compact summaries")
	accounts.AddCommand(list)
	accounts.AddCommand(newAccountsPersonsCommand(globals))
	accounts.AddCommand(newAccountCapabilitiesCommand(globals))
	accounts.AddCommand(newAccountExternalAccountsCommand(globals))
	root.AddCommand(accounts)
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
		copyStringAs(out, dashboard, "type", "stripe_dashboard_type")
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
