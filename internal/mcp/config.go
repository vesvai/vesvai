package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/vesvai/vesvai/internal/filesystem"
)

type MCPConfigFile struct {
	MCPServers map[string]MCPServerEntry `json:"mcpServers"`
}

type MCPServerEntry struct {
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     []string          `json:"env,omitempty"`
	URL     string            `json:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

func LoadMCPConfigFile(dir string, fs *filesystem.FileSystem) (*MCPConfigFile, error) {
	configPath := filepath.Join(dir, ".mcp.json")
	return LoadMCPConfigFileFromPath(configPath, fs)
}

func LoadMCPConfigFileFromPath(path string, fs *filesystem.FileSystem) (*MCPConfigFile, error) {
	ctx := context.Background()
	data, err := fs.Read(ctx, path)
	if err != nil {
		return &MCPConfigFile{MCPServers: make(map[string]MCPServerEntry)}, nil
	}

	var config MCPConfigFile
	if err := json.Unmarshal([]byte(data), &config); err != nil {
		return nil, fmt.Errorf("failed to parse .mcp.json: %w", err)
	}

	if config.MCPServers == nil {
		config.MCPServers = make(map[string]MCPServerEntry)
	}

	return &config, nil
}

func CreateTransportForEntry(entry MCPServerEntry) (Transport, error) {
	if entry.URL != "" {
		return NewSSETransport(entry.URL, entry.Headers), nil
	}

	if entry.Command != "" {
		transport := NewStdioTransport(entry.Command, entry.Args...)
		if len(entry.Env) > 0 {
			transport.SetEnv(entry.Env)
		}
		return transport, nil
	}

	return nil, fmt.Errorf("MCP server entry must have either 'command' or 'url'")
}
