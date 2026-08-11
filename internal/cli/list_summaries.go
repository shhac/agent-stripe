package cli

func customerListSummary(customer map[string]any) map[string]any {
	summary := map[string]any{}
	copyString(summary, customer, "id")
	copyString(summary, customer, "object")
	copyNumber(summary, customer, "created")
	copyBool(summary, customer, "delinquent")
	copyNumber(summary, customer, "balance")
	copyString(summary, customer, "currency")
	copySubset(summary, customer, "invoice_settings", "default_payment_method")
	return summary
}

func paymentMethodListSummary(pm map[string]any) map[string]any {
	summary := map[string]any{}
	copyString(summary, pm, "id")
	copyString(summary, pm, "object")
	copyString(summary, pm, "type")
	copyString(summary, pm, "customer")
	copyNumber(summary, pm, "created")
	if card, ok := pm["card"].(map[string]any); ok {
		summary["card"] = cardSummary(card)
	}
	copySubset(summary, pm, "us_bank_account", "bank_name", "last4", "account_type", "account_holder_type")
	return summary
}

func paymentIntentListSummary(pi map[string]any) map[string]any {
	summary := map[string]any{}
	copyString(summary, pi, "id")
	copyString(summary, pi, "object")
	copyNumber(summary, pi, "created")
	copyNumber(summary, pi, "amount")
	copyNumber(summary, pi, "amount_received")
	copyNumber(summary, pi, "amount_capturable")
	copyString(summary, pi, "currency")
	copyString(summary, pi, "status")
	copyString(summary, pi, "customer")
	copyString(summary, pi, "payment_method")
	copyExpandableID(summary, pi, "latest_charge")
	copyString(summary, pi, "invoice")
	copyString(summary, pi, "capture_method")
	if lastErr, ok := pi["last_payment_error"].(map[string]any); ok {
		out := map[string]any{}
		copyString(out, lastErr, "code")
		copyString(out, lastErr, "decline_code")
		copyString(out, lastErr, "type")
		copyString(out, lastErr, "payment_method_type")
		copyString(out, lastErr, "message")
		if len(out) > 0 {
			summary["last_payment_error"] = out
		}
	}
	return summary
}

func chargeListSummary(charge map[string]any) map[string]any {
	summary := map[string]any{}
	copyString(summary, charge, "id")
	copyString(summary, charge, "object")
	copyNumber(summary, charge, "created")
	copyNumber(summary, charge, "amount")
	copyNumber(summary, charge, "amount_captured")
	copyNumber(summary, charge, "amount_refunded")
	copyString(summary, charge, "currency")
	copyString(summary, charge, "status")
	copyBool(summary, charge, "paid")
	copyBool(summary, charge, "refunded")
	copyBool(summary, charge, "disputed")
	copyString(summary, charge, "customer")
	copyString(summary, charge, "payment_method")
	copyExpandableID(summary, charge, "payment_intent")
	copyExpandableID(summary, charge, "balance_transaction")
	copyString(summary, charge, "failure_code")
	copyString(summary, charge, "failure_message")
	if details, ok := charge["payment_method_details"].(map[string]any); ok {
		out := map[string]any{}
		copyString(out, details, "type")
		if card, ok := details["card"].(map[string]any); ok {
			out["card"] = cardSummary(card)
		}
		if len(out) > 0 {
			summary["payment_method_details"] = out
		}
	}
	if outcome, ok := charge["outcome"].(map[string]any); ok {
		out := map[string]any{}
		copyString(out, outcome, "type")
		copyString(out, outcome, "network_status")
		copyString(out, outcome, "risk_level")
		copyString(out, outcome, "seller_message")
		if len(out) > 0 {
			summary["outcome"] = out
		}
	}
	return summary
}

func invoiceListSummary(invoice map[string]any) map[string]any {
	summary := map[string]any{}
	copyString(summary, invoice, "id")
	copyString(summary, invoice, "object")
	copyString(summary, invoice, "number")
	copyNumber(summary, invoice, "created")
	copyString(summary, invoice, "status")
	copyBool(summary, invoice, "paid")
	copyNumber(summary, invoice, "amount_due")
	copyNumber(summary, invoice, "amount_paid")
	copyNumber(summary, invoice, "amount_remaining")
	copyString(summary, invoice, "currency")
	copyString(summary, invoice, "customer")
	copyExpandableID(summary, invoice, "subscription")
	copyExpandableID(summary, invoice, "payment_intent")
	copyNumber(summary, invoice, "attempt_count")
	copyNumber(summary, invoice, "next_payment_attempt")
	copyNumber(summary, invoice, "due_date")
	return summary
}

func subscriptionListSummary(sub map[string]any) map[string]any {
	summary := map[string]any{}
	copyString(summary, sub, "id")
	copyString(summary, sub, "object")
	copyNumber(summary, sub, "created")
	copyString(summary, sub, "status")
	copyString(summary, sub, "customer")
	copyString(summary, sub, "collection_method")
	copyExpandableID(summary, sub, "latest_invoice")
	copyExpandableID(summary, sub, "default_payment_method")
	copyNumber(summary, sub, "current_period_start")
	copyNumber(summary, sub, "current_period_end")
	copyBool(summary, sub, "cancel_at_period_end")
	copyNumber(summary, sub, "cancel_at")
	copyNumber(summary, sub, "canceled_at")
	copyNumber(summary, sub, "trial_end")
	addListDataCount(summary, sub, "items")
	return summary
}

func setupIntentListSummary(seti map[string]any) map[string]any {
	summary := map[string]any{}
	copyString(summary, seti, "id")
	copyString(summary, seti, "object")
	copyNumber(summary, seti, "created")
	copyString(summary, seti, "status")
	copyString(summary, seti, "customer")
	copyString(summary, seti, "payment_method")
	copyString(summary, seti, "usage")
	copyString(summary, seti, "cancellation_reason")
	return summary
}

func checkoutSessionListSummary(session map[string]any) map[string]any {
	summary := map[string]any{}
	copyString(summary, session, "id")
	copyString(summary, session, "object")
	copyNumber(summary, session, "created")
	copyString(summary, session, "status")
	copyString(summary, session, "mode")
	copyString(summary, session, "payment_status")
	copyString(summary, session, "customer")
	copyExpandableID(summary, session, "payment_intent")
	copyExpandableID(summary, session, "subscription")
	copyExpandableID(summary, session, "payment_link")
	copyNumber(summary, session, "amount_total")
	copyString(summary, session, "currency")
	copyNumber(summary, session, "expires_at")
	return summary
}

func paymentLinkListSummary(link map[string]any) map[string]any {
	summary := map[string]any{}
	copyString(summary, link, "id")
	copyString(summary, link, "object")
	copyBool(summary, link, "active")
	copyString(summary, link, "url")
	copyNumber(summary, link, "application_fee_amount")
	copyNumber(summary, link, "application_fee_percent")
	addListDataCount(summary, link, "line_items")
	return summary
}

func eventListSummary(event map[string]any) map[string]any {
	summary := map[string]any{}
	copyString(summary, event, "id")
	copyString(summary, event, "object")
	copyString(summary, event, "type")
	copyNumber(summary, event, "created")
	copyBool(summary, event, "livemode")
	copyString(summary, event, "api_version")
	copyString(summary, event, "account")
	if request, ok := event["request"].(map[string]any); ok {
		out := map[string]any{}
		copyString(out, request, "id")
		copyString(out, request, "idempotency_key")
		if len(out) > 0 {
			summary["request"] = out
		}
	}
	if data, ok := event["data"].(map[string]any); ok {
		if object, ok := data["object"].(map[string]any); ok {
			out := map[string]any{}
			copyString(out, object, "id")
			copyString(out, object, "object")
			copyString(out, object, "status")
			copyString(out, object, "customer")
			if len(out) > 0 {
				summary["data_object"] = out
			}
		}
	}
	return summary
}

func cardSummary(card map[string]any) map[string]any {
	out := map[string]any{}
	copyString(out, card, "brand")
	copyString(out, card, "last4")
	copyNumber(out, card, "exp_month")
	copyNumber(out, card, "exp_year")
	copyString(out, card, "funding")
	return out
}

func copyExpandableID(out, in map[string]any, key string) {
	switch value := in[key].(type) {
	case string:
		if value != "" {
			out[key] = value
		}
	case map[string]any:
		if id, ok := value["id"].(string); ok && id != "" {
			out[key] = id
		}
	}
}

func addListDataCount(out, in map[string]any, key string) {
	list, ok := in[key].(map[string]any)
	if !ok {
		return
	}
	data, ok := list["data"].([]any)
	if ok && len(data) > 0 {
		out[key+"_count"] = len(data)
	}
}
