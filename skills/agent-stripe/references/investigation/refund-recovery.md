# refund-recovery

Run `agent-stripe investigate refund-recovery re_...|ch_...|pi_...|trr_... [--transfer tr_...]` when refund funding or connected-account recovery failed.

Use `--transfer` for transfer reversal IDs because Stripe nests reversals under their parent transfer.
