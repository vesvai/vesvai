package llm

import (
	"context"
	"fmt"
	"sync"

	"github.com/vesvai/vesvai/internal/config"
	"github.com/vesvai/vesvai/internal/hook"
)

type ProviderFactory func(cfg config.LLMConfig) (Provider, error)

type ProviderRegistry struct {
	mu        sync.RWMutex
	factories map[string]ProviderFactory
}

func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		factories: make(map[string]ProviderFactory),
	}
}

func (r *ProviderRegistry) Register(name string, factory ProviderFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.factories[name] = factory
}

func (r *ProviderRegistry) Create(cfg config.LLMConfig) (Provider, error) {
	r.mu.RLock()
	factory, ok := r.factories[cfg.Provider]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unknown provider: %s", cfg.Provider)
	}

	return factory(cfg)
}

func (r *ProviderRegistry) Supported() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.factories))
	for name := range r.factories {
		names = append(names, name)
	}
	return names
}

const HookProviderRegistry = "provider:registry"

func RegisterProviderHook(hooks *hook.Hooks, registry *ProviderRegistry) {
	hooks.AddAction(HookProviderRegistry, func(ctx context.Context, args ...interface{}) error {
		return nil
	}, 50)

	hooks.AddFilter(hook.HookProvidersCollect, func(ctx context.Context, value interface{}, args ...interface{}) interface{} {
		return registry
	}, 50)
}

func CollectProviderRegistry(hooks *hook.Hooks, ctx context.Context) *ProviderRegistry {
	registry := NewProviderRegistry()

	hooks.ApplyFilter(ctx, hook.HookProvidersCollect, registry)

	return registry
}

func CreateProviderFromConfig(registry *ProviderRegistry, cfg *config.Config) (Provider, error) {
	if len(cfg.Providers) == 0 {
		return nil, fmt.Errorf("no providers configured")
	}

	providerCfg := cfg.Providers[0]
	return registry.Create(providerCfg)
}

func ListModelsFromConfig(registry *ProviderRegistry, cfg config.LLMConfig) ([]Model, error) {
	provider, err := registry.Create(cfg)
	if err != nil {
		return nil, err
	}
	return provider.ListModels(context.Background())
}
