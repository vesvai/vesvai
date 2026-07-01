package hook

import (
	"context"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/vesvai/vesvai/internal/event"
)

type HookType string

const (
	HookTypeAction HookType = "action"
	HookTypeFilter HookType = "filter"
)

type Hook func(ctx context.Context, args ...interface{}) interface{}

type Filter func(ctx context.Context, value interface{}, args ...interface{}) interface{}

type Callback struct {
	ID       string
	Hook     string
	Priority int
	Once     bool

	action ActionCallback
	filter FilterCallback

	mu       sync.RWMutex
	disabled bool
	runCount atomic.Int32
}

type ActionCallback func(ctx context.Context, args ...interface{}) error
type FilterCallback func(ctx context.Context, value interface{}, args ...interface{}) interface{}

func (c *Callback) IsAction() bool {
	return c.action != nil
}

func (c *Callback) IsFilter() bool {
	return c.filter != nil
}

func (c *Callback) Disable() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.disabled = true
}

func (c *Callback) Enable() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.disabled = false
}

func (c *Callback) IsDisabled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.disabled
}

func (c *Callback) RunCount() int {
	return int(c.runCount.Load())
}

type Hooks struct {
	mu        sync.RWMutex
	actions   map[string][]*Callback
	filters   map[string][]*Callback
	eventBus  event.EventBus
	globalCtx context.Context
	stats     *hookStats
}

type hookStats struct {
	actionsTriggered atomic.Int64
	filtersTriggered atomic.Int64
	hooksRegistered  atomic.Int64
}

func New(eventBus event.EventBus) *Hooks {
	return &Hooks{
		actions:   make(map[string][]*Callback),
		filters:   make(map[string][]*Callback),
		eventBus:  eventBus,
		globalCtx: context.Background(),
		stats:     &hookStats{},
	}
}

func (h *Hooks) AddAction(hookName string, callback ActionCallback, priority int) *Callback {
	return h.registerHook(hookName, callback, nil, priority, false)
}

func (h *Hooks) AddActionOnce(hookName string, callback ActionCallback, priority int) *Callback {
	return h.registerHook(hookName, callback, nil, priority, true)
}

func (h *Hooks) AddFilter(hookName string, callback FilterCallback, priority int) *Callback {
	return h.registerHook(hookName, nil, callback, priority, false)
}

func (h *Hooks) AddFilterOnce(hookName string, callback FilterCallback, priority int) *Callback {
	return h.registerHook(hookName, nil, callback, priority, true)
}

func (h *Hooks) registerHook(hookName string, action ActionCallback, filter FilterCallback, priority int, once bool) *Callback {
	h.mu.Lock()
	defer h.mu.Unlock()

	cb := &Callback{
		ID:       uuid.New().String(),
		Hook:     hookName,
		Priority: priority,
		Once:     once,
		action:   action,
		filter:   filter,
	}

	if action != nil {
		if _, ok := h.actions[hookName]; !ok {
			h.actions[hookName] = []*Callback{}
		}
		h.actions[hookName] = append(h.actions[hookName], cb)
		h.sortHooks(h.actions, hookName)
	}

	if filter != nil {
		if _, ok := h.filters[hookName]; !ok {
			h.filters[hookName] = []*Callback{}
		}
		h.filters[hookName] = append(h.filters[hookName], cb)
		h.sortHooks(h.filters, hookName)
	}

	h.stats.hooksRegistered.Add(1)

	if h.eventBus != nil {
		h.eventBus.Publish(h.globalCtx, event.NewSystemEvent(
			event.EventSystemConfig,
			map[string]interface{}{
				"hook_registered": hookName,
				"type":            h.getHookType(action),
			},
		))
	}

	return cb
}

func (h *Hooks) sortHooks(hooksMap map[string][]*Callback, hookName string) {
	hooks := hooksMap[hookName]
	sort.Slice(hooks, func(i, j int) bool {
		return hooks[i].Priority > hooks[j].Priority
	})
}

func (h *Hooks) getHookType(action ActionCallback) HookType {
	if action != nil {
		return HookTypeAction
	}
	return HookTypeFilter
}

func (h *Hooks) RemoveAction(hookName string, callback ActionCallback) {
	h.removeHook(hookName, func(cb *Callback) bool {
		return cb.action != nil && &cb.action == &callback
	})
}

func (h *Hooks) RemoveFilter(hookName string, callback FilterCallback) {
	h.removeHook(hookName, func(cb *Callback) bool {
		return cb.filter != nil && &cb.filter == &callback
	})
}

func (h *Hooks) RemoveCallback(id string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for hookName, cbs := range h.actions {
		for i, cb := range cbs {
			if cb.ID == id {
				h.actions[hookName] = append(cbs[:i], cbs[i+1:]...)
				return
			}
		}
	}

	for hookName, cbs := range h.filters {
		for i, cb := range cbs {
			if cb.ID == id {
				h.filters[hookName] = append(cbs[:i], cbs[i+1:]...)
				return
			}
		}
	}
}

func (h *Hooks) removeHook(hookName string, match func(*Callback) bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if cbs, ok := h.actions[hookName]; ok {
		for i, cb := range cbs {
			if match(cb) {
				h.actions[hookName] = append(cbs[:i], cbs[i+1:]...)
				return
			}
		}
	}

	if cbs, ok := h.filters[hookName]; ok {
		for i, cb := range cbs {
			if match(cb) {
				h.filters[hookName] = append(cbs[:i], cbs[i+1:]...)
				return
			}
		}
	}
}

func (h *Hooks) HasAction(hookName string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.actions[hookName]) > 0
}

func (h *Hooks) HasFilter(hookName string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.filters[hookName]) > 0
}

func (h *Hooks) DoAction(ctx context.Context, hookName string, args ...interface{}) {
	h.mu.RLock()
	callbacks := h.actions[hookName]
	h.mu.RUnlock()

	if len(callbacks) == 0 {
		return
	}

	h.stats.actionsTriggered.Add(1)

	for _, cb := range callbacks {
		if cb.IsDisabled() {
			continue
		}

		if err := cb.action(ctx, args...); err != nil {
			if h.eventBus != nil {
				h.eventBus.Publish(ctx, event.NewSystemEvent(
					event.EventSystemError,
					map[string]interface{}{
						"hook":  hookName,
						"id":    cb.ID,
						"error": err.Error(),
						"type":  "action",
					},
				))
			}
		}

		cb.runCount.Add(1)

		if cb.Once {
			h.RemoveCallback(cb.ID)
		}
	}
}

func (h *Hooks) ApplyFilter(ctx context.Context, hookName string, value interface{}, args ...interface{}) interface{} {
	h.mu.RLock()
	callbacks := h.filters[hookName]
	h.mu.RUnlock()

	if len(callbacks) == 0 {
		return value
	}

	h.stats.filtersTriggered.Add(1)

	result := value
	for _, cb := range callbacks {
		if cb.IsDisabled() {
			continue
		}

		result = cb.filter(ctx, result, args...)

		cb.runCount.Add(1)

		if cb.Once {
			h.RemoveCallback(cb.ID)
		}
	}

	return result
}

func (h *Hooks) DoActionAsync(hookName string, args ...interface{}) {
	h.mu.RLock()
	callbacks := h.actions[hookName]
	h.mu.RUnlock()

	for _, cb := range callbacks {
		go func(callback *Callback) {
			defer func() {
				if r := recover(); r != nil {
				}
			}()
			callback.action(context.Background(), args...)
		}(cb)
	}
}

func (h *Hooks) GetActions(hookName string) []*Callback {
	h.mu.RLock()
	defer h.mu.RUnlock()

	cbs := h.actions[hookName]
	result := make([]*Callback, len(cbs))
	copy(result, cbs)
	return result
}

func (h *Hooks) GetFilters(hookName string) []*Callback {
	h.mu.RLock()
	defer h.mu.RUnlock()

	cbs := h.filters[hookName]
	result := make([]*Callback, len(cbs))
	copy(result, cbs)
	return result
}

func (h *Hooks) GetAllHooks() (actions, filters map[string][]*Callback) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	actions = make(map[string][]*Callback, len(h.actions))
	for k, v := range h.actions {
		actions[k] = v
	}

	filters = make(map[string][]*Callback, len(h.filters))
	for k, v := range h.filters {
		filters[k] = v
	}

	return
}

func (h *Hooks) RemoveAll(hookName string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.actions, hookName)
	delete(h.filters, hookName)
}

func (h *Hooks) Stats() (registered, actionsTriggered, filtersTriggered int64) {
	return h.stats.hooksRegistered.Load(),
		h.stats.actionsTriggered.Load(),
		h.stats.filtersTriggered.Load()
}

func (h *Hooks) SetContext(ctx context.Context) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.globalCtx = ctx
}

func (h *Hooks) EventBus() event.EventBus {
	return h.eventBus
}
