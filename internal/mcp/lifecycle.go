package mcp

import (
	"context"
	"fmt"
	"sync"

	"github.com/vesvai/vesvai/internal/config"
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
	mu      sync.RWMutex
	servers map[string]*MCPServer
	config  *config.Config
	lc      *lifecycle.Lifecycle
}

func NewManager(lc *lifecycle.Lifecycle, cfg *config.Config) *Manager {
	return &Manager{
		servers: make(map[string]*MCPServer),
		config:  cfg,
		lc:      lc,
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
	if m.config == nil {
		return nil
	}

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
	}

	m.lc.SetComponentPhase("mcp", lifecycle.PhaseMounted)
	return nil
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
	var transport Transport
	if cfg.Url != "" {
		transport = NewSSETransport(cfg.Url, cfg.Headers)
	} else if len(cfg.Command) > 0 {
		transport = NewStdioTransport(cfg.Command[0], cfg.Command[1:]...)
		if len(cfg.Environment) > 0 {
			if stdio, ok := transport.(*StdioTransport); ok {
				stdio.SetEnv(cfg.Environment)
			}
		}
	} else {
		return nil, fmt.Errorf("no command or URL provided for MCP server %s", name)
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
