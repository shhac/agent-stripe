# agent-stripe

Stripe incident triage CLI for AI agents. It is designed for read-heavy investigation workflows where an LLM needs compact structured output, actionable error hints, and no direct access to Stripe secrets.

## Features

- **Keychain-first credentials**: API keys are stored in macOS Keychain and are never printed back to the caller.
- **Multi-profile support**: configure aliases for sandbox, live, organization, or holding-account workflows.
- **Stripe context aware**: supports `Stripe-Context` for organization keys and related-account requests.
- **LLM-shaped output**: lists default to NDJSON, single resources default to JSON, and errors include `fixable_by` plus hints.
- **Read-first triage**: balance, events, PaymentIntents, charges, disputes, accounts, and a GET-only raw API escape hatch.
- **Subscription investigation**: inspect subscriptions, subscription items, invoices, and payment failures from one command group.

## Quick Start

```bash
make build
./agent-stripe auth add sandbox --form --context acct_...
./agent-stripe auth check sandbox
./agent-stripe events list --type charge.failed --limit 20
./agent-stripe payment-intents get pi_... --expand latest_charge
./agent-stripe subscriptions get sub_... --expand latest_invoice --expand latest_invoice.payment_intent
./agent-stripe subscriptions invoices sub_... --status open
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

Use `--debug` to emit extra JSON records to stderr while commands run. Debug output includes client setup details, credential source labels, request URLs, status codes, request IDs, and response bodies, but not raw API keys.

## Development

```bash
make test
make build
make build-mock
make dev ARGS="usage"
```

## Mock Stripe

Run a local mock server for safe e2e development:

```bash
make build-mock
./mockstripe --addr 127.0.0.1:12111
AGENT_STRIPE_BASE_URL=http://127.0.0.1:12111 agent-stripe --api-key sk_test_mock events list
```

## License

MIT
