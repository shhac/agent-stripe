# agent-stripe initial design

## Goal

Build a Go CLI that makes Stripe incident triage easy for an LLM while keeping Stripe credentials out of model-visible output.

The first version biases toward read-first investigation:

- identify the active Stripe account/context
- inspect balance and funds availability
- list and retrieve common billing, payment, Connect, catalog, and event objects
- search supported Stripe Search resources
- inspect disputes, refunds, payouts, transfers, and connected-account requirements
- inspect platform/connected accounts
- provide a GET-only raw API escape hatch for endpoints not yet wrapped
- use safe Stripe read-like helpers where needed for investigation, such as invoice previews

## Auth and profile model

Use `profile` as the user-facing alias because Stripe's current docs distinguish several account/key scopes:

- account-level sandbox and live keys
- restricted API keys
- organization API keys
- related-account context via `Stripe-Context`
- Connect connected accounts

Profiles store non-secret metadata in `${XDG_CONFIG_HOME}/agent-stripe/config.json`, or `~/.config/agent-stripe/config.json` when `XDG_CONFIG_HOME` is not set:

- alias
- default `Stripe-Context`
- default Stripe API version
- optional global defaults such as `timeout_ms` and `max_retries`

The API key itself is stored in macOS Keychain under the profile alias. A non-secret `credentials.json` index in the config directory records which profiles are Keychain-managed, but never stores the API key. The credential package deliberately has no method for listing or printing secret values. The `--api-key` flag is accepted as a non-persisted override for automation and tests, but output must never echo it.

The CLI should edit this file through validated commands, not by encouraging manual JSON edits:

- `agent-stripe auth add <profile> --api-key <key> [--context <context>] [--api-version <version>]`
- `agent-stripe auth add <profile> --form [--context <context>] [--api-version <version>]`
- `agent-stripe auth update <profile>` for profile metadata
- `agent-stripe auth check|list|default|remove`
- `agent-stripe config show|path|get|set|unset` for non-secret global defaults

When an LLM is guiding a human through setup, use `agent-stripe auth add <profile> --form`. The native OS dialog asks for the API key outside the agent's terminal/chat context, then the CLI stores it in Keychain and prints only a redacted receipt.

## Stripe context

Stripe's current account-scoping direction is `Stripe-Context`. The CLI exposes this as `--context` and stores an optional default on the profile.

Examples:

- standalone or account-key request: no context
- connected account from a platform key: `acct_connected`
- organization key to platform account: `acct_platform`
- organization key to connected account: `acct_platform/acct_connected`

## Output contract

Lists default to NDJSON so an LLM can stream, truncate, and resume investigation without parsing a large JSON array. Single-resource commands default to pretty JSON. Errors are JSON on stderr with:

- `error`
- `fixable_by`: `agent`, `human`, or `retry`
- optional `hint`

Stripe errors often include request IDs, request-log URLs, error codes, and decline codes. The API client should surface those in messages or hints when present.

`--debug` is the global diagnostic switch. It prints structured JSON records to stderr for client setup and HTTP responses. Debug output may include Stripe response bodies and request URLs, but must not include raw API keys.

Stripe rate limits and lock timeouts use HTTP 429. The CLI retries those responses with bounded exponential backoff and jitter, then emits a `fixable_by=retry` error with Stripe's rate-limit reason header when present. `--max-retries 0` disables automatic retries for callers that need one-shot behavior.

Known Stripe ID prefixes are preflighted before direct retrieval commands and before investigations whose input type is constrained. A confidently wrong known prefix returns a `fixable_by=agent` JSON error with a better command suggestion. Unknown values still pass through to Stripe or search flows so invoice numbers and unusual future Stripe IDs are not blocked prematurely. Investigation commands can deliberately accept related IDs when the relationship is recoverable, such as `invoice`, `charge`, and `payment_intent` IDs for `investigate incoming-payment`.

## Command surface

```text
agent-stripe auth add <profile> --api-key <key> [--context <context>] [--api-version <version>]
agent-stripe auth add <profile> --form [--context <context>] [--api-version <version>]
agent-stripe auth check|list|default|remove|update

agent-stripe config show|path|get|set|unset

agent-stripe balance get
agent-stripe events list|get
agent-stripe customers list|get|search
agent-stripe payment-intents list|get|search
agent-stripe charges list|get|search
agent-stripe disputes list|get
agent-stripe invoices list|get|search|line-items
agent-stripe subscriptions list|get|search|items|invoices
agent-stripe payment-methods list|get
agent-stripe refunds list|get
agent-stripe transfers list|get
agent-stripe payouts list|get
agent-stripe balance-transactions list|get
agent-stripe application-fees list|get
agent-stripe products list|get|search
agent-stripe prices list|get|search
agent-stripe setup-intents list|get
agent-stripe payment-links list|get
agent-stripe checkout-sessions list|get|line-items
agent-stripe early-fraud-warnings list|get
agent-stripe accounts self|list|get

agent-stripe investigate usage
agent-stripe investigate resolve <stripe-id-or-invoice-number>
agent-stripe investigate customer-context --customer cus_...
agent-stripe investigate customer-card-payment --customer cus_... --last4 4242
agent-stripe investigate webhook-event evt_...
agent-stripe investigate dispute-response dp_...
agent-stripe investigate invoice-payment in_...
agent-stripe investigate invoice-metadata in_...|--number ABC-0001
agent-stripe investigate subscription-renewal --subscription sub_...|--customer cus_...|--metadata key=value
agent-stripe investigate subscription-items --subscription sub_...
agent-stripe investigate subscription-amount-change --subscription sub_...
agent-stripe investigate collection-risk --days 30
agent-stripe investigate subscription-cancel-risk --days 30
agent-stripe investigate incoming-payment pi_...|ch_...|in_...
agent-stripe investigate outgoing-payment tr_...|po_...|acct_...
agent-stripe investigate refund-status re_...
agent-stripe investigate payout-failure po_...
agent-stripe investigate refund-recovery re_...|ch_...|pi_...|trr_... --transfer tr_...

agent-stripe payments usage
agent-stripe connect usage
agent-stripe usage
agent-stripe api get <path>
```

## Stripe docs checked

- API keys: sandbox/live modes, restricted keys, organization keys, key protection, and request logs.
- Authentication: API keys, restricted keys, per-request keys, HTTPS-only requests.
- Stripe-Context: account scoping for organization and related-account requests.
- Connect authentication: connected account IDs and the older `Stripe-Account` model.
- Request IDs: request identifiers are returned in `Request-Id` and are useful during support/debugging.
- Events: API v1 events include snapshot payloads, Connect events can identify the connected account, and events are retrievable for 30 days.
- PaymentIntent search: search is eventually consistent and should not be used for strict read-after-write flows.
- Disputes: disputes can be listed by PaymentIntent or charge, useful during incident investigation.
