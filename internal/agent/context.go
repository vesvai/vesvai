package agent

import "context"

type agentCtxKey struct{}

func WithAgent(ctx context.Context, a *Agent) context.Context {
	if a == nil {
		return ctx
	}
	return context.WithValue(ctx, agentCtxKey{}, a)
}

func FromContext(ctx context.Context) *Agent {
	a, _ := ctx.Value(agentCtxKey{}).(*Agent)
	return a
}
