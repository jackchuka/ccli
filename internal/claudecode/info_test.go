package claudecode_test

import (
	"testing"

	"github.com/jackchuka/ccli/internal/claudecode"
)

func TestInfo(t *testing.T) {
	t.Setenv("ANTHROPIC_MODEL", "")

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

func TestResolveModel(t *testing.T) {
	tests := []struct {
		name          string
		envModel      string
		settingsModel string
		want          string
	}{
		{
			name:          "settings model when set",
			settingsModel: "opus",
			want:          "opus",
		},
		{
			name:     "env var used when settings empty",
			envModel: "sonnet",
			want:     "sonnet",
		},
		{
			name:          "env var takes precedence over settings",
			envModel:      "haiku",
			settingsModel: "opus",
			want:          "haiku",
		},
		{
			name: "fallback when nothing set",
			want: "default (recommended)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("ANTHROPIC_MODEL", tt.envModel)
			got := claudecode.ResolveModel(tt.settingsModel)
			if got != tt.want {
				t.Errorf("ResolveModel(%q) with env %q = %q, want %q",
					tt.settingsModel, tt.envModel, got, tt.want)
			}
		})
	}
}
