package tools

import (
	"github.com/vesvai/vesvai/internal/builtin/tools/ask"
	"github.com/vesvai/vesvai/internal/builtin/tools/file"
	"github.com/vesvai/vesvai/internal/builtin/tools/shell"
	"github.com/vesvai/vesvai/internal/builtin/tools/subagent"
	"github.com/vesvai/vesvai/internal/builtin/tools/todo"
	"github.com/vesvai/vesvai/internal/builtin/tools/web"
	"github.com/vesvai/vesvai/internal/session"
	"github.com/vesvai/vesvai/internal/vfs"
)

func Create(fs *vfs.VFS, sess *session.Manager) {
	file.FileTools(fs)
	todo.TodoTools(fs)
	shell.ShellTools(fs)
	web.WebTools(fs)
	subagent.SubAgentTools(sess)
	ask.AskTool()
}
