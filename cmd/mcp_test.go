package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackchuka/ccli/internal/agent"
)

func runCLI(t *testing.T, args ...string) string {
	t.Helper()
	format, noColor = "text", false
	t.Cleanup(func() { format, noColor = "text", false })

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout := os.Stdout
	os.Stdout = w
	rootCmd.SetArgs(args)
	runErr := rootCmd.Execute()
	os.Stdout = stdout
	_ = w.Close()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	if runErr != nil {
		t.Fatalf("ccli %v: %v", args, runErr)
	}
	return buf.String()
}

func TestMCPStructuredOutputMasksSecrets(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Chdir(home)
	cfg := `{"mcpServers": {"pg": {"command": "mcp-postgres", "env": {"PGPASSWORD": "hunter2", "PGHOST": "localhost"}}}}`
	if err := os.WriteFile(filepath.Join(home, ".claude.json"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, args := range [][]string{
		{"mcp", "get", "pg", "--format", "json"},
		{"mcp", "list", "--format", "json"},
	} {
		out := runCLI(t, args...)
		var servers []agent.MCPServer
		if args[1] == "get" {
			var s agent.MCPServer
			if err := json.Unmarshal([]byte(out), &s); err != nil {
				t.Fatalf("%v: decoding %q: %v", args, out, err)
			}
			servers = append(servers, s)
		} else if err := json.Unmarshal([]byte(out), &servers); err != nil {
			t.Fatalf("%v: decoding %q: %v", args, out, err)
		}
		if len(servers) != 1 {
			t.Fatalf("%v: got %d servers, want 1", args, len(servers))
		}
		env := servers[0].Env
		if env["PGPASSWORD"] == "hunter2" {
			t.Errorf("%v: PGPASSWORD printed unmasked", args)
		}
		if env["PGHOST"] != "localhost" {
			t.Errorf("%v: PGHOST = %q, want it unmasked", args, env["PGHOST"])
		}
	}
}
