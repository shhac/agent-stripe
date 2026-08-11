package cli

import (
	"fmt"
	"strings"
)

func invoicePaymentSummary(invoice, pi, charge map[string]any) string {
	card := cardLast4(charge)
	cardText := "card details unavailable"
	if card != "" {
		cardText = "card ending " + card
	}
	return fmt.Sprintf("Invoice %s paid %s with %s through PaymentIntent %s.", mapString(invoice, "id"), formatAmountPaid(invoice), cardText, mapString(pi, "id"))
}

// paymentFailureSummary names the fields it reports. Interpolating the whole
// outcome or last_payment_error map with %v pulled every nested field into the
// summary — and last_payment_error nests a full PaymentMethod, so billing
// name/email/phone and the card fingerprint rode along. Summaries are free text
// and are not run through the redaction policy, so only named scalars go in.
func paymentFailureSummary(pi, charge map[string]any) string {
	if charge != nil {
		if message := mapString(charge, "failure_message"); message != "" {
			return fmt.Sprintf("Charge %s failed: %s", mapString(charge, "id"), message)
		}
		if outcome := mapAnyMap(charge, "outcome"); len(outcome) > 0 {
			return fmt.Sprintf("Charge %s status is %s; outcome %s.",
				mapString(charge, "id"), mapString(charge, "status"), describeOutcome(outcome))
		}
	}
	if pi != nil {
		if lastErr := mapAnyMap(pi, "last_payment_error"); len(lastErr) > 0 {
			return fmt.Sprintf("PaymentIntent %s failed: %s.", mapString(pi, "id"), describePaymentError(lastErr))
		}
		return fmt.Sprintf("PaymentIntent %s status is %s.", mapString(pi, "id"), mapString(pi, "status"))
	}
	return "Payment status could not be determined."
}

func describeOutcome(outcome map[string]any) string {
	return describeFields(outcome, "type", "network_status", "risk_level", "seller_message")
}

func describePaymentError(lastErr map[string]any) string {
	return describeFields(lastErr, "code", "decline_code", "message")
}

// describeFields renders the named fields that are present, as key=value, so a
// summary never widens when Stripe adds a field to the object.
func describeFields(item map[string]any, fields ...string) string {
	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		if value := mapString(item, field); value != "" {
			parts = append(parts, field+"="+value)
		}
	}
	if len(parts) == 0 {
		return "no details reported"
	}
	return strings.Join(parts, ", ")
}

func moneyMovementFinding(object string, item map[string]any) evidenceRecord {
	status := mapString(item, "status")
	severity := "info"
	if status == "failed" || status == "canceled" || status == "reversed" {
		severity = "warning"
	}
	summary := fmt.Sprintf("%s %s status is %s.", object, mapString(item, "id"), status)
	if failure := firstNonEmpty(mapString(item, "failure_message"), mapString(item, "failure_code"), mapString(item, "failure_reason"), mapString(item, "failure_balance_transaction")); failure != "" {
		summary += " Failure detail: " + failure + "."
		severity = "warning"
	}
	return evidenceRecord{Type: "finding", Severity: severity, Summary: summary}
}

func severityForPayment(pi, charge map[string]any) string {
	if charge != nil {
		if mapString(charge, "status") == "failed" || !mapBool(charge, "paid") {
			return "warning"
		}
	}
	if pi != nil {
		status := mapString(pi, "status")
		if status != "" && status != "succeeded" {
			return "warning"
		}
	}
	return "info"
}

func cardLast4(charge map[string]any) string {
	if charge == nil {
		return ""
	}
	pmd := mapAnyMap(charge, "payment_method_details")
	card, _ := pmd["card"].(map[string]any)
	return mapString(card, "last4")
}

func formatAmount(item map[string]any) string {
	amount, ok := mapInt64(item, "amount")
	if !ok {
		amount, ok = mapInt64(item, "amount_due")
	}
	if !ok {
		return "unknown amount"
	}
	currency := strings.ToUpper(mapString(item, "currency"))
	if currency == "" {
		currency = "UNKNOWN"
	}
	return fmt.Sprintf("%d %s minor units", amount, currency)
}

func formatAmountPaid(invoice map[string]any) string {
	amount, ok := mapInt64(invoice, "amount_paid")
	if !ok {
		return formatAmount(invoice)
	}
	currency := strings.ToUpper(mapString(invoice, "currency"))
	if currency == "" {
		currency = "UNKNOWN"
	}
	return fmt.Sprintf("%d %s minor units", amount, currency)
}
