# agent-stripe

Stripe incident triage CLI for AI agents. It is designed for read-heavy investigation workflows where an LLM needs compact structured output, actionable error hints, and no direct access to Stripe secrets.

## Features

- **Keychain-first credentials**: API keys are stored in macOS Keychain and are never printed back to the caller.
- **Multi-profile support**: configure aliases for sandbox, live, organization, or holding-account workflows.
- **Stripe context aware**: supports `Stripe-Context` for organization keys and related-account requests.
- **LLM-shaped output**: lists default to NDJSON, single resources default to JSON, and errors include `fixable_by` plus hints.
- **Read-first triage**: balance, events, PaymentIntents, charges, disputes, accounts, and a GET-only raw API escape hatch.

## Quick Start

```bash
make build
./agent-stripe auth add sandbox --api-key rk_test_... --context acct_...
./agent-stripe auth check sandbox
./agent-stripe events list --type charge.failed --limit 20
./agent-stripe payment-intents get pi_... --expand latest_charge
```

For organization API keys, store the organization key under a profile and provide a `Stripe-Context` value:

```bash
agent-stripe auth add org-prod --api-key sk_org_... --context acct_platform/acct_connected
agent-stripe -p org-prod balance get
```

## Output

Lists stream one JSON object per line:

```bash
agent-stripe disputes list --payment-intent pi_... --format jsonl
```

Errors are written to stderr as JSON:

```json
{"error":"Permission denied: ...","fixable_by":"human","hint":"The key may need read permissions for this Stripe resource..."}
```

## Development

```bash
make test
make build
make dev ARGS="usage"
```

## License

MIT
