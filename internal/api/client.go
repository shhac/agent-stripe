package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	agenterrors "github.com/shhac/agent-stripe/internal/errors"
)

const defaultBaseURL = "https://api.stripe.com"

type Client struct {
	baseURL    string
	apiKey     string
	context    string
	apiVersion string
	http       *http.Client
	debug      bool
}

type Options struct {
	APIKey     string
	Context    string
	APIVersion string
	BaseURL    string
}

func NewClient(opts Options) *Client {
	baseURL := opts.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     opts.APIKey,
		context:    opts.Context,
		apiVersion: opts.APIVersion,
		http:       &http.Client{},
	}
}

func (c *Client) SetDebug(enabled bool) {
	c.debug = enabled
}

func (c *Client) Get(ctx context.Context, path string, params url.Values) (json.RawMessage, error) {
	return c.do(ctx, http.MethodGet, buildPath(path, params), nil)
}

func (c *Client) PostForm(ctx context.Context, path string, params url.Values) (json.RawMessage, error) {
	return c.do(ctx, http.MethodPost, path, params)
}

func (c *Client) do(ctx context.Context, method, path string, form url.Values) (json.RawMessage, error) {
	req, err := c.buildRequest(ctx, method, path, form)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, agenterrors.Wrap(err, agenterrors.FixableByRetry).WithHint("Network error: check connectivity and retry")
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, agenterrors.Wrap(err, agenterrors.FixableByRetry)
	}

	if c.debug {
		c.logDebug(method, req.URL.String(), resp.StatusCode, resp.Header.Get("Request-Id"), respBody)
	}

	if resp.StatusCode >= 400 {
		return nil, classifyHTTPError(resp.StatusCode, resp.Header.Get("Request-Id"), respBody)
	}

	return json.RawMessage(respBody), nil
}

func (c *Client) buildRequest(ctx context.Context, method, path string, form url.Values) (*http.Request, error) {
	var bodyReader io.Reader
	if form != nil {
		bodyReader = strings.NewReader(form.Encode())
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, agenterrors.Wrap(err, agenterrors.FixableByAgent)
	}

	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(c.apiKey+":")))
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if c.context != "" {
		req.Header.Set("Stripe-Context", c.context)
	}
	if c.apiVersion != "" {
		req.Header.Set("Stripe-Version", c.apiVersion)
	}
	return req, nil
}

func buildPath(base string, params url.Values) string {
	if encoded := params.Encode(); encoded != "" {
		return base + "?" + encoded
	}
	return base
}

type ListResponse struct {
	Object   string            `json:"object"`
	URL      string            `json:"url,omitempty"`
	HasMore  bool              `json:"has_more"`
	Data     []json.RawMessage `json:"data"`
	NextPage string            `json:"next_page,omitempty"`
}

func DecodeList(raw json.RawMessage) (*ListResponse, error) {
	var list ListResponse
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil, agenterrors.Wrap(err, agenterrors.FixableByAgent)
	}
	return &list, nil
}

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

func classifyHTTPError(status int, requestID string, body []byte) *agenterrors.APIError {
	msg, errType, code, requestLogURL := extractErrorMessage(status, body)
	if requestID != "" {
		msg = msg + " (request_id: " + requestID + ")"
	}

	hintParts := []string{}
	if code != "" {
		hintParts = append(hintParts, "Stripe code: "+code)
	}
	if requestLogURL != "" {
		hintParts = append(hintParts, "request log: "+requestLogURL)
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
			append(hintParts, "Wait and retry with a smaller --limit or narrower time range")...)
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
		err.WithHint(strings.Join(parts, "; "))
	}
	return err
}

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
		entry["body"] = parsed
	} else {
		entry["body_raw"] = string(body)
	}
	enc := json.NewEncoder(os.Stderr)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(entry)
}
