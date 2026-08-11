# action-required

Run `agent-stripe investigate action-required` for payments that are stalled rather than failed — the customer still has to complete SCA, 3-D Secure, or a bank authorization.

It finds PaymentIntents in `requires_action`, `requires_confirmation`, or `requires_source_action`, follows each to its invoice, and reports where the customer can finish.

The completion URLs (`hosted_invoice_url`) are redacted by the default policy. The finding reports that a hosted page exists and names the flag to reveal it rather than putting the URL in a summary, because summaries are not redacted.
