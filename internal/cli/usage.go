package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func registerUsageCommand(root *cobra.Command) {
	root.AddCommand(&cobra.Command{
		Use:   "usage",
		Short: "LLM-optimized reference card",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Print(usageText)
		},
	})
}

const usageText = `agent-stripe — Stripe triage CLI for AI agents

PROFILE SETUP
  agent-stripe auth add <profile> --form [--context <acct_or_path>] [--api-version <version>]
  agent-stripe auth add <profile> --api-key <rk_or_sk> [--context <acct_or_path>] [--api-version <version>]
  agent-stripe auth check [profile]
  agent-stripe auth list
  agent-stripe auth default <profile>
  agent-stripe auth remove <profile>

CONTEXT
  --context <acct_...> applies Stripe-Context for organization keys and related-account requests.
  For a platform connected account, use the connected account ID.
  For organization key access to a platform-connected account, use <platform_acct>/<connected_acct>.
  The API key is stored in macOS Keychain; this CLI never prints it back.
  Prefer --form when an LLM is guiding setup so the user types the key into an OS popup.

TRIAGE STARTERS
  agent-stripe balance get
  agent-stripe events list [--type charge.failed] [--created-gte <unix>] [--limit N]
  agent-stripe events get <evt_id>
  agent-stripe customers list [--email <email>]
  agent-stripe invoices get <in_id> [--expand payment_intent]
  agent-stripe invoices line-items <in_id>
  agent-stripe payment-intents get <pi_id> [--expand latest_charge] [--expand customer]
  agent-stripe payment-intents search --query "metadata['order_id']:'123'"
  agent-stripe charges get <ch_id> [--expand payment_intent] [--expand balance_transaction]
  agent-stripe charges search --query "payment_method_details.card.last4:'4242'"
  agent-stripe subscriptions get <sub_id> --expand latest_invoice --expand latest_invoice.payment_intent
  agent-stripe subscriptions list [--customer <cus_id>] [--status active|past_due|unpaid|all]
  agent-stripe subscriptions invoices <sub_id> [--status open]
  agent-stripe payment-methods list --customer <cus_id> --type card
  agent-stripe disputes list [--charge <ch_id>] [--payment-intent <pi_id>]
  agent-stripe refunds list [--payment-intent <pi_id>]
  agent-stripe transfers list [--destination <acct_id>]
  agent-stripe payouts get <po_id>
  agent-stripe balance-transactions get <txn_id>
  agent-stripe accounts list
  agent-stripe accounts get <acct_id>

INVESTIGATIONS
  agent-stripe investigate usage
  agent-stripe investigate customer-card-payment --customer <cus_id> --last4 4242
  agent-stripe investigate invoice-payment <in_id>
  agent-stripe investigate invoice-metadata <in_id>
  agent-stripe investigate subscription-renewal --subscription <sub_id>
  agent-stripe investigate subscription-renewal --metadata tenant_id=acme
  agent-stripe investigate collection-risk --days 30
  agent-stripe investigate incoming-payment <pi_id|ch_id|in_id>
  agent-stripe investigate outgoing-payment <tr_id|po_id|acct_id>
  agent-stripe investigate refund-recovery <re_id|trr_id|ch_id|pi_id> [--transfer <tr_id>]

RAW READ-ONLY API
  agent-stripe api get /v1/payment_intents/pi_... [--query expand[]=latest_charge]
  Only GET is exposed initially so agents can investigate without mutation risk.

OUTPUT
  Lists default to NDJSON/jsonl, one object per line, with @pagination when there is another page.
  Single-object reads default to pretty JSON.
  Errors are JSON on stderr with fixable_by: agent|human|retry and a hint where possible.

GLOBAL FLAGS
  -p, --profile <alias>
  --context <Stripe-Context>
  --api-version <version>
  --format json|yaml|jsonl
  --timeout <ms>
  --debug   Emit structured debug records to stderr (client setup + HTTP)
`
