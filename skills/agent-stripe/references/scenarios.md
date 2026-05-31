# agent-stripe Scenario Reference

Use this when translating a human Stripe incident question into commands. Prefer `investigate` commands for multi-stage questions.

## Customer Gave Card Last4

Question: "A customer told me the last 4 digits of their card. What is the most recent payment they made?"

```bash
agent-stripe investigate customer-card-payment --customer cus_... --last4 4242
```

Card last4 is not globally unique, so include the customer.

## Invoice Payment Details

Question: "I see an invoice for a customer. What card did they pay with, and how much did they pay?"

```bash
agent-stripe investigate invoice-payment in_...
```

This walks Invoice -> PaymentIntent -> latest Charge and reports amount paid plus card last4 when present.

## Invoice Copy To Internal Metadata

Question: "A customer sent me a copy of a Stripe invoice. I need internal product IDs from PaymentIntent metadata."

```bash
agent-stripe investigate invoice-metadata --number ABC-0001
agent-stripe investigate invoice-metadata in_...
```

Use the invoice number variant when the customer copy has no Stripe ID.

## Subscription Renewal Or Amount

Question: "A customer is on a subscription. When did they last pay, how much, when will they next pay, and how much?"

```bash
agent-stripe investigate subscription-renewal --subscription sub_...
agent-stripe investigate subscription-renewal --customer cus_...
agent-stripe investigate subscription-renewal --metadata tenant_id=acme
```

For item/product metadata:

```bash
agent-stripe investigate subscription-items --subscription sub_...
```

For invoice amount changes:

```bash
agent-stripe investigate subscription-amount-change --subscription sub_...
```

## Upcoming Collection Risk

Question: "Which customers should I outreach to about updating payment details before upcoming payments?"

```bash
agent-stripe investigate collection-risk --days 30 --limit 25
```

This flags missing default payment methods, expiring cards, open unpaid latest invoices, action-required states, and past-due/unpaid subscriptions.

## Failed Customer Payment

Question: "A payment to me went wrong. What happened?"

```bash
agent-stripe investigate incoming-payment pi_...
agent-stripe investigate incoming-payment ch_...
agent-stripe investigate incoming-payment in_...
```

This pulls PaymentIntent, Charge, Invoice, refunds, and disputes when related.

## Failed Connect Money Movement

Question: "A payment from me to a business I work with went wrong."

```bash
agent-stripe investigate outgoing-payment tr_...
agent-stripe investigate outgoing-payment po_...
agent-stripe investigate outgoing-payment acct_...
```

Question: "A business lets us draw money from their Stripe account for refunds, but it went wrong."

```bash
agent-stripe investigate refund-recovery re_...
agent-stripe investigate refund-recovery trr_... --transfer tr_...
```

Use `--context` when the incident is scoped to a connected account or organization related-account path.

## Webhook Or Dispute

```bash
agent-stripe investigate webhook-event evt_...
agent-stripe investigate dispute-response dp_...
```

Webhook event investigation fetches the event and the underlying object. Dispute investigation summarizes reason, status, due date, charge, customer, and PaymentIntent.

## Unknown ID Or Invoice Number

```bash
agent-stripe investigate resolve <stripe-id-or-invoice-number>
```

Use `resolve` when the user gives an ID but the object type is unclear, or when they give an invoice number rather than an `in_...` ID.

## Redacted Field Needed

If output contains an `@redacted` note and the user explicitly needs that hidden field:

```bash
agent-stripe payment-intents get pi_... --expose client_secret
agent-stripe invoices get in_... --expose hosted_invoice_url
```

Do not use `--expose` casually. Stored profile API keys cannot be exposed.
