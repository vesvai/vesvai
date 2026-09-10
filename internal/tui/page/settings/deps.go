package settings

import (
	"github.com/vesvai/vesvai/internal/agent"
	"github.com/vesvai/vesvai/internal/core/cache"
	"github.com/vesvai/vesvai/internal/core/config"
	"github.com/vesvai/vesvai/internal/core/event"
	"github.com/vesvai/vesvai/internal/llm"
	"github.com/vesvai/vesvai/internal/mcp"
	"github.com/vesvai/vesvai/internal/plugin"
	"github.com/vesvai/vesvai/internal/session"
	"github.com/vesvai/vesvai/internal/vfs"
)

type Deps struct {
	Config   *config.Config
	LLM      *llm.Manager
	MCP      *mcp.Manager
	Sessions *session.Manager
	Agent    *agent.Agent
	Bus      event.Bus
	VFS      *vfs.VFS
	Cache    cache.Cache
	Plugin   *plugin.Manager
}
