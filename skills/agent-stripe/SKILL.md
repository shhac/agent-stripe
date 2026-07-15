---
name: agent-stripe
description: |
  Triage and investigate Stripe payments, invoices, and subscriptions. Covers incidents, disputes, refunds, balances, connected accounts, transfers, payouts, checkout, catalog/pricing, payment links, Radar warnings, and organization-account context. Use when:
  - Explaining failed or successful customer payments
  - Finding invoice payment details, card last4, or PaymentIntent metadata
  - Investigating subscriptions, renewal timing, collection risk, or past-due invoices
  - Investigating Connect transfers, payouts, refund recovery, or connected-account requirements
  - Looking up Stripe events, customers, payment methods, charges, refunds, balance transactions, accounts, or application fees
  Triggers: "stripe", "payment intent", "payment_intent", "charge failed", "invoice paid", "subscription", "card last4", "refund", "transfer", "payout", "connected account", "stripe connect", "stripe metadata", "collection risk", "checkout session", "payment link", "dispute", "early fraud warning"
allowed-tools: Bash(agent-stripe *) Bash(mockstripe *) Read Grep Glob
---

# agent-stripe

Use `agent-stripe` when investigating Stripe payment incidents, invoice questions, subscription billing, webhook/event questions, disputes, refunds, balances, connected accounts, or organization-account context.

## Safety

- Never ask the tool to reveal an API key.
- Never accept pasted Stripe API keys in chat. Ask the user to run `agent-stripe auth add <profile> --form` locally so the key goes directly into an OS dialog.
- Use `agent-stripe auth update <profile> --form` when a stored key needs to be replaced.
- Prefer read-only commands.
- Use `--context` when the incident is scoped to a connected account or organization account path.
- Treat live-mode actions as high stakes; this CLI is read-first by design.
- Use `--expose <path,key>` only when the user explicitly needs a redacted Stripe response field. Stored profile API keys are never exposed by this flag.

## Start Here

```bash
agent-stripe usage
agent-stripe investigate usage
agent-stripe auth list
agent-stripe auth check
agent-stripe config show
agent-stripe balance get
```

Prefer `investigate` commands when the user asks a question in incident language rather than asking for a specific Stripe object:

```bash
agent-stripe investigate resolve <stripe-id-or-invoice-number>
agent-stripe investigate customer-context --customer cus_...
agent-stripe investigate customer-card-payment --customer cus_... --last4 4242
agent-stripe investigate invoice-payment in_...
agent-stripe investigate invoice-metadata in_...
agent-stripe investigate invoice-collection in_...
agent-stripe investigate subscription-renewal --subscription sub_...
agent-stripe investigate collection-risk --days 30
agent-stripe investigate incoming-payment <pi_id|ch_id|in_id>
agent-stripe investigate checkout-session cs_...
agent-stripe investigate outgoing-payment <tr_id|po_id|acct_id>
agent-stripe investigate refund <re_id|ch_id|pi_id>
agent-stripe investigate ledger <ch_id|pi_id|re_id|tr_id|po_id|txn_id|fee_id>
agent-stripe investigate payout-failure po_...
agent-stripe investigate refund-recovery <re_id|trr_id|ch_id|pi_id> [--transfer tr_...]
```

For direct exploration, use resource commands:

```bash
agent-stripe customers list --email buyer@example.com
agent-stripe payment-intents get pi_... --expand latest_charge
agent-stripe charges get ch_... --expand payment_intent --expand balance_transaction
agent-stripe invoices get in_... --expand payment_intent
agent-stripe subscriptions get sub_... --expand latest_invoice --expand latest_invoice.payment_intent
agent-stripe payment-methods list --customer cus_... --type card
agent-stripe accounts self
agent-stripe accounts list
agent-stripe accounts list --full
agent-stripe accounts get acct_...
agent-stripe api get /v1/payment_intents/pi_... --query expand[]=latest_charge
```

For local testing, run `mockstripe` and set `AGENT_STRIPE_BASE_URL` to its base URL.

## Output

Lists and investigation output default to NDJSON. Single-resource `get` commands also default to NDJSON (one line); pass `--format json` for the pretty object. Errors include `fixable_by` and usually a `hint`.

List commands that commonly carry bulky nested payloads or sensitive person/payment details return compact summaries by default. This includes customers, payment methods, PaymentIntents, charges, invoices, subscriptions, setup intents, Checkout Sessions, Payment Links, Events, and connected accounts. Add `--full` to those list commands only when raw redacted Stripe objects are needed.

Sensitive Stripe fields are redacted by default with `"[REDACTED]"` leaf values and a top-level `@redacted` path list. Use `--expose <path,key>` only when the user explicitly needs that field for the investigation; `--expose` can be comma-separated or repeated.

Investigation output uses evidence records:

```json
{"type":"entity","object":"invoice","id":"in_...","data":{}}
{"type":"finding","severity":"warning","summary":"...","data":{}}
```

Expanded nested Stripe objects are emitted as separate `entity` records and replaced by ID in the parent `data`, so navigation IDs stay visible and downstream commands can use the same Stripe-shaped fields. Long strings may be truncated with `truncated_fields`; rerun with `--expand-field <path>` or `--full`. Truncation controls do not override redaction.

**Get (single + multi).** `get <id>...` takes one or more ids and returns one result per id, in input order. Default output is NDJSON: one line per id — the record, or `{"@unresolved":{"id","reason","fixable_by","hint"?}}` for an id that couldn't be resolved (e.g. not found / bad id). `--format json|yaml` collapses to one `{"data":[…], "@unresolved":[…]}` envelope. A single `get <id>` is just the one-element case (NDJSON one line by default; was pretty JSON before — pass `--format json` for the object). Item-level misses stay on stdout and exit 0; only a command-level failure (auth, network) goes to stderr with exit 1 and empty stdout.

A wrong ID prefix on a `get` (e.g. `invoices get pi_...`) yields an `@unresolved` record on stdout (exit 0) instead of a stderr error. Redaction (`@redacted` / `[REDACTED]`) is unchanged and applies inside resolved records.

Commands excluded from multi-get (take no id arg, so multi does not apply): `balance get` and `accounts self` (no id; default to NDJSON like all other gets — pass `--format json` for the object), `invoice/checkout line-items`, `invoice preview`. Raw passthroughs (`api get`, `get --full` raw dumps) output pretty JSON rather than NDJSON. `config get <key>...` accepts one or more keys and returns one NDJSON line per key; misses produce `{"@unresolved":{"id","reason"}}` entries (exit 0).

`accounts list` is compact by default and omits full Account KYC/profile/settings/external-account data. Use `accounts get acct_...` for one account or `accounts list --full` only when raw list objects are needed.

Stripe `429` responses retry automatically with bounded exponential backoff and jitter. Use `--max-retries 0` for one-shot behavior or `--debug` to see retry records on stderr.

Non-secret profile/config metadata lives in XDG config. API keys live in Keychain. `auth list` and `auth check` may show `credential_type` (`rk_live`, `rk_test`, `sk_live`, `sk_test`, `pk_live`, `pk_test`, or `unknown`) but never the key. Use `agent-stripe config show` or `agent-stripe config path` for config inspection; use `auth update` rather than editing JSON by hand.

## Incremental References

Load these only when you need more detail:

- [references/commands.md](references/commands.md): command map, domain usage commands, and flags.
- [references/investigations.md](references/investigations.md): investigation chooser table with args, use cases, and per-command detail links.
- [references/output.md](references/output.md): output, redaction, truncation, pagination, errors, and debug records.
- [references/scenarios.md](references/scenarios.md): common incident questions and recommended command sequences.
