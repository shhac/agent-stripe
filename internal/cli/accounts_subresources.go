package cli

import (
	"context"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/api"
	"github.com/shhac/agent-stripe/internal/cli/shared"
	agenterrors "github.com/shhac/agent-stripe/internal/errors"
)

// The /v1 account sub-resources. These answer the questions the Account object
// itself only hints at: requirements say "external_account" is missing — what
// external accounts does it have? A capability is inactive — what is that
// capability waiting on? Note that these matter for Accounts v2 too: the v2 API
// has no external-accounts hash, so external accounts are managed through the
// v1 endpoint whichever namespace the account belongs to.
func newAccountsPersonsCommand(globals shared.GlobalsFunc) *cobra.Command {
	persons := &cobra.Command{
		Use:   "persons",
		Short: "People on a Connect v1 account's identity (owners, representatives, directors)",
	}

	var limit int
	var relationship string
	var cursor cursorFlags
	list := &cobra.Command{
		Use:   "list <account-id>",
		Short: "List the persons on a Connect v1 account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateExpectedStripeID(args[0], "account"); err != nil {
				return err
			}
			params := url.Values{}
			shared.AddLimit(params, limit)
			cursor.AddTo(params)
			switch relationship {
			case "":
			case "representative", "owner", "director", "executive":
				params.Set("relationship["+relationship+"]", "true")
			default:
				return unknownRelationshipError(relationship)
			}
			return shared.GetRawList(globals(), accountSubPath(args[0], "persons"), params)
		},
	}
	list.Flags().IntVar(&limit, "limit", 10, "Maximum results to return (1-100)")
	list.Flags().StringVar(&relationship, "relationship", "", "Filter by role: representative, owner, director, or executive")
	cursor.AddFlags(list)
	persons.AddCommand(list)

	persons.AddCommand(&cobra.Command{
		Use:   "get <account-id> <person-id>...",
		Short: "Retrieve persons on a Connect v1 account by ID",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := globals()
			if err := validateExpectedStripeID(args[0], "account"); err != nil {
				return err
			}
			base := accountSubPath(args[0], "persons") + "/"
			return shared.GetEntities(flags, args[1:], func(ctx context.Context, client *api.Client, id string) (any, error) {
				return shared.FetchItem(ctx, client, flags, base+url.PathEscape(id), url.Values{})
			})
		},
	})
	return persons
}

func newAccountCapabilitiesCommand(globals shared.GlobalsFunc) *cobra.Command {
	return &cobra.Command{
		Use:   "capabilities <account-id>",
		Short: "List a Connect v1 account's capabilities with their own requirements",
		Long: "The Account object carries capability names and statuses only. The Capability\n" +
			"objects carry the requirements each one is waiting on, which is what explains\n" +
			"an inactive capability.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateExpectedStripeID(args[0], "account"); err != nil {
				return err
			}
			return shared.GetRawList(globals(), accountSubPath(args[0], "capabilities"), url.Values{})
		},
	}
}

func newAccountExternalAccountsCommand(globals shared.GlobalsFunc) *cobra.Command {
	var limit int
	var object string
	var cursor cursorFlags
	cmd := &cobra.Command{
		Use:     "external-accounts <account-id>",
		Aliases: []string{"external_accounts"},
		Short:   "List the bank accounts and cards an account is paid out to",
		Long: "Answers what a 'external_account' requirement is asking for. Accounts v2 has no\n" +
			"external-accounts hash, so this v1 endpoint is the one to use for accounts in\n" +
			"either namespace.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateExpectedStripeID(args[0], "account"); err != nil {
				return err
			}
			params := url.Values{}
			shared.AddLimit(params, limit)
			cursor.AddTo(params)
			shared.AddString(params, "object", object)
			return getSummarizedList(globals(), accountSubPath(args[0], "external_accounts"), params, externalAccountListSummary)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 10, "Maximum results to return (1-100)")
	cmd.Flags().StringVar(&object, "object", "", "Filter by type: bank_account or card")
	cursor.AddFlags(cmd)
	return cmd
}

func unknownRelationshipError(value string) error {
	return agenterrors.Newf(agenterrors.FixableByAgent, "unknown --relationship value %q", value).
		WithHint("Valid roles: representative, owner, director, executive")
}

func accountSubPath(accountID, sub string) string {
	return "/v1/accounts/" + url.PathEscape(accountID) + "/" + sub
}

// externalAccountListSummary keeps what decides whether a payout can land, and
// leaves the rest to --full. last4 stays visible for triage the way card last4
// does elsewhere; the account number never appears in a Stripe response.
func externalAccountListSummary(item map[string]any) map[string]any {
	summary := map[string]any{}
	copyString(summary, item, "id")
	copyString(summary, item, "object")
	copyString(summary, item, "account")
	copyString(summary, item, "status")
	copyString(summary, item, "currency")
	copyString(summary, item, "country")
	copyBool(summary, item, "default_for_currency")
	copyString(summary, item, "bank_name")
	copyString(summary, item, "routing_number")
	copyString(summary, item, "last4")
	copyString(summary, item, "brand")
	copyNumber(summary, item, "exp_month")
	copyNumber(summary, item, "exp_year")
	return summary
}
