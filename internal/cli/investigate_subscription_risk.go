package cli

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func newInvestigateCollectionRisk(globals shared.GlobalsFunc, outputOpts *investigationOutputOptions) *cobra.Command {
	var days int
	var limit int
	cmd := &cobra.Command{
		Use:   "collection-risk",
		Short: "Find customers likely to need payment-method outreach before upcoming collection",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithInvestigator(globals(), outputOpts, func(inv investigator) ([]evidenceRecord, error) {
				params := upcomingSubscriptionParams(days, limit)
				subs, err := inv.list("/v1/subscriptions", params)
				if err != nil {
					return nil, err
				}
				records := []evidenceRecord{}
				for _, sub := range subs {
					risk := inv.collectionRisk(sub)
					if risk == "" {
						continue
					}
					records = inv.appendEvidence(records, entityRecord("subscription", sub))
					records = inv.appendEvidence(records, subscriptionRiskFinding("warning", risk, sub))
				}
				if len(records) == 0 {
					records = inv.appendEvidence(records, evidenceRecord{Type: "finding", Severity: "info", Summary: "No collection-risk subscriptions found in the inspected window."})
				}
				return records, nil
			})
		},
	}
	cmd.Flags().IntVar(&days, "days", 30, "Upcoming renewal window in days")
	cmd.Flags().IntVar(&limit, "limit", 25, "Maximum subscriptions to inspect")
	return cmd
}

func newInvestigateSubscriptionCancelRisk(globals shared.GlobalsFunc, outputOpts *investigationOutputOptions) *cobra.Command {
	var days int
	var limit int
	cmd := &cobra.Command{
		Use:   "subscription-cancel-risk",
		Short: "Find subscriptions likely to cancel, end trial, or stop billing soon",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithInvestigator(globals(), outputOpts, func(inv investigator) ([]evidenceRecord, error) {
				params := upcomingSubscriptionParams(days, limit)
				subs, err := inv.list("/v1/subscriptions", params)
				if err != nil {
					return nil, err
				}
				records := []evidenceRecord{}
				for _, sub := range subs {
					risk := subscriptionCancelRisk(sub)
					if risk == "" {
						continue
					}
					records = inv.appendEvidence(records, entityRecord("subscription", sub))
					records = inv.appendEvidence(records, subscriptionRiskFinding("warning", risk, sub))
				}
				if len(records) == 0 {
					records = inv.appendEvidence(records, evidenceRecord{Type: "finding", Severity: "info", Summary: "No subscription cancellation risks found in the inspected window."})
				}
				return records, nil
			})
		},
	}
	cmd.Flags().IntVar(&days, "days", 30, "Upcoming window in days")
	cmd.Flags().IntVar(&limit, "limit", 25, "Maximum subscriptions to inspect")
	return cmd
}

func upcomingSubscriptionParams(days, limit int) url.Values {
	params := url.Values{"status": []string{"all"}}
	shared.AddLimit(params, limit)
	if days > 0 {
		params.Set("current_period_end[lte]", strconv.FormatInt(time.Now().Add(time.Duration(days)*24*time.Hour).Unix(), 10))
	}
	return params
}

func subscriptionRiskFinding(severity, summary string, sub map[string]any) evidenceRecord {
	return evidenceRecord{
		Type:     "finding",
		Severity: severity,
		Summary:  summary,
		Data: map[string]any{
			"customer":     mapString(sub, "customer"),
			"subscription": mapString(sub, "id"),
		},
	}
}

func (i investigator) collectionRisk(sub map[string]any) string {
	return i.collectionRiskAt(sub, time.Now())
}

func (i investigator) collectionRiskAt(sub map[string]any, now time.Time) string {
	ctx := newCollectionRiskContext(sub)
	if risk := statusCollectionRisk(sub, ctx); risk != "" {
		return risk
	}
	pmID := i.subscriptionDefaultPaymentMethodID(sub, ctx.customerID)
	if pmID == "" {
		return missingPaymentMethodRisk(ctx)
	}
	if i.paymentMethodExpiresSoon(pmID, now) {
		return expiringPaymentMethodRisk(ctx, pmID)
	}
	if risk := i.latestInvoiceCollectionRisk(sub, ctx); risk != "" {
		return risk
	}
	return ""
}

func (i investigator) subscriptionDefaultPaymentMethodID(sub map[string]any, customerID string) string {
	if pmID := idFromValue(sub["default_payment_method"]); pmID != "" {
		return pmID
	}
	customer, err := i.get("/v1/customers/"+url.PathEscape(customerID), url.Values{})
	if err != nil {
		return ""
	}
	settings := mapAnyMap(customer, "invoice_settings")
	return idFromValue(settings["default_payment_method"])
}

func (i investigator) paymentMethodExpiresSoon(paymentMethodID string, now time.Time) bool {
	pm, err := i.get("/v1/payment_methods/"+url.PathEscape(paymentMethodID), url.Values{})
	return err == nil && cardExpiresSoon(pm, now)
}

func (i investigator) latestInvoiceCollectionRisk(sub map[string]any, ctx collectionRiskContext) string {
	invoiceID := idFromValue(sub["latest_invoice"])
	if invoiceID == "" {
		return ""
	}
	invoice, err := i.get("/v1/invoices/"+url.PathEscape(invoiceID), url.Values{})
	if err != nil {
		return ""
	}
	if !mapBool(invoice, "paid") && mapString(invoice, "status") == "open" {
		return openInvoiceRisk(ctx, invoiceID)
	}
	if idFromValue(invoice["payment_intent"]) == "" {
		return ""
	}
	pi, err := i.paymentIntentForInvoice(invoice)
	if err == nil && mapString(pi, "status") == "requires_action" {
		return latestInvoiceActionRisk(ctx)
	}
	return ""
}

type collectionRiskContext struct {
	customerID     string
	subscriptionID string
}

func newCollectionRiskContext(sub map[string]any) collectionRiskContext {
	return collectionRiskContext{
		customerID:     mapString(sub, "customer"),
		subscriptionID: mapString(sub, "id"),
	}
}

func statusCollectionRisk(sub map[string]any, ctx collectionRiskContext) string {
	status := mapString(sub, "status")
	if status == "past_due" || status == "unpaid" || status == "incomplete" {
		return fmt.Sprintf("Customer %s has subscription %s in %s status; outreach about payment details is recommended.", ctx.customerID, ctx.subscriptionID, status)
	}
	return ""
}

func missingPaymentMethodRisk(ctx collectionRiskContext) string {
	return fmt.Sprintf("Customer %s has subscription %s with no default payment method visible.", ctx.customerID, ctx.subscriptionID)
}

func expiringPaymentMethodRisk(ctx collectionRiskContext, pmID string) string {
	return fmt.Sprintf("Customer %s has subscription %s with card payment method %s expiring soon.", ctx.customerID, ctx.subscriptionID, pmID)
}

func openInvoiceRisk(ctx collectionRiskContext, invoiceID string) string {
	return fmt.Sprintf("Customer %s has an open unpaid invoice %s for subscription %s.", ctx.customerID, invoiceID, ctx.subscriptionID)
}

func latestInvoiceActionRisk(ctx collectionRiskContext) string {
	return fmt.Sprintf("Customer %s has subscription %s with latest invoice PaymentIntent requiring action.", ctx.customerID, ctx.subscriptionID)
}

func subscriptionCancelRisk(sub map[string]any) string {
	status := mapString(sub, "status")
	if status == "canceled" || status == "unpaid" {
		return fmt.Sprintf("Subscription %s for customer %s is %s.", mapString(sub, "id"), mapString(sub, "customer"), status)
	}
	if mapBool(sub, "cancel_at_period_end") {
		return fmt.Sprintf("Subscription %s for customer %s is set to cancel at period end %v.", mapString(sub, "id"), mapString(sub, "customer"), mapValue(sub, "current_period_end"))
	}
	if trialEnd, ok := mapInt64(sub, "trial_end"); ok && trialEnd > 0 {
		return fmt.Sprintf("Subscription %s for customer %s trial ends at %d.", mapString(sub, "id"), mapString(sub, "customer"), trialEnd)
	}
	return ""
}

func cardExpiresSoon(paymentMethod map[string]any, now time.Time) bool {
	card := mapAnyMap(paymentMethod, "card")
	year, yearOK := mapInt64(card, "exp_year")
	month, monthOK := mapInt64(card, "exp_month")
	if !yearOK || !monthOK || month < 1 || month > 12 {
		return false
	}
	expiry := time.Date(int(year), time.Month(month)+1, 1, 0, 0, 0, 0, time.UTC)
	return expiry.Before(now.AddDate(0, 2, 0))
}
