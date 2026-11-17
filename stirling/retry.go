package stirling

import (
	"context"
	"net/http"
	"strconv"
	"time"
)

// RetryPolicy controls automatic retries. The zero value disables retries.
type RetryPolicy struct {
	MaxRetries int
	MinWait    time.Duration
	MaxWait    time.Duration
}

// WithRetryPolicy replaces the full retry policy.
func WithRetryPolicy(policy RetryPolicy) Option {
	return func(c *Client) error {
		if policy.MaxRetries < 0 {
			policy.MaxRetries = 0
		}
		if policy.MinWait < 0 {
			policy.MinWait = 0
		}
		if policy.MaxWait < policy.MinWait {
			policy.MaxWait = policy.MinWait
		}
		c.retry = policy
		return nil
	}
}

func (p RetryPolicy) delay(attempt int, resp *http.Response) time.Duration {
	if resp != nil {
		if value := resp.Header.Get("Retry-After"); value != "" {
			if seconds, err := strconv.Atoi(value); err == nil && seconds >= 0 {
				return time.Duration(seconds) * time.Second
			}
			if when, err := http.ParseTime(value); err == nil {
				if wait := time.Until(when); wait > 0 {
					return wait
				}
			}
		}
	}
	wait := p.MinWait
	for i := 0; i < attempt; i++ {
		wait *= 2
		if p.MaxWait > 0 && wait >= p.MaxWait {
			return p.MaxWait
		}
	}
	if p.MaxWait > 0 && wait > p.MaxWait {
		return p.MaxWait
	}
	return wait
}

func sleepContext(ctx context.Context, wait time.Duration) error {
	if wait <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
