# agent-stripe

Stripe incident triage CLI for AI agents. It is designed for read-heavy investigation workflows where an LLM needs compact structured output, actionable error hints, and no direct access to Stripe secrets.

## Features

- **Keychain-first credentials**: API keys are stored in macOS Keychain and are never printed back to the caller.
- **Multi-profile support**: configure aliases for sandbox, live, organization, or holding-account workflows.
- **Stripe context aware**: supports `Stripe-Context` for organization keys and related-account requests.
- **LLM-shaped output**: lists default to NDJSON, single resources default to JSON, sensitive Stripe fields are redacted by default, and errors include `fixable_by` plus hints.
- **Bounded Stripe retries**: transient Stripe `429` responses retry with exponential backoff and jitter before returning a retryable error.
- **Read-first triage**: balance, events, PaymentIntents, charges, disputes, accounts, and a GET-only raw API escape hatch.
- **Subscription investigation**: inspect subscriptions, subscription items, invoices, and payment failures from one command group.
- **Scenario investigations**: invoice payment evidence, customer card-last4 lookup, subscription renewal summaries, collection-risk outreach, failed incoming payments, and Connect money-movement failures.

## Quick Start

```bash
make build
./agent-stripe auth add sandbox --form --context acct_...
./agent-stripe auth check sandbox
./agent-stripe auth update sandbox --context acct_...
./agent-stripe config show
./agent-stripe events list --type charge.failed --limit 20
./agent-stripe investigate resolve MOCK-0001
./agent-stripe investigate customer-context --customer cus_...
./agent-stripe payment-intents get pi_... --expand latest_charge
./agent-stripe subscriptions get sub_... --expand latest_invoice --expand latest_invoice.payment_intent
./agent-stripe subscriptions invoices sub_... --status open
./agent-stripe subscriptions usage
./agent-stripe payments usage
./agent-stripe connect usage
./agent-stripe investigate usage
./agent-stripe investigate invoice-payment in_...
./agent-stripe investigate customer-card-payment --customer cus_... --last4 4242
./agent-stripe investigate subscription-renewal --metadata tenant_id=acme
./agent-stripe investigate subscription-items --subscription sub_...
./agent-stripe investigate subscription-amount-change --subscription sub_...
./agent-stripe investigate collection-risk --days 30
```

For organization API keys, store the organization key under a profile and provide a `Stripe-Context` value:

```bash
agent-stripe auth add org-prod --form --context acct_platform/acct_connected
agent-stripe -p org-prod balance get
```

When an LLM is guiding setup, prefer `--form` over `--api-key`. A native OS dialog asks the user for the key, and the CLI returns only a redacted setup receipt.

## Output

Lists stream one JSON object per line:

```bash
agent-stripe disputes list --payment-intent pi_... --format jsonl
```

Errors are written to stderr as JSON:

```json
{"error":"Permission denied: ...","fixable_by":"human","hint":"The key may need read permissions for this Stripe resource..."}
```

Sensitive Stripe response fields are redacted by default. Redacted string values are replaced with `"[REDACTED]"`, and the containing top-level object includes an `@redacted` list with the field path and expose hint. Use `--expose <field-or-path>` when a human has decided the field is safe for the current investigation:

```bash
agent-stripe payment-intents get pi_... --expose client_secret
agent-stripe payment-intents get pi_... --expose metadata.internal_order_token,receipt_url
```

`--expose` accepts comma-separated values and can be repeated. It applies to Stripe response output and debug response bodies, but never exposes stored profile API keys.

Use `--debug` to emit extra JSON records to stderr while commands run. Debug output includes client setup details, credential source labels, request URLs, status codes, request IDs, and redacted response bodies, but not raw API keys.

Stripe `429` responses are retried automatically up to `--max-retries` times, which defaults to 2. After retries are exhausted, the CLI returns a JSON error with `fixable_by:"retry"` and includes Stripe's rate-limit reason header when present.

Profile metadata is stored at `${XDG_CONFIG_HOME}/agent-stripe/config.json`, or `~/.config/agent-stripe/config.json` when `XDG_CONFIG_HOME` is not set. A non-secret `credentials.json` index in the same directory records which profiles are Keychain-managed. API keys are stored separately in macOS Keychain and are not written to either file.

Use `agent-stripe auth update <profile>` to change non-secret profile metadata, and `agent-stripe config set max_retries|timeout_ms <value>` to persist global defaults. Command-line flags still override persisted defaults.

## Commands

Resource commands expose Stripe objects directly for agent-controlled exploration:

```bash
agent-stripe customers list --email buyer@example.com
agent-stripe checkout-sessions list --customer cus_...
agent-stripe products list --active true
agent-stripe prices list --product prod_...
agent-stripe invoices get in_... --expand payment_intent
agent-stripe invoices line-items in_...
agent-stripe setup-intents list --customer cus_...
agent-stripe payment-methods list --customer cus_... --type card
agent-stripe refunds list --payment-intent pi_...
agent-stripe transfers list --destination acct_...
agent-stripe payouts get po_...
agent-stripe balance-transactions get txn_...
agent-stripe application-fees list --charge ch_...
agent-stripe payment-links list --active true
agent-stripe early-fraud-warnings list --charge ch_...
agent-stripe accounts self
agent-stripe accounts list
agent-stripe accounts get acct_...
agent-stripe api get /v1/payment_intents/pi_... --query expand[]=latest_charge
agent-stripe payments usage
agent-stripe connect usage
agent-stripe auth update prod --context acct_...
agent-stripe config path
agent-stripe config show
agent-stripe config set max_retries 2
```

Investigation commands walk common Stripe object graphs and emit evidence records plus findings:

```bash
agent-stripe investigate invoice-payment in_...
agent-stripe investigate invoice-metadata in_...
agent-stripe investigate invoice-metadata --number ABC-0001
agent-stripe investigate subscription-items --subscription sub_...
agent-stripe investigate subscription-amount-change --subscription sub_...
agent-stripe investigate webhook-event evt_...
agent-stripe investigate dispute-response dp_...
agent-stripe investigate incoming-payment pi_...
agent-stripe investigate outgoing-payment tr_...
agent-stripe investigate refund-status re_...
agent-stripe investigate payout-failure po_...
agent-stripe investigate refund-recovery trr_... --transfer tr_...
```

Investigation output keeps Stripe-shaped `data` while emitting nested expanded Stripe objects as their own `entity` records. Long strings are truncated by default with `truncated_fields`; use `--expand-field <path>` or `--full` when the hidden content matters. Sensitive fields are still redacted unless explicitly exposed with `--expose`.

## Development

```bash
make test
make build
make build-mock
make mock
make mock-dev ARGS="events list --type charge.failed"
make dev ARGS="usage"
```

## LLM Skill

The bundled skill lives at `skills/agent-stripe/SKILL.md`. It keeps the core safety and workflow guidance short, with deeper command, output, and scenario references in `skills/agent-stripe/references/`.

## Mock Stripe

Run a local mock server for safe e2e development:

```bash
make build-mock
./mockstripe --routes
./mockstripe --addr 127.0.0.1:12111
AGENT_STRIPE_BASE_URL=http://127.0.0.1:12111 agent-stripe --api-key sk_test_mock events list
```

`make mock-dev ARGS="..."` runs `agent-stripe` against the default mock server URL with a mock API key.

## License

MIT
