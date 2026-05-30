package mockstripe

func events() []map[string]any {
	return []map[string]any{
		{
			"id":          "evt_mock_charge_failed",
			"object":      "event",
			"type":        "charge.failed",
			"api_version": "2025-06-30.basil",
			"created":     1760000000,
			"livemode":    false,
			"request": map[string]any{
				"id":              "req_mock_decline",
				"idempotency_key": "order_123",
			},
			"data": map[string]any{
				"object": charges()[1],
			},
		},
		{
			"id":          "evt_mock_payment_succeeded",
			"object":      "event",
			"type":        "payment_intent.succeeded",
			"api_version": "2025-06-30.basil",
			"created":     1760000300,
			"livemode":    false,
			"data": map[string]any{
				"object": paymentIntents()[0],
			},
		},
	}
}

func paymentIntents() []map[string]any {
	return []map[string]any{
		{
			"id":            "pi_mock_succeeded",
			"object":        "payment_intent",
			"amount":        4200,
			"currency":      "usd",
			"status":        "succeeded",
			"customer":      "cus_mock_123",
			"latest_charge": "ch_mock_succeeded",
			"metadata": map[string]string{
				"order_id": "order_123",
			},
		},
		{
			"id":            "pi_mock_failed",
			"object":        "payment_intent",
			"amount":        9900,
			"currency":      "usd",
			"status":        "requires_payment_method",
			"customer":      "cus_mock_456",
			"latest_charge": "ch_mock_failed",
			"last_payment_error": map[string]any{
				"code":         "card_declined",
				"decline_code": "insufficient_funds",
				"message":      "Your card has insufficient funds.",
			},
			"metadata": map[string]string{
				"order_id": "order_failed",
			},
		},
	}
}

func charges() []map[string]any {
	return []map[string]any{
		{
			"id":                  "ch_mock_succeeded",
			"object":              "charge",
			"amount":              4200,
			"currency":            "usd",
			"status":              "succeeded",
			"paid":                true,
			"payment_intent":      "pi_mock_succeeded",
			"balance_transaction": "txn_mock_succeeded",
			"outcome": map[string]any{
				"type":           "authorized",
				"risk_level":     "normal",
				"seller_message": "Payment complete.",
			},
		},
		{
			"id":              "ch_mock_failed",
			"object":          "charge",
			"amount":          9900,
			"currency":        "usd",
			"status":          "failed",
			"paid":            false,
			"payment_intent":  "pi_mock_failed",
			"failure_code":    "card_declined",
			"failure_message": "Your card has insufficient funds.",
			"outcome": map[string]any{
				"type":           "issuer_declined",
				"risk_level":     "normal",
				"network_status": "declined_by_network",
			},
		},
	}
}

func disputes() []map[string]any {
	return []map[string]any{
		{
			"id":             "dp_mock_needs_response",
			"object":         "dispute",
			"charge":         "ch_mock_succeeded",
			"payment_intent": "pi_mock_succeeded",
			"amount":         4200,
			"currency":       "usd",
			"reason":         "fraudulent",
			"status":         "needs_response",
			"evidence_details": map[string]any{
				"due_by":           1760500000,
				"submission_count": 0,
			},
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
