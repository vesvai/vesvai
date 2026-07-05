package plugin

import (
	"context"
	"fmt"
	"sync"

	"github.com/vesvai/vesvai/internal/agent"
	"github.com/vesvai/vesvai/internal/event"
	"github.com/vesvai/vesvai/internal/hook"
)

type Base struct {
	mu      sync.RWMutex
	meta    PluginMeta
	state   PluginState
	ctx     PluginContext
	initFn  func(PluginContext) error
	startFn func() error
	stopFn  func() error
	err     error
}

func NewBase(meta PluginMeta) *Base {
	return &Base{
		meta:  meta,
		state: StateRegistered,
	}
}

func (b *Base) Meta() PluginMeta {
	return b.meta
}

func (b *Base) State() PluginState {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.state
}

func (b *Base) Err() error {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.err
}

func (b *Base) Context() PluginContext {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.ctx
}

func (b *Base) SetInitFn(fn func(PluginContext) error) {
	b.initFn = fn
}

func (b *Base) SetStartFn(fn func() error) {
	b.startFn = fn
}

func (b *Base) SetStopFn(fn func() error) {
	b.stopFn = fn
}

func (b *Base) Init(ctx PluginContext) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.state != StateRegistered {
		return fmt.Errorf("plugin %s cannot init in state %s", b.meta.Name, b.state)
	}

	b.ctx = ctx

	if b.initFn != nil {
		if err := b.initFn(ctx); err != nil {
			b.state = StateError
			b.err = err
			return fmt.Errorf("plugin %s init failed: %w", b.meta.Name, err)
		}
	}

	b.state = StateInitialized
	return nil
}

func (b *Base) Start() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.state != StateInitialized {
		return fmt.Errorf("plugin %s cannot start in state %s", b.meta.Name, b.state)
	}

	if b.startFn != nil {
		if err := b.startFn(); err != nil {
			b.state = StateError
			b.err = err
			return fmt.Errorf("plugin %s start failed: %w", b.meta.Name, err)
		}
	}

	b.state = StateRunning
	return nil
}

func (b *Base) Stop() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.state != StateRunning {
		return fmt.Errorf("plugin %s cannot stop in state %s", b.meta.Name, b.state)
	}

	if b.stopFn != nil {
		if err := b.stopFn(); err != nil {
			b.state = StateError
			b.err = err
			return fmt.Errorf("plugin %s stop failed: %w", b.meta.Name, err)
		}
	}

	b.state = StateStopped
	return nil
}

func (b *Base) RegisterTool(t agent.Tool) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.ctx.ToolRegistry == nil {
		return fmt.Errorf("tool registry not available")
	}
	b.ctx.ToolRegistry.Register(t)
	return nil
}

func (b *Base) RegisterTools(tools ...agent.Tool) error {
	for _, t := range tools {
		if err := b.RegisterTool(t); err != nil {
			return err
		}
	}
	return nil
}

func (b *Base) AddAction(hookName string, callback hook.ActionCallback, priority int) *hook.Callback {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.ctx.Hooks == nil {
		return nil
	}
	return b.ctx.Hooks.AddAction(hookName, callback, priority)
}

func (b *Base) AddFilter(hookName string, callback hook.FilterCallback, priority int) *hook.Callback {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.ctx.Hooks == nil {
		return nil
	}
	return b.ctx.Hooks.AddFilter(hookName, callback, priority)
}

func (b *Base) PublishEvent(ctx context.Context, eventType event.EventType, data any) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.ctx.EventBus == nil {
		return nil
	}
	evt := &event.SystemEvent{
		BaseEvent: event.NewBaseEvent(eventType),
		Data:      data,
	}
	return b.ctx.EventBus.Publish(ctx, evt)
}
