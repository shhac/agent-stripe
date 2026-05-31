# Investigation Reference Index

Use this table to choose the smallest investigation that matches the user's question. Load the linked detail file only when you need the extra path/interpretation notes.

| Name | Args | When to use | Detail |
| --- | --- | --- | --- |
| `resolve` | `<stripe-id-or-invoice-number>` | Unknown Stripe ID, invoice number, or "what command should I run?" | [resolve](investigation/resolve.md) |
| `customer-context` | `--customer cus_... [--limit N]` | Broad customer support context before choosing a narrower path. | [customer-context](investigation/customer-context.md) |
| `customer-card-payment` | `--customer cus_... --last4 4242 [--limit N]` | Customer only knows the card last4 and asks for the latest payment. | [customer-card-payment](investigation/customer-card-payment.md) |
| `webhook-event` | `<evt_...>` | Explain an event and the object embedded in it. | [webhook-event](investigation/webhook-event.md) |
| `webhook-delivery` | `<evt_...|we_...> [--endpoint we_...]` | Webhook did not arrive, endpoint may be disabled, or event has pending webhooks. | [webhook-delivery](investigation/webhook-delivery.md) |
| `dispute-response` | `<dp_...>` | Need evidence due date, dispute reason/status, and response posture. | [dispute-response](investigation/dispute-response.md) |
| `dispute-impact` | `<dp_...|ch_...|cus_...>` | Need revenue exposure and related payment/refund context. | [dispute-impact](investigation/dispute-impact.md) |
| `invoice-payment` | `<in_...>` | "How was this invoice paid, card last4, amount?" | [invoice-payment](investigation/invoice-payment.md) |
| `invoice-collection` | `<in_...|cus_...|sub_...> [--limit N]` | Open/past-due invoice, dunning/retry, next payment attempt. | [invoice-collection](investigation/invoice-collection.md) |
| `invoice-metadata` | `[in_...] [--number ABC-0001]` | Customer sent invoice copy and internal IDs live on PaymentIntent metadata. | [invoice-metadata](investigation/invoice-metadata.md) |
| `subscription-renewal` | `--subscription sub_...|--customer cus_...|--metadata key=value` | Last/next subscription payment timing and amount. | [subscription-renewal](investigation/subscription-renewal.md) |
| `subscription-items` | `--subscription sub_...` | Direct item/price/product metadata for a subscription. | [subscription-items](investigation/subscription-items.md) |
| `subscription-amount-change` | `--subscription sub_...` | Why latest/upcoming invoice amount differs. | [subscription-amount-change](investigation/subscription-amount-change.md) |
| `entitlement` | `--subscription/--customer/--metadata/--invoice/--checkout-session` | Internal product entitlement mismatch across billing surfaces. | [entitlement](investigation/entitlement.md) |
| `collection-risk` | `--days N [--limit N]` | Which upcoming subscription customers need payment-method outreach. | [collection-risk](investigation/collection-risk.md) |
| `subscription-cancel-risk` | `--days N [--limit N]` | Subscriptions ending trial, canceling, or stopping billing soon. | [subscription-cancel-risk](investigation/subscription-cancel-risk.md) |
| `incoming-payment` | `<pi_...|ch_...|in_...>` | Customer payment to you failed or needs explanation. | [incoming-payment](investigation/incoming-payment.md) |
| `checkout-session` | `<cs_...>` | Checkout completion, line items, resulting PI/subscription/invoice. | [checkout-session](investigation/checkout-session.md) |
| `payment-method-readiness` | `<cus_...|pm_...>` | Saved payment method missing, detached, expiring, or unclear. | [payment-method-readiness](investigation/payment-method-readiness.md) |
| `setup` | `<seti_...|pm_...|cus_...>` | Saving a payment method or mandate/setup flow failed or needs confirmation. | [setup](investigation/setup.md) |
| `timeline` | `<cus_...> [--limit N]` | Need chronological "what happened to this customer?" context. | [timeline](investigation/timeline.md) |
| `outgoing-payment` | `<tr_...|po_...|acct_...>` | Money from platform to connected business went wrong. | [outgoing-payment](investigation/outgoing-payment.md) |
| `account-health` | `<acct_...>` | Connected account capability/requirements blocker. | [account-health](investigation/account-health.md) |
| `ledger` | `<ch_...|pi_...|re_...|tr_...|po_...|txn_...|fee_...>` | Reconcile amount, fee, net, payout/transfer/refund ledger evidence. | [ledger](investigation/ledger.md) |
| `refund` | `<re_...|ch_...|pi_...>` | Customer-visible refund state. | [refund](investigation/refund.md) |
| `refund-recovery` | `<re_...|ch_...|pi_...|trr_...> [--transfer tr_...]` | Refund funding, reverse transfer, connected account recovery. | [refund-recovery](investigation/refund-recovery.md) |
| `payout-failure` | `<po_...>` | Payout failed and ledger/failure reason is needed. | [payout-failure](investigation/payout-failure.md) |
| `fraud-review` | `<issfr_...|ch_...|pi_...>` | Radar early fraud warning, risk outcome, disputes/refunds. | [fraud-review](investigation/fraud-review.md) |
