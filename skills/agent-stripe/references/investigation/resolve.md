# resolve

Run `agent-stripe investigate resolve <stripe-id-or-invoice-number>` when the user gives an unknown Stripe ID, a copied invoice number, or asks what to inspect next.

It classifies known ID prefixes, retrieves the object when possible, resolves invoice numbers through invoice search, and emits a finding with suggested next investigation commands.

For an `acct_` ID it also answers a question the prefix cannot: which account namespace the ID belongs to. It reads Accounts v2 first (a v2 account ID also answers on v1 endpoints, so a v1-first probe would hide the richer model) and falls back to Connect v1, then names the namespace and the command group to use.
