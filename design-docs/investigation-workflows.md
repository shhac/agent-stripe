# Investigation Workflows

`agent-stripe investigate` exists for cases where an LLM should not need to manually walk the Stripe object graph. Commands emit NDJSON evidence records:

- `entity`: a Stripe object retrieved during the investigation.
- `finding`: a concise conclusion, warning, or next step.

The resource commands remain available for direct exploration, but investigation commands should encode common incident paths.

Investigation output preserves Stripe-shaped `data` as much as possible. When Stripe returns an expanded nested object in a field that can also be an ID string, the parent field is replaced by the nested object's ID and the nested object is emitted as its own `entity` record. Long strings are truncated by default, with `truncated_fields` pointing at `--expand-field <path>` or `--full`. Sensitive Stripe fields are redacted by default with `"[REDACTED]"` leaf values and a top-level `@redacted` path list; `--expose <field-or-path>` is required to reveal them.

## Adding A Workflow

Each investigation class should be mostly independent:

1. Add a focused file in `internal/cli/investigate_<domain>.go`.
2. Add a `newInvestigate<Name>(globals shared.GlobalsFunc, outputOpts *investigationOutputOptions) *cobra.Command` factory.
3. Register the factory in `investigationCommands` in `internal/cli/investigate.go`.
4. Keep Stripe traversal methods near the workflow that owns them unless they are clearly shared.
5. Emit raw Stripe objects as `entity` records and conclusions as `finding` records.
6. Add mock fixtures/routes in `internal/mockstripe`.
7. Add a mock-backed e2e test that proves the user-facing scenario works.
8. Update `agent-stripe investigate usage`, top-level `usage`, this design doc, and the skill.
9. Add or update ID-prefix validation when the workflow accepts only specific object types. Use `validateAllowedStripeID` when related prefixes are recoverable, and `validateExpectedStripeID` when the input must be one object type.

Good workflows accept incident-language inputs (`in_...`, `pi_...`, `last4`, metadata, customer, account) and return enough evidence for an LLM to answer without issuing five follow-up raw API calls.

## ID Handling

Direct `get` commands reject known wrong Stripe ID prefixes before making a request. For example, `invoices get pi_...` returns a JSON error that points at `payment-intents get`, `investigate incoming-payment`, and `investigate resolve`.

Investigation commands should be relationship-aware:

- Accept alternate IDs when the workflow can recover by traversing related objects, such as `incoming-payment` accepting `in_`, `ch_`, and `pi_`.
- Reject unrelated known prefixes with `fixable_by=agent` and a suggested command.
- Let unknown strings pass through when a workflow intentionally supports invoice numbers, search terms, or future Stripe ID shapes.
- Keep navigable IDs visible in output even when an expanded object is emitted separately.

Current workflow families:

- `resolve`: identify a Stripe ID or invoice number and suggest next command.
- `customer-context`: gather recent objects around a customer.
- `customer-card-payment`: bounded last4 lookup for a customer.
- `webhook-event`: fetch event and underlying object.
- `webhook-delivery`: event `pending_webhooks` plus endpoint config.
- `dispute-response`: summarize dispute response status and related payment.
- `dispute-impact`: dispute exposure from dispute, charge, or customer.
- `invoice-payment`: invoice to PaymentIntent to latest Charge.
- `invoice-collection`: invoice retry/collection state from invoice, customer, or subscription.
- `invoice-metadata`: invoice to PaymentIntent metadata.
- `subscription-renewal`: latest and next invoice/payment.
- `subscription-items`: subscription items, prices, products, and product metadata.
- `subscription-amount-change`: latest invoice, invoice lines, preview, and current item subtotal.
- `entitlement`: product/price metadata across subscriptions, invoices, and Checkout.
- `collection-risk`: payment-method outreach candidates.
- `subscription-cancel-risk`: cancellations and trial/period end risk.
- `incoming-payment`: failed/successful customer payment explanation.
- `checkout-session`: Checkout Session, line items, and resulting payment/subscription.
- `payment-method-readiness`: saved payment method attachment/readiness.
- `setup`: SetupIntent/payment method setup status.
- `timeline`: chronological customer context from recent Stripe objects.
- `outgoing-payment`: transfers, payouts, and connected account readiness.
- `account-health`: connected account requirements and blockers.
- `ledger`: balance transaction and reconciliation evidence.
- `refund` and `refund-recovery`: refund state and transfer reversal recovery.
- `payout-failure`: payout failure plus ledger movement.
- `fraud-review`: Radar early fraud warnings, charge outcome, disputes, and refunds.

## Command Shape Decisions

Keep commands separate when the user's question implies a different starting object, vocabulary, or answer shape. Combine internally when the graph traversal is the same.

- `refund` is the broad public command for refund state from `re_`, `ch_`, or `pi_`; pre-major releases do not keep old alias names.
- `account-health` owns connected account requirement/capability checks. `outgoing-payment acct_...` delegates to it, while `outgoing-payment tr_...|po_...` remains money-movement focused.
- `dispute-response` stays narrow for evidence deadline/response state. `dispute-impact` is broader and can start from a dispute, charge, or customer.
- `subscription-items` stays direct and compact. `entitlement` is the broader "what internal product should they have?" workflow across subscription, invoice, and Checkout evidence.
- `webhook-event` answers "what event was this?" while `webhook-delivery` answers "did our webhook receive it?"

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

## Invoice Collection

Command:

```bash
agent-stripe investigate invoice-collection in_...
agent-stripe investigate invoice-collection cus_...
agent-stripe investigate invoice-collection sub_...
```

Path:

```text
Invoice(s) -> PaymentIntent -> latest Charge
```

This answers open/past-due invoice questions with status, paid flag, amount due, attempt count, next payment attempt, and hosted invoice URL.

## Subscription Renewal

Command:

```bash
agent-stripe investigate subscription-renewal --subscription sub_...
agent-stripe investigate subscription-renewal --customer cus_...
agent-stripe investigate subscription-renewal --metadata tenant_id=acme
agent-stripe investigate subscription-items --subscription sub_...
agent-stripe investigate subscription-amount-change --subscription sub_...
```

Path:

```text
Subscription -> latest Invoice -> PaymentIntent -> latest Charge
Subscription -> invoice preview
Subscription -> subscription items -> Price -> Product metadata
Subscription -> latest Invoice lines and preview amount
```

This is aimed at support questions such as "when did they last pay, how much, when will they next pay, and how much?"
Use `subscription-items` when the important question is "which internal product IDs or prices are attached?"
Use `subscription-amount-change` when the important question is "why is this invoice amount different?"
Use `entitlement` when the user asks whether the customer has the right internal product/plan across subscription items, invoice lines, or Checkout line items.

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

This scans upcoming subscription renewals and flags subscriptions that are already `past_due`, `unpaid`, `incomplete`, have no visible default payment method, have a default card expiring soon, require customer action, or have an open unpaid latest invoice. Future versions should add deeper checks for invoice retry policy.

## Checkout, Setup, And Timeline

Commands:

```bash
agent-stripe investigate checkout-session cs_...
agent-stripe investigate payment-method-readiness cus_...|pm_...
agent-stripe investigate setup seti_...|pm_...|cus_...
agent-stripe investigate timeline cus_...
```

Checkout follows the session to line items, prices/products, PaymentIntent, Charge, Subscription, Invoice, Customer, and Payment Link when present. Payment method readiness checks visible saved cards and related SetupIntents. Setup focuses on SetupIntent status and saved payment method usability. Timeline reuses customer context and emits ordered finding records for timestamped customer objects.

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
agent-stripe investigate account-health acct_...
agent-stripe investigate outgoing-payment tr_...
agent-stripe investigate outgoing-payment po_...
agent-stripe investigate outgoing-payment acct_...
agent-stripe investigate ledger ch_...|pi_...|re_...|tr_...|po_...|txn_...|fee_...
agent-stripe investigate refund re_...|ch_...|pi_...
agent-stripe investigate refund-recovery re_...
agent-stripe investigate refund-recovery trr_... --transfer tr_...
```

These distinguish Transfers, Payouts, connected Accounts, Refunds, and Transfer Reversals. For transfer reversals, Stripe nests the reversal under its parent transfer, so the command requires `--transfer` when given only a `trr_` ID.
Use `ledger` when finance/support needs amount, fee, net, and balance transaction evidence.

## Webhooks, Disputes, And Fraud

Commands:

```bash
agent-stripe investigate webhook-event evt_...
agent-stripe investigate webhook-delivery evt_... [--endpoint we_...]
agent-stripe investigate webhook-delivery we_...
agent-stripe investigate dispute-response dp_...
agent-stripe investigate dispute-impact dp_...|ch_...|cus_...
agent-stripe investigate fraud-review issfr_...|ch_...|pi_...
```

Webhook delivery uses event `pending_webhooks` and webhook endpoint configuration. It does not claim per-endpoint delivery-attempt logs when Stripe has not returned them. Fraud review follows Early Fraud Warning or Charge/PaymentIntent to charge outcome, disputes, and refunds.

## Improvements To Prioritize

The next likely common triage scenarios:

- Duplicate charge: customer, amount, card last4, and time window to identify repeated PaymentIntents/Charges.
- Unknown statement descriptor: descriptor to Charge/PaymentIntent/Customer candidates.
- Refund missing from bank: Refund -> BalanceTransaction -> Charge -> bank/card network status.
- SCA required: PaymentIntent or Invoice requiring customer action, with hosted invoice/payment links when available.
- Subscription quantity/price drift: Subscription items -> Price/Product metadata over time.
- Tax or total mismatch: Invoice line items, discounts, tax amounts, customer tax IDs, and final amount paid.
- Payment Link issue: Payment Link -> Checkout Sessions -> line items -> resulting PaymentIntent/Subscription.
- Connect refund liability: Refund with reverse transfer/application fee refund and connected account balance state.

Potential output improvements:

- `--summary-only` for high-level finding records without entity payloads.
- `--include-object <type>` / `--exclude-object <type>` for large investigations.
- Redaction policy presets for stricter customer PII contexts, building on default `@redacted` and `--expose`.
- `--limit-related N` to cap fan-out per related collection.
