package claudecode_test

import (
	"testing"

	"github.com/jackchuka/ccli/internal/agent"
	"github.com/jackchuka/ccli/internal/claudecode"
)

func TestListMCPServers(t *testing.T) {
	a := claudecode.NewAgent(claudecode.Paths{
		ConfigFile: "testdata/claude.json",
		MCPFile:    "testdata/mcp.json",
		HomeDir:    "testdata",
	})

	servers, err := a.ListMCPServers()
	if err != nil {
		t.Fatalf("ListMCPServers: %v", err)
	}

	if len(servers) != 2 {
		t.Fatalf("got %d servers, want 2", len(servers))
	}

	scopes := map[agent.Scope]int{}
	for _, s := range servers {
		scopes[s.Scope]++
	}
	if scopes[agent.ScopeGlobal] != 1 {
		t.Errorf("global servers = %d, want 1", scopes[agent.ScopeGlobal])
	}
	if scopes[agent.ScopeProject] != 1 {
		t.Errorf("project servers = %d, want 1", scopes[agent.ScopeProject])
	}
}

func TestGetMCPServer(t *testing.T) {
	a := claudecode.NewAgent(claudecode.Paths{
		ConfigFile: "testdata/claude.json",
		MCPFile:    "testdata/mcp.json",
		HomeDir:    "testdata",
	})

	server, err := a.GetMCPServer("slack")
	if err != nil {
		t.Fatalf("GetMCPServer: %v", err)
	}
	if server.Name != "slack" {
		t.Errorf("name = %q, want %q", server.Name, "slack")
	}

	_, err = a.GetMCPServer("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent server")
	}
}
