package mockstripe

func transfers() []map[string]any {
	return []map[string]any{
		{
			"id":                  "tr_mock_failed",
			"object":              "transfer",
			"amount":              4200,
			"currency":            "usd",
			"destination":         "acct_mock_connected",
			"transfer_group":      "order_123",
			"reversed":            false,
			"status":              "failed",
			"failure_code":        "account_closed",
			"failure_message":     "The destination account external account is closed.",
			"balance_transaction": "txn_mock_transfer_failed",
		},
	}
}

func transferReversals(transferID string) []map[string]any {
	return []map[string]any{
		{
			"id":                          "trr_mock_failed",
			"object":                      "transfer_reversal",
			"transfer":                    transferID,
			"amount":                      4200,
			"currency":                    "usd",
			"status":                      "failed",
			"failure_balance_transaction": "txn_mock_reversal_failed",
		},
	}
}

func payouts() []map[string]any {
	return []map[string]any{
		{
			"id":                  "po_mock_failed",
			"object":              "payout",
			"amount":              125000,
			"currency":            "usd",
			"status":              "failed",
			"destination":         "ba_mock_closed",
			"failure_code":        "account_closed",
			"failure_message":     "The bank account has been closed.",
			"balance_transaction": "txn_mock_payout_failed",
		},
	}
}

func balanceTransactions() []map[string]any {
	return []map[string]any{
		{
			"id":       "txn_mock_succeeded",
			"object":   "balance_transaction",
			"amount":   4200,
			"currency": "usd",
			"type":     "charge",
			"net":      3920,
			"fee":      280,
		},
		{
			"id":       "txn_mock_payout_failed",
			"object":   "balance_transaction",
			"amount":   -125000,
			"currency": "usd",
			"type":     "payout_failure",
			"payout":   "po_mock_failed",
		},
		{
			"id":       "txn_mock_refund",
			"object":   "balance_transaction",
			"amount":   -4200,
			"currency": "usd",
			"type":     "refund",
			"net":      -4200,
			"fee":      0,
		},
		{
			"id":       "txn_mock_transfer_failed",
			"object":   "balance_transaction",
			"amount":   0,
			"currency": "usd",
			"type":     "transfer_failure",
			"net":      0,
			"fee":      0,
		},
	}
}

func applicationFees() []map[string]any {
	return []map[string]any{
		{
			"id":       "fee_mock_123",
			"object":   "application_fee",
			"amount":   420,
			"currency": "usd",
			"charge":   "ch_mock_succeeded",
			"account":  "acct_mock_connected",
		},
	}
}

func accounts() []map[string]any {
	return []map[string]any{
		{
			"id":                "acct_mock_connected",
			"object":            "account",
			"charges_enabled":   true,
			"payouts_enabled":   false,
			"details_submitted": true,
			"type":              "express",
			"requirements": map[string]any{
				"currently_due": []string{"external_account"},
			},
		},
	}
}
