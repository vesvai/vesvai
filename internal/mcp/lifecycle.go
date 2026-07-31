package mcp

import (
	"context"
	"fmt"
	"sync"

	"github.com/vesvai/vesvai/internal/agent"
	"github.com/vesvai/vesvai/internal/config"
	"github.com/vesvai/vesvai/internal/filesystem"
	"github.com/vesvai/vesvai/internal/hook"
	"github.com/vesvai/vesvai/internal/lifecycle"
)

type MCPServer struct {
	Name    string
	Client  *Client
	Tools   []AgentTool
	Config  config.MCPConfig
	running bool
}

type Manager struct {
	mu         sync.RWMutex
	servers    map[string]*MCPServer
	config     *config.Config
	projectDir string
	fs         *filesystem.FileSystem
	lc         *lifecycle.Lifecycle
	hooks      *hook.Hooks
}

func NewManager(lc *lifecycle.Lifecycle, cfg *config.Config, hooks *hook.Hooks, projectDir string, fs *filesystem.FileSystem) *Manager {
	return &Manager{
		servers:    make(map[string]*MCPServer),
		config:     cfg,
		projectDir: projectDir,
		fs:         fs,
		lc:         lc,
		hooks:      hooks,
	}
}

func (m *Manager) RegisterHooks() {
	m.lc.On(lifecycle.HookCreate).Priority(80).Do(m.onCreate)
	m.lc.On(lifecycle.HookMount).Priority(80).Do(m.onMount)
	m.lc.On(lifecycle.HookUnmount).Priority(80).Do(m.onUnmount)
	m.lc.On(lifecycle.HookDelete).Priority(80).Do(m.onDelete)
}

func (m *Manager) onCreate(ctx context.Context, args ...interface{}) error {
	m.lc.SetComponentPhase("mcp", lifecycle.PhaseCreated)
	return nil
}

func (m *Manager) onMount(ctx context.Context, args ...interface{}) error {
	if m.config != nil {
		for _, mcpCfg := range m.config.MCPs {
			if !mcpCfg.Enabled {
				continue
			}

			name := deriveServerName(mcpCfg)
			if _, exists := m.servers[name]; exists {
				continue
			}

			server, err := m.startServer(ctx, name, mcpCfg)
			if err != nil {
				fmt.Printf("MCP: failed to start server %s: %v\n", name, err)
				continue
			}

			m.servers[name] = server
			m.registerServerTools(server)
		}
	}

	if m.projectDir != "" {
		m.mountFromMCPFile(ctx, m.projectDir)
	}

	m.lc.SetComponentPhase("mcp", lifecycle.PhaseMounted)
	return nil
}

func (m *Manager) mountFromMCPFile(ctx context.Context, dir string) {
	configFile, err := LoadMCPConfigFile(dir, m.fs)
	if err != nil {
		fmt.Printf("MCP: failed to load .mcp.json from %s: %v\n", dir, err)
		return
	}

	for name, entry := range configFile.MCPServers {
		if _, exists := m.servers[name]; exists {
			continue
		}

		transport, err := CreateTransportForEntry(entry)
		if err != nil {
			fmt.Printf("MCP: failed to create transport for %q: %v\n", name, err)
			continue
		}

		client := NewClient(transport, WithServerName(name))
		if err := client.Connect(ctx); err != nil {
			fmt.Printf("MCP: failed to connect to %q: %v\n", name, err)
			continue
		}

		tools, err := client.ListTools(ctx)
		if err != nil {
			client.Close()
			fmt.Printf("MCP: failed to list tools for %q: %v\n", name, err)
			continue
		}

		agentTools := make([]AgentTool, len(tools))
		for i, tool := range tools {
			agentTools[i] = NewMCPTool(client, tool)
		}

		server := &MCPServer{
			Name:    name,
			Client:  client,
			Tools:   agentTools,
			running: true,
		}
		m.servers[name] = server
		m.registerServerTools(server)
	}
}

func (m *Manager) registerServerTools(server *MCPServer) {
	if m.hooks == nil || len(server.Tools) == 0 {
		return
	}

	m.hooks.AddFilter(hook.HookToolsCollect, func(ctx context.Context, value interface{}, args ...interface{}) interface{} {
		existing, _ := value.([]agent.Tool)
		for _, t := range server.Tools {
			existing = append(existing, t)
		}
		return existing
	}, 40)
}

func (m *Manager) onUnmount(ctx context.Context, args ...interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, server := range m.servers {
		if server.running && server.Client != nil {
			server.Client.Close()
			server.running = false
		}
		delete(m.servers, name)
	}

	m.lc.SetComponentPhase("mcp", lifecycle.PhaseUnmounted)
	return nil
}

func (m *Manager) onDelete(ctx context.Context, args ...interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.servers = make(map[string]*MCPServer)
	m.lc.SetComponentPhase("mcp", lifecycle.PhaseDeleted)
	return nil
}

func (m *Manager) startServer(ctx context.Context, name string, cfg config.MCPConfig) (*MCPServer, error) {
	transport, err := CreateTransportFromConfig(cfg)
	if err != nil {
		return nil, err
	}

	client := NewClient(transport, WithServerName(name))
	if err := client.Connect(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	tools, err := client.ListTools(ctx)
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}

	agentTools := make([]AgentTool, len(tools))
	for i, tool := range tools {
		agentTools[i] = NewMCPTool(client, tool)
	}

	return &MCPServer{
		Name:    name,
		Client:  client,
		Tools:   agentTools,
		Config:  cfg,
		running: true,
	}, nil
}

func (m *Manager) GetServer(name string) (*MCPServer, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	server, ok := m.servers[name]
	return server, ok
}

func (m *Manager) Servers() map[string]*MCPServer {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[string]*MCPServer, len(m.servers))
	for k, v := range m.servers {
		result[k] = v
	}
	return result
}

func (m *Manager) AllTools() []AgentTool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var tools []AgentTool
	for _, server := range m.servers {
		tools = append(tools, server.Tools...)
	}
	return tools
}
