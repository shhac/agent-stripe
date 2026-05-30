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

TRIAGE STARTERS
  agent-stripe balance get
  agent-stripe events list [--type charge.failed] [--created-gte <unix>] [--limit N]
  agent-stripe events get <evt_id>
  agent-stripe payment-intents get <pi_id> [--expand latest_charge] [--expand customer]
  agent-stripe payment-intents search --query "metadata['order_id']:'123'"
  agent-stripe charges get <ch_id> [--expand payment_intent] [--expand balance_transaction]
  agent-stripe charges search --query "payment_method_details.card.last4:'4242'"
  agent-stripe disputes list [--charge <ch_id>] [--payment-intent <pi_id>]
  agent-stripe accounts list
  agent-stripe accounts get <acct_id>

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
`
