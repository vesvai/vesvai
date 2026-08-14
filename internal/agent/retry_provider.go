package agent

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/vesvai/vesvai/internal/llm"
)

type RetryableProvider struct {
	inner llm.Provider
	cfg   RetryConfig
}

func NewRetryableProvider(inner llm.Provider, cfg ...RetryConfig) *RetryableProvider {
	c := DefaultRetryConfig()
	if len(cfg) > 0 {
		c = cfg[0]
	}
	return &RetryableProvider{inner: inner, cfg: c}
}

func (p *RetryableProvider) Name() string { return p.inner.Name() }

func (p *RetryableProvider) ListModels(ctx context.Context) ([]llm.Model, error) {
	return p.inner.ListModels(ctx)
}

func (p *RetryableProvider) Chat(ctx context.Context, req *llm.Request) (*llm.Response, error) {
	var lastErr error
	for attempt := 0; attempt <= p.cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := p.calculateDelay(attempt - 1)
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		resp, err := p.inner.Chat(ctx, req)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if !p.cfg.RetryOn(err) {
			return nil, err
		}
	}
	return nil, fmt.Errorf("llm chat failed after %d retries: %w", p.cfg.MaxRetries, lastErr)
}

func (p *RetryableProvider) ChatStream(ctx context.Context, req *llm.Request, handler llm.StreamHandler) error {
	var lastErr error
	delivered := false
	for attempt := 0; attempt <= p.cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := p.calculateDelay(attempt - 1)
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		err := p.inner.ChatStream(ctx, req, func(chunk llm.StreamChunk) error {
			delivered = true
			return handler(chunk)
		})
		if err == nil {
			return nil
		}
		lastErr = err
		if delivered || !p.cfg.RetryOn(err) {
			return err
		}
	}
	return fmt.Errorf("llm stream failed after %d retries: %w", p.cfg.MaxRetries, lastErr)
}

func (p *RetryableProvider) calculateDelay(attempt int) time.Duration {
	delay := float64(p.cfg.BaseDelay) * math.Pow(p.cfg.Multiplier, float64(attempt))
	if delay > float64(p.cfg.MaxDelay) {
		delay = float64(p.cfg.MaxDelay)
	}
	return time.Duration(delay)
}
