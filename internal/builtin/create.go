package builtin

import (
	"github.com/vesvai/vesvai/internal/builtin/agents"
	"github.com/vesvai/vesvai/internal/builtin/middlewares"
	"github.com/vesvai/vesvai/internal/builtin/tools"
	"github.com/vesvai/vesvai/internal/session"
	"github.com/vesvai/vesvai/internal/vfs"
)

func Create(fs *vfs.VFS, sess *session.Manager) error {
	tools.Create(fs, sess)
	middlewares.Create()
	agents.Create(fs)
	return nil
}
