# invoice-metadata

Run `agent-stripe investigate invoice-metadata in_...` or `agent-stripe investigate invoice-metadata --number ABC-0001` when internal product/order IDs live on PaymentIntent metadata.

It resolves an invoice by ID or number, follows to PaymentIntent, and emits a finding with PaymentIntent metadata.
