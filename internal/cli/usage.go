package cli

import "github.com/spf13/cobra"

func registerUsageCommand(root *cobra.Command) {
	root.AddCommand(newUsageCommand(usageText))
}

const usageText = `agent-stripe — Stripe triage CLI for AI agents

PROFILE SETUP
  agent-stripe auth add <profile> --form [--context <acct_or_path>] [--api-version <version>]
  agent-stripe auth add <profile> --api-key <rk_or_sk> [--context <acct_or_path>] [--api-version <version>]
  agent-stripe auth check [profile]
  agent-stripe auth list
  agent-stripe auth default <profile>
  agent-stripe auth update <profile> [--context <ctx>|--clear-context] [--api-version <version>] [--default]
  agent-stripe auth remove <profile>
  agent-stripe config path
  agent-stripe config show
  agent-stripe config get max_retries|timeout_ms
  agent-stripe config set max_retries|timeout_ms <non-negative-int>
  agent-stripe config unset max_retries|timeout_ms
  agent-stripe payments usage
  agent-stripe connect usage

CONTEXT
  --context <acct_...> applies Stripe-Context for organization keys and related-account requests.
  For a platform connected account, use the connected account ID.
  For organization key access to a platform-connected account, use <platform_acct>/<connected_acct>.
  The API key is stored in macOS Keychain; this CLI never prints it back.
  Non-secret config is stored in XDG config, usually ~/.config/agent-stripe/config.json.
  Prefer --form when an LLM is guiding setup so the user types the key into an OS popup.

TRIAGE STARTERS
  agent-stripe balance get
  agent-stripe checkout-sessions list [--customer <cus_id>] [--payment-intent <pi_id>]
  agent-stripe checkout-sessions line-items <cs_id>
  agent-stripe events list [--type charge.failed] [--created-gte <unix>] [--limit N]
  agent-stripe events get <evt_id>
  agent-stripe customers list [--email <email>]
  agent-stripe products list [--active true]
  agent-stripe prices list [--product <prod_id>] [--active true]
  agent-stripe invoices get <in_id> [--expand payment_intent]
  agent-stripe invoices line-items <in_id>
  agent-stripe invoices usage
  agent-stripe payment-intents get <pi_id> [--expand latest_charge] [--expand customer]
  agent-stripe payment-intents search --query "metadata['order_id']:'123'"
  agent-stripe setup-intents list [--customer <cus_id>] [--payment-method <pm_id>]
  agent-stripe charges get <ch_id> [--expand payment_intent] [--expand balance_transaction]
  agent-stripe charges search --query "payment_method_details.card.last4:'4242'"
  agent-stripe subscriptions get <sub_id> --expand latest_invoice --expand latest_invoice.payment_intent
  agent-stripe subscriptions list [--customer <cus_id>] [--status active|past_due|unpaid|all]
  agent-stripe subscriptions invoices <sub_id> [--status open]
  agent-stripe subscriptions usage
  agent-stripe payment-methods list --customer <cus_id> --type card
  agent-stripe disputes list [--charge <ch_id>] [--payment-intent <pi_id>]
  agent-stripe refunds list [--payment-intent <pi_id>]
  agent-stripe transfers list [--destination <acct_id>]
  agent-stripe payouts get <po_id>
  agent-stripe balance-transactions get <txn_id>
  agent-stripe payment-links list [--active true]
  agent-stripe early-fraud-warnings list [--charge <ch_id>] [--payment-intent <pi_id>]
  agent-stripe accounts list
  agent-stripe accounts get <acct_id>

INVESTIGATIONS
  agent-stripe investigate usage
  agent-stripe investigate resolve <stripe-id-or-invoice-number>
  agent-stripe investigate customer-context --customer <cus_id> [--limit N]
  agent-stripe investigate customer-card-payment --customer <cus_id> --last4 4242
  agent-stripe investigate webhook-event <evt_id>
  agent-stripe investigate dispute-response <dp_id>
  agent-stripe investigate invoice-payment <in_id>
  agent-stripe investigate invoice-metadata <in_id>
  agent-stripe investigate subscription-renewal --subscription <sub_id>
  agent-stripe investigate subscription-renewal --metadata tenant_id=acme
  agent-stripe investigate subscription-items --subscription <sub_id>
  agent-stripe investigate subscription-amount-change --subscription <sub_id>
  agent-stripe investigate collection-risk --days 30
  agent-stripe investigate subscription-cancel-risk --days 30
  agent-stripe investigate incoming-payment <pi_id|ch_id|in_id>
  agent-stripe investigate outgoing-payment <tr_id|po_id|acct_id>
  agent-stripe investigate refund-status <re_id>
  agent-stripe investigate payout-failure <po_id>
  agent-stripe investigate refund-recovery <re_id|trr_id|ch_id|pi_id> [--transfer <tr_id>]
  Investigation output is evidence NDJSON: entity records plus finding records.
  Nested expanded Stripe objects are emitted as separate entity records and replaced by ID in the parent.
  Use --max-string N, --expand-field <path>, or --full for verbose fields.

RAW READ-ONLY API
  agent-stripe api get /v1/payment_intents/pi_... [--query expand[]=latest_charge]
  Only GET is exposed initially so agents can investigate without mutation risk.

OUTPUT
  Lists default to NDJSON/jsonl, one object per line, with @pagination when there is another page.
  Single-object reads default to pretty JSON.
  Sensitive Stripe fields are replaced by {"@redacted":true,...}; use --expose <path,key> to opt in.
  Errors are JSON on stderr with fixable_by: agent|human|retry and a hint where possible.
  Stripe 429s retry automatically with backoff and jitter before returning fixable_by=retry.

GLOBAL FLAGS
  -p, --profile <alias>
  --context <Stripe-Context>
  --api-version <version>
  --format json|yaml|jsonl
  --expose <path,key>  Reveal redacted Stripe response fields by path or key; comma-separated/repeatable
  --timeout <ms>
  --max-retries <N>  Maximum automatic retries for transient Stripe 429 responses (default 2)
  --debug   Emit structured debug records to stderr (client setup + HTTP)
`
