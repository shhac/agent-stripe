# invoice-payment

Run `agent-stripe investigate invoice-payment in_...` for "how was this invoice paid?"

It walks Invoice -> PaymentIntent -> latest Charge and reports invoice amount paid plus card last4 when a card charge is available.
