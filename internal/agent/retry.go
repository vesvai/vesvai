package agent

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net"
	"strings"
	"time"

	"github.com/vesvai/vesvai/internal/llm"
)

type RetryConfig struct {
	MaxRetries int
	BaseDelay  time.Duration
	MaxDelay   time.Duration
	Multiplier float64
	RetryOn    func(error) bool
}

func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries: 3,
		BaseDelay:  1 * time.Second,
		MaxDelay:   30 * time.Second,
		Multiplier: 2.0,
		RetryOn:    defaultRetryCondition,
	}
}

func defaultRetryCondition(err error) bool {
	if err == nil {
		return false
	}

	var providerErr *llm.ProviderError
	if errors.As(err, &providerErr) {
		return providerErr.Temporary()
	}

	errStr := err.Error()
	if strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "deadline exceeded") ||
		strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "broken pipe") ||
		strings.Contains(errStr, "EOF") {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout()
	}

	return false
}

func RetryMiddleware(config ...RetryConfig) Middleware {
	cfg := DefaultRetryConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	return func(ctx context.Context, agent Agent, next MiddlewareFunc) error {
		var lastErr error

		for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
			if attempt > 0 {
				delay := calculateDelay(cfg, attempt-1)
				select {
				case <-time.After(delay):
				case <-ctx.Done():
					return ctx.Err()
				}
			}

			lastErr = next(ctx)
			if lastErr == nil {
				return nil
			}

			if !cfg.RetryOn(lastErr) {
				return lastErr
			}

			if attempt == cfg.MaxRetries {
				break
			}
		}

		return fmt.Errorf("failed after %d retries: %w", cfg.MaxRetries, lastErr)
	}
}

func calculateDelay(cfg RetryConfig, attempt int) time.Duration {
	delay := float64(cfg.BaseDelay) * math.Pow(cfg.Multiplier, float64(attempt))
	if delay > float64(cfg.MaxDelay) {
		delay = float64(cfg.MaxDelay)
	}
	return time.Duration(delay)
}
