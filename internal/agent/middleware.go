package agent

import "context"

type Middleware func(ctx context.Context, agent Agent, next MiddlewareFunc) error

type MiddlewareFunc func(ctx context.Context) error

type MiddlewareChain struct {
	middlewares []Middleware
}

func NewMiddlewareChain(mws ...Middleware) *MiddlewareChain {
	return &MiddlewareChain{
		middlewares: mws,
	}
}

func (c *MiddlewareChain) Use(mw Middleware) {
	c.middlewares = append(c.middlewares, mw)
}

func (c *MiddlewareChain) Execute(ctx context.Context, agent Agent, final func(ctx context.Context) error) error {
	if len(c.middlewares) == 0 {
		return final(ctx)
	}

	var handler MiddlewareFunc
	handler = final

	for i := len(c.middlewares) - 1; i >= 0; i-- {
		mw := c.middlewares[i]
		next := handler
		handler = func(ctx context.Context) error {
			return mw(ctx, agent, next)
		}
	}

	return handler(ctx)
}

func PreToolCallMiddleware(fn func(ctx context.Context, toolName string, params map[string]any) error) Middleware {
	return func(ctx context.Context, agent Agent, next MiddlewareFunc) error {
		return next(ctx)
	}
}

func PostToolCallMiddleware(fn func(ctx context.Context, toolName string, result string, err error) (string, error)) Middleware {
	return func(ctx context.Context, agent Agent, next MiddlewareFunc) error {
		return next(ctx)
	}
}
