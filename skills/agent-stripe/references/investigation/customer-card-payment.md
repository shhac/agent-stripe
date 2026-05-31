# customer-card-payment

Run `agent-stripe investigate customer-card-payment --customer cus_... --last4 4242 [--limit N]` when the customer only knows the last four card digits.

Card last4 is not unique, so the command requires a customer and scans recent charges for that customer. It reports the most recent matching charge and payment context.
