package tools

import (
	"context"

	"github.com/vesvai/vesvai/internal/agent"
	"github.com/vesvai/vesvai/internal/filesystem"
	"github.com/vesvai/vesvai/internal/hook"
)

func RegisterCoreTools(hooks *hook.Hooks, fs *filesystem.FileSystem) {
	hooks.AddFilter(hook.HookToolsCollect, func(ctx context.Context, value interface{}, args ...interface{}) interface{} {
		existing, _ := value.([]agent.Tool)
		return append(existing, CoreTools(fs)...)
	}, 50)
}

func CoreTools(fs *filesystem.FileSystem) []agent.Tool {
	return []agent.Tool{
		newReadTool(fs),
		newEditTool(fs),
		newWriteTool(fs),
		newListTool(fs),
		newGlobTool(fs),
		newGrepTool(fs),
		newBashTool(fs.Root()),
		newSetTodoTool(fs),
		newGetTodoTool(fs),
		newListTodosTool(fs),
		newUpdateTodoTool(fs),
		newDeleteTodoTool(fs),
	}
}

func CollectToolsFromHooks(hooks *hook.Hooks, ctx context.Context) []agent.Tool {
	result := hooks.ApplyFilter(ctx, hook.HookToolsCollect, []agent.Tool{})
	tools, _ := result.([]agent.Tool)
	return tools
}

func PopulateRegistryFromHooks(registry *agent.ToolRegistry, hooks *hook.Hooks, ctx context.Context) {
	tools := CollectToolsFromHooks(hooks, ctx)
	for _, t := range tools {
		registry.Register(t)
	}
}
