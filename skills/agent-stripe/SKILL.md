---
name: agent-stripe
description: Triage and investigate Stripe incidents, payments, events, disputes, balances, accounts, and connected-account context.
---

# agent-stripe

Use `agent-stripe` when investigating Stripe payment incidents, webhook/event questions, disputes, balances, connected accounts, or organization-account context.

## Safety

- Never ask the tool to reveal an API key.
- Prefer read-only commands.
- Use `--context` when the incident is scoped to a connected account or organization account path.
- Treat live-mode actions as high stakes; this initial CLI is read-first by design.

## Common Commands

```bash
agent-stripe auth list
agent-stripe auth check
agent-stripe balance get
agent-stripe events list --type charge.failed --limit 20
agent-stripe events get evt_...
agent-stripe payment-intents get pi_... --expand latest_charge
agent-stripe payment-intents search --query "metadata['order_id']:'123'"
agent-stripe charges get ch_... --expand payment_intent --expand balance_transaction
agent-stripe disputes list --payment-intent pi_...
agent-stripe accounts self
agent-stripe api get /v1/payment_intents/pi_... --query expand[]=latest_charge
```

## Output

Lists default to NDJSON. Single resources default to JSON. Errors include `fixable_by` and usually a `hint`.
