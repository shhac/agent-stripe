package api

import (
	"encoding/json"
	"fmt"
	"strings"

	agenterrors "github.com/shhac/agent-stripe/internal/errors"
)

func extractErrorMessage(status int, body []byte) (string, string, string, string) {
	var parsed struct {
		Error struct {
			Type          string `json:"type"`
			Code          string `json:"code"`
			DeclineCode   string `json:"decline_code"`
			Message       string `json:"message"`
			Param         string `json:"param"`
			DocURL        string `json:"doc_url"`
			RequestLogURL string `json:"request_log_url"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		if len(body) > 0 && len(body) <= 200 {
			return fmt.Sprintf("HTTP %d: %s", status, string(body)), "", "", ""
		}
		return fmt.Sprintf("HTTP %d", status), "", "", ""
	}

	msg := parsed.Error.Message
	if msg == "" {
		msg = fmt.Sprintf("HTTP %d", status)
	}
	if parsed.Error.Param != "" {
		msg = msg + " (param: " + parsed.Error.Param + ")"
	}
	code := parsed.Error.Code
	if code == "" {
		code = parsed.Error.DeclineCode
	}
	return msg, parsed.Error.Type, code, parsed.Error.RequestLogURL
}

func classifyHTTPError(status int, requestID, rateLimitedReason string, maxRetries int, body []byte) *agenterrors.APIError {
	msg, errType, code, requestLogURL := extractErrorMessage(status, body)
	if requestID != "" {
		msg = msg + " (request_id: " + requestID + ")"
	}

	hintParts := []string{}
	if code != "" {
		hintParts = append(hintParts, "Stripe code: "+code)
	}
	if requestLogURL != "" {
		hintParts = append(hintParts, "request log URL redacted; use request_id in Stripe Dashboard")
	}
	if rateLimitedReason != "" {
		hintParts = append(hintParts, "rate limit reason: "+rateLimitedReason)
	}

	switch {
	case status == 401:
		return withHint(agenterrors.New("Authentication failed: "+msg, agenterrors.FixableByHuman),
			append(hintParts, "Check the stored profile with 'agent-stripe auth check' or re-add it with a valid restricted/secret key")...)
	case status == 403:
		return withHint(agenterrors.New("Permission denied: "+msg, agenterrors.FixableByHuman),
			append(hintParts, "The key may need read permissions for this Stripe resource, or the profile may need --context for an organization key")...)
	case status == 404:
		return withHint(agenterrors.New("Not found: "+msg, agenterrors.FixableByAgent),
			append(hintParts, "Check the ID, live/sandbox mode, and Stripe context")...)
	case status == 429:
		return withHint(agenterrors.New("Rate limited: "+msg, agenterrors.FixableByRetry),
			append(hintParts, retryExhaustedHint(maxRetries))...)
	case status >= 500:
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
