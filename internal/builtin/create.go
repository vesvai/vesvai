package builtin

import (
	"github.com/vesvai/vesvai/internal/builtin/agents"
	"github.com/vesvai/vesvai/internal/builtin/middlewares"
	"github.com/vesvai/vesvai/internal/builtin/reminders/usage"
	"github.com/vesvai/vesvai/internal/builtin/tools"
	"github.com/vesvai/vesvai/internal/core/config"
	"github.com/vesvai/vesvai/internal/core/event"
	"github.com/vesvai/vesvai/internal/llm"
	"github.com/vesvai/vesvai/internal/session"
	"github.com/vesvai/vesvai/internal/vfs"
)

type Options struct {
	LLM    *llm.Manager
	Config *config.Config
	Bus    event.Bus
}

func Create(fs *vfs.VFS, sess *session.Manager, opts Options) error {
	tools.Create(fs, sess)
	middlewares.Create(fs, middlewares.Deps{
		Config:   opts.Config,
		LLM:      opts.LLM,
		Sessions: sess,
	})
	agents.Create(fs)
	if opts.Bus != nil {
		rem := usage.New()
		if err := rem.Start(opts.Bus); err != nil {
			return err
		}
	}
	return nil
}
