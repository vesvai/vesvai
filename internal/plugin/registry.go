package plugin

import (
	"fmt"
	"sync"
)

type Registry struct {
	mu        sync.RWMutex
	factories map[string]PluginFactory
	plugins   map[string]Plugin
	order     []string
}

func NewRegistry() *Registry {
	return &Registry{
		factories: make(map[string]PluginFactory),
		plugins:   make(map[string]Plugin),
	}
}

func (r *Registry) Register(factory PluginFactory) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	instance := factory()
	meta := instance.Meta()

	if meta.Name == "" {
		return fmt.Errorf("plugin name is required")
	}

	if _, exists := r.factories[meta.Name]; exists {
		return fmt.Errorf("plugin %s already registered", meta.Name)
	}

	r.factories[meta.Name] = factory
	r.plugins[meta.Name] = instance
	r.order = append(r.order, meta.Name)

	return nil
}

func (r *Registry) MustRegister(factory PluginFactory) {
	if err := r.Register(factory); err != nil {
		panic(err)
	}
}

func (r *Registry) Get(name string) (Plugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.plugins[name]
	return p, ok
}

func (r *Registry) Factory(name string) (PluginFactory, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	f, ok := r.factories[name]
	return f, ok
}

func (r *Registry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, ok := r.factories[name]
	return ok
}

func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]string, len(r.order))
	copy(result, r.order)
	return result
}

func (r *Registry) All() []Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Plugin, 0, len(r.order))
	for _, name := range r.order {
		result = append(result, r.plugins[name])
	}
	return result
}

func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.factories)
}

func (r *Registry) Create(name string) (Plugin, error) {
	r.mu.RLock()
	factory, ok := r.factories[name]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("plugin %s not found", name)
	}

	return factory(), nil
}

func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.factories, name)
	delete(r.plugins, name)

	for i, n := range r.order {
		if n == name {
			r.order = append(r.order[:i], r.order[i+1:]...)
			break
		}
	}
}
