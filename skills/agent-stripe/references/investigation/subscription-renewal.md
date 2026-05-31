# subscription-renewal

Run `agent-stripe investigate subscription-renewal --subscription sub_...|--customer cus_...|--metadata key=value` for last/next payment timing and amount.

It finds matching subscriptions, follows latest invoice and payment evidence, and uses invoice preview to estimate the next amount.
