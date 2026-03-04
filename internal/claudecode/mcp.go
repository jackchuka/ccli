package claudecode

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/jackchuka/ccli/internal/agent"
)

// ListMCPServers collects MCP servers from all scopes.
func (a *Agent) ListMCPServers() ([]agent.MCPServer, error) {
	var servers []agent.MCPServer

	// Global: root-level mcpServers in ~/.claude.json
	cfg, err := LoadConfig(a.paths.ConfigFile)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	if cfg != nil {
		for name, entry := range cfg.MCPServers {
			servers = append(servers, toMCPServer(name, entry, agent.ScopeGlobal))
		}
	}

	// Project: from .mcp.json
	if a.paths.MCPFile != "" {
		project, err := loadMCPFile(a.paths.MCPFile, agent.ScopeProject)
		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("loading .mcp.json: %w", err)
		}
		servers = append(servers, project...)
	}

	return servers, nil
}

// GetMCPServer finds an MCP server by name across all scopes.
func (a *Agent) GetMCPServer(name string) (*agent.MCPServer, error) {
	servers, err := a.ListMCPServers()
	if err != nil {
		return nil, err
	}
	return findByName(servers, name, func(s agent.MCPServer) string { return s.Name }, "MCP server")
}

func loadMCPFile(path string, scope agent.Scope) ([]agent.MCPServer, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var wrapper struct {
		MCPServers map[string]MCPServerEntry `json:"mcpServers"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, err
	}
	var servers []agent.MCPServer
	for name, entry := range wrapper.MCPServers {
		servers = append(servers, toMCPServer(name, entry, scope))
	}
	return servers, nil
}

func toMCPServer(name string, entry MCPServerEntry, scope agent.Scope) agent.MCPServer {
	typ := entry.Type
	if typ == "" && entry.Command != "" {
		typ = "stdio"
	}
	cmd := entry.Command
	if len(entry.Args) > 0 {
		cmd = cmd + " " + strings.Join(entry.Args, " ")
	}
	return agent.MCPServer{
		Name:    name,
		Scope:   scope,
		Type:    typ,
		Command: cmd,
		Args:    entry.Args,
		URL:     entry.URL,
		Env:     entry.Env,
	}
}
