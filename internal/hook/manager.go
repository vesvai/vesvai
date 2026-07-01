package hook

import (
	"context"
	"sync"

	"github.com/vesvai/vesvai/internal/event"
)

var (
	defaultHooks *Hooks
	hooksOnce    sync.Once
)

func Default() *Hooks {
	hooksOnce.Do(func() {
		defaultHooks = New(nil)
	})
	return defaultHooks
}

func Init(eventBus event.EventBus) *Hooks {
	hooks := New(eventBus)
	defaultHooks = hooks
	return hooks
}

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

func (b *HookBuilder) DoAsync(fn func(ctx context.Context, args ...interface{}) error) *Callback {
	cb := b.Do(fn)
	return cb
}

func (b *HookBuilder) Filter(fn func(ctx context.Context, value interface{}, args ...interface{}) interface{}) *Callback {
	if b.once {
		return b.hooks.AddFilterOnce(b.hookName, fn, b.priority)
	}
	return b.hooks.AddFilter(b.hookName, fn, b.priority)
}

type HookManager struct {
	mu     sync.RWMutex
	hooks  map[string]*Hooks
	active *Hooks
	bus    event.EventBus
}

func NewManager(bus event.EventBus) *HookManager {
	return &HookManager{
		hooks:  make(map[string]*Hooks),
		active: New(bus),
		bus:    bus,
	}
}

func (m *HookManager) GetHooks(name string) *Hooks {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.hooks[name]
}

func (m *HookManager) RegisterNamespace(name string) *Hooks {
	m.mu.Lock()
	defer m.mu.Unlock()

	if h, ok := m.hooks[name]; ok {
		return h
	}

	h := New(m.bus)
	m.hooks[name] = h
	return h
}

func (m *HookManager) Active() *Hooks {
	return m.active
}

func (m *HookManager) RemoveNamespace(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.hooks, name)
}

type HookCollection struct {
	mu    sync.RWMutex
	hooks []*Hooks
}

func NewCollection() *HookCollection {
	return &HookCollection{
		hooks: make([]*Hooks, 0),
	}
}

func (c *HookCollection) Add(h *Hooks) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.hooks = append(c.hooks, h)
}

func (c *HookCollection) DoAction(ctx context.Context, hookName string, args ...interface{}) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, h := range c.hooks {
		if h.HasAction(hookName) {
			h.DoAction(ctx, hookName, args...)
		}
	}
}

func (c *HookCollection) ApplyFilter(ctx context.Context, hookName string, value interface{}, args ...interface{}) interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := value
	for _, h := range c.hooks {
		if h.HasFilter(hookName) {
			result = h.ApplyFilter(ctx, hookName, result, args...)
		}
	}
	return result
}
