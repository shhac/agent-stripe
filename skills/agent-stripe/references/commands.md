# agent-stripe Command Reference

Run `agent-stripe usage` for the concise in-CLI reference. Run domain usage commands before a focused investigation:

```bash
agent-stripe invoices usage
agent-stripe subscriptions usage
agent-stripe payments usage
agent-stripe connect usage
agent-stripe investigate usage
```

## Auth And Config

- `agent-stripe auth add <profile> --form [--context <ctx>] [--api-version <version>]` - LLM-safe setup. The user types the API key into a native OS dialog.
- `agent-stripe auth add <profile> --api-key <key> [--context <ctx>] [--api-version <version>]` - direct setup when the key is already in the user's shell, not chat.
- `agent-stripe auth check [profile]` - verify the active or named profile and refresh stored `credential_type` metadata when possible.
- `agent-stripe auth list` - list profile metadata without secrets, including `credential_type` and hints for missing, unrecognized, or publishable keys.
- `agent-stripe auth default <profile>` - set the default profile.
- `agent-stripe auth update <profile> [--api-key <key>|--form] [--context <ctx>|--clear-context] [--api-version <version>] [--default]` - replace a stored key or edit non-secret profile metadata.
- `agent-stripe auth remove <profile>` - remove a stored profile.
- `agent-stripe config path|show|get|set|unset` - inspect or edit non-secret config.

## Direct Resource Exploration

- `agent-stripe balance get`
- `agent-stripe events list|get`
- `agent-stripe customers list|get|search`
- `agent-stripe checkout-sessions list|get|line-items`
- `agent-stripe products list|get|search`
- `agent-stripe prices list|get|search`
- `agent-stripe invoices list|get|search|line-items`
- `agent-stripe payment-intents list|get|search`
- `agent-stripe setup-intents list|get`
- `agent-stripe charges list|get|search`
- `agent-stripe payment-methods list|get`
- `agent-stripe subscriptions list|get|search|items|invoices`
- `agent-stripe disputes list|get`
- `agent-stripe refunds list|get`
- `agent-stripe transfers list|get`
- `agent-stripe payouts list|get`
- `agent-stripe balance-transactions list|get`
- `agent-stripe application-fees list|get`
- `agent-stripe payment-links list|get`
- `agent-stripe early-fraud-warnings list|get`
- `agent-stripe accounts self|list|get` - `accounts list` returns compact status summaries by default; use `accounts list --full` for full redacted Account objects.

Most list commands accept `--limit`, `--created-gte`, `--created-lte`, `--starting-after`, and `--ending-before`. List commands with compact defaults also accept `--full` for full redacted Stripe objects; when such a command supports `--expand`, use `--full` with `--expand`. Search commands accept `--query`, `--limit`, and `--page`.

### Get Contract (single + multi)

`get <id>...` takes one or more ids and returns one result per id, in input order. Default output is NDJSON: one line per id — the record, or `{"@unresolved":{"id","reason","fixable_by","hint"?}}` for an id that couldn't be resolved (e.g. not found / bad id). `--format json|yaml` collapses to one `{"data":[…], "@unresolved":[…]}` envelope. A single `get <id>` is just the one-element case (NDJSON one line by default; was pretty JSON before — pass `--format json` for the object). Item-level misses stay on stdout and exit 0; only a command-level failure (auth, network) goes to stderr with exit 1 and empty stdout.

A wrong ID prefix on a `get` (e.g. `invoices get pi_...`) yields an `@unresolved` record on stdout (exit 0) instead of a stderr error. Redaction (`@redacted` / `[REDACTED]`) is unchanged and applies inside resolved records.

Commands excluded from multi-get (take no id arg, so multi does not apply): `balance get` and `accounts self` (no id; default to NDJSON like all other gets — pass `--format json` for the object), invoice/checkout `line-items`, `invoice preview`. Raw passthroughs (`api get`, `get --full` raw dumps) output pretty JSON rather than NDJSON. `config get <key>...` accepts one or more keys and returns one NDJSON line per key; misses produce `{"@unresolved":{"id","reason"}}` entries (exit 0).

## Investigations

- `agent-stripe investigate resolve <stripe-id-or-invoice-number>` - identify object type and suggest next commands.
- `agent-stripe investigate customer-context --customer cus_... [--limit N]` - gather customer, payment methods, subscriptions, invoices, PaymentIntents, charges, disputes, and refunds.
- `agent-stripe investigate customer-card-payment --customer cus_... --last4 4242 [--limit N]` - find the most recent matching customer payment by card last4.
- `agent-stripe investigate webhook-event evt_...` - fetch event and underlying object.
- `agent-stripe investigate webhook-delivery evt_... [--endpoint we_...]|we_...` - inspect event pending webhooks and endpoint config.
- `agent-stripe investigate dispute-response dp_...` - summarize dispute status, due date, reason, charge, customer, and PaymentIntent.
- `agent-stripe investigate dispute-impact dp_...|ch_...|cus_...` - dispute exposure and related payment/refund evidence.
- `agent-stripe investigate invoice-payment in_...` - walk Invoice -> PaymentIntent -> latest Charge.
- `agent-stripe investigate invoice-collection in_...|cus_...|sub_...` - invoice retry, attempt count, next payment attempt, and collection state.
- `agent-stripe investigate invoice-metadata in_...` or `--number ABC-0001` - find PaymentIntent metadata for internal product/order IDs.
- `agent-stripe investigate subscription-renewal --subscription sub_...|--customer cus_...|--metadata key=value` - latest and next payment summary.
- `agent-stripe investigate subscription-items --subscription sub_...` - subscription items, prices, products, and metadata.
- `agent-stripe investigate subscription-amount-change --subscription sub_...` - latest invoice, invoice lines, preview, and current item subtotal.
- `agent-stripe investigate entitlement --subscription sub_...|--customer cus_...|--metadata key=value|--invoice in_...|--checkout-session cs_...` - product/price metadata for entitlement mismatches.
- `agent-stripe investigate collection-risk --days 30 [--limit N]` - upcoming subscriptions needing payment detail outreach.
- `agent-stripe investigate subscription-cancel-risk --days 30 [--limit N]` - subscriptions ending soon or set to cancel.
- `agent-stripe investigate incoming-payment <pi_id|ch_id|in_id>` - failed or successful customer payment explanation.
- `agent-stripe investigate checkout-session cs_...` - Checkout completion, line items, and resulting payment/subscription.
- `agent-stripe investigate payment-method-readiness cus_...|pm_...` - saved payment method attachment and card readiness.
- `agent-stripe investigate setup seti_...|pm_...|cus_...` - SetupIntent status and reusable payment method evidence.
- `agent-stripe investigate timeline cus_...` - chronological customer activity context.
- `agent-stripe investigate outgoing-payment <tr_id|po_id|acct_id>` - Connect transfer, payout, or account readiness issue.
- `agent-stripe investigate account-health acct_...` - connected account requirements/capability blockers.
- `agent-stripe investigate ledger ch_...|pi_...|re_...|tr_...|po_...|txn_...|fee_...` - balance transaction and reconciliation evidence.
- `agent-stripe investigate refund <re_id|ch_id|pi_id>` - refund state and related movement.
- `agent-stripe investigate payout-failure po_...` - payout failure plus balance transaction.
- `agent-stripe investigate refund-recovery <re_id|trr_id|ch_id|pi_id> [--transfer tr_...]` - refund and transfer reversal recovery.
- `agent-stripe investigate fraud-review issfr_...|ch_...|pi_...` - Radar early fraud warning and charge risk evidence.

For a chooser table and per-investigation details, see [investigations.md](investigations.md).

## Raw Read-Only API

Use when a needed read endpoint has no first-class command yet:

```bash
agent-stripe api get /v1/payment_intents/pi_... --query expand[]=latest_charge
```

Only GET is exposed.

## Global Flags

- `-p, --profile <alias>` - Stripe profile alias.
- `--context <Stripe-Context>` - organization or related-account request context.
- `--api-version <version>` - Stripe API version override.
- `-f, --format json|yaml|jsonl` - output format.
- `--expose <path,key>` - reveal redacted Stripe response fields by path or key; comma-separated/repeatable.
- `-t, --timeout <ms>` - request timeout.
- `--max-retries <N>` - automatic retries for transient Stripe 429 responses. Default: 2.
- `-d, --debug` - structured debug records to stderr.
