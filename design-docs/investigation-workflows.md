# Investigation Workflows

`agent-stripe investigate` exists for cases where an LLM should not need to manually walk the Stripe object graph. Commands emit NDJSON evidence records:

- `entity`: a Stripe object retrieved during the investigation.
- `finding`: a concise conclusion, warning, or next step.

The resource commands remain available for direct exploration, but investigation commands should encode common incident paths.

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
