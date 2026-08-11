package cli

import "github.com/spf13/cobra"

func registerUsageCommand(root *cobra.Command) {
	root.AddCommand(newUsageCommand(usageText))
}

const usageText = `agent-stripe — Stripe triage CLI for AI agents

PROFILE SETUP
  agent-stripe auth add <profile> --form [--context <acct_or_path>] [--api-version <version>] [--v2-api-version <version>]
  agent-stripe auth add <profile> --api-key <rk_or_sk> [--context <acct_or_path>] [--api-version <version>] [--v2-api-version <version>]
  agent-stripe auth check [profile]
  agent-stripe auth list
  agent-stripe auth default <profile>
  agent-stripe auth update <profile> [--api-key <rk_or_sk>|--form] [--context <ctx>|--clear-context] [--api-version <version>] [--v2-api-version <version>] [--default]
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
  auth list/check show credential_type only: rk_live|rk_test|sk_live|sk_test|pk_live|pk_test|unknown.
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
  agent-stripe accounts self
  agent-stripe accounts list [--full]
  agent-stripe accounts get <acct_id>
  agent-stripe accounts-v2 get <acct_id> [--include requirements]
  agent-stripe accounts-v2 list [--applied-configuration merchant|customer|recipient] [--page <token>]
  agent-stripe accounts-v2 persons list <acct_id>
  agent-stripe accounts-v2 usage
  agent-stripe events-v2 list [--object-id <acct_id>] [--type <v2.core...>]
  agent-stripe events-v2 get <evt_id>
  agent-stripe events-v2 usage

TWO ACCOUNT NAMESPACES
  Connect v1 (/v1/accounts) and Accounts v2 / UA2 (/v2/core/accounts) are different object models that
  share the acct_ prefix. 'accounts' reads v1 only; 'accounts-v2' reads v2 only; neither retries the other.
  When the namespace is unknown, ask instead of guessing:
    agent-stripe investigate resolve <acct_id>
    agent-stripe investigate account-health <acct_id>
  A v2 account ID works on v1 money-movement endpoints; a v1-only ID is rejected by /v2 with
  v1_account_instead_of_v2_account. See 'agent-stripe connect usage'.

INVESTIGATIONS
  agent-stripe investigate usage
  agent-stripe investigate resolve <stripe-id-or-invoice-number>
  agent-stripe investigate customer-context --customer <cus_id> [--limit N]
  agent-stripe investigate customer-card-payment --customer <cus_id> --last4 4242
  agent-stripe investigate webhook-event <evt_id>
  agent-stripe investigate webhook-delivery <evt_id|we_id>
  agent-stripe investigate dispute-response <dp_id>
  agent-stripe investigate dispute-impact <dp_id|ch_id|cus_id>
  agent-stripe investigate invoice-payment <in_id>
  agent-stripe investigate invoice-collection <in_id|cus_id|sub_id>
  agent-stripe investigate invoice-metadata <in_id>
  agent-stripe investigate subscription-renewal --subscription <sub_id>
  agent-stripe investigate subscription-renewal --metadata tenant_id=acme
  agent-stripe investigate subscription-items --subscription <sub_id>
  agent-stripe investigate subscription-amount-change --subscription <sub_id>
  agent-stripe investigate entitlement --subscription <sub_id>
  agent-stripe investigate collection-risk --days 30
  agent-stripe investigate subscription-cancel-risk --days 30
  agent-stripe investigate incoming-payment <pi_id|ch_id|in_id>
  agent-stripe investigate checkout-session <cs_id>
  agent-stripe investigate payment-method-readiness <cus_id|pm_id>
  agent-stripe investigate setup <seti_id|pm_id|cus_id>
  agent-stripe investigate timeline <cus_id>
  agent-stripe investigate outgoing-payment <tr_id|po_id|acct_id>
  agent-stripe investigate account-health <acct_id> [--namespace auto|v1|v2]
  agent-stripe investigate account-events <acct_id>
  agent-stripe investigate ledger <ch_id|pi_id|re_id|tr_id|po_id|txn_id|fee_id>
  agent-stripe investigate refund <re_id|ch_id|pi_id>
  agent-stripe investigate payout-failure <po_id>
  agent-stripe investigate refund-recovery <re_id|trr_id|ch_id|pi_id> [--transfer <tr_id>]
  agent-stripe investigate fraud-review <issfr_id|ch_id|pi_id>
  Investigation output is evidence NDJSON: entity records plus finding records.
  Nested expanded Stripe objects are emitted as separate entity records and replaced by ID in the parent.
  Use --max-string N, --expand-field <path>, or --full for verbose fields.
  Redaction still applies; use --expose <path,key> only when the hidden value is required.

RAW READ-ONLY API
  agent-stripe api get /v1/payment_intents/pi_... [--query expand[]=latest_charge]
  agent-stripe api get /v2/core/accounts/acct_... [--query include[0]=requirements]
  Only GET is exposed initially so agents can investigate without mutation risk.
  /v2 paths automatically use Bearer auth and the v2 API version. /v2 needs indexed arrays
  (include[0]=...), not the v1 include[]/expand[] form. Prefer the wrapped accounts-v2 and
  events-v2 commands: they validate includes, summarize, and paginate for you.

OUTPUT
  Lists default to NDJSON/jsonl, one object per line, with @pagination when there is another page.
  Bulky or sensitive list surfaces use compact summaries by default; add --full for raw redacted objects.
  On compact list commands, --expand requires --full; use get <id> for focused expanded reads.
  Get (single + multi). get <id>... takes one or more ids and returns one result per id, in input order.
  Default output is NDJSON: one line per id — the record, or {"@unresolved":{"id","reason","fixable_by","hint"?}}
  for an id that couldn't be resolved (e.g. not found / bad id). --format json|yaml collapses to one
  {"data":[...], "@unresolved":[...]} envelope. A single get <id> is just the one-element case (NDJSON one
  line by default; was pretty JSON before — pass --format json for the object). Item-level misses stay on
  stdout and exit 0; only a command-level failure (auth, network) goes to stderr with exit 1 and empty stdout.
  A wrong ID prefix on a get (e.g. invoices get pi_...) yields an @unresolved record (exit 0) instead of a
  stderr error. Redaction (@redacted / [REDACTED]) is unchanged and applies inside resolved records.
  Excluded from multi-get (take no id arg, so multi does not apply): balance get and accounts self (no id;
  default to NDJSON like all other gets — pass --format json for the object), invoice/checkout line-items,
  invoice preview. Raw passthroughs (api get, get --full raw dumps) output pretty JSON rather than NDJSON.
  config get <key>... accepts one or more keys and returns one NDJSON line per key; misses produce
  {"@unresolved":{"id","reason"}} entries (exit 0).
  Sensitive Stripe fields are replaced by "[REDACTED]" and indexed in @redacted; use --expose <path,key> to opt in.
  Errors are JSON on stderr with fixable_by: agent|human|retry and a hint where possible.
  Stripe 429s retry automatically with backoff and jitter before returning fixable_by=retry.

GLOBAL FLAGS
  -p, --profile <alias>
  --context <Stripe-Context>
  --api-version <version>     Stripe-Version for /v1 requests
  --v2-api-version <version>  Stripe-Version for /v2 requests (Accounts v2, v2 core events)
  -f, --format json|yaml|jsonl
  --expose <path,key>  Reveal redacted Stripe response fields by path or key; comma-separated/repeatable
  -t, --timeout <ms>
  --max-retries <N>  Maximum automatic retries for transient Stripe 429 responses (default 2)
  -d, --debug   Emit structured debug records to stderr (client setup + HTTP)
`
