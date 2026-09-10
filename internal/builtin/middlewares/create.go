package middlewares

import (
	"github.com/vesvai/vesvai/internal/agent/middlewares"
	"github.com/vesvai/vesvai/internal/builtin/middlewares/permission"
	"github.com/vesvai/vesvai/internal/core/config"
	"github.com/vesvai/vesvai/internal/llm"
	"github.com/vesvai/vesvai/internal/vfs"
)

type Deps struct {
	Config *config.Config
	LLM    *llm.Manager
}

func Create(fs *vfs.VFS, deps Deps) {
	middlewares.Register("loop-detector", NewLoopDetector())
	middlewares.Register("redaction", NewRedaction())
	middlewares.Register("retry", NewRetry())
	perm := permission.New(permission.Deps{
		Config: permissionConfig(deps.Config),
		LLM:    deps.LLM,
	})
	middlewares.Register("permission", perm)
	if fs != nil {
		fs.OnAccessCheck(perm.AccessChecker)
	}
}

func permissionConfig(cfg *config.Config) *config.PermissionConfig {
	if cfg == nil {
		return nil
	}
	return cfg.Permission
}
