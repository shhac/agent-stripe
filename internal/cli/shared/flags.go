package shared

import (
	"context"
	"time"

	agenterrors "github.com/shhac/agent-stripe/internal/errors"
	libcli "github.com/shhac/lib-agent-cli/cli"
)

type GlobalFlags struct {
	libcli.Globals // Format, TimeoutMS, Debug

	Profile      string
	Context      string
	APIKey       string
	BaseURL      string
	MaxRetries   int
	APIVersion   string
	V2APIVersion string
}

type GlobalsFunc = func() *GlobalFlags

// RequireFlag returns nil when value is present, or a structured
// fixable_by:agent error when it is empty. Callers bubble the error so the
// single sink in libcli.Run renders it once and exits non-zero.
func RequireFlag(flag, value, hint string) error {
	if value != "" {
		return nil
	}
	err := agenterrors.Newf(agenterrors.FixableByAgent, "--%s is required", flag)
	if hint != "" {
		err = err.WithHint(hint)
	}
	return err
}

func ContextWithTimeout(parent context.Context, ms int) (context.Context, context.CancelFunc) {
	if ms <= 0 {
		return parent, func() {}
	}
	return context.WithTimeout(parent, time.Duration(ms)*time.Millisecond)
}
