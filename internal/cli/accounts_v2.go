package cli

import (
	"context"
	"net/url"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/api"
	"github.com/shhac/agent-stripe/internal/cli/shared"
	agenterrors "github.com/shhac/agent-stripe/internal/errors"
)

func registerAccountsV2(root *cobra.Command, globals shared.GlobalsFunc) {
	accounts := &cobra.Command{
		Use:     "accounts-v2",
		Aliases: []string{"accounts_v2", "av2"},
		Short:   "Accounts v2 (UA2) lookup: configurations, capabilities, requirements, persons",
	}
	accounts.AddCommand(newAccountsV2GetCommand(globals))
	accounts.AddCommand(newAccountsV2ListCommand(globals))
	accounts.AddCommand(newAccountsV2PersonsCommand(globals))
	accounts.AddCommand(newUsageCommand(accountsV2UsageText))
	root.AddCommand(accounts)
}

func newAccountsV2GetCommand(globals shared.GlobalsFunc) *cobra.Command {
	var include []string
	cmd := &cobra.Command{
		Use:   "get <account-id>...",
		Short: "Retrieve v2 accounts with configurations, identity, and requirements",
		Long: "Retrieve v2 accounts. Stripe returns null for configuration, identity, requirements,\n" +
			"future_requirements, and defaults unless they are requested, so every include is\n" +
			"requested by default; --include narrows that.",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := globals()
			params, err := v2AccountIncludeParams(include)
			if err != nil {
				return err
			}
			return shared.GetEntities(flags, args, func(ctx context.Context, client *api.Client, id string) (any, error) {
				if err := validateExpectedStripeID(id, "account"); err != nil {
					return nil, err
				}
				return shared.FetchItem(ctx, client, flags, v2AccountPath(id), params)
			})
		},
	}
	cmd.Flags().StringArrayVar(&include, "include", nil, "Field to include; repeatable. Defaults to every include. One of: "+strings.Join(v2AccountIncludes, ", "))
	return cmd
}

func newAccountsV2ListCommand(globals shared.GlobalsFunc) *cobra.Command {
	var limit int
	var page string
	var configurations []string
	var closed bool
	var full bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List v2 accounts, newest page first",
		Long: "List v2 accounts. Stripe's list endpoint does not support include, so configuration,\n" +
			"identity, and requirements are always null here — use 'accounts-v2 get <id>' for those.",
		RunE: func(cmd *cobra.Command, args []string) error {
			params := url.Values{}
			shared.AddLimit(params, limit)
			shared.AddString(params, "page", page)
			if err := validateV2Configurations(configurations); err != nil {
				return err
			}
			shared.AddIndexed(params, "applied_configurations", configurations)
			if closed {
				params.Set("closed", "true")
			}
			if full {
				return shared.GetRawList(globals(), "/v2/core/accounts", params)
			}
			return getSummarizedList(globals(), "/v2/core/accounts", params, v2AccountListSummary)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 10, "Maximum results to return")
	cmd.Flags().StringVar(&page, "page", "", "Page token from a previous @pagination next_page")
	cmd.Flags().StringArrayVar(&configurations, "applied-configuration", nil, "Only accounts with this configuration; repeatable and AND-ed: customer, merchant, recipient")
	cmd.Flags().BoolVar(&closed, "closed", false, "Return closed accounts instead of open ones")
	cmd.Flags().BoolVar(&full, "full", false, "Return full Stripe objects instead of compact summaries")
	return cmd
}

func newAccountsV2PersonsCommand(globals shared.GlobalsFunc) *cobra.Command {
	persons := &cobra.Command{
		Use:   "persons",
		Short: "People attached to a v2 account's identity (owners, representatives, directors)",
	}
	persons.AddCommand(newAccountsV2PersonsListCommand(globals))
	persons.AddCommand(newAccountsV2PersonsGetCommand(globals))
	return persons
}

func newAccountsV2PersonsListCommand(globals shared.GlobalsFunc) *cobra.Command {
	var limit int
	var page string
	var full bool
	cmd := &cobra.Command{
		Use:   "list <account-id>",
		Short: "List the persons on a v2 account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateExpectedStripeID(args[0], "account"); err != nil {
				return err
			}
			params := url.Values{}
			shared.AddLimit(params, limit)
			shared.AddString(params, "page", page)
			path := v2AccountPersonsPath(args[0])
			if full {
				return shared.GetRawList(globals(), path, params)
			}
			return getSummarizedList(globals(), path, params, v2PersonListSummary)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 10, "Maximum results to return")
	cmd.Flags().StringVar(&page, "page", "", "Page token from a previous @pagination next_page")
	cmd.Flags().BoolVar(&full, "full", false, "Return full Stripe objects instead of compact summaries")
	return cmd
}

func newAccountsV2PersonsGetCommand(globals shared.GlobalsFunc) *cobra.Command {
	return &cobra.Command{
		Use:   "get <account-id> <person-id>...",
		Short: "Retrieve persons on a v2 account by ID",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := globals()
			if err := validateExpectedStripeID(args[0], "account"); err != nil {
				return err
			}
			basePath := v2AccountPersonsPath(args[0]) + "/"
			return shared.GetEntities(flags, args[1:], func(ctx context.Context, client *api.Client, id string) (any, error) {
				if err := validateExpectedStripeID(id, "account_person"); err != nil {
					return nil, err
				}
				return shared.FetchItem(ctx, client, flags, basePath+url.PathEscape(id), url.Values{})
			})
		},
	}
}

func v2AccountPath(id string) string {
	return "/v2/core/accounts/" + url.PathEscape(id)
}

func v2AccountPersonsPath(accountID string) string {
	return v2AccountPath(accountID) + "/persons"
}

// v2AccountIncludeParams defaults to every include because an omitted field is
// returned as null, which reads like "not set" to an agent that did not ask.
func v2AccountIncludeParams(include []string) (url.Values, error) {
	params := url.Values{}
	if len(include) == 0 {
		shared.AddIndexed(params, "include", v2AccountIncludes)
		return params, nil
	}
	for _, value := range include {
		if !slices.Contains(v2AccountIncludes, value) {
			return nil, agenterrors.Newf(agenterrors.FixableByAgent, "unknown --include value %q", value).
				WithHint("Valid includes: " + strings.Join(v2AccountIncludes, ", "))
		}
	}
	shared.AddIndexed(params, "include", include)
	return params, nil
}

func validateV2Configurations(values []string) error {
	for _, value := range values {
		if !slices.Contains(v2Configurations, value) {
			return agenterrors.Newf(agenterrors.FixableByAgent, "unknown --applied-configuration value %q", value).
				WithHint("Valid configurations: customer, merchant, recipient")
		}
	}
	return nil
}

const accountsV2UsageText = `agent-stripe accounts-v2 — Accounts v2 (UA2) lookup

WHEN TO USE THIS INSTEAD OF 'accounts'
  'accounts' reads Connect v1 (/v1/accounts): charges_enabled, payouts_enabled, requirements arrays.
  'accounts-v2' reads Accounts v2 (/v2/core/accounts): configurations, nested capabilities, requirement entries.
  Both use the acct_ prefix, so the ID cannot tell you which one applies. If you do not know, run
  'agent-stripe investigate account-health acct_...' — it probes v2, falls back to v1, and says which answered.

COMMANDS
  agent-stripe accounts-v2 get acct_... [--include requirements]
  agent-stripe accounts-v2 list [--applied-configuration merchant] [--closed] [--page <token>]
  agent-stripe accounts-v2 persons list acct_...
  agent-stripe accounts-v2 persons get acct_... person_...

INCLUDES
  Stripe returns null for configuration, identity, requirements, future_requirements, and defaults
  unless requested. 'get' requests all of them by default; null after that means genuinely unset.
  Valid: configuration.customer, configuration.merchant, configuration.recipient, defaults,
  future_requirements, identity, requirements.

PAGINATION
  v2 lists have no cursor IDs. Read @pagination.next_page and pass it back as --page <token>.

API VERSION
  /v2 requests use their own Stripe-Version (--v2-api-version, profile v2_api_version). Pin a preview
  train when a field or endpoint is preview-only.
`
