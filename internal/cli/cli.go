package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/vesvai/vesvai/internal/agent/agents"
	"github.com/vesvai/vesvai/internal/core/cache"
	"github.com/vesvai/vesvai/internal/core/config"
	"github.com/vesvai/vesvai/internal/core/event"
	"github.com/vesvai/vesvai/internal/core/hook"
	"github.com/vesvai/vesvai/internal/core/logger"
	"github.com/vesvai/vesvai/internal/llm"
	"github.com/vesvai/vesvai/internal/lsp"
	"github.com/vesvai/vesvai/internal/mcp"
	"github.com/vesvai/vesvai/internal/session"
	"github.com/vesvai/vesvai/internal/tui"
	"github.com/vesvai/vesvai/internal/tui/page/settings"
	"github.com/vesvai/vesvai/internal/vfs"
)

type CLI struct {
	bus      event.Bus
	cfg      *config.Config
	log      *logger.Logger
	sessions *session.Manager
	llmMgr   *llm.Manager
	mcpMgr   *mcp.Manager
	lspMgr   *lsp.Manager
	fs       *vfs.VFS
	cache    cache.Cache
	root     *cobra.Command
	commands hook.Hook[[]*cobra.Command]
	picker   func(items []string, label string) (int, error)
}

func New(bus event.Bus, cfg *config.Config, log *logger.Logger, vfs *vfs.VFS, sessions *session.Manager, llmMgr *llm.Manager, mcpMgr *mcp.Manager, lspMgr *lsp.Manager, cache cache.Cache) *CLI {
	c := &CLI{
		bus:      bus,
		cfg:      cfg,
		log:      log,
		fs:       vfs,
		cache:    cache,
		sessions: sessions,
		llmMgr:   llmMgr,
		mcpMgr:   mcpMgr,
		lspMgr:   lspMgr,
		root:     newRootCommand(),
		picker:   defaultPicker,
	}

	c.root.RunE = func(cmd *cobra.Command, args []string) error {
		if !isTerminal(cmd.InOrStdin()) {
			var parts []string
			if argMsg := strings.TrimSpace(strings.Join(args, " ")); argMsg != "" {
				parts = append(parts, argMsg)
			}
			stdinMsg, err := readStdinMessage(cmd.InOrStdin())
			if err != nil {
				return err
			}
			if stdinMsg != "" {
				parts = append(parts, stdinMsg)
			}
			message := strings.Join(parts, "\n\n")
			return c.runRun(cmd.OutOrStdout(), cmd.InOrStdin(), message, runOptions{})
		}
		deps, err := c.tuiDeps()
		if err != nil {
			return err
		}
		return tui.Run(bus, deps)
	}

	c.registerDefaultCommands()

	return c
}

func newRootCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "vesvai",
		Short: "vesvai command line interface",
		Args:  cobra.ArbitraryArgs,
	}
}

func (c *CLI) OnRegisterCommand(fn func([]*cobra.Command) []*cobra.Command) {
	c.commands.Add(fn)
}

func (c *CLI) registerDefaultCommands() {
	c.OnRegisterCommand(func(cmds []*cobra.Command) []*cobra.Command {
		return append(cmds,
			c.newLoginCommand(),
			c.newLogsCommand(),
			c.newFilesCommand(),
			c.newCacheCommand(),
			c.newProviderCommand(),
			c.newModelsCommand(),
			c.newDoctorCommand(),
			c.newConfigCommand(),
			c.newSessionCommand(),
			c.newRunCommand(),
			c.newMCPCommand(),
			c.newLSPCommand(),
			c.newTUICommand(),
			c.newServeCommand(),
			c.newVersionCommand(),
			c.newUpdateCommand(),
		)
	})
}

func (c *CLI) tuiDeps() (settings.Deps, error) {
	orch, err := agents.New("orchestrator")
	if err != nil {
		return settings.Deps{}, fmt.Errorf("cli: create orchestrator: %w", err)
	}
	orch.Bus = c.bus
	return settings.Deps{
		Config:   c.cfg,
		LLM:      c.llmMgr,
		MCP:      c.mcpMgr,
		Sessions: c.sessions,
		Agent:    orch,
		Bus:      c.bus,
		VFS:      c.fs,
		Cache:    c.cache,
	}, nil
}

func (c *CLI) Execute(args []string) error {
	for _, cmd := range c.commands.Apply(nil) {
		c.root.AddCommand(cmd)
	}

	c.root.SetArgs(args)
	return c.root.Execute()
}

func readStdinMessage(in io.Reader) (string, error) {
	data, err := io.ReadAll(in)
	if err != nil {
		return "", fmt.Errorf("cli: read stdin: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}
