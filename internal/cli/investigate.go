package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/api"
	"github.com/shhac/agent-stripe/internal/cli/shared"
	agenterrors "github.com/shhac/agent-stripe/internal/errors"
	"github.com/shhac/agent-stripe/internal/output"
)

type evidenceRecord struct {
	Type     string         `json:"type"`
	Object   string         `json:"object,omitempty"`
	ID       string         `json:"id,omitempty"`
	Severity string         `json:"severity,omitempty"`
	Summary  string         `json:"summary,omitempty"`
	Data     map[string]any `json:"data,omitempty"`
	Command  string         `json:"command,omitempty"`
}

type investigator struct {
	ctx    context.Context
	client *api.Client
}

func registerInvestigate(root *cobra.Command, globals shared.GlobalsFunc) {
	investigate := &cobra.Command{
		Use:     "investigate",
		Aliases: []string{"invest"},
		Short:   "Opinionated Stripe incident investigations",
	}
	investigate.AddCommand(newInvestigateCustomerCardPayment(globals))
	investigate.AddCommand(newInvestigateInvoicePayment(globals))
	investigate.AddCommand(newInvestigateSubscriptionRenewal(globals))
	investigate.AddCommand(newInvestigateInvoiceMetadata(globals))
	investigate.AddCommand(newInvestigateCollectionRisk(globals))
	investigate.AddCommand(newInvestigateIncomingPayment(globals))
	investigate.AddCommand(newInvestigateOutgoingPayment(globals))
	investigate.AddCommand(newInvestigateRefundRecovery(globals))
	root.AddCommand(investigate)
}

func newInvestigateCustomerCardPayment(globals shared.GlobalsFunc) *cobra.Command {
	var customer string
	var last4 string
	var limit int
	cmd := &cobra.Command{
		Use:   "customer-card-payment",
		Short: "Find a customer's most recent charge by card last4",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !shared.RequireFlag("customer", customer, "Last4 is not unique; provide a Customer ID to keep the investigation bounded") {
				return nil
			}
			if !shared.RequireFlag("last4", last4, "Use the final four digits the customer supplied") {
				return nil
			}
			return runInvestigation(globals(), func(ctx context.Context, client *api.Client) ([]evidenceRecord, error) {
				inv := investigator{ctx: ctx, client: client}
				params := url.Values{"customer": []string{customer}}
				api.AddLimit(params, limit)
				charges, err := inv.list("/v1/charges", params)
				if err != nil {
					return nil, err
				}
				records := []evidenceRecord{}
				for _, charge := range charges {
					if cardLast4(charge) != last4 {
						continue
					}
					records = append(records, entityRecord("charge", charge))
					records = append(records, evidenceRecord{
						Type:     "finding",
						Severity: "info",
						Summary:  fmt.Sprintf("Most recent payment for customer %s using card ending %s is charge %s for %s.", customer, last4, mapString(charge, "id"), formatAmount(charge)),
					})
					return records, nil
				}
				return append(records, evidenceRecord{
					Type:     "finding",
					Severity: "warning",
					Summary:  fmt.Sprintf("No recent charge for customer %s matched card ending %s in the first %d charges.", customer, last4, limit),
				}), nil
			})
		},
	}
	cmd.Flags().StringVar(&customer, "customer", "", "Customer ID")
	cmd.Flags().StringVar(&last4, "last4", "", "Card last4")
	cmd.Flags().IntVar(&limit, "limit", 25, "Maximum recent charges to inspect")
	return cmd
}

func newInvestigateInvoicePayment(globals shared.GlobalsFunc) *cobra.Command {
	return &cobra.Command{
		Use:   "invoice-payment <invoice-id>",
		Short: "Explain how an invoice was paid, including card last4 when available",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInvestigation(globals(), func(ctx context.Context, client *api.Client) ([]evidenceRecord, error) {
				inv := investigator{ctx: ctx, client: client}
				return inv.invoicePayment(args[0])
			})
		},
	}
}

func newInvestigateSubscriptionRenewal(globals shared.GlobalsFunc) *cobra.Command {
	var subscription string
	var metadata string
	var customer string
	var limit int
	cmd := &cobra.Command{
		Use:   "subscription-renewal",
		Short: "Summarize last and next payment for subscriptions found by ID, customer, or metadata",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInvestigation(globals(), func(ctx context.Context, client *api.Client) ([]evidenceRecord, error) {
				inv := investigator{ctx: ctx, client: client}
				subs, err := inv.findSubscriptions(subscription, customer, metadata, limit)
				if err != nil {
					return nil, err
				}
				if len(subs) == 0 {
					return []evidenceRecord{{Type: "finding", Severity: "warning", Summary: "No subscriptions matched the supplied filters."}}, nil
				}
				records := []evidenceRecord{}
				for _, sub := range subs {
					records = append(records, entityRecord("subscription", sub))
					records = append(records, inv.subscriptionPaymentSummary(sub)...)
				}
				return records, nil
			})
		},
	}
	cmd.Flags().StringVar(&subscription, "subscription", "", "Subscription ID")
	cmd.Flags().StringVar(&metadata, "metadata", "", "Metadata equality filter as key=value")
	cmd.Flags().StringVar(&customer, "customer", "", "Customer ID")
	cmd.Flags().IntVar(&limit, "limit", 5, "Maximum matching subscriptions to inspect")
	return cmd
}

func newInvestigateInvoiceMetadata(globals shared.GlobalsFunc) *cobra.Command {
	var number string
	cmd := &cobra.Command{
		Use:   "invoice-metadata [invoice-id]",
		Short: "Find PaymentIntent metadata from an invoice ID or invoice number",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInvestigation(globals(), func(ctx context.Context, client *api.Client) ([]evidenceRecord, error) {
				inv := investigator{ctx: ctx, client: client}
				invoiceID := ""
				if len(args) == 1 {
					invoiceID = args[0]
				}
				if invoiceID == "" {
					if !shared.RequireFlag("number", number, "Use --number when the customer sent an invoice number instead of an invoice ID") {
						return nil, nil
					}
					found, err := inv.list("/v1/invoices/search", url.Values{"query": []string{"number:'" + number + "'"}, "limit": []string{"1"}})
					if err != nil {
						return nil, err
					}
					if len(found) == 0 {
						return []evidenceRecord{{Type: "finding", Severity: "warning", Summary: "No invoice matched number " + number + "."}}, nil
					}
					invoiceID = mapString(found[0], "id")
				}
				invoice, err := inv.get("/v1/invoices/"+url.PathEscape(invoiceID), url.Values{})
				if err != nil {
					return nil, err
				}
				records := []evidenceRecord{entityRecord("invoice", invoice)}
				pi, err := inv.paymentIntentForInvoice(invoice)
				if err != nil {
					return nil, err
				}
				if pi == nil {
					return append(records, evidenceRecord{Type: "finding", Severity: "warning", Summary: "Invoice has no PaymentIntent."}), nil
				}
				records = append(records, entityRecord("payment_intent", pi))
				records = append(records, evidenceRecord{
					Type:     "finding",
					Severity: "info",
					Summary:  "PaymentIntent metadata is available for internal product lookup.",
					Data: map[string]any{
						"payment_intent": mapString(pi, "id"),
						"metadata":       mapAnyMap(pi, "metadata"),
					},
				})
				return records, nil
			})
		},
	}
	cmd.Flags().StringVar(&number, "number", "", "Invoice number from a customer copy")
	return cmd
}

func newInvestigateCollectionRisk(globals shared.GlobalsFunc) *cobra.Command {
	var days int
	var limit int
	cmd := &cobra.Command{
		Use:   "collection-risk",
		Short: "Find customers likely to need payment-method outreach before upcoming collection",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInvestigation(globals(), func(ctx context.Context, client *api.Client) ([]evidenceRecord, error) {
				inv := investigator{ctx: ctx, client: client}
				params := url.Values{"status": []string{"all"}}
				api.AddLimit(params, limit)
				if days > 0 {
					params.Set("current_period_end[lte]", strconv.FormatInt(time.Now().Add(time.Duration(days)*24*time.Hour).Unix(), 10))
				}
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
					records = append(records, entityRecord("subscription", sub))
					records = append(records, evidenceRecord{
						Type:     "finding",
						Severity: "warning",
						Summary:  risk,
						Data: map[string]any{
							"customer":     mapString(sub, "customer"),
							"subscription": mapString(sub, "id"),
						},
					})
				}
				if len(records) == 0 {
					records = append(records, evidenceRecord{Type: "finding", Severity: "info", Summary: "No collection-risk subscriptions found in the inspected window."})
				}
				return records, nil
			})
		},
	}
	cmd.Flags().IntVar(&days, "days", 30, "Upcoming renewal window in days")
	cmd.Flags().IntVar(&limit, "limit", 25, "Maximum subscriptions to inspect")
	return cmd
}

func newInvestigateIncomingPayment(globals shared.GlobalsFunc) *cobra.Command {
	return &cobra.Command{
		Use:   "incoming-payment <payment-intent-id|charge-id|invoice-id>",
		Short: "Explain what happened to a customer payment to you",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInvestigation(globals(), func(ctx context.Context, client *api.Client) ([]evidenceRecord, error) {
				inv := investigator{ctx: ctx, client: client}
				return inv.incomingPayment(args[0])
			})
		},
	}
}

func newInvestigateOutgoingPayment(globals shared.GlobalsFunc) *cobra.Command {
	return &cobra.Command{
		Use:   "outgoing-payment <transfer-id|payout-id|account-id>",
		Short: "Explain what happened to money moving from you to a connected business",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInvestigation(globals(), func(ctx context.Context, client *api.Client) ([]evidenceRecord, error) {
				inv := investigator{ctx: ctx, client: client}
				return inv.outgoingPayment(args[0])
			})
		},
	}
}

func newInvestigateRefundRecovery(globals shared.GlobalsFunc) *cobra.Command {
	var transfer string
	cmd := &cobra.Command{
		Use:   "refund-recovery <refund-id|charge-id|payment-intent-id|transfer-reversal-id>",
		Short: "Explain failed refund funding or Connect transfer reversal recovery",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInvestigation(globals(), func(ctx context.Context, client *api.Client) ([]evidenceRecord, error) {
				inv := investigator{ctx: ctx, client: client}
				return inv.refundRecovery(args[0], transfer)
			})
		},
	}
	cmd.Flags().StringVar(&transfer, "transfer", "", "Parent transfer ID when investigating a transfer reversal ID")
	return cmd
}

func runInvestigation(flags *shared.GlobalFlags, fn func(context.Context, *api.Client) ([]evidenceRecord, error)) error {
	return shared.WithClient(flags, func(ctx context.Context, client *api.Client) error {
		records, err := fn(ctx, client)
		if err != nil {
			return err
		}
		writeEvidence(records, flags.Format)
		return nil
	})
}

func (i investigator) get(path string, params url.Values) (map[string]any, error) {
	raw, err := i.client.Get(i.ctx, path, params)
	if err != nil {
		return nil, err
	}
	var item map[string]any
	if err := json.Unmarshal(raw, &item); err != nil {
		return nil, agenterrors.Wrap(err, agenterrors.FixableByAgent)
	}
	return item, nil
}

func (i investigator) postForm(path string, params url.Values) (map[string]any, error) {
	raw, err := i.client.PostForm(i.ctx, path, params)
	if err != nil {
		return nil, err
	}
	var item map[string]any
	if err := json.Unmarshal(raw, &item); err != nil {
		return nil, agenterrors.Wrap(err, agenterrors.FixableByAgent)
	}
	return item, nil
}

func (i investigator) list(path string, params url.Values) ([]map[string]any, error) {
	raw, err := i.client.Get(i.ctx, path, params)
	if err != nil {
		return nil, err
	}
	list, err := api.DecodeList(raw)
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(list.Data))
	for _, rawItem := range list.Data {
		var item map[string]any
		if err := json.Unmarshal(rawItem, &item); err != nil {
			return nil, agenterrors.Wrap(err, agenterrors.FixableByAgent)
		}
		items = append(items, item)
	}
	return items, nil
}

func (i investigator) invoicePayment(invoiceID string) ([]evidenceRecord, error) {
	invoice, err := i.get("/v1/invoices/"+url.PathEscape(invoiceID), url.Values{})
	if err != nil {
		return nil, err
	}
	records := []evidenceRecord{entityRecord("invoice", invoice)}
	pi, err := i.paymentIntentForInvoice(invoice)
	if err != nil {
		return nil, err
	}
	if pi == nil {
		return append(records, evidenceRecord{Type: "finding", Severity: "warning", Summary: "Invoice has no PaymentIntent, so no card details are available from a charge."}), nil
	}
	records = append(records, entityRecord("payment_intent", pi))
	charge, err := i.latestChargeForPaymentIntent(pi)
	if err != nil {
		return nil, err
	}
	if charge != nil {
		records = append(records, entityRecord("charge", charge))
	}
	records = append(records, evidenceRecord{
		Type:     "finding",
		Severity: severityForPayment(pi, charge),
		Summary:  invoicePaymentSummary(invoice, pi, charge),
	})
	return records, nil
}

func (i investigator) findSubscriptions(subscription, customer, metadata string, limit int) ([]map[string]any, error) {
	switch {
	case subscription != "":
		sub, err := i.get("/v1/subscriptions/"+url.PathEscape(subscription), url.Values{})
		if err != nil {
			return nil, err
		}
		return []map[string]any{sub}, nil
	case metadata != "":
		key, value, ok := strings.Cut(metadata, "=")
		if !ok || key == "" || value == "" {
			return nil, agenterrors.New("--metadata must be key=value", agenterrors.FixableByAgent).
				WithHint("Example: --metadata tenant_id=acme")
		}
		params := url.Values{"query": []string{"metadata['" + key + "']:'" + value + "'"}}
		api.AddLimit(params, limit)
		return i.list("/v1/subscriptions/search", params)
	default:
		params := url.Values{}
		api.AddLimit(params, limit)
		shared.AddString(params, "customer", customer)
		return i.list("/v1/subscriptions", params)
	}
}

func (i investigator) subscriptionPaymentSummary(sub map[string]any) []evidenceRecord {
	records := []evidenceRecord{}
	if latestInvoiceID := idFromValue(sub["latest_invoice"]); latestInvoiceID != "" {
		invoiceRecords, err := i.invoicePayment(latestInvoiceID)
		if err == nil {
			records = append(records, invoiceRecords...)
		} else {
			records = append(records, evidenceRecord{Type: "finding", Severity: "warning", Summary: "Could not retrieve latest invoice " + latestInvoiceID + ": " + err.Error()})
		}
	}
	nextAmount := "unknown"
	if preview, err := i.postForm("/v1/invoices/create_preview", url.Values{"subscription": []string{mapString(sub, "id")}}); err == nil {
		records = append(records, entityRecord("invoice_preview", preview))
		nextAmount = formatAmount(preview)
	}
	records = append(records, evidenceRecord{
		Type:     "finding",
		Severity: "info",
		Summary: fmt.Sprintf("Subscription %s last invoice is %s; next renewal is at %v and preview amount is %s.",
			mapString(sub, "id"), idFromValue(sub["latest_invoice"]), mapValue(sub, "current_period_end"), nextAmount),
	})
	return records
}

func (i investigator) collectionRisk(sub map[string]any) string {
	status := mapString(sub, "status")
	if status == "past_due" || status == "unpaid" || status == "incomplete" {
		return fmt.Sprintf("Customer %s has subscription %s in %s status; outreach about payment details is recommended.", mapString(sub, "customer"), mapString(sub, "id"), status)
	}
	if invoiceID := idFromValue(sub["latest_invoice"]); invoiceID != "" {
		invoice, err := i.get("/v1/invoices/"+url.PathEscape(invoiceID), url.Values{})
		if err == nil && !mapBool(invoice, "paid") && mapString(invoice, "status") == "open" {
			return fmt.Sprintf("Customer %s has an open unpaid invoice %s for subscription %s.", mapString(sub, "customer"), invoiceID, mapString(sub, "id"))
		}
	}
	return ""
}

func (i investigator) incomingPayment(id string) ([]evidenceRecord, error) {
	switch {
	case strings.HasPrefix(id, "in_"):
		return i.invoicePayment(id)
	case strings.HasPrefix(id, "ch_"):
		charge, err := i.get("/v1/charges/"+url.PathEscape(id), url.Values{})
		if err != nil {
			return nil, err
		}
		return i.paymentIncidentFromCharge(charge)
	default:
		pi, err := i.get("/v1/payment_intents/"+url.PathEscape(id), url.Values{})
		if err != nil {
			return nil, err
		}
		return i.paymentIncidentFromPI(pi)
	}
}

func (i investigator) paymentIncidentFromPI(pi map[string]any) ([]evidenceRecord, error) {
	records := []evidenceRecord{entityRecord("payment_intent", pi)}
	charge, err := i.latestChargeForPaymentIntent(pi)
	if err != nil {
		return nil, err
	}
	if charge != nil {
		records = append(records, entityRecord("charge", charge))
	}
	records = append(records, i.relatedDisputesAndRefunds(pi, charge)...)
	records = append(records, evidenceRecord{
		Type:     "finding",
		Severity: severityForPayment(pi, charge),
		Summary:  paymentFailureSummary(pi, charge),
	})
	return records, nil
}

func (i investigator) paymentIncidentFromCharge(charge map[string]any) ([]evidenceRecord, error) {
	records := []evidenceRecord{entityRecord("charge", charge)}
	if piID := idFromValue(charge["payment_intent"]); piID != "" {
		pi, err := i.get("/v1/payment_intents/"+url.PathEscape(piID), url.Values{})
		if err == nil {
			records = append(records, entityRecord("payment_intent", pi))
		}
	}
	records = append(records, i.relatedDisputesAndRefunds(nil, charge)...)
	records = append(records, evidenceRecord{
		Type:     "finding",
		Severity: severityForPayment(nil, charge),
		Summary:  paymentFailureSummary(nil, charge),
	})
	return records, nil
}

func (i investigator) relatedDisputesAndRefunds(pi, charge map[string]any) []evidenceRecord {
	records := []evidenceRecord{}
	params := url.Values{}
	if charge != nil {
		shared.AddString(params, "charge", mapString(charge, "id"))
	}
	if pi != nil {
		shared.AddString(params, "payment_intent", mapString(pi, "id"))
	}
	if len(params) == 0 {
		return records
	}
	if disputes, err := i.list("/v1/disputes", params); err == nil {
		for _, dispute := range disputes {
			records = append(records, entityRecord("dispute", dispute))
		}
	}
	if refunds, err := i.list("/v1/refunds", params); err == nil {
		for _, refund := range refunds {
			records = append(records, entityRecord("refund", refund))
		}
	}
	return records
}

func (i investigator) outgoingPayment(id string) ([]evidenceRecord, error) {
	switch {
	case strings.HasPrefix(id, "tr_"):
		transfer, err := i.get("/v1/transfers/"+url.PathEscape(id), url.Values{})
		if err != nil {
			return nil, err
		}
		return []evidenceRecord{entityRecord("transfer", transfer), moneyMovementFinding("transfer", transfer)}, nil
	case strings.HasPrefix(id, "po_"):
		payout, err := i.get("/v1/payouts/"+url.PathEscape(id), url.Values{})
		if err != nil {
			return nil, err
		}
		return []evidenceRecord{entityRecord("payout", payout), moneyMovementFinding("payout", payout)}, nil
	default:
		account, err := i.get("/v1/accounts/"+url.PathEscape(id), url.Values{})
		if err != nil {
			return nil, err
		}
		records := []evidenceRecord{entityRecord("account", account)}
		if !mapBool(account, "payouts_enabled") || !mapBool(account, "charges_enabled") {
			records = append(records, evidenceRecord{
				Type:     "finding",
				Severity: "warning",
				Summary:  fmt.Sprintf("Connected account %s is not fully enabled for charges/payouts; inspect account requirements.", mapString(account, "id")),
				Data: map[string]any{
					"requirements": mapAnyMap(account, "requirements"),
				},
			})
		}
		transfers, _ := i.list("/v1/transfers", url.Values{"destination": []string{id}, "limit": []string{"5"}})
		for _, transfer := range transfers {
			records = append(records, entityRecord("transfer", transfer))
		}
		return records, nil
	}
}

func (i investigator) refundRecovery(id, transferID string) ([]evidenceRecord, error) {
	switch {
	case strings.HasPrefix(id, "re_"):
		refund, err := i.get("/v1/refunds/"+url.PathEscape(id), url.Values{})
		if err != nil {
			return nil, err
		}
		return []evidenceRecord{entityRecord("refund", refund), moneyMovementFinding("refund", refund)}, nil
	case strings.HasPrefix(id, "trr_"):
		if transferID == "" {
			return nil, agenterrors.New("--transfer is required for transfer reversal IDs", agenterrors.FixableByAgent).
				WithHint("Stripe transfer reversals are nested under /v1/transfers/{transfer}/reversals/{reversal}")
		}
		reversal, err := i.get("/v1/transfers/"+url.PathEscape(transferID)+"/reversals/"+url.PathEscape(id), url.Values{})
		if err != nil {
			return nil, err
		}
		return []evidenceRecord{entityRecord("transfer_reversal", reversal), moneyMovementFinding("transfer_reversal", reversal)}, nil
	default:
		return i.incomingPayment(id)
	}
}

func (i investigator) paymentIntentForInvoice(invoice map[string]any) (map[string]any, error) {
	piID := idFromValue(invoice["payment_intent"])
	if piID == "" {
		return nil, nil
	}
	if pi, ok := invoice["payment_intent"].(map[string]any); ok {
		return pi, nil
	}
	return i.get("/v1/payment_intents/"+url.PathEscape(piID), url.Values{})
}

func (i investigator) latestChargeForPaymentIntent(pi map[string]any) (map[string]any, error) {
	if pi == nil {
		return nil, nil
	}
	if charge, ok := pi["latest_charge"].(map[string]any); ok {
		return charge, nil
	}
	chargeID := idFromValue(pi["latest_charge"])
	if chargeID != "" {
		return i.get("/v1/charges/"+url.PathEscape(chargeID), url.Values{})
	}
	charges, err := i.list("/v1/charges", url.Values{"payment_intent": []string{mapString(pi, "id")}, "limit": []string{"1"}})
	if err != nil || len(charges) == 0 {
		return nil, err
	}
	return charges[0], nil
}

func writeEvidence(records []evidenceRecord, format string) {
	f := output.ResolveFormat(format, output.FormatNDJSON)
	if f == output.FormatNDJSON {
		w := output.NewNDJSONWriter(os.Stdout)
		for _, record := range records {
			_ = w.WriteItem(record)
		}
		return
	}
	items := make([]any, len(records))
	for idx, record := range records {
		items[idx] = record
	}
	shared.WritePaginatedList(items, nil, format)
}

func entityRecord(object string, data map[string]any) evidenceRecord {
	return evidenceRecord{
		Type:   "entity",
		Object: object,
		ID:     mapString(data, "id"),
		Data:   data,
	}
}

func invoicePaymentSummary(invoice, pi, charge map[string]any) string {
	card := cardLast4(charge)
	cardText := "card details unavailable"
	if card != "" {
		cardText = "card ending " + card
	}
	return fmt.Sprintf("Invoice %s paid %s with %s through PaymentIntent %s.", mapString(invoice, "id"), formatAmountPaid(invoice), cardText, mapString(pi, "id"))
}

func paymentFailureSummary(pi, charge map[string]any) string {
	if charge != nil {
		if message := mapString(charge, "failure_message"); message != "" {
			return fmt.Sprintf("Charge %s failed: %s", mapString(charge, "id"), message)
		}
		if outcome := mapAnyMap(charge, "outcome"); len(outcome) > 0 {
			return fmt.Sprintf("Charge %s status is %s; outcome=%v.", mapString(charge, "id"), mapString(charge, "status"), outcome)
		}
	}
	if pi != nil {
		if lastErr := mapAnyMap(pi, "last_payment_error"); len(lastErr) > 0 {
			return fmt.Sprintf("PaymentIntent %s failed: %v.", mapString(pi, "id"), lastErr)
		}
		return fmt.Sprintf("PaymentIntent %s status is %s.", mapString(pi, "id"), mapString(pi, "status"))
	}
	return "Payment status could not be determined."
}

func moneyMovementFinding(object string, item map[string]any) evidenceRecord {
	status := mapString(item, "status")
	severity := "info"
	if status == "failed" || status == "canceled" || status == "reversed" {
		severity = "warning"
	}
	summary := fmt.Sprintf("%s %s status is %s.", object, mapString(item, "id"), status)
	if failure := firstNonEmpty(mapString(item, "failure_message"), mapString(item, "failure_code"), mapString(item, "failure_balance_transaction")); failure != "" {
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

func idFromValue(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case map[string]any:
		return mapString(v, "id")
	default:
		return ""
	}
}

func mapAnyMap(m map[string]any, key string) map[string]any {
	if m == nil {
		return nil
	}
	value, _ := m[key].(map[string]any)
	return value
}

func mapString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	value, _ := m[key].(string)
	return value
}

func mapBool(m map[string]any, key string) bool {
	if m == nil {
		return false
	}
	value, _ := m[key].(bool)
	return value
}

func mapValue(m map[string]any, key string) any {
	if m == nil {
		return nil
	}
	return m[key]
}

func mapInt64(m map[string]any, key string) (int64, bool) {
	if m == nil {
		return 0, false
	}
	switch value := m[key].(type) {
	case float64:
		return int64(value), true
	case int64:
		return value, true
	case int:
		return int64(value), true
	default:
		return 0, false
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
