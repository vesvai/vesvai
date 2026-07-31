package plugin

import (
	"context"
	"fmt"
	"sync"

	"github.com/vesvai/vesvai/internal/agent"
	"github.com/vesvai/vesvai/internal/event"
	"github.com/vesvai/vesvai/internal/hook"
)

type Manager struct {
	mu           sync.RWMutex
	registry     *Registry
	loaded       map[string]Plugin
	ordered      []string
	hooks        *hook.Hooks
	eventBus     event.EventBus
	toolRegistry *agent.ToolRegistry
	config       map[string]any
}

type ManagerOption func(*Manager)

func WithHooks(h *hook.Hooks) ManagerOption {
	return func(m *Manager) {
		m.hooks = h
	}
}

func WithEventBus(bus event.EventBus) ManagerOption {
	return func(m *Manager) {
		m.eventBus = bus
	}
}

func WithToolRegistry(reg *agent.ToolRegistry) ManagerOption {
	return func(m *Manager) {
		m.toolRegistry = reg
	}
}

func WithConfig(config map[string]any) ManagerOption {
	return func(m *Manager) {
		m.config = config
	}
}

func NewManager(opts ...ManagerOption) *Manager {
	m := &Manager{
		registry: NewRegistry(),
		loaded:   make(map[string]Plugin),
		config:   make(map[string]any),
	}

	for _, opt := range opts {
		opt(m)
	}

	return m
}

func (m *Manager) Registry() *Registry {
	return m.registry
}

func (m *Manager) Load(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.loaded[name]; exists {
		return fmt.Errorf("plugin %s already loaded", name)
	}

	p, ok := m.registry.Get(name)
	if !ok {
		return fmt.Errorf("plugin %s not registered", name)
	}

	meta := p.Meta()
	for _, dep := range meta.Dependencies {
		if _, loaded := m.loaded[dep]; !loaded {
			return fmt.Errorf("plugin %s depends on %s which is not loaded", name, dep)
		}
	}

	ctx := PluginContext{
		Hooks:        m.hooks,
		EventBus:     m.eventBus,
		ToolRegistry: m.toolRegistry,
		Config:       m.config,
	}

	if err := p.Init(ctx); err != nil {
		return fmt.Errorf("failed to init plugin %s: %w", name, err)
	}

	if registrar, ok := p.(HookRegistrar); ok && m.hooks != nil {
		registrar.RegisterHooks(m.hooks)
	}

	if subscriber, ok := p.(EventSubscriber); ok && m.eventBus != nil {
		subscriber.SubscribeEvents(m.eventBus)
	}

	if provider, ok := p.(ToolProvider); ok && m.toolRegistry != nil {
		for _, t := range provider.Tools() {
			m.toolRegistry.Register(t)
		}
	}

	if err := p.Start(); err != nil {
		return fmt.Errorf("failed to start plugin %s: %w", name, err)
	}

	m.loaded[name] = p
	m.ordered = append(m.ordered, name)

	if m.eventBus != nil {
		m.eventBus.Publish(context.Background(), NewPluginEvent(
			EventPluginLoaded,
			name,
			meta.Version,
		))
	}

	return nil
}

func (m *Manager) LoadAll() error {
	for _, name := range m.registry.List() {
		if err := m.Load(name); err != nil {
			return fmt.Errorf("failed to load plugin %s: %w", name, err)
		}
	}
	return nil
}

func (m *Manager) Unload(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	p, exists := m.loaded[name]
	if !exists {
		return fmt.Errorf("plugin %s not loaded", name)
	}

	for _, loaded := range m.loaded {
		meta := loaded.Meta()
		for _, dep := range meta.Dependencies {
			if dep == name {
				return fmt.Errorf("cannot unload %s: plugin %s depends on it", name, meta.Name)
			}
		}
	}

	if err := p.Stop(); err != nil {
		return fmt.Errorf("failed to stop plugin %s: %w", name, err)
	}

	delete(m.loaded, name)

	for i, n := range m.ordered {
		if n == name {
			m.ordered = append(m.ordered[:i], m.ordered[i+1:]...)
			break
		}
	}

	if m.eventBus != nil {
		m.eventBus.Publish(context.Background(), NewPluginEvent(
			EventPluginUnloaded,
			name,
			p.Meta().Version,
		))
	}

	return nil
}

func (m *Manager) UnloadAll() error {
	m.mu.Lock()
	order := make([]string, len(m.ordered))
	copy(order, m.ordered)
	m.mu.Unlock()

	for i := len(order) - 1; i >= 0; i-- {
		if err := m.Unload(order[i]); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) Get(name string) (Plugin, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	p, ok := m.loaded[name]
	return p, ok
}

func (m *Manager) IsLoaded(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, ok := m.loaded[name]
	return ok
}

func (m *Manager) Loaded() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]string, len(m.ordered))
	copy(result, m.ordered)
	return result
}

func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.loaded)
}

func (m *Manager) Tools() []agent.Tool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var tools []agent.Tool
	for _, p := range m.loaded {
		if provider, ok := p.(ToolProvider); ok {
			tools = append(tools, provider.Tools()...)
		}
	}
	return tools
}

func (m *Manager) Agents() []agent.Agent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var agents []agent.Agent
	for _, p := range m.loaded {
		if provider, ok := p.(AgentProvider); ok {
			agents = append(agents, provider.Agents()...)
		}
	}
	return agents
}

func (m *Manager) Prompts() map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	prompts := make(map[string]string)
	for _, p := range m.loaded {
		if provider, ok := p.(PromptProvider); ok {
			for k, v := range provider.Prompts() {
				prompts[k] = v
			}
		}
	}
	return prompts
}
