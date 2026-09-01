package agents

import (
	"github.com/vesvai/vesvai/internal/builtin/agents/developer"
	"github.com/vesvai/vesvai/internal/builtin/agents/explorer"
	"github.com/vesvai/vesvai/internal/builtin/agents/orchestrator"
	"github.com/vesvai/vesvai/internal/builtin/agents/planner"
	"github.com/vesvai/vesvai/internal/vfs"
)

func Create(fs *vfs.VFS) {
	explorer.Register()
	planner.Register(fs)
	developer.Register(fs)
	orchestrator.Register(fs)
}