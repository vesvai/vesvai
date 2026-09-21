package middlewares

import (
	"github.com/vesvai/vesvai/internal/agent/middlewares"
	"github.com/vesvai/vesvai/internal/builtin/middlewares/compaction"
	"github.com/vesvai/vesvai/internal/builtin/middlewares/permission"
	"github.com/vesvai/vesvai/internal/core/config"
	"github.com/vesvai/vesvai/internal/llm"
	"github.com/vesvai/vesvai/internal/session"
	"github.com/vesvai/vesvai/internal/vfs"
)

type Deps struct {
	Config   *config.Config
	LLM      *llm.Manager
	Sessions *session.Manager
}

func Create(fs *vfs.VFS, deps Deps) {
	middlewares.Register("loop-detector", NewLoopDetector())
	middlewares.Register("redaction", NewRedaction())
	middlewares.Register("retry", NewRetry())
	var permCfg *config.PermissionConfig
	if deps.Config != nil {
		permCfg = deps.Config.Permission
	}
	perm := permission.New(permission.Deps{
		Config: permCfg,
		LLM:    deps.LLM,
	})
	middlewares.Register("permission", perm)
	if fs != nil {
		fs.OnAccessCheck(perm.AccessChecker)
	}
	var compCfg *config.CompactionConfig
	if deps.Config != nil {
		compCfg = deps.Config.Compaction
	}
	middlewares.Register("compaction", compaction.New(compaction.Deps{
		Config: compCfg,
		LLM:    deps.LLM,
	}))
}
