package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/vesvai/vesvai/internal/agent"
	"github.com/vesvai/vesvai/internal/agents/orchestrator"
	"github.com/vesvai/vesvai/internal/bootstrap"
	"github.com/vesvai/vesvai/internal/config"
	"github.com/vesvai/vesvai/internal/filesystem"
	"github.com/vesvai/vesvai/internal/hook"
	"github.com/vesvai/vesvai/internal/llm"
	allProviders "github.com/vesvai/vesvai/internal/llm/providers/all"
	"github.com/vesvai/vesvai/internal/permission"
	"github.com/vesvai/vesvai/internal/session"
	"github.com/vesvai/vesvai/internal/skill"
	tui_app "github.com/vesvai/vesvai/internal/tui/app"
)

func main() {
	if len(os.Args) < 2 {
		runTUI()
		return
	}

	cmd := os.Args[1]

	switch cmd {
	case "run":
		cmdRun(os.Args[2:])
	case "session":
		cmdSession(os.Args[2:])
	case "skill":
		cmdSkill(os.Args[2:])
	case "config":
		cmdConfig(os.Args[2:])
	case "provider":
		cmdProvider(os.Args[2:])
	case "version", "--version", "-v":
		cmdVersion()
	case "help", "--help", "-h":
		cmdHelp()
	case "--debug", "-debug":
		runTUI(cmd)
	default:
		if cmd[0] == '-' {
			runTUI(cmd)
		} else {
			fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", cmd)
			cmdHelp()
			os.Exit(1)
		}
	}
}

func runTUI(args ...string) {
	demo := false
	for _, a := range args {
		if a == "--demo" || a == "-demo" {
			demo = true
		}
	}
	if demo {
		if err := tui_app.New().Run(); err != nil {
			fmt.Fprintf(os.Stderr, "tui error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	app := bootstrap.New(cfg)
	ctx := context.Background()

	if err := app.Init(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "init error: %v\n", err)
		os.Exit(1)
	}
	defer app.Shutdown(ctx)

	if len(cfg.Providers) == 0 {
		fmt.Fprintln(os.Stderr, "no providers configured — run: vesvai provider add <name> <api-key>")
		fmt.Fprintln(os.Stderr, "or launch in demo mode: vesvai --demo")
		os.Exit(1)
	}

	provider, err := app.CreateProvider()
	if err != nil {
		fmt.Fprintf(os.Stderr, "provider error: %v\n", err)
		os.Exit(1)
	}

	approver := tui_app.NewTUIApprover()
	runner := app.CreateRunnerWithApprover(provider, approver)

	cwd, _ := os.Getwd()
	store, err := session.NewFileStore(cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "session store error: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	driver := tui_app.NewAgentDriver(tui_app.AgentDriverConfig{
		Runner:   runner,
		Store:    store,
		App:      app,
		Approver: approver,
	})

	if err := tui_app.NewWithDriver(driver).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "tui error: %v\n", err)
		os.Exit(1)
	}
}

func cmdRun(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	model := fs.String("model", "", "model to use for the run")
	provider := fs.String("provider", "", "provider to route to (defaults to the first configured provider)")
	fs.Parse(args)

	prompt := strings.Join(fs.Args(), " ")
	if prompt == "" {
		fmt.Fprintln(os.Stderr, "usage: vesvai run --model <model> <prompt>")
		os.Exit(1)
	}

	if *model == "" {
		fmt.Fprintln(os.Stderr, "error: --model flag is required")
		fmt.Fprintln(os.Stderr, "usage: vesvai run --model <model> <prompt>")
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	app := bootstrap.New(cfg)
	ctx := context.Background()

	if err := app.Init(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "init error: %v\n", err)
		os.Exit(1)
	}
	defer app.Shutdown(ctx)

	var providerForRun llm.Provider
	if *provider != "" {
		providerForRun, err = app.CreateProviderByName(*provider)
		if err != nil {
			fmt.Fprintf(os.Stderr, "provider error: %v\n", err)
			os.Exit(1)
		}
	} else {
		providerForRun, err = app.CreateProvider()
		if err != nil {
			fmt.Fprintf(os.Stderr, "provider error: %v\n", err)
			os.Exit(1)
		}
	}

	runner := app.CreateRunnerWithApprover(providerForRun, permission.TerminalApprover{})

	orchestratorAgent := orchestrator.New(agent.WithModel(*model))

	fmt.Fprintf(os.Stderr, "running: %s\n", prompt)

	_, err = runner.RunStream(ctx, orchestratorAgent, prompt, func(chunk agent.StreamChunk) error {
		if chunk.Content != "" {
			fmt.Print(chunk.Content)
		}
		if chunk.Reasoning != "" {
			fmt.Fprintf(os.Stderr, "\n[reasoning] %s", chunk.Reasoning)
		}
		if chunk.ToolCall != nil {
			fmt.Fprintf(os.Stderr, "\n[tool: %s]: %s\n", chunk.ToolCall.ToolName, chunk.ToolCall.Args)
		}
		if chunk.ToolResult != nil {
			if chunk.ToolResult.Error != nil {
				fmt.Fprintf(os.Stderr, "\n[tool error: %s] %v\n", chunk.ToolResult.ToolName, chunk.ToolResult.Error)
			}
		}
		return nil
	})

	fmt.Println()

	if err != nil {
		fmt.Fprintf(os.Stderr, "run error: %v\n", err)
		os.Exit(1)
	}
}

func cmdSession(args []string) {
	if len(args) == 0 {
		cmdSessionList()
		return
	}

	switch args[0] {
	case "list", "ls":
		cmdSessionList()
	case "delete", "rm":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: vesvai session delete <id>")
			os.Exit(1)
		}
		cmdSessionDelete(args[1])
	default:
		fmt.Fprintf(os.Stderr, "unknown session subcommand: %s\n", args[0])
		fmt.Fprintln(os.Stderr, "usage: vesvai session {list|delete}")
		os.Exit(1)
	}
}

func cmdSessionList() {
	store, err := session.NewFileStore("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	result, err := store.List(session.ListOptions{
		Page:     1,
		PageSize: 50,
		Reverse:  true,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error listing sessions: %v\n", err)
		os.Exit(1)
	}

	if len(result.Sessions) == 0 {
		fmt.Println("no sessions found")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTITLE\tMESSAGES\tMODEL\tUPDATED")
	fmt.Fprintln(w, "---\t-----\t--------\t-----\t-------")

	for _, s := range result.Sessions {
		title := s.Title
		if title == "" {
			title = "(untitled)"
		}
		fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n",
			s.ID,
			title,
			s.MessageCount,
			s.Model,
			s.UpdatedAt.Format("2006-01-02 15:04"),
		)
	}

	w.Flush()
	fmt.Printf("\n%d session(s)\n", result.Total)
}

func cmdSessionDelete(id string) {
	store, err := session.NewFileStore("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	if !store.Exists(id) {
		fmt.Fprintf(os.Stderr, "session not found: %s\n", id)
		os.Exit(1)
	}

	if err := store.Delete(id); err != nil {
		fmt.Fprintf(os.Stderr, "error deleting session: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("deleted session: %s\n", id)
}

func cmdSkill(args []string) {
	if len(args) == 0 {
		cmdSkillList()
		return
	}

	switch args[0] {
	case "list", "ls":
		cmdSkillList()
	case "show", "get":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: vesvai skill show <name>")
			os.Exit(1)
		}
		cmdSkillShow(args[1])
	default:
		fmt.Fprintf(os.Stderr, "unknown skill subcommand: %s\n", args[0])
		fmt.Fprintln(os.Stderr, "usage: vesvai skill {list|show}")
		os.Exit(1)
	}
}

func cmdSkillList() {
	cwd, _ := os.Getwd()
	fs, _ := filesystem.New(filesystem.Config{RootDir: cwd})
	mgr, err := skill.NewManager(cwd, fs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	skills, err := mgr.List()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error listing skills: %v\n", err)
		os.Exit(1)
	}

	if len(skills) == 0 {
		fmt.Println("no skills found")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tLOCATION\tDESCRIPTION")
	fmt.Fprintln(w, "----\t--------\t-----------")

	for _, s := range skills {
		desc := s.Description
		if len(desc) > 60 {
			desc = desc[:57] + "..."
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", s.Name, s.Location, desc)
	}

	w.Flush()
	fmt.Printf("\n%d skill(s)\n", len(skills))
}

func cmdSkillShow(name string) {
	cwd, _ := os.Getwd()
	fs, _ := filesystem.New(filesystem.Config{RootDir: cwd})
	mgr, err := skill.NewManager(cwd, fs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	s, err := mgr.Read(name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("--- %s (%s) ---\n", s.Name, s.Location)
	fmt.Println(s.Content)
}

func cmdConfig(args []string) {
	if len(args) == 0 {
		args = []string{"show"}
	}

	switch args[0] {
	case "show", "get":
		cmdConfigShow()
	case "path":
		configPath, err := config.GetConfigPath()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(configPath)
	default:
		fmt.Fprintf(os.Stderr, "unknown config subcommand: %s\n", args[0])
		fmt.Fprintln(os.Stderr, "usage: vesvai config {show|path}")
		os.Exit(1)
	}
}

func cmdConfigShow() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
		os.Exit(1)
	}

	configPath, _ := config.GetConfigPath()
	fmt.Printf("config path: %s\n\n", configPath)

	if len(cfg.Providers) == 0 {
		fmt.Println("providers: (none configured)")
	} else {
		fmt.Println("providers:")
		for _, p := range cfg.Providers {
			key := p.APIKey
			if key != "" {
				key = key[:4] + "..." + key[len(key)-4:]
			}
			fmt.Printf("  - %s (key: %s)\n", p.Provider, key)
		}
	}

	if len(cfg.MCPs) == 0 {
		fmt.Println("mcps: (none configured)")
	} else {
		fmt.Println("mcps:")
		for _, m := range cfg.MCPs {
			status := "disabled"
			if m.Enabled {
				status = "enabled"
			}
			fmt.Printf("  - %s [%s] (%s)\n", m.Type, status, m.Url)
		}
	}
}

func supportedProviders() string {
	hooks := hook.New(nil)
	allProviders.RegisterAll(hooks)
	registry := llm.NewProviderRegistry()
	hooks.ApplyFilter(context.Background(), llm.HookProviderRegistry, registry)
	return strings.Join(registry.Supported(), ", ")
}

func cmdProvider(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: vesvai provider add <name> <api-key>")
		fmt.Fprintln(os.Stderr, "supported providers: "+supportedProviders())
		os.Exit(1)
	}

	switch args[0] {
	case "add":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: vesvai provider add <name> <api-key>")
			fmt.Fprintln(os.Stderr, "supported providers: "+supportedProviders())
			os.Exit(1)
		}
		cmdProviderAdd(args[1], args[2])
	default:
		fmt.Fprintf(os.Stderr, "unknown provider subcommand: %s\n", args[0])
		fmt.Fprintln(os.Stderr, "usage: vesvai provider {add}")
		os.Exit(1)
	}
}

func cmdProviderAdd(name, apiKey string) {
	providerCfg := config.LLMConfig{
		Provider: name,
		APIKey:   apiKey,
	}

	hooks := hook.New(nil)
	allProviders.RegisterAll(hooks)

	registry := llm.NewProviderRegistry()
	result := hooks.ApplyFilter(context.Background(), llm.HookProviderRegistry, registry)
	if r, ok := result.(*llm.ProviderRegistry); ok {
		registry = r
	}

	provider, err := registry.Create(providerCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n\nsupported providers: %s\n", err, strings.Join(registry.Supported(), ", "))
		os.Exit(1)
	}

	if err := config.AddProvider(providerCfg); err != nil {
		fmt.Fprintf(os.Stderr, "error saving provider: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "fetching models from %s...\n", name)
	models, err := provider.ListModels(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not fetch models (provider still saved): %v\n", err)
		fmt.Printf("added provider: %s (0 models cached)\n", name)
		return
	}

	modelIDs := make([]string, len(models))
	for i, m := range models {
		modelIDs[i] = m.ID
	}

	if err := config.SaveProviderModels(name, modelIDs); err != nil {
		fmt.Fprintf(os.Stderr, "error saving models cache: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("added provider: %s (%d models cached)\n", name, len(models))
}

func cmdVersion() {
	fmt.Printf("%s %s\n", config.AppName, config.AppVersion)
}

func cmdHelp() {
	fmt.Println(`vesvai - AI-powered development assistant

USAGE:
  vesvai                                          Launch the TUI (default)
  vesvai --demo                                   Launch the TUI with the demo driver
  vesvai run --model <model> <prompt>             Run a prompt non-interactively
  vesvai session list                             List saved sessions
  vesvai session delete <id>                      Delete a session
  vesvai skill list                               List installed skills
  vesvai skill show <skill>                       Show a skill's content
  vesvai provider add <provider> <api-key>        Add a provider
  vesvai config show                              Show current configuration
  vesvai config path                              Print config file path
  vesvai version                                  Print version
  vesvai help                                     Show this help message

EXAMPLES:
  vesvai                                          # start interactive TUI
  vesvai --demo                                   # start with simulated runs
  vesvai run --model gemini-1.5-flash "explain code"
  vesvai run --model claude-3-opus "refactor this"
  vesvai session list                             # list sessions
  vesvai skill list                               # list skills
  vesvai provider add openrouter sk-or-xxx          # add openrouter provider
  vesvai provider add openai sk-xxx                 # add openai provider
  vesvai config show                              # show config`)
}
