package bootstrap

import (
	"context"
	"fmt"
	"os"

	"github.com/vesvai/vesvai/internal/agent"
	"github.com/vesvai/vesvai/internal/agents"
	"github.com/vesvai/vesvai/internal/config"
	"github.com/vesvai/vesvai/internal/event"
	"github.com/vesvai/vesvai/internal/filesystem"
	"github.com/vesvai/vesvai/internal/hook"
	"github.com/vesvai/vesvai/internal/lifecycle"
	"github.com/vesvai/vesvai/internal/llm"
	"github.com/vesvai/vesvai/internal/llm/providers/openai"
	"github.com/vesvai/vesvai/internal/llm/providers/openrouter"
	"github.com/vesvai/vesvai/internal/mcp"
	"github.com/vesvai/vesvai/internal/memory"
	"github.com/vesvai/vesvai/internal/permission"
	"github.com/vesvai/vesvai/internal/plugin"
	"github.com/vesvai/vesvai/internal/prompt"
	"github.com/vesvai/vesvai/internal/skill"
	"github.com/vesvai/vesvai/internal/tools"
)

type App struct {
	Hooks         *hook.Hooks
	EventBus      event.EventBus
	Lifecycle     *lifecycle.Lifecycle
	ToolRegistry  *agent.ToolRegistry
	AgentRegistry *agents.Registry
	Config        *config.Config
	Providers     *llm.ProviderRegistry
	FileSystem    *filesystem.FileSystem
	Permissions   *permission.Manager

	mcpManager      *mcp.Manager
	memoryManager   *memory.Manager
	pluginLifecycle *plugin.LifecycleManager
	skillsManager   *skill.Manager
}

func New(cfg *config.Config) *App {
	eventBus := event.NewEventBus()
	hooks := hook.New(eventBus)
	lifecycleMgr := lifecycle.New(hooks)
	toolRegistry := agent.NewToolRegistry()
	providerRegistry := llm.NewProviderRegistry()

	openrouter.RegisterHooks(hooks)
	openai.RegisterHooks(hooks)

	return &App{
		Hooks:        hooks,
		EventBus:     eventBus,
		Lifecycle:    lifecycleMgr,
		ToolRegistry: toolRegistry,
		Config:       cfg,
		Providers:    providerRegistry,
	}
}

func (a *App) Init(ctx context.Context) error {
	cwd, _ := os.Getwd()

	fs, err := filesystem.New(filesystem.Config{
		RootDir:        cwd,
		IgnoreDotfiles: true,
	})
	if err != nil {
		return fmt.Errorf("failed to create filesystem: %w", err)
	}
	a.FileSystem = fs

	a.Permissions = permission.NewManager(cwd)

	prompt.SetHooks(a.Hooks)
	if sm, err := skill.NewManager(cwd, fs); err == nil {
		a.skillsManager = sm
		skill.RegisterHooks(a.Hooks, sm)
	}

	a.registerCoreHooks(fs)

	a.mcpManager = mcp.NewManager(a.Lifecycle, a.Config, a.Hooks, cwd, fs)
	a.mcpManager.RegisterHooks()

	a.memoryManager = memory.NewManager(a.Lifecycle, a.Hooks, fs)
	a.memoryManager.RegisterHooks()

	a.pluginLifecycle = plugin.NewLifecycleManager(a.Lifecycle, a.EventBus, a.Hooks)
	a.pluginLifecycle.RegisterHooks()

	if err := a.Lifecycle.Create(ctx); err != nil {
		return fmt.Errorf("lifecycle create failed: %w", err)
	}

	if err := a.Lifecycle.Mount(ctx); err != nil {
		return fmt.Errorf("lifecycle mount failed: %w", err)
	}

	a.collectTools(ctx)
	a.collectAgents(ctx)
	a.collectProviders(ctx)

	return nil
}

func (a *App) Shutdown(ctx context.Context) error {
	if err := a.Lifecycle.Unmount(ctx); err != nil {
		return fmt.Errorf("lifecycle unmount failed: %w", err)
	}

	if err := a.Lifecycle.Delete(ctx); err != nil {
		return fmt.Errorf("lifecycle delete failed: %w", err)
	}

	return nil
}

func (a *App) registerCoreHooks(fs *filesystem.FileSystem) {
	tools.RegisterCoreTools(a.Hooks, fs)
	agents.RegisterCoreAgents(a.Hooks)
}

func (a *App) collectTools(ctx context.Context) {
	tools.PopulateRegistryFromHooks(a.ToolRegistry, a.Hooks, ctx)
}

func (a *App) collectAgents(ctx context.Context) {
	a.AgentRegistry = agents.NewRegistryFromHooks(nil, a.Hooks, ctx)

	for _, t := range a.AgentRegistry.SubagentTools() {
		if !a.ToolRegistry.Has(t.Name()) {
			a.ToolRegistry.Register(t)
		}
	}
}

func (a *App) registerMessageTools(runner *agent.Runner) {
	if !a.ToolRegistry.Has("message") {
		a.ToolRegistry.Register(agent.NewMessageTool(runner))
	}
	if !a.ToolRegistry.Has("collect-messages") {
		a.ToolRegistry.Register(agent.NewCollectMessagesTool(runner))
	}
}

func (a *App) collectProviders(ctx context.Context) {
	result := a.Hooks.ApplyFilter(ctx, llm.HookProviderRegistry, a.Providers)
	if registry, ok := result.(*llm.ProviderRegistry); ok {
		a.Providers = registry
	}
}

func (a *App) CreateProvider() (llm.Provider, error) {
	return llm.CreateProviderFromConfig(a.Providers, a.Config)
}

func (a *App) CreateRunner(provider llm.Provider, middlewares ...agent.Middleware) *agent.Runner {
	permMW := permission.PermissionMiddleware(a.Permissions, permission.NopApprover)
	retryMW := agent.RetryMiddleware()
	retryProvider := agent.NewRetryableProvider(provider)
	allMiddlewares := []agent.Middleware{permMW, retryMW}
	allMiddlewares = append(allMiddlewares, middlewares...)
	runner := agent.NewRunner(retryProvider, a.EventBus, a.ToolRegistry, allMiddlewares...)
	a.bindRunnerToSubagents(runner)
	return runner
}

func (a *App) CreateRunnerWithApprover(provider llm.Provider, approver permission.Approver, middlewares ...agent.Middleware) *agent.Runner {
	permMW := permission.PermissionMiddleware(a.Permissions, approver)
	retryMW := agent.RetryMiddleware()
	retryProvider := agent.NewRetryableProvider(provider)
	allMiddlewares := []agent.Middleware{permMW, retryMW}
	allMiddlewares = append(allMiddlewares, middlewares...)
	runner := agent.NewRunner(retryProvider, a.EventBus, a.ToolRegistry, allMiddlewares...)
	a.bindRunnerToSubagents(runner)
	return runner
}

func (a *App) bindRunnerToSubagents(runner *agent.Runner) {
	if a.AgentRegistry == nil {
		return
	}
	for _, t := range a.AgentRegistry.SubagentTools() {
		if st, ok := t.(interface{ SetRunner(*agent.Runner) }); ok {
			st.SetRunner(runner)
		}
	}
	a.registerMessageTools(runner)
}
