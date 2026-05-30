---
name: agent-stripe
description: |
  Triage and investigate Stripe incidents, payments, invoices, subscriptions, disputes, refunds, balances, connected accounts, transfers, payouts, checkout, catalog/pricing, payment links, Radar warnings, and organization-account context. Use when:
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
- Prefer read-only commands.
- Use `--context` when the incident is scoped to a connected account or organization account path.
- Treat live-mode actions as high stakes; this initial CLI is read-first by design.

## Common Commands

```bash
agent-stripe auth list
agent-stripe auth add prod --form --context acct_...
agent-stripe auth check
agent-stripe usage
agent-stripe investigate usage
agent-stripe balance get
agent-stripe checkout-sessions list --customer cus_...
agent-stripe checkout-sessions line-items cs_...
agent-stripe events list --type charge.failed --limit 20
agent-stripe events get evt_...
agent-stripe customers list --email buyer@example.com
agent-stripe products list --active true
agent-stripe prices list --product prod_...
agent-stripe invoices get in_... --expand payment_intent
agent-stripe invoices line-items in_...
agent-stripe payment-intents get pi_... --expand latest_charge
agent-stripe payment-intents search --query "metadata['order_id']:'123'"
agent-stripe setup-intents list --customer cus_...
agent-stripe charges get ch_... --expand payment_intent --expand balance_transaction
agent-stripe payment-methods list --customer cus_... --type card
agent-stripe subscriptions get sub_... --expand latest_invoice --expand latest_invoice.payment_intent
agent-stripe subscriptions invoices sub_... --status open
agent-stripe disputes list --payment-intent pi_...
agent-stripe refunds list --payment-intent pi_...
agent-stripe transfers list --destination acct_...
agent-stripe payouts get po_...
agent-stripe balance-transactions get txn_...
agent-stripe payment-links list --active true
agent-stripe early-fraud-warnings list --charge ch_...
agent-stripe accounts self
agent-stripe api get /v1/payment_intents/pi_... --query expand[]=latest_charge
```

## Investigation Workflows

Prefer `investigate` commands when the user asks a question in incident language rather than asking for a specific Stripe object.

```bash
agent-stripe investigate resolve in_...
agent-stripe investigate resolve MOCK-0001
agent-stripe investigate customer-context --customer cus_...
agent-stripe investigate customer-card-payment --customer cus_... --last4 4242
agent-stripe investigate webhook-event evt_...
agent-stripe investigate dispute-response dp_...
agent-stripe investigate invoice-payment in_...
agent-stripe investigate invoice-metadata in_...
agent-stripe investigate invoice-metadata --number ABC-0001
agent-stripe investigate subscription-renewal --subscription sub_...
agent-stripe investigate subscription-renewal --metadata tenant_id=acme
agent-stripe investigate collection-risk --days 30
agent-stripe investigate subscription-cancel-risk --days 30
agent-stripe investigate incoming-payment pi_...
agent-stripe investigate incoming-payment ch_...
agent-stripe investigate outgoing-payment tr_...
agent-stripe investigate outgoing-payment po_...
agent-stripe investigate outgoing-payment acct_...
agent-stripe investigate refund-status re_...
agent-stripe investigate payout-failure po_...
agent-stripe investigate refund-recovery re_...
agent-stripe investigate refund-recovery trr_... --transfer tr_...
```

For local testing, run `mockstripe` and set `AGENT_STRIPE_BASE_URL` to its base URL.

## Output

Lists and investigation output default to NDJSON. Single resources default to JSON. Errors include `fixable_by` and usually a `hint`.

Investigation output uses evidence records:

```json
{"type":"entity","object":"invoice","id":"in_...","data":{}}
{"type":"finding","severity":"warning","summary":"...","data":{}}
```

Expanded nested Stripe objects are emitted as separate `entity` records and replaced by ID in the parent `data`, so navigation IDs stay visible and downstream commands can use the same Stripe-shaped fields. Long strings may be truncated with `truncated_fields`; rerun with `--expand-field <path>` or `--full`.
