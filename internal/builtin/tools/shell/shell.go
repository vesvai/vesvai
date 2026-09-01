package shell

import (
	"github.com/vesvai/vesvai/internal/agent/tools"
	"github.com/vesvai/vesvai/internal/vfs"
)

func ShellTools(fs *vfs.VFS) {
	tools.Register(bashTool(fs))
}
