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

func CreateTransportFromConfig(mcpCfg config.MCPConfig) (Transport, error) {
	if mcpCfg.Url != "" {
		return NewSSETransport(mcpCfg.Url, mcpCfg.Headers), nil
	}
	if len(mcpCfg.Command) > 0 {
		t := NewStdioTransport(mcpCfg.Command[0], mcpCfg.Command[1:]...)
		if len(mcpCfg.Environment) > 0 {
			t.SetEnv(mcpCfg.Environment)
		}
		return t, nil
	}
	return nil, fmt.Errorf("no command or URL provided")
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
