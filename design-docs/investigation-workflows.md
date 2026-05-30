# Investigation Workflows

`agent-stripe investigate` exists for cases where an LLM should not need to manually walk the Stripe object graph. Commands emit NDJSON evidence records:

- `entity`: a Stripe object retrieved during the investigation.
- `finding`: a concise conclusion, warning, or next step.

The resource commands remain available for direct exploration, but investigation commands should encode common incident paths.

## Adding A Workflow

Each investigation class should be mostly independent:

1. Add a focused file in `internal/cli/investigate_<domain>.go`.
2. Add a `newInvestigate<Name>(globals shared.GlobalsFunc) *cobra.Command` factory.
3. Register the factory in `investigationCommands` in `internal/cli/investigate.go`.
4. Keep Stripe traversal methods near the workflow that owns them unless they are clearly shared.
5. Emit raw Stripe objects as `entity` records and conclusions as `finding` records.
6. Add mock fixtures/routes in `internal/mockstripe`.
7. Add a mock-backed e2e test that proves the user-facing scenario works.
8. Update `agent-stripe investigate usage`, top-level `usage`, this design doc, and the skill.

Good workflows accept incident-language inputs (`in_...`, `pi_...`, `last4`, metadata, customer, account) and return enough evidence for an LLM to answer without issuing five follow-up raw API calls.

## Customer Card Last4

Command:

```bash
agent-stripe investigate customer-card-payment --customer cus_... --last4 4242
```

Card last4 is not globally unique, so this command requires a customer. It lists recent charges for that customer and filters locally on `payment_method_details.card.last4`.

## Invoice Payment

Command:

```bash
agent-stripe investigate invoice-payment in_...
```

Path:

```text
Invoice -> PaymentIntent -> latest Charge
```

The finding includes invoice amount paid and card last4 when a card charge is present.

## Subscription Renewal

Command:

```bash
agent-stripe investigate subscription-renewal --subscription sub_...
agent-stripe investigate subscription-renewal --customer cus_...
agent-stripe investigate subscription-renewal --metadata tenant_id=acme
```

Path:

```text
Subscription -> latest Invoice -> PaymentIntent -> latest Charge
Subscription -> invoice preview
```

This is aimed at support questions such as "when did they last pay, how much, when will they next pay, and how much?"

## Invoice Metadata

Command:

```bash
agent-stripe investigate invoice-metadata in_...
agent-stripe investigate invoice-metadata --number ABC-0001
```

Path:

```text
Invoice -> PaymentIntent.metadata
```

This handles customer-provided invoice copies where internal product IDs are stored on the PaymentIntent.

## Collection Risk

Command:

```bash
agent-stripe investigate collection-risk --days 30
```

This scans upcoming subscription renewals and flags subscriptions that are already `past_due`, `unpaid`, `incomplete`, or have an open unpaid latest invoice. Future versions should add deeper checks for expiring default payment methods and invoice retry policy.

## Failed Customer Payment

Command:

```bash
agent-stripe investigate incoming-payment pi_...
agent-stripe investigate incoming-payment ch_...
agent-stripe investigate incoming-payment in_...
```

Path:

```text
PaymentIntent or Charge or Invoice -> related charge -> disputes/refunds -> failure fields
```

The finding summarizes charge failure messages, PaymentIntent `last_payment_error`, and related refunds/disputes.

## Connect Money Movement

Commands:

```bash
agent-stripe investigate outgoing-payment tr_...
agent-stripe investigate outgoing-payment po_...
agent-stripe investigate outgoing-payment acct_...
agent-stripe investigate refund-recovery re_...
agent-stripe investigate refund-recovery trr_... --transfer tr_...
```

These distinguish Transfers, Payouts, connected Accounts, Refunds, and Transfer Reversals. For transfer reversals, Stripe nests the reversal under its parent transfer, so the command requires `--transfer` when given only a `trr_` ID.

## Improvements To Prioritize

- Add a `resolve` investigation that accepts any Stripe ID or invoice number and emits the likely object type plus recommended commands.
- Add `customer-context` to gather customer, default payment method, recent invoices, recent PaymentIntents, open disputes, and active subscriptions.
- Add `webhook-event` to explain what an event means and fetch the underlying object.
- Add `dispute-response` to summarize dispute status, due date, reason, evidence fields, and related charge/customer.
- Add `refund-status` to distinguish pending, failed, succeeded, reversed, and Connect `reverse_transfer` cases.
- Add `payout-failure` to include balance transactions and connected-account external account requirements.
- Add `subscription-cancel-risk` to summarize cancellations, trial endings, unpaid status, and upcoming invoice amount.
- Add richer collection-risk checks for expiring cards, missing default payment methods, invoice retry windows, and `requires_action` PaymentIntents.
- Add optional `--include-raw=false` or `--summary-only` once evidence payloads become too large for routine LLM use.
