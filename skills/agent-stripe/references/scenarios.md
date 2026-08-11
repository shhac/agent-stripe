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

## Invoice Collection Or Dunning

Question: "This invoice is open or past due. When will Stripe retry and why did collection fail?"

```bash
agent-stripe investigate invoice-collection in_...
agent-stripe investigate invoice-collection cus_...
agent-stripe investigate invoice-collection sub_...
```

Use this instead of `invoice-payment` when the invoice was not successfully paid.

## Failed Customer Payment

Question: "A payment to me went wrong. What happened?"

```bash
agent-stripe investigate incoming-payment pi_...
agent-stripe investigate incoming-payment ch_...
agent-stripe investigate incoming-payment in_...
```

This pulls PaymentIntent, Charge, Invoice, refunds, and disputes when related.

## Checkout Or Setup Did Not Complete

```bash
agent-stripe investigate checkout-session cs_...
agent-stripe investigate setup seti_...
agent-stripe investigate payment-method-readiness cus_...
```

Use Checkout for hosted checkout completion and line-item/product metadata. Use Setup for saved-payment setup status. Use payment-method-readiness when the question is whether future billing can use a customer's saved card.

## Entitlements Or Product IDs

```bash
agent-stripe investigate entitlement --subscription sub_...
agent-stripe investigate entitlement --invoice in_...
agent-stripe investigate entitlement --checkout-session cs_...
```

This gathers subscription items, invoice lines, Checkout line items, prices, products, and metadata.

## Failed Connect Money Movement

Question: "A payment from me to a business I work with went wrong."

```bash
agent-stripe investigate outgoing-payment tr_...
agent-stripe investigate outgoing-payment po_...
agent-stripe investigate outgoing-payment acct_...
agent-stripe investigate ledger tr_...
```

Question: "A business lets us draw money from their Stripe account for refunds, but it went wrong."

```bash
agent-stripe investigate refund-recovery re_...
agent-stripe investigate refund-recovery trr_... --transfer tr_...
```

Use `--context` when the incident is scoped to a connected account or organization related-account path.

## Connected Account Cannot Take Payments Or Get Paid

Question: "This business says they can't accept cards / haven't been paid out. Why?"

```bash
agent-stripe investigate account-health acct_...
```

Do not assume which account model the ID uses — Connect v1 and Accounts v2 share the `acct_` prefix, and `account-health` resolves that for you and says which namespace answered.

If it answered with Accounts v2, the blocker is a capability whose status is not `active` (for example `merchant.card_payments`, `merchant.stripe_balance.payouts`) plus the requirement entries that restrict it. Report which entries are `awaiting_action_from: user` — those are the ones anyone can act on — and which are with Stripe. A v2 account has no `charges_enabled` / `payouts_enabled`; never phrase a v2 answer in those terms.

If it answered with Connect v1, `charges_enabled` / `payouts_enabled` plus `requirements.currently_due` are the answer.

Question: "What changed on this account recently?"

```bash
agent-stripe investigate account-events acct_...
```

Accounts v2 capability and requirement changes exist only in the v2 event stream, so `events list` will not show them. The events are thin — the account state shown alongside them is current, not point-in-time.

Question: "Who is on this account's identity, and what is missing?"

```bash
agent-stripe accounts-v2 get acct_... --include requirements
agent-stripe accounts-v2 persons list acct_...
```

A requirement entry with `reference.resource` pointing at a `person_...` ID tells you which person the missing information belongs to. Person names, dates of birth, and contact details are redacted by default; ask before using `--expose`.

For customer-visible refund state:

```bash
agent-stripe investigate refund re_...
agent-stripe investigate refund ch_...
```

For finance reconciliation:

```bash
agent-stripe investigate ledger ch_...
agent-stripe investigate ledger txn_...
```

## Webhook Or Dispute

```bash
agent-stripe investigate webhook-event evt_...
agent-stripe investigate webhook-delivery evt_...
agent-stripe investigate dispute-response dp_...
agent-stripe investigate dispute-impact dp_...
agent-stripe investigate fraud-review issfr_...
```

Webhook event investigation fetches the event and the underlying object. Webhook delivery checks event pending webhooks and endpoint configuration. Dispute response summarizes reason, status, due date, charge, customer, and PaymentIntent. Dispute impact and fraud review are broader risk/exposure workflows.

## Customer Timeline

Question: "Give me a concise timeline of what happened to this customer."

```bash
agent-stripe investigate timeline cus_...
```

This is a good first command when the user has a broad or confusing support narrative.

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
