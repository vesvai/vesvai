package tools

import "github.com/vesvai/vesvai/internal/agent"

func NewGeneralTools() []agent.Tool {
	return []agent.Tool{
		newReadTool(),
		newEditTool(),
		newWriteTool(),
		newListTool(),
		newGlobTool(),
		newGrepTool(),
		newBashTool(),
	}
}
