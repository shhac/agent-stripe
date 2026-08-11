# refund-settlement

Run `agent-stripe investigate refund-settlement re_...` (or a charge ID) for "we refunded them and nothing has arrived".

Stripe issuing a refund and the customer's bank posting it are different events, and the gap between them is what the customer is experiencing. The finding separates them:

- `failed` / `canceled` — the money never left Stripe; it went back to your balance, with the failure reason.
- `pending` — Stripe has not finished sending it, so nothing will show on a statement yet.
- `succeeded` — Stripe has sent it; posting is up to the bank and typically takes several business days.

When Stripe has an acquirer reference, the finding surfaces it: that is the number the customer's bank can actually trace. When there is none yet, it says so, so nobody promises the customer a reference that does not exist.
