# outgoing-payment

Run `agent-stripe investigate outgoing-payment tr_...|po_...|acct_...` when money moving from you to a connected business failed.

Transfers and payouts return money-movement status/failure evidence. Account input delegates to `account-health`, which resolves the account namespace (Connect v1 or Accounts v2) itself.

Transfers, payouts, balance transactions, and application fees are `/v1` endpoints for both account models, and they accept an Accounts v2 account ID, so this workflow does not change with the namespace.
