package mockstripe

func paymentIntents() []map[string]any {
	return []map[string]any{
		{
			"id":             "pi_mock_succeeded",
			"object":         "payment_intent",
			"amount":         4200,
			"currency":       "usd",
			"status":         "succeeded",
			"customer":       "cus_mock_123",
			"payment_method": "pm_mock_visa",
			"latest_charge":  "ch_mock_succeeded",
			"metadata": map[string]string{
				"order_id": "order_123",
			},
		},
		{
			"id":             "pi_mock_failed",
			"object":         "payment_intent",
			"amount":         9900,
			"currency":       "usd",
			"status":         "requires_payment_method",
			"customer":       "cus_mock_456",
			"payment_method": "pm_mock_declined",
			"latest_charge":  "ch_mock_failed",
			"last_payment_error": map[string]any{
				"code":         "card_declined",
				"decline_code": "insufficient_funds",
				"message":      "Your card has insufficient funds.",
			},
			"metadata": map[string]string{
				"order_id": "order_failed",
			},
		},
		{
			"id":             "pi_mock_requires_action",
			"object":         "payment_intent",
			"amount":         5500,
			"currency":       "usd",
			"status":         "requires_action",
			"customer":       "cus_mock_expiring",
			"payment_method": "pm_mock_expiring",
			"latest_charge":  "ch_mock_requires_action",
			"metadata": map[string]string{
				"order_id": "order_action",
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
			"customer":            "cus_mock_123",
			"payment_method":      "pm_mock_visa",
			"payment_intent":      "pi_mock_succeeded",
			"balance_transaction": "txn_mock_succeeded",
			"payment_method_details": map[string]any{
				"type": "card",
				"card": map[string]any{
					"brand":     "visa",
					"last4":     "4242",
					"exp_month": 12,
					"exp_year":  2030,
				},
			},
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
			"customer":        "cus_mock_456",
			"payment_method":  "pm_mock_declined",
			"payment_intent":  "pi_mock_failed",
			"failure_code":    "card_declined",
			"failure_message": "Your card has insufficient funds.",
			"outcome": map[string]any{
				"type":           "issuer_declined",
				"risk_level":     "normal",
				"network_status": "declined_by_network",
			},
			"payment_method_details": map[string]any{
				"type": "card",
				"card": map[string]any{
					"brand":     "visa",
					"last4":     "0002",
					"exp_month": 1,
					"exp_year":  2025,
				},
			},
		},
		{
			"id":             "ch_mock_requires_action",
			"object":         "charge",
			"amount":         5500,
			"currency":       "usd",
			"status":         "failed",
			"paid":           false,
			"customer":       "cus_mock_expiring",
			"payment_method": "pm_mock_expiring",
			"payment_intent": "pi_mock_requires_action",
			"outcome": map[string]any{
				"type":           "issuer_declined",
				"network_status": "requires_action",
			},
			"payment_method_details": map[string]any{
				"type": "card",
				"card": map[string]any{
					"brand":     "visa",
					"last4":     "0341",
					"exp_month": 1,
					"exp_year":  2026,
				},
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

func refunds() []map[string]any {
	return []map[string]any{
		{
			"id":                  "re_mock_pending",
			"object":              "refund",
			"charge":              "ch_mock_succeeded",
			"payment_intent":      "pi_mock_succeeded",
			"amount":              4200,
			"currency":            "usd",
			"status":              "requires_action",
			"failure_reason":      "lost_or_stolen_card",
			"balance_transaction": "txn_mock_refund",
			"transfer":            "tr_mock_failed",
			"transfer_reversal":   "trr_mock_failed",
		},
	}
}

func setupIntents() []map[string]any {
	return []map[string]any{
		{
			"id":             "seti_mock_succeeded",
			"object":         "setup_intent",
			"customer":       "cus_mock_123",
			"payment_method": "pm_mock_visa",
			"status":         "succeeded",
			"usage":          "off_session",
		},
	}
}
