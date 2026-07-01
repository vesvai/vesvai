package llm

import "context"

type ProviderError struct {
	StatusCode int
	Message    string
	Body       string
}

func (e *ProviderError) Error() string {
	return e.Message
}

func (e *ProviderError) Temporary() bool {
	return e.StatusCode == 429 || e.StatusCode >= 500
}

func (e *ProviderError) RateLimited() bool {
	return e.StatusCode == 429
}

type StreamHandler func(chunk StreamChunk) error

type Provider interface {
	Name() string
	Chat(ctx context.Context, req *Request) (*Response, error)
	ChatStream(ctx context.Context, req *Request, handler StreamHandler) error
}
