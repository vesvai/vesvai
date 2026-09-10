package middlewares

import (
	"github.com/vesvai/vesvai/internal/agent/middlewares"
	"github.com/vesvai/vesvai/internal/builtin/middlewares/permission"
	"github.com/vesvai/vesvai/internal/core/config"
	"github.com/vesvai/vesvai/internal/llm"
)

type Deps struct {
	Config *config.Config
	LLM    *llm.Manager
}

func Create(deps Deps) {
	middlewares.Register("loop-detector", NewLoopDetector())
	middlewares.Register("redaction", NewRedaction())
	middlewares.Register("retry", NewRetry())
	middlewares.Register("permission", permission.New(permission.Deps{
		Config: permissionConfig(deps.Config),
		LLM:    deps.LLM,
	}))
}

func permissionConfig(cfg *config.Config) *config.PermissionConfig {
	if cfg == nil {
		return nil
	}
	return cfg.Permission
}
