# resolve

Run `agent-stripe investigate resolve <stripe-id-or-invoice-number>` when the user gives an unknown Stripe ID, a copied invoice number, or asks what to inspect next.

It classifies known ID prefixes, retrieves the object when possible, resolves invoice numbers through invoice search, and emits a finding with suggested next investigation commands.
