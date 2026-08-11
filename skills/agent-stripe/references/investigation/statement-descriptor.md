# statement-descriptor

Run `agent-stripe investigate statement-descriptor --descriptor "TEXT"` when someone asks what a line on their bank statement is.

Stripe's charge search does **not** index statement descriptors, so this scans recent charges and matches `statement_descriptor`, `calculated_statement_descriptor`, and `statement_descriptor_suffix` client-side. Two consequences worth stating when you report the result:

- Matching is substring in both directions, because banks truncate the descriptor and prefix their own text. A fragment will match.
- The scan only sees the page it fetched. A miss means "not among the N most recent", not "no such charge" — the finding says so, and suggests narrowing with `--customer` or raising `--limit`.
