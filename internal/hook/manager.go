package hook

import (
	"context"
)

type HookBuilder struct {
	hooks    *Hooks
	hookName string
	priority int
	once     bool
	action   ActionCallback
	filter   FilterCallback
}

func (h *Hooks) On(hookName string) *HookBuilder {
	return &HookBuilder{
		hooks:    h,
		hookName: hookName,
		priority: 50,
	}
}

func (b *HookBuilder) Priority(p int) *HookBuilder {
	b.priority = p
	return b
}

func (b *HookBuilder) Once() *HookBuilder {
	b.once = true
	return b
}

func (b *HookBuilder) Do(fn func(ctx context.Context, args ...interface{}) error) *Callback {
	if b.once {
		return b.hooks.AddActionOnce(b.hookName, fn, b.priority)
	}
	return b.hooks.AddAction(b.hookName, fn, b.priority)
}

func (b *HookBuilder) Filter(fn func(ctx context.Context, value interface{}, args ...interface{}) interface{}) *Callback {
	if b.once {
		return b.hooks.AddFilterOnce(b.hookName, fn, b.priority)
	}
	return b.hooks.AddFilter(b.hookName, fn, b.priority)
}
