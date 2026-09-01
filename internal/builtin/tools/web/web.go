package web

import (
	"github.com/vesvai/vesvai/internal/agent/tools"
	"github.com/vesvai/vesvai/internal/vfs"
)

func WebTools(fs *vfs.VFS) {
	tools.Register(fetchTool(fs))
	tools.Register(searchTool(fs))
}