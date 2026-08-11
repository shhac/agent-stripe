# duplicate-charge

Run `agent-stripe investigate duplicate-charge --customer cus_...` when a customer believes they were charged twice.

Charges are grouped by amount, currency, and card last4, then clustered by time. All three parts matter:

- **Grouping alone is not enough.** Two legitimate monthly charges of the same amount would look identical; `--window-hours` (default 24) is what makes "duplicate" mean something.
- **The card is part of the key.** The same amount on two different cards is a customer retrying after a decline, not a double charge.
- `--customer` is required, because neither amount nor last4 is unique on its own.

Failed charges are excluded — a decline followed by a successful retry is not a duplicate.

Every inspected charge is emitted as evidence, as in the other scans; the claim is the finding, whose `charges` array lists exactly the cluster. The finding also counts how many of the cluster are already refunded, and points at `investigate refund` for the most recent one.
