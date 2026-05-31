package api

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var retryBaseDelay = 250 * time.Millisecond

func shouldRetry(status, attempt, maxRetries int) bool {
	return status == http.StatusTooManyRequests && attempt < maxRetries
}

func retryDelay(retryAfter string, attempt int) time.Duration {
	if parsed := retryAfterDelay(retryAfter); parsed > 0 {
		return parsed
	}
	base := retryBaseDelay * time.Duration(1<<attempt)
	if base <= 0 {
		return 0
	}
	return base + randomJitter(base/2)
}

func retryAfterDelay(value string) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(value); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	if when, err := http.ParseTime(value); err == nil {
		if delay := time.Until(when); delay > 0 {
			return delay
		}
	}
	return 0
}

func sleepWithContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func retryExhaustedHint(maxRetries int) string {
	if maxRetries <= 0 {
		return "Wait and retry with a smaller --limit or narrower time range"
	}
	return fmt.Sprintf("Retried %d time(s); wait and retry with a smaller --limit, narrower time range, or fewer expansions", maxRetries)
}

func randomJitter(max time.Duration) time.Duration {
	if max <= 0 {
		return 0
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return max / 2
	}
	return time.Duration(n.Int64())
}

func nonNegative(value int) int {
	if value < 0 {
		return 0
	}
	return value
}
