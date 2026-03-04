package claudecode_test

import (
	"testing"

	"github.com/jackchuka/ccli/internal/claudecode"
)

func TestInfo(t *testing.T) {
	a := claudecode.NewAgent(claudecode.Paths{
		ConfigFile:   "testdata/claude.json",
		SettingsFile: "testdata/settings.json",
		MCPFile:      "testdata/mcp.json",
		HomeDir:      "testdata",
		SkillsDir:    "testdata/skills",
		PluginsDir:   "testdata/plugins",
		ProjectDir:   "testdata/project",
	})

	info, err := a.Info()
	if err != nil {
		t.Fatalf("Info: %v", err)
	}

	if info.Model != "opus" {
		t.Errorf("model = %q, want %q", info.Model, "opus")
	}
	if info.MCPCount < 1 {
		t.Errorf("mcpCount = %d, want >= 1", info.MCPCount)
	}
	if info.SkillCount < 1 {
		t.Errorf("skillCount = %d, want >= 1", info.SkillCount)
	}
}
