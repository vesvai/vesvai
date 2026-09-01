package mcp

import (
	"fmt"

	"github.com/vesvai/vesvai/internal/core/config"
	"github.com/vesvai/vesvai/internal/core/logger"
)

func Module(global map[string]config.MCPServerConfig, log *logger.Logger) (*Manager, error) {
	servers, err := ConfiguredServers(global)
	if err != nil {
		return nil, fmt.Errorf("mcp: load project servers: %w", err)
	}

	mgr := NewManager(log)
	mgr.Start(servers)

	if len(servers) == 0 {
		log.Debug("mcp: no servers configured")
	} else {
		log.Finfo("mcp: starting %d server(s)", len(servers))
	}
	return mgr, nil
}
