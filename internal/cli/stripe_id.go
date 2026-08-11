package cli

import (
	"fmt"
	"strings"

	agenterrors "github.com/shhac/agent-stripe/internal/errors"
)

type stripeIDKind struct {
	Kind               string
	Display            string
	Prefixes           []string
	GetCommand         string
	InvestigateCommand string
	ResolveObject      string
	APIPath            string
}

var stripeIDKinds = []stripeIDKind{
	{Kind: "transfer_reversal", Display: "transfer reversal", Prefixes: []string{"trr_"}, InvestigateCommand: "agent-stripe investigate refund-recovery --transfer <transfer-id>", ResolveObject: "transfer_reversal"},
	{Kind: "payment_intent", Display: "PaymentIntent", Prefixes: []string{"pi_"}, GetCommand: "agent-stripe payment-intents get", InvestigateCommand: "agent-stripe investigate incoming-payment", APIPath: "/v1/payment_intents"},
	{Kind: "setup_intent", Display: "SetupIntent", Prefixes: []string{"seti_"}, GetCommand: "agent-stripe setup-intents get", InvestigateCommand: "agent-stripe investigate setup", APIPath: "/v1/setup_intents"},
	{Kind: "checkout_session", Display: "Checkout Session", Prefixes: []string{"cs_"}, GetCommand: "agent-stripe checkout-sessions get", InvestigateCommand: "agent-stripe investigate checkout-session", ResolveObject: "checkout.session", APIPath: "/v1/checkout/sessions"},
	{Kind: "webhook_endpoint", Display: "webhook endpoint", Prefixes: []string{"we_"}, InvestigateCommand: "agent-stripe investigate webhook-delivery", APIPath: "/v1/webhook_endpoints"},
	{Kind: "early_fraud_warning", Display: "early fraud warning", Prefixes: []string{"issfr_"}, GetCommand: "agent-stripe early-fraud-warnings get", InvestigateCommand: "agent-stripe investigate fraud-review", APIPath: "/v1/radar/early_fraud_warnings"},
	{Kind: "balance_transaction", Display: "balance transaction", Prefixes: []string{"txn_"}, GetCommand: "agent-stripe balance-transactions get", InvestigateCommand: "agent-stripe investigate ledger", APIPath: "/v1/balance_transactions"},
	{Kind: "application_fee", Display: "application fee", Prefixes: []string{"fee_"}, GetCommand: "agent-stripe application-fees get", InvestigateCommand: "agent-stripe investigate ledger", APIPath: "/v1/application_fees"},
	{Kind: "payment_link", Display: "Payment Link", Prefixes: []string{"plink_"}, GetCommand: "agent-stripe payment-links get", APIPath: "/v1/payment_links"},
	{Kind: "payment_method", Display: "PaymentMethod", Prefixes: []string{"pm_"}, GetCommand: "agent-stripe payment-methods get", InvestigateCommand: "agent-stripe investigate payment-method-readiness", APIPath: "/v1/payment_methods"},
	{Kind: "subscription_item", Display: "subscription item", Prefixes: []string{"si_"}},
	{Kind: "customer", Display: "customer", Prefixes: []string{"cus_"}, GetCommand: "agent-stripe customers get", InvestigateCommand: "agent-stripe investigate customer-context --customer", APIPath: "/v1/customers"},
	{Kind: "invoice", Display: "invoice", Prefixes: []string{"in_"}, GetCommand: "agent-stripe invoices get", InvestigateCommand: "agent-stripe investigate invoice-payment", APIPath: "/v1/invoices"},
	{Kind: "charge", Display: "charge", Prefixes: []string{"ch_"}, GetCommand: "agent-stripe charges get", InvestigateCommand: "agent-stripe investigate incoming-payment", APIPath: "/v1/charges"},
	{Kind: "subscription", Display: "subscription", Prefixes: []string{"sub_"}, GetCommand: "agent-stripe subscriptions get", InvestigateCommand: "agent-stripe investigate subscription-renewal --subscription", APIPath: "/v1/subscriptions"},
	{Kind: "dispute", Display: "dispute", Prefixes: []string{"dp_"}, GetCommand: "agent-stripe disputes get", InvestigateCommand: "agent-stripe investigate dispute-response", APIPath: "/v1/disputes"},
	{Kind: "refund", Display: "refund", Prefixes: []string{"re_"}, GetCommand: "agent-stripe refunds get", InvestigateCommand: "agent-stripe investigate refund", APIPath: "/v1/refunds"},
	{Kind: "transfer", Display: "transfer", Prefixes: []string{"tr_"}, GetCommand: "agent-stripe transfers get", InvestigateCommand: "agent-stripe investigate outgoing-payment", APIPath: "/v1/transfers"},
	{Kind: "payout", Display: "payout", Prefixes: []string{"po_"}, GetCommand: "agent-stripe payouts get", InvestigateCommand: "agent-stripe investigate payout-failure", APIPath: "/v1/payouts"},
	{Kind: "account_person", Display: "v2 account person", Prefixes: []string{"person_"}, InvestigateCommand: "agent-stripe accounts-v2 persons get <account-id>", ResolveObject: objectV2Person},
	{Kind: "account", Display: "account", Prefixes: []string{"acct_"}, GetCommand: "agent-stripe accounts get", InvestigateCommand: "agent-stripe investigate account-health", APIPath: "/v1/accounts"},
	{Kind: "event", Display: "event", Prefixes: []string{"evt_"}, GetCommand: "agent-stripe events get", InvestigateCommand: "agent-stripe investigate webhook-event", APIPath: "/v1/events"},
	{Kind: "price", Display: "price", Prefixes: []string{"price_"}, GetCommand: "agent-stripe prices get", APIPath: "/v1/prices"},
	{Kind: "product", Display: "product", Prefixes: []string{"prod_"}, GetCommand: "agent-stripe products get", APIPath: "/v1/products"},
}

func classifyStripeID(id string) (stripeIDKind, bool) {
	for _, kind := range stripeIDKinds {
		for _, prefix := range kind.Prefixes {
			if strings.HasPrefix(id, prefix) {
				return kind, true
			}
		}
	}
	return stripeIDKind{}, false
}

func stripeIDKindByName(name string) (stripeIDKind, bool) {
	for _, kind := range stripeIDKinds {
		if kind.Kind == name {
			return kind, true
		}
	}
	return stripeIDKind{}, false
}

func (k stripeIDKind) resolvedObject() string {
	if k.ResolveObject != "" {
		return k.ResolveObject
	}
	if k.APIPath != "" {
		return k.Kind
	}
	return ""
}

func (k stripeIDKind) resolveCommandPrefix() string {
	command := k.InvestigateCommand
	if command == "" {
		command = k.GetCommand
	}
	if command == "" {
		return ""
	}
	return command + " "
}

func validateExpectedStripeID(id, expectedKind string) error {
	if expectedKind == "" {
		return nil
	}
	actual, ok := classifyStripeID(id)
	if !ok || actual.Kind == expectedKind {
		return nil
	}
	expected, ok := stripeIDKindByName(expectedKind)
	if !ok {
		return nil
	}
	return unexpectedStripeIDError(id, actual, expected)
}

func validateAllowedStripeID(id string, allowedKinds ...string) error {
	actual, ok := classifyStripeID(id)
	if !ok {
		return nil
	}
	for _, allowed := range allowedKinds {
		if actual.Kind == allowed {
			return nil
		}
	}
	allowed := make([]stripeIDKind, 0, len(allowedKinds))
	for _, name := range allowedKinds {
		if kind, ok := stripeIDKindByName(name); ok {
			allowed = append(allowed, kind)
		}
	}
	return unexpectedStripeIDForInvestigationError(id, actual, allowed)
}

func unexpectedStripeIDError(id string, actual, expected stripeIDKind) *agenterrors.APIError {
	msg := fmt.Sprintf("%s looks like a %s ID (%s), but this command expects %s ID (%s)",
		id, actual.Display, strings.Join(actual.Prefixes, " or "), expected.Display, strings.Join(expected.Prefixes, " or "))
	return agenterrors.New(msg, agenterrors.FixableByAgent).WithHint(idMismatchHint(id, actual, expected))
}

func unexpectedStripeIDForInvestigationError(id string, actual stripeIDKind, allowed []stripeIDKind) *agenterrors.APIError {
	expected := make([]string, 0, len(allowed))
	for _, kind := range allowed {
		expected = append(expected, kind.Display+" ID ("+strings.Join(kind.Prefixes, " or ")+")")
	}
	msg := fmt.Sprintf("%s looks like a %s ID (%s), but this investigation expects %s",
		id, actual.Display, strings.Join(actual.Prefixes, " or "), strings.Join(expected, ", "))
	return agenterrors.New(msg, agenterrors.FixableByAgent).WithHint(idMismatchHint(id, actual, stripeIDKind{}))
}

func idMismatchHint(id string, actual, expected stripeIDKind) string {
	hints := []string{}
	if actual.GetCommand != "" {
		hints = append(hints, "use '"+actual.GetCommand+" "+id+"'")
	}
	if actual.InvestigateCommand != "" {
		hints = append(hints, "or '"+actual.InvestigateCommand+" "+id+"'")
	}
	if expected.InvestigateCommand != "" {
		hints = append(hints, "for relationship-aware triage, try '"+expected.InvestigateCommand+" <"+expected.Kind+"-id>'")
	}
	if len(hints) == 0 {
		return "Run 'agent-stripe investigate resolve " + id + "' to identify the object and choose the next command"
	}
	hints = append(hints, "or run 'agent-stripe investigate resolve "+id+"'")
	return strings.Join(hints, "; ")
}
