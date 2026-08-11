# invoice-total

Run `agent-stripe investigate invoice-total in_...` when an invoice total is not what someone expected.

It walks the same arithmetic Stripe does and names the first step that disagrees, rather than printing the fields and leaving the reader to do the sum:

1. Do the line items sum to the subtotal?
2. Does subtotal − discounts + tax equal the total?
3. Does the amount paid match the total?

A reconciling invoice is reported as `info` with the figures; a mismatch is a `warning` naming each step that failed. It also notes when tax is charged at invoice level with no per-line tax amounts, which is a common source of surprise.
