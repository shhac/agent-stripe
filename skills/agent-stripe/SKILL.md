---
name: agent-stripe
description: |
  Triage and investigate Stripe payments, invoices, subscriptions, disputes,
  refunds, balances, connected accounts, transfers, payouts, checkout,
  catalog and pricing, payment links, Radar warnings, and organization-
  account context. Use when explaining a failed or successful charge,
  finding invoice payment details, card last4, or PaymentIntent metadata,
  investigating renewal timing, collection risk, or past-due invoices,
  tracing Connect transfers, payouts, or connected-account requirements, or
  working with Accounts v2 (UA2) configurations, capabilities, and persons.
  Triggers: early fraud warning, v2.core.account, onboarding blocked.
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
agent-stripe investigate duplicate-charge --customer cus_... [--window-hours 24]
agent-stripe investigate statement-descriptor --descriptor "FUREVER"
agent-stripe investigate action-required [--customer cus_...]
agent-stripe investigate refund-settlement <re_id|ch_id>
agent-stripe investigate invoice-total in_...
agent-stripe investigate checkout-session cs_...
agent-stripe investigate outgoing-payment <tr_id|po_id|acct_id>
agent-stripe investigate refund <re_id|ch_id|pi_id>
agent-stripe investigate ledger <ch_id|pi_id|re_id|tr_id|po_id|txn_id|fee_id>
agent-stripe investigate payout-failure po_...
agent-stripe investigate refund-recovery <re_id|trr_id|ch_id|pi_id> [--transfer tr_...]
agent-stripe investigate account-health acct_... [--namespace auto|v1|v2]
agent-stripe investigate account-events acct_...
agent-stripe investigate connect-readiness [--limit N]
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
agent-stripe accounts-v2 get acct_...
agent-stripe accounts-v2 list --applied-configuration merchant
agent-stripe accounts persons list acct_...
agent-stripe accounts capabilities acct_...
agent-stripe accounts external-accounts acct_...
agent-stripe accounts-v2 persons list acct_...
agent-stripe accounts-v2 payout-methods acct_...
agent-stripe events-v2 list --object-id acct_...
agent-stripe api get /v1/payment_intents/pi_... --query expand[]=latest_charge
```

For local testing, run `mockstripe` and set `AGENT_STRIPE_BASE_URL` to its base URL.

## Two Account Namespaces

Stripe has two connected-account models and both use the `acct_` prefix, so the ID never tells you which one applies:

- **Connect v1** (`/v1/accounts`) — the `accounts` commands. `charges_enabled`, `payouts_enabled`, `requirements.currently_due[]`.
- **Accounts v2 / UA2** (`/v2/core/accounts`) — the `accounts-v2` commands. Configurations (`customer`, `merchant`, `recipient`), per-capability status, and `requirements.entries[]`.

Do not guess. When the namespace is unknown, run `agent-stripe investigate resolve acct_...` or `agent-stripe investigate account-health acct_...` — both probe v2 first, fall back to v1, and state which namespace answered. `accounts` and `accounts-v2` deliberately stay in their own namespace, so a v1-only ID passed to `accounts-v2` returns `v1_account_instead_of_v2_account` with the v1 command in the hint.

Never describe a v2 account with v1 vocabulary. A `v2.core.account` has no `charges_enabled` or `payouts_enabled`; report the capability that is restricted (for example `merchant.card_payments`) and the requirement entries blocking it, noting whether each is `awaiting_action_from: user` or `stripe`.

When a requirement names something, look at what the account actually has rather than guessing: `accounts external-accounts` for `external_account`, `accounts capabilities` for an inactive capability (the v1 Capability objects carry their own requirements; the Account object carries only statuses), `accounts persons list` / `accounts-v2 persons list` for a person, and `accounts-v2 payout-methods` for where a v2 recipient would be paid. External accounts are a v1 endpoint even for v2 accounts — Accounts v2 has no external-accounts hash.

For the platform-wide question — which connected accounts need attention — use `investigate connect-readiness`. It retrieves each account because neither list endpoint carries capability or requirement detail.

Money movement is namespace-independent: transfers, payouts, balance transactions, and application fees are `/v1` endpoints that accept v2 account IDs, so `ledger`, `outgoing-payment`, `payout-failure`, and `refund-recovery` need no namespace choice.

Accounts v2 capability and requirement changes are emitted only as v2 thin events. `agent-stripe events list` will never show them; use `agent-stripe investigate account-events acct_...` or `agent-stripe events-v2 list --object-id acct_...`. Thin events carry no snapshot, so any object fetched for one is current state, not state at event time — say so when reporting it.

## Output

Lists and investigation output default to NDJSON. Single-resource `get` commands also default to NDJSON (one line); pass `--format json` for the pretty object. Errors include `fixable_by` and usually a `hint`.

List commands that commonly carry bulky nested payloads or sensitive person/payment details return compact summaries by default. This includes customers, payment methods, PaymentIntents, charges, invoices, subscriptions, setup intents, Checkout Sessions, Payment Links, Events, connected accounts, v2 accounts, v2 account persons, and v2 core events. Add `--full` to those list commands only when raw redacted Stripe objects are needed.

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

On `/v2` surfaces, two behaviours differ from `/v1`. Stripe returns `null` for `configuration`, `identity`, `requirements`, `future_requirements`, and `defaults` unless they are requested — `accounts-v2 get` requests every include by default, so a `null` in its output means genuinely unset, but a `null` in `accounts-v2 list` output only means the list endpoint cannot return it. And v2 lists have no cursor IDs: read `@pagination.next_page` and pass it back as `--page <token>`. `/v2` requests use their own `Stripe-Version` (`--v2-api-version`, profile `v2_api_version`); pin a preview train only when a field or endpoint is preview-only.

Stripe `429` responses retry automatically with bounded exponential backoff and jitter. Use `--max-retries 0` for one-shot behavior or `--debug` to see retry records on stderr.

Non-secret profile/config metadata lives in XDG config. API keys live in Keychain. `auth list` and `auth check` may show `credential_type` (`rk_live`, `rk_test`, `sk_live`, `sk_test`, `pk_live`, `pk_test`, or `unknown`) but never the key. Use `agent-stripe config show` or `agent-stripe config path` for config inspection; use `auth update` rather than editing JSON by hand.

## Incremental References

Load these only when you need more detail:

- [references/commands.md](references/commands.md): command map, domain usage commands, and flags.
- [references/investigations.md](references/investigations.md): investigation chooser table with args, use cases, and per-command detail links.
- [references/output.md](references/output.md): output, redaction, truncation, pagination, errors, and debug records.
- [references/scenarios.md](references/scenarios.md): common incident questions and recommended command sequences.
