# agent-stripe

Stripe incident triage CLI for AI agents. It is designed for read-heavy investigation workflows where an LLM needs compact structured output, actionable error hints, and no direct access to Stripe secrets.

## Features

- **Keychain-first credentials**: API keys are stored in macOS Keychain and are never printed back to the caller.
- **Multi-profile support**: configure aliases for sandbox, live, organization, or holding-account workflows.
- **Stripe context aware**: supports `Stripe-Context` for organization keys and related-account requests.
- **Both account namespaces**: Connect v1 (`/v1/accounts`) and Accounts v2 / UA2 (`/v2/core/accounts`), including v2 configurations, capabilities, requirement entries, persons, and v2 core events — with the `/v2` transport (Bearer auth, its own API version, indexed array parameters, token pagination) handled for you.
- **LLM-shaped output**: lists default to NDJSON, single `get` commands also default to NDJSON (one line; pass `--format json` for the object), sensitive Stripe fields are redacted by default, and errors include `fixable_by` plus hints.
- **Bounded Stripe retries**: transient Stripe `429` responses retry with exponential backoff and jitter before returning a retryable error.
- **Read-first triage**: balance, events, PaymentIntents, charges, disputes, accounts, and a GET-only raw API escape hatch.
- **Subscription investigation**: inspect subscriptions, subscription items, invoices, and payment failures from one command group.
- **Scenario investigations**: invoice payment evidence, Checkout completion, customer card-last4 lookup, subscription renewal summaries, collection-risk outreach, failed incoming payments, refund/dispute/fraud triage, ledger reconciliation, Connect money-movement failures, and connected-account health across both account namespaces.

## Quick Start

```bash
make build
./agent-stripe auth add sandbox --form --context acct_...
./agent-stripe auth check sandbox
./agent-stripe auth update sandbox --context acct_...
./agent-stripe auth update sandbox --form
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
./agent-stripe investigate checkout-session cs_...
./agent-stripe investigate invoice-collection in_...
./agent-stripe investigate timeline cus_...
./agent-stripe investigate account-health acct_...
./agent-stripe investigate account-events acct_...
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

`auth add` and `auth update` store only the credential type, such as `rk_test` or `sk_live`, as non-secret metadata. `auth list` and `auth check` show `credential_type`; legacy profiles without stored metadata render as `unknown` with a hint to run `auth check`, while genuinely unrecognized saved formats also render as `unknown` with a test-it hint. Use `agent-stripe auth update <profile> --form` to replace a stored key without exposing it to the LLM.

Use `agent-stripe auth update <profile>` to change profile key or non-secret metadata, and `agent-stripe config set max_retries|timeout_ms <value>` to persist global defaults. Command-line flags still override persisted defaults.

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
agent-stripe accounts list        # compact connected-account status summaries
agent-stripe accounts list --full # full Stripe account objects, redacted
agent-stripe accounts get acct_...
agent-stripe accounts-v2 get acct_...              # v2 account with every include requested
agent-stripe accounts-v2 list --applied-configuration merchant
agent-stripe accounts persons list acct_...        # v1 account sub-resources
agent-stripe accounts capabilities acct_...
agent-stripe accounts external-accounts acct_...
agent-stripe accounts-v2 persons list acct_...
agent-stripe accounts-v2 payout-methods acct_...   # v2 recipient payout routes (preview API)
agent-stripe events-v2 list --object-id acct_...   # v2 thin events for an account
agent-stripe api get /v1/payment_intents/pi_... --query expand[]=latest_charge
agent-stripe payments usage
agent-stripe connect usage
agent-stripe auth update prod --context acct_...
agent-stripe auth update prod --form
agent-stripe config path
agent-stripe config show
agent-stripe config set max_retries 2
```

List commands that commonly carry bulky nested payloads or sensitive person/payment details return compact summaries by default. This includes customers, payment methods, PaymentIntents, charges, invoices, subscriptions, setup intents, Checkout Sessions, Payment Links, Events, connected accounts, v2 accounts, v2 account persons, and v2 core events. Add `--full` to those list commands when you need the full redacted Stripe object; use `get <id>` for focused inspection. On compact list commands, `--expand` requires `--full`.

Investigation commands walk common Stripe object graphs and emit evidence records plus findings:

```bash
agent-stripe investigate invoice-payment in_...
agent-stripe investigate invoice-metadata in_...
agent-stripe investigate invoice-metadata --number ABC-0001
agent-stripe investigate subscription-items --subscription sub_...
agent-stripe investigate subscription-amount-change --subscription sub_...
agent-stripe investigate entitlement --subscription sub_...
agent-stripe investigate checkout-session cs_...
agent-stripe investigate webhook-event evt_...
agent-stripe investigate webhook-delivery evt_...
agent-stripe investigate dispute-response dp_...
agent-stripe investigate dispute-impact dp_...
agent-stripe investigate incoming-payment pi_...
agent-stripe investigate duplicate-charge --customer cus_...
agent-stripe investigate statement-descriptor --descriptor "FUREVER"
agent-stripe investigate action-required
agent-stripe investigate refund-settlement re_...
agent-stripe investigate invoice-total in_...
agent-stripe investigate invoice-collection in_...
agent-stripe investigate payment-method-readiness cus_...
agent-stripe investigate setup seti_...
agent-stripe investigate timeline cus_...
agent-stripe investigate outgoing-payment tr_...
agent-stripe investigate account-health acct_...
agent-stripe investigate account-events acct_...
agent-stripe investigate connect-readiness
agent-stripe investigate ledger ch_...
agent-stripe investigate refund re_...
agent-stripe investigate fraud-review issfr_...
agent-stripe investigate payout-failure po_...
agent-stripe investigate refund-recovery trr_... --transfer tr_...
```

### Connected accounts: Connect v1 and Accounts v2 (UA2)

Stripe has two connected-account models and they share the `acct_` prefix, so an ID alone cannot tell you which applies:

| | Connect v1 | Accounts v2 (UA2) |
| --- | --- | --- |
| Endpoint | `/v1/accounts` | `/v2/core/accounts` |
| Commands | `accounts` | `accounts-v2` |
| Enablement | `charges_enabled`, `payouts_enabled` | per-capability status inside each configuration |
| Outstanding info | `requirements.currently_due[]` field names | `requirements.entries[]` with owner, deadline, and restricted capabilities |
| Events | `/v1/events` snapshots | `/v2/core/events` thin events |

`accounts` and `accounts-v2` each stay in their own namespace and never silently retry the other, so output is always unambiguous. When you don't know which namespace an account is in, ask:

```bash
agent-stripe investigate resolve acct_...          # names the namespace
agent-stripe investigate account-health acct_...   # probes v2, falls back to v1, reports which answered
```

`account-health` reads each model with its own logic — a v2 account has no `charges_enabled`, so it is never reported for one. The v2 finding names restricted capabilities (`merchant.card_payments`, `merchant.stripe_balance.payouts`), which requirement entries are waiting on you rather than on Stripe, and which capabilities each entry is holding back.

A v2 account ID is accepted by the v1 money-movement endpoints, so `transfers`, `payouts`, `balance-transactions`, `application-fees`, and the `ledger` / `outgoing-payment` / `refund-recovery` investigations work for both models unchanged. A v1-only ID sent to `/v2` is rejected with `v1_account_instead_of_v2_account`, and the error hint names the v1 command to use instead.

Requirements name what is missing but not what exists. `accounts external-accounts`, `accounts capabilities`, and `accounts persons list` answer that for Connect v1 — and the external-accounts one matters for v2 accounts too, because Accounts v2 has no external-accounts hash and Stripe directs platforms to the v1 endpoint. For a v2 recipient, `accounts-v2 payout-methods` shows where a payout would actually go; that surface is preview-only, so pass `--v2-api-version <preview train>`.

`investigate connect-readiness` sweeps connected accounts and reports only the blocked ones. Neither list endpoint returns capability or requirement detail — Stripe's v2 list supports no `include` at all — so it retrieves each account, which is why `--limit` defaults low.

Two `/v2` details worth knowing, both handled by the wrapped commands: Stripe returns `null` for `configuration`, `identity`, `requirements`, `future_requirements`, and `defaults` unless you request them (`accounts-v2 get` requests all of them by default), and v2 lists paginate by token — read `@pagination.next_page` and pass it back as `--page`.

`/v2` requests use their own `Stripe-Version`, separate from the v1 one, because Stripe versions the namespaces on different release trains:

```bash
agent-stripe auth add sandbox --form --api-version 2025-06-30.basil --v2-api-version 2026-07-29.dahlia
agent-stripe accounts-v2 get acct_... --v2-api-version 2026-07-29.preview   # pin a preview train
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

## Claude Code / AI agent skill

```bash
npx skills add shhac/agent-skills --skill agent-stripe --global
```

Installs the `agent-stripe` skill globally so Claude Code (and other AI agents) can discover and use it automatically. It ships from [`shhac/agent-skills`](https://github.com/shhac/agent-skills) — the whole family's skills in one repo, so `npx skills update` checks a single source no matter how many you use. Want several at once? Run `npx skills add shhac/agent-skills --global` and pick from the list.

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

PolyForm Perimeter License 1.0.0 — see [LICENSE](LICENSE).
