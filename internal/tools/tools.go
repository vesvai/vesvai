package tools

import "github.com/vesvai/vesvai/internal/agent"

const HookTools = "tools:collect"

func CollectTools(existing interface{}, args ...interface{}) interface{} {
	tools, _ := existing.([]agent.Tool)
	return tools
}
