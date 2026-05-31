package api

import (
	"encoding/json"
	"time"

	"github.com/shhac/agent-stripe/internal/output"
)

func (c *Client) logDebug(method, requestURL string, status int, requestID string, body []byte) {
	entry := map[string]any{
		"@debug": "http",
		"method": method,
		"url":    requestURL,
		"status": status,
	}
	if requestID != "" {
		entry["request_id"] = requestID
	}
	var parsed any
	if json.Unmarshal(body, &parsed) == nil {
		entry["body"] = output.Redact(parsed, c.redaction)
	} else {
		entry["body_raw"] = output.RedactedString
	}
	enc := json.NewEncoder(output.Stderr())
	enc.SetEscapeHTML(false)
	_ = enc.Encode(entry)
}

func (c *Client) logRetry(method, requestURL string, status int, requestID, rateLimitedReason string, attempt, maxRetries int, delay time.Duration) {
	entry := map[string]any{
		"@debug":      "retry",
		"method":      method,
		"url":         requestURL,
		"status":      status,
		"attempt":     attempt,
		"max_retries": maxRetries,
		"delay_ms":    delay.Milliseconds(),
	}
	if requestID != "" {
		entry["request_id"] = requestID
	}
	if rateLimitedReason != "" {
		entry["rate_limited_reason"] = rateLimitedReason
	}
	enc := json.NewEncoder(output.Stderr())
	enc.SetEscapeHTML(false)
	_ = enc.Encode(entry)
}
