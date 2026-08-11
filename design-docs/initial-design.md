# agent-stripe initial design

## Goal

Build a Go CLI that makes Stripe incident triage easy for an LLM while keeping Stripe credentials out of model-visible output.

The first version biases toward read-first investigation:

- identify the active Stripe account/context
- inspect balance and funds availability
- list and retrieve common billing, payment, Connect, catalog, and event objects
- search supported Stripe Search resources
- inspect disputes, refunds, payouts, transfers, and connected-account requirements
- inspect platform/connected accounts in both Stripe account namespaces: Connect v1 (`/v1/accounts`) and Accounts v2 / UA2 (`/v2/core/accounts`)
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
- default Stripe API version for `/v1` requests, and a separate default for `/v2` requests (the namespaces are versioned on different release trains; see `accounts-v2.md`)
- optional global defaults such as `timeout_ms` and `max_retries`
- non-secret credential classification: `rk_live`, `rk_test`, `sk_live`, `sk_test`, `pk_live`, `pk_test`, or `unknown`

The API key itself is stored in macOS Keychain under the profile alias. A non-secret `credentials.json` index in the config directory records which profiles are Keychain-managed, but never stores the API key. The credential package deliberately has no method for listing or printing secret values. The `--api-key` flag is accepted as a non-persisted override for automation and tests, but output must never echo it.

The CLI should edit this file through validated commands, not by encouraging manual JSON edits:

- `agent-stripe auth add <profile> --api-key <key> [--context <context>] [--api-version <version>]`
- `agent-stripe auth add <profile> --form [--context <context>] [--api-version <version>]`
- `agent-stripe auth update <profile> [--api-key <key>|--form]` for key replacement or profile metadata
- `agent-stripe auth check|list|default|remove`
- `agent-stripe config show|path|get|set|unset` for non-secret global defaults

When an LLM is guiding a human through setup, use `agent-stripe auth add <profile> --form` or `agent-stripe auth update <profile> --form`. The native OS dialog asks for the API key outside the agent's terminal/chat context, then the CLI stores it in Keychain and prints only a redacted receipt.

`auth add` and key-changing `auth update` classify the key prefix into non-secret `credential_type` metadata. `auth check` refreshes that value after reading the key internally. An omitted value means a legacy profile has not been classified yet; a saved `unknown` value means the stored key format was inspected but not recognized, so the profile should be tested rather than assumed invalid.

## Stripe context

Stripe's current account-scoping direction is `Stripe-Context`. The CLI exposes this as `--context` and stores an optional default on the profile.

Examples:

- standalone or account-key request: no context
- connected account from a platform key: `acct_connected`
- organization key to platform account: `acct_platform`
- organization key to connected account: `acct_platform/acct_connected`

## Output contract

Lists default to NDJSON so an LLM can stream, truncate, and resume investigation without parsing a large JSON array. List commands that commonly carry bulky nested payloads or sensitive person/payment details use compact summaries by default and expose `--full` for full redacted Stripe objects. On compact list commands, `--expand` requires `--full` so expanded payloads are never silently discarded. Single-resource `get` commands also default to NDJSON (one line per id); pass `--format json` for the pretty object. `get <id>...` accepts one or more ids and returns one result per id in input order — the record, or an `@unresolved` line for each id that couldn't be resolved (not found / bad id). Item-level misses stay on stdout with exit 0; only command-level failures go to stderr with exit 1. Errors are JSON on stderr with:

- `error`
- `fixable_by`: `agent`, `human`, or `retry`
- optional `hint`

Stripe errors often include request IDs, request-log URLs, error codes, and decline codes. The API client should surface those in messages or hints when present.

Sensitive Stripe response fields are redacted by default in resource output, investigation evidence, and debug response bodies. Redacted string values use a `"[REDACTED]"` marker so common Stripe scalar field shapes stay scalar, and the containing top-level object carries an `@redacted` path list so an LLM can tell which fields exist without seeing their values. `--expose <field-or-path>` is the explicit opt-in escape hatch; it accepts comma-separated values and repeated flags. It must never expose stored profile API keys.

`--debug` is the global diagnostic switch. It prints structured JSON records to stderr for client setup and HTTP responses. Debug output may include redacted Stripe response bodies and request URLs, but must not include raw API keys.

Stripe rate limits and lock timeouts use HTTP 429. The CLI retries those responses with bounded exponential backoff and jitter, then emits a `fixable_by=retry` error with Stripe's rate-limit reason header when present. `--max-retries 0` disables automatic retries for callers that need one-shot behavior.

Known Stripe ID prefixes are preflighted before direct retrieval commands and before investigations whose input type is constrained. On a `get` command, a confidently wrong known prefix returns an `@unresolved` record on stdout (exit 0) with `fixable_by=agent` and a better command suggestion — it is treated as an item-level miss, not a command-level error. Unknown values still pass through to Stripe or search flows so invoice numbers and unusual future Stripe IDs are not blocked prematurely. Investigation commands can deliberately accept related IDs when the relationship is recoverable, such as `invoice`, `charge`, and `payment_intent` IDs for `investigate incoming-payment`.

## Command surface

```text
agent-stripe auth add <profile> --api-key <key> [--context <context>] [--api-version <version>]
agent-stripe auth add <profile> --form [--context <context>] [--api-version <version>]
agent-stripe auth update <profile> [--api-key <key>|--form] [--context <context>|--clear-context] [--api-version <version>] [--default]
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
agent-stripe accounts-v2 get|list|persons
agent-stripe events-v2 list|get

agent-stripe investigate usage
agent-stripe investigate resolve <stripe-id-or-invoice-number>
agent-stripe investigate customer-context --customer cus_...
agent-stripe investigate customer-card-payment --customer cus_... --last4 4242
agent-stripe investigate webhook-event evt_...
agent-stripe investigate webhook-delivery evt_...|we_...
agent-stripe investigate dispute-response dp_...
agent-stripe investigate dispute-impact dp_...|ch_...|cus_...
agent-stripe investigate invoice-payment in_...
agent-stripe investigate invoice-collection in_...|cus_...|sub_...
agent-stripe investigate invoice-metadata in_...|--number ABC-0001
agent-stripe investigate subscription-renewal --subscription sub_...|--customer cus_...|--metadata key=value
agent-stripe investigate subscription-items --subscription sub_...
agent-stripe investigate subscription-amount-change --subscription sub_...
agent-stripe investigate entitlement --subscription sub_...|--invoice in_...|--checkout-session cs_...
agent-stripe investigate collection-risk --days 30
agent-stripe investigate subscription-cancel-risk --days 30
agent-stripe investigate incoming-payment pi_...|ch_...|in_...
agent-stripe investigate checkout-session cs_...
agent-stripe investigate payment-method-readiness cus_...|pm_...
agent-stripe investigate setup seti_...|pm_...|cus_...
agent-stripe investigate timeline cus_...
agent-stripe investigate outgoing-payment tr_...|po_...|acct_...
agent-stripe investigate account-health acct_... [--namespace auto|v1|v2]
agent-stripe investigate account-events acct_...
agent-stripe investigate ledger ch_...|pi_...|re_...|tr_...|po_...|txn_...|fee_...
agent-stripe investigate refund re_...|ch_...|pi_...
agent-stripe investigate payout-failure po_...
agent-stripe investigate refund-recovery re_...|ch_...|pi_...|trr_... --transfer tr_...
agent-stripe investigate fraud-review issfr_...|ch_...|pi_...

agent-stripe payments usage
agent-stripe connect usage
agent-stripe usage
agent-stripe api get <path>
```

The compact-summary list commands preserve navigation IDs and operational status while leaving bulky or sensitive full-object details to `get <id>` or explicit `list --full`. This currently covers customers, payment methods, PaymentIntents, charges, invoices, subscriptions, setup intents, Checkout Sessions, Payment Links, Events, connected accounts, v2 accounts, v2 account persons, and v2 core events. `accounts list` is intentionally compact rather than a raw Account-object dump: it preserves navigation IDs and operational enablement/requirements/capability counts, while leaving KYC/profile/settings/external-account details to `accounts get <acct_id>` or `accounts list --full`.

`accounts` and `accounts-v2` are deliberately separate command groups rather than one auto-detecting group. Both namespaces use the `acct_` prefix, so an ID cannot select the namespace, and silently retrying the other namespace would make it unclear which object model the output came from. Auto-detection lives only in `investigate`, where the caller is asking a question rather than naming an endpoint. See `accounts-v2.md`.

## Stripe docs checked

- API keys: sandbox/live modes, restricted keys, organization keys, key protection, and request logs.
- Authentication: API keys, restricted keys, per-request keys, HTTPS-only requests.
- Stripe-Context: account scoping for organization and related-account requests.
- Connect authentication: connected account IDs and the older `Stripe-Account` model.
- Request IDs: request identifiers are returned in `Request-Id` and are useful during support/debugging.
- Events: API v1 events include snapshot payloads, Connect events can identify the connected account, and events are retrievable for 30 days.
- PaymentIntent search: search is eventually consistent and should not be used for strict read-after-write flows.
- Disputes: disputes can be listed by PaymentIntent or charge, useful during incident investigation.
- API v2 namespace: JSON encoding, Bearer auth, a required `Stripe-Version` header, indexed array query parameters, and `page`/`next_page_url` pagination.
- Accounts v2: configurations, nested capabilities, requirement entries with capability impact, sparse `include` responses, and the documented v1/v2 interop error codes.
- v2 core events: thin events with `related_object`, 30-day retention, and account event types that only exist in the v2 namespace.
