# agent-stripe initial design

## Goal

Build a Go CLI that makes Stripe incident triage easy for an LLM while keeping Stripe credentials out of model-visible output.

The first version should bias toward read-only investigation:

- identify the active Stripe account/context
- inspect balance and funds availability
- list and retrieve events
- search and retrieve PaymentIntents and charges
- inspect disputes
- inspect platform/connected accounts
- provide a GET-only raw API escape hatch for endpoints not yet wrapped

## Auth and profile model

Use `profile` as the user-facing alias because Stripe's current docs distinguish several account/key scopes:

- account-level sandbox and live keys
- restricted API keys
- organization API keys
- related-account context via `Stripe-Context`
- Connect connected accounts

Profiles store non-secret metadata in `~/.config/agent-stripe/config.json`:

- alias
- default `Stripe-Context`
- default Stripe API version

The API key itself is stored in macOS Keychain under the profile alias. The credential package deliberately has no method for listing or printing secret values. The `--api-key` flag is accepted as a non-persisted override for automation and tests, but output must never echo it.

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

## Initial command surface

```text
agent-stripe auth add <profile> --api-key <key> [--context <context>] [--api-version <version>]
agent-stripe auth add <profile> --form [--context <context>] [--api-version <version>]
agent-stripe auth check [profile]
agent-stripe auth list
agent-stripe auth default <profile>
agent-stripe auth remove <profile>

agent-stripe balance get
agent-stripe events list|get
agent-stripe payment-intents list|get|search
agent-stripe charges list|get|search
agent-stripe disputes list|get
agent-stripe subscriptions list|get|search|items|invoices
agent-stripe accounts self|list|get
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
