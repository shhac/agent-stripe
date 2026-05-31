# dispute-impact

Run `agent-stripe investigate dispute-impact dp_...|ch_...|cus_...` when the question is revenue exposure or account/customer impact.

It gathers disputes from the starting object, then follows each dispute to charge, PaymentIntent, and related refunds.
