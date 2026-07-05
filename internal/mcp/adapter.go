package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/vesvai/vesvai/internal/config"
)

type AgentTool interface {
	Name() string
	Description() string
	Schema() map[string]any
	Handle(ctx context.Context, params map[string]any) (string, error)
}

type MCPTool struct {
	client      *Client
	name        string
	description string
	schema      map[string]any
}

func NewMCPTool(client *Client, tool Tool) *MCPTool {
	var schema map[string]any
	if len(tool.InputSchema) > 0 {
		_ = json.Unmarshal(tool.InputSchema, &schema)
	}

	return &MCPTool{
		client:      client,
		name:        tool.Name,
		description: tool.Description,
		schema:      schema,
	}
}

func (t *MCPTool) Name() string        { return t.name }
func (t *MCPTool) Description() string { return t.description }
func (t *MCPTool) Schema() map[string]any {
	if t.schema == nil {
		return map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		}
	}
	return t.schema
}

func (t *MCPTool) Handle(ctx context.Context, params map[string]any) (string, error) {
	result, err := t.client.CallTool(ctx, t.name, params)
	if err != nil {
		return "", fmt.Errorf("MCP tool %q failed: %w", t.name, err)
	}

	var texts []string
	for _, c := range result.Content {
		if c.Type == "text" && c.Text != "" {
			texts = append(texts, c.Text)
		}
	}

	return strings.Join(texts, "\n"), nil
}

func DiscoverTools(ctx context.Context, transport Transport) ([]AgentTool, error) {
	client := NewClient(transport)
	if err := client.Connect(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to MCP server: %w", err)
	}
	defer client.Close()

	tools, err := client.ListTools(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}

	agentTools := make([]AgentTool, len(tools))
	for i, tool := range tools {
		agentTools[i] = NewMCPTool(client, tool)
	}

	return agentTools, nil
}

func DiscoverToolsFromCommand(ctx context.Context, command string, args ...string) ([]AgentTool, error) {
	transport := NewStdioTransport(command, args...)
	return DiscoverTools(ctx, transport)
}

type MCPServerConfig struct {
	Command string   `json:"command"`
	Args    []string `json:"args,omitempty"`
	Env     []string `json:"env,omitempty"`
}

func DiscoverToolsFromConfig(ctx context.Context, config MCPServerConfig) ([]AgentTool, error) {
	transport := NewStdioTransport(config.Command, config.Args...)
	if len(config.Env) > 0 {
		transport.SetEnv(config.Env)
	}
	return DiscoverTools(ctx, transport)
}

func DiscoverToolsFromMCPFile(ctx context.Context, dir string) (map[string][]AgentTool, error) {
	configFile, err := LoadMCPConfigFile(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to load .mcp.json: %w", err)
	}

	result := make(map[string][]AgentTool)
	for name, entry := range configFile.MCPServers {
		transport, err := CreateTransportForEntry(entry)
		if err != nil {
			return nil, fmt.Errorf("failed to create transport for %q: %w", name, err)
		}

		tools, err := DiscoverTools(ctx, transport)
		if err != nil {
			return nil, fmt.Errorf("failed to discover tools for %q: %w", name, err)
		}

		result[name] = tools
	}

	return result, nil
}

func DiscoverToolsFromConfigFile(ctx context.Context, mcps []config.MCPConfig) (map[string][]AgentTool, error) {
	result := make(map[string][]AgentTool)

	for _, mcp := range mcps {
		if !mcp.Enabled {
			continue
		}

		name := deriveServerName(mcp)

		var transport Transport
		if mcp.Url != "" {
			transport = NewSSETransport(mcp.Url, mcp.Headers)
		} else if len(mcp.Command) > 0 {
			transport = NewStdioTransport(mcp.Command[0], mcp.Command[1:]...)
			if len(mcp.Environment) > 0 {
				if stdio, ok := transport.(*StdioTransport); ok {
					stdio.SetEnv(mcp.Environment)
				}
			}
		} else {
			continue
		}

		tools, err := DiscoverTools(ctx, transport)
		if err != nil {
			return nil, fmt.Errorf("failed to discover tools for %q: %w", name, err)
		}

		result[name] = tools
	}

	return result, nil
}

func deriveServerName(mcp config.MCPConfig) string {
	if mcp.Url != "" {
		parts := strings.Split(strings.TrimRight(mcp.Url, "/"), "/")
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
		return "remote"
	}
	if len(mcp.Command) > 0 {
		return mcp.Command[0]
	}
	return "unknown"
}
