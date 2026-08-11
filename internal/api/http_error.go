package api

import (
	"encoding/json"
	stderrors "errors"
	"fmt"
	"strings"

	agenterrors "github.com/shhac/agent-stripe/internal/errors"
)

// StripeError is attached as the cause of every classified HTTP failure so
// callers can branch on Stripe's own classification instead of matching on
// message text. Namespace fallback (v2 account -> v1 account) needs the code.
type StripeError struct {
	Status int
	Type   string
	Code   string
}

func (e *StripeError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("stripe %d %s", e.Status, e.Code)
	}
	return fmt.Sprintf("stripe %d", e.Status)
}

// ErrorCode returns Stripe's error code for a classified failure, or "" when
// the error did not come from a Stripe response.
func ErrorCode(err error) string {
	var stripeErr *StripeError
	if stderrors.As(err, &stripeErr) {
		return stripeErr.Code
	}
	return ""
}

// ErrorStatus returns the HTTP status of a classified failure, or 0.
func ErrorStatus(err error) int {
	var stripeErr *StripeError
	if stderrors.As(err, &stripeErr) {
		return stripeErr.Status
	}
	return 0
}

type stripeErrorBody struct {
	Type          string `json:"type"`
	Code          string `json:"code"`
	DeclineCode   string `json:"decline_code"`
	Message       string `json:"message"`
	Param         string `json:"param"`
	DocURL        string `json:"doc_url"`
	RequestLogURL string `json:"request_log_url"`
}

func extractErrorMessage(status int, body []byte) (string, stripeErrorBody) {
	var parsed struct {
		Error stripeErrorBody `json:"error"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		// A non-JSON body is usually a proxy or gateway page rather than Stripe.
		// Echo a bounded, single-line excerpt: enough to recognise what answered,
		// without pasting an unbounded response into the error message.
		if excerpt := errorBodyExcerpt(body); excerpt != "" {
			return fmt.Sprintf("HTTP %d: %s", status, excerpt), stripeErrorBody{}
		}
		return fmt.Sprintf("HTTP %d", status), stripeErrorBody{}
	}

	msg := parsed.Error.Message
	if msg == "" {
		msg = fmt.Sprintf("HTTP %d", status)
	}
	if parsed.Error.Param != "" {
		msg = msg + " (param: " + parsed.Error.Param + ")"
	}
	if parsed.Error.Code == "" {
		parsed.Error.Code = parsed.Error.DeclineCode
	}
	return msg, parsed.Error
}

const errorBodyExcerptLimit = 200

func errorBodyExcerpt(body []byte) string {
	excerpt := strings.TrimSpace(string(body))
	if excerpt == "" {
		return ""
	}
	excerpt = strings.Join(strings.Fields(excerpt), " ")
	if len(excerpt) > errorBodyExcerptLimit {
		excerpt = excerpt[:errorBodyExcerptLimit] + "…"
	}
	return excerpt
}

type httpErrorInput struct {
	status            int
	requestID         string
	rateLimitedReason string
	maxRetries        int
	body              []byte
	v2                bool
}

func classifyHTTPError(in httpErrorInput) *agenterrors.APIError {
	msg, parsed := extractErrorMessage(in.status, in.body)
	if in.requestID != "" {
		msg = msg + " (request_id: " + in.requestID + ")"
	}

	hintParts := []string{}
	if parsed.Code != "" {
		hintParts = append(hintParts, "Stripe code: "+parsed.Code)
	}
	if parsed.RequestLogURL != "" {
		hintParts = append(hintParts, "request log URL redacted; use request_id in Stripe Dashboard")
	}
	if in.rateLimitedReason != "" {
		hintParts = append(hintParts, "rate limit reason: "+in.rateLimitedReason)
	}
	if namespaceHint := v2NamespaceHint(in, parsed.Code); namespaceHint != "" {
		hintParts = append(hintParts, namespaceHint)
	}

	err := classifyByStatus(in, msg, parsed.Type, hintParts)
	return err.WithCause(&StripeError{Status: in.status, Type: parsed.Type, Code: parsed.Code})
}

func classifyByStatus(in httpErrorInput, msg, errType string, hintParts []string) *agenterrors.APIError {
	switch {
	case in.status == 401:
		return withHint(agenterrors.New("Authentication failed: "+msg, agenterrors.FixableByHuman),
			append(hintParts, "Check the stored profile with 'agent-stripe auth check' or re-add it with a valid restricted/secret key")...)
	case in.status == 403:
		return withHint(agenterrors.New("Permission denied: "+msg, agenterrors.FixableByHuman),
			append(hintParts, "The key may need read permissions for this Stripe resource, or the profile may need --context for an organization key")...)
	case in.status == 404:
		return withHint(agenterrors.New("Not found: "+msg, agenterrors.FixableByAgent),
			append(hintParts, "Check the ID, live/sandbox mode, and Stripe context")...)
	case in.status == 429:
		return withHint(agenterrors.New("Rate limited: "+msg, agenterrors.FixableByRetry),
			append(hintParts, retryExhaustedHint(in.maxRetries))...)
	case in.status >= 500:
		return withHint(agenterrors.New("Stripe API error: "+msg, agenterrors.FixableByRetry),
			append(hintParts, "Stripe returned a server error; retry later")...)
	default:
		fixable := agenterrors.FixableByAgent
		if errType == "authentication_error" || errType == "permission_error" {
			fixable = agenterrors.FixableByHuman
		}
		return withHint(agenterrors.New(msg, fixable), hintParts...)
	}
}

// v2NamespaceHint turns Stripe's documented Accounts v2 interop codes into the
// command an agent should run instead. Both namespaces use the acct_ prefix, so
// without this the caller only learns that "something" was wrong with the ID.
func v2NamespaceHint(in httpErrorInput, code string) string {
	switch code {
	case "v1_account_instead_of_v2_account", "account_not_yet_compatible_with_v2":
		return "this is a Connect v1 account: use 'agent-stripe accounts get <acct_id>' or 'agent-stripe investigate account-health <acct_id> --namespace v1'"
	case "accounts_v2_access_blocked", "non_connect_platform_accounts_v2_access_blocked":
		return "Accounts v2 is not enabled for this platform: use the v1 'agent-stripe accounts' commands"
	case "v1_customer_instead_of_v2_account":
		return "this is a v1 Customer ID: use 'agent-stripe customers get <cus_id>'"
	}
	if in.v2 && in.status == 400 {
		return "if this endpoint or field is preview-only, pin a preview train with --v2-api-version <version>"
	}
	return ""
}

func withHint(err *agenterrors.APIError, hints ...string) *agenterrors.APIError {
	parts := make([]string, 0, len(hints))
	for _, hint := range hints {
		if hint != "" {
			parts = append(parts, hint)
		}
	}
	if len(parts) > 0 {
		err = err.WithHint(strings.Join(parts, "; "))
	}
	return err
}
