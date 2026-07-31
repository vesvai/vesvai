package plugin

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/vesvai/vesvai/internal/agent"
	"github.com/vesvai/vesvai/internal/config"
	"github.com/vesvai/vesvai/internal/event"
	"github.com/vesvai/vesvai/internal/hook"
	"github.com/vesvai/vesvai/internal/lifecycle"
)

type LifecycleManager struct {
	mu       sync.RWMutex
	mgr      *Manager
	registry *Registry
	lc       *lifecycle.Lifecycle
	eventBus event.EventBus
	hooks    *hook.Hooks
}

func NewLifecycleManager(lc *lifecycle.Lifecycle, bus event.EventBus, hooks *hook.Hooks) *LifecycleManager {
	return &LifecycleManager{
		registry: NewRegistry(),
		lc:       lc,
		eventBus: bus,
		hooks:    hooks,
	}
}

func (m *LifecycleManager) RegisterHooks() {
	m.lc.On(lifecycle.HookCreate).Priority(60).Do(m.onCreate)
	m.lc.On(lifecycle.HookMount).Priority(60).Do(m.onMount)
	m.lc.On(lifecycle.HookUnmount).Priority(60).Do(m.onUnmount)
	m.lc.On(lifecycle.HookDelete).Priority(60).Do(m.onDelete)
}

func (m *LifecycleManager) onCreate(ctx context.Context, args ...interface{}) error {
	pluginsDir, err := config.GetPluginsDir()
	if err != nil {
		return fmt.Errorf("failed to get plugins dir: %w", err)
	}

	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		return fmt.Errorf("failed to create plugins directory: %w", err)
	}

	loader := NewDiskLoader(pluginsDir)
	plugins, err := loader.Load()
	if err != nil {
		return fmt.Errorf("failed to load plugins from disk: %w", err)
	}

	for _, p := range plugins {
		if err := m.registry.Register(func() Plugin { return p }); err != nil {
			fmt.Printf("Plugin: failed to register %s: %v\n", p.Meta().Name, err)
		}
	}

	m.lc.SetComponentPhase("plugin", lifecycle.PhaseCreated)
	return nil
}

func (m *LifecycleManager) onMount(ctx context.Context, args ...interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.mgr == nil {
		opts := []ManagerOption{}
		if m.hooks != nil {
			opts = append(opts, WithHooks(m.hooks))
		}
		if m.eventBus != nil {
			opts = append(opts, WithEventBus(m.eventBus))
		}
		m.mgr = NewManager(opts...)
		m.mgr.registry = m.registry
	}

	if err := m.mgr.LoadAll(); err != nil {
		return fmt.Errorf("failed to load all plugins: %w", err)
	}

	m.lc.SetComponentPhase("plugin", lifecycle.PhaseMounted)
	return nil
}

func (m *LifecycleManager) onUnmount(ctx context.Context, args ...interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.mgr != nil {
		if err := m.mgr.UnloadAll(); err != nil {
			fmt.Printf("Plugin: failed to unload all: %v\n", err)
		}
	}

	m.lc.SetComponentPhase("plugin", lifecycle.PhaseUnmounted)
	return nil
}

func (m *LifecycleManager) onDelete(ctx context.Context, args ...interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.mgr = nil
	m.registry = NewRegistry()
	m.lc.SetComponentPhase("plugin", lifecycle.PhaseDeleted)
	return nil
}

func (m *LifecycleManager) Registry() *Registry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.registry
}

func (m *LifecycleManager) Manager() *Manager {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.mgr
}

func (m *LifecycleManager) Tools() []agent.Tool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.mgr == nil {
		return nil
	}
	return m.mgr.Tools()
}
