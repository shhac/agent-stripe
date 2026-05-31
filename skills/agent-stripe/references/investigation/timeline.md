# timeline

Run `agent-stripe investigate timeline cus_... [--limit N]` when the user asks "what happened to this customer?"

It reuses customer context and emits chronological finding records for timestamped objects such as PaymentIntents, charges, invoices, refunds, and disputes.
