package mockstripe

func earlyFraudWarnings() []map[string]any {
	return []map[string]any{
		{
			"id":             "issfr_mock_123",
			"object":         "radar.early_fraud_warning",
			"charge":         "ch_mock_succeeded",
			"payment_intent": "pi_mock_succeeded",
			"fraud_type":     "misc",
			"actionable":     true,
		},
	}
}
