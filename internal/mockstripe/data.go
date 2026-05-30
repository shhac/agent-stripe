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

func subscriptions() []map[string]any {
	return []map[string]any{
		{
			"id":                   "sub_mock_active",
			"object":               "subscription",
			"customer":             "cus_mock_123",
			"status":               "active",
			"collection_method":    "charge_automatically",
			"latest_invoice":       "in_mock_paid",
			"current_period_start": 1760000000,
			"current_period_end":   1762592000,
			"cancel_at_period_end": false,
			"items": map[string]any{
				"object":   "list",
				"has_more": false,
				"data":     []map[string]any{subscriptionItems()[0]},
			},
			"metadata": map[string]string{
				"tenant_id": "acme",
			},
		},
		{
			"id":                   "sub_mock_past_due",
			"object":               "subscription",
			"customer":             "cus_mock_456",
			"status":               "past_due",
			"collection_method":    "charge_automatically",
			"latest_invoice":       "in_mock_open_failed",
			"current_period_start": 1760000000,
			"current_period_end":   1762592000,
			"cancel_at_period_end": false,
			"metadata": map[string]string{
				"tenant_id": "delinquent",
			},
		},
	}
}

func subscriptionItems() []map[string]any {
	return []map[string]any{
		{
			"id":           "si_mock_basic",
			"object":       "subscription_item",
			"subscription": "sub_mock_active",
			"quantity":     1,
			"price": map[string]any{
				"id":          "price_mock_basic",
				"object":      "price",
				"currency":    "usd",
				"unit_amount": 4200,
				"recurring": map[string]string{
					"interval": "month",
				},
				"product": "prod_mock_basic",
			},
		},
		{
			"id":           "si_mock_pro",
			"object":       "subscription_item",
			"subscription": "sub_mock_past_due",
			"quantity":     3,
			"price": map[string]any{
				"id":          "price_mock_pro",
				"object":      "price",
				"currency":    "usd",
				"unit_amount": 9900,
				"recurring": map[string]string{
					"interval": "month",
				},
				"product": "prod_mock_pro",
			},
		},
	}
}

func invoices() []map[string]any {
	return []map[string]any{
		{
			"id":             "in_mock_paid",
			"object":         "invoice",
			"subscription":   "sub_mock_active",
			"customer":       "cus_mock_123",
			"status":         "paid",
			"paid":           true,
			"amount_due":     4200,
			"amount_paid":    4200,
			"currency":       "usd",
			"number":         "MOCK-0001",
			"payment_intent": "pi_mock_succeeded",
			"attempt_count":  1,
		},
		{
			"id":                   "in_mock_open_failed",
			"object":               "invoice",
			"subscription":         "sub_mock_past_due",
			"customer":             "cus_mock_456",
			"status":               "open",
			"paid":                 false,
			"amount_due":           29700,
			"amount_paid":          0,
			"currency":             "usd",
			"number":               "MOCK-0002",
			"payment_intent":       "pi_mock_failed",
			"attempt_count":        3,
			"next_payment_attempt": 1760100000,
		},
	}
}

func customers() []map[string]any {
	return []map[string]any{
		{
			"id":     "cus_mock_123",
			"object": "customer",
			"email":  "buyer@example.com",
			"name":   "Mock Buyer",
			"invoice_settings": map[string]any{
				"default_payment_method": "pm_mock_visa",
			},
			"metadata": map[string]string{
				"tenant_id": "acme",
			},
		},
		{
			"id":     "cus_mock_456",
			"object": "customer",
			"email":  "delinquent@example.com",
			"name":   "Mock Delinquent",
			"invoice_settings": map[string]any{
				"default_payment_method": "pm_mock_declined",
			},
			"metadata": map[string]string{
				"tenant_id": "delinquent",
			},
		},
	}
}

func paymentMethods() []map[string]any {
	return []map[string]any{
		{
			"id":       "pm_mock_visa",
			"object":   "payment_method",
			"type":     "card",
			"customer": "cus_mock_123",
			"card": map[string]any{
				"brand":     "visa",
				"last4":     "4242",
				"exp_month": 12,
				"exp_year":  2030,
			},
		},
		{
			"id":       "pm_mock_declined",
			"object":   "payment_method",
			"type":     "card",
			"customer": "cus_mock_456",
			"card": map[string]any{
				"brand":     "visa",
				"last4":     "0002",
				"exp_month": 1,
				"exp_year":  2025,
			},
		},
	}
}

func invoiceLines(invoiceID string) []map[string]any {
	return []map[string]any{
		{
			"id":       "il_mock_" + invoiceID,
			"object":   "line_item",
			"invoice":  invoiceID,
			"amount":   4200,
			"currency": "usd",
			"metadata": map[string]string{
				"internal_product_id": "prod_internal_basic",
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
			"transfer_reversal":   "trr_mock_failed",
		},
	}
}

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
