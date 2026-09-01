package bootstrap

import (
	"fmt"
	"os"

	"github.com/vesvai/vesvai/internal/builtin"
	_ "github.com/vesvai/vesvai/internal/builtin/agents"
	_ "github.com/vesvai/vesvai/internal/builtin/lsps"
	"github.com/vesvai/vesvai/internal/builtin/tools/subagent"
	"github.com/vesvai/vesvai/internal/cli"
	"github.com/vesvai/vesvai/internal/core/cache"
	"github.com/vesvai/vesvai/internal/core/config"
	"github.com/vesvai/vesvai/internal/core/event"
	"github.com/vesvai/vesvai/internal/core/logger"
	"github.com/vesvai/vesvai/internal/llm"
	_ "github.com/vesvai/vesvai/internal/llm/drivers"
	_ "github.com/vesvai/vesvai/internal/llm/providers"
	"github.com/vesvai/vesvai/internal/lsp"
	"github.com/vesvai/vesvai/internal/mcp"
	"github.com/vesvai/vesvai/internal/session"
	"github.com/vesvai/vesvai/internal/skill"
	"github.com/vesvai/vesvai/internal/utils/query"
	"github.com/vesvai/vesvai/internal/vfs"
)

func Run(args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("bootstrap: load config: %w", err)
	}

	if err := config.SaveIfNotExists(cfg); err != nil {
		return fmt.Errorf("bootstrap: save default config: %w", err)
	}

	if err := config.EnsureProjectConfigDir(); err != nil {
		return fmt.Errorf("bootstrap: create project config dir: %w", err)
	}

	log, err := logger.LoggerModule(cfg.Logger)
	if err != nil {
		return fmt.Errorf("bootstrap: init logger: %w", err)
	}
	defer log.Close()

	bus := event.New()

	cacheStore, err := cache.CacheModule(cfg.Cache)
	if err != nil {
		return fmt.Errorf("bootstrap: init cache: %w", err)
	}
	defer cacheStore.Close()

	mgr := llm.NewManager(bus, log, cacheStore)
	if err := mgr.Start(); err != nil {
		return fmt.Errorf("bootstrap: init llm manager: %w", err)
	}
	defer mgr.Shutdown()

	bus.Publish(event.TopicAppMounted, cfg)

	sess, err := session.SessionModule(cfg.Session, bus, log)
	if err != nil {
		return fmt.Errorf("bootstrap: init session manager: %w", err)
	}
	defer sess.Close()

	mgr.SetSessionResolver(func() (string, string, bool) {
		dir, err := os.Getwd()
		if err != nil {
			return "", "", false
		}
		q := query.Query{
			Page: query.Page{Number: 1, Size: 20},
			Sort: []query.Sort{{Column: "updated_at", Dir: query.Desc}},
			Filters: []query.Filter{
				{Column: "project_dir", Operator: query.OpEqual, Value: dir},
			},
		}
		sessions, _, err := sess.List(q)
		if err != nil {
			return "", "", false
		}
		for _, s := range sessions {
			if subagent.IsSubagentSession(s.ID) {
				continue
			}
			if s.Provider == "" || s.Model == "" {
				continue
			}
			return s.Provider, s.Model, true
		}
		return "", "", false
	})

	rec := session.NewRecorder(sess, log)
	if err := rec.Start(bus); err != nil {
		return fmt.Errorf("bootstrap: init session recorder: %w", err)
	}
	defer rec.Stop(bus)

	fs, err := vfs.VFSModule(log)
	if err != nil {
		return fmt.Errorf("bootstrap: init vfs: %w", err)
	}
	if err := skill.SkillModule(); err != nil {
		return fmt.Errorf("bootstrap: init skills: %w", err)
	}
	if err := builtin.Create(fs, sess); err != nil {
		return fmt.Errorf("bootstrap: builtin: %w", err)
	}
	log.Finfo("vfs: workspace mounted at %s", fs.Root())

	mcpMgr, err := mcp.Module(cfg.MCPServers, log)
	if err != nil {
		return fmt.Errorf("bootstrap: init mcp: %w", err)
	}
	defer mcpMgr.Close()

	lspMgr, err := lsp.Module(cfg.LanguageServers, fs, log)
	if err != nil {
		return fmt.Errorf("bootstrap: init lsp: %w", err)
	}
	defer lspMgr.Close()

	log.Info("application started")

	app := cli.New(bus, cfg, log, fs, sess, mgr, mcpMgr, lspMgr)
	if err := app.Execute(args); err != nil {
		return fmt.Errorf("bootstrap: cli: %w", err)
	}

	log.Info("application stopped")
	return nil
}
