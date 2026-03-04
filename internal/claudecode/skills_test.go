package claudecode_test

import (
	"testing"

	"github.com/jackchuka/ccli/internal/agent"
	"github.com/jackchuka/ccli/internal/claudecode"
)

func TestListSkills(t *testing.T) {
	a := claudecode.NewAgent(claudecode.Paths{
		ConfigFile:   "testdata/claude.json",
		SettingsFile: "testdata/settings.json",
		MCPFile:      "testdata/mcp.json",
		HomeDir:      "testdata",
		SkillsDir:    "testdata/skills",
		PluginsDir:   "testdata/plugins",
		ProjectDir:   "testdata/project",
	})

	skills, err := a.ListSkills()
	if err != nil {
		t.Fatalf("ListSkills: %v", err)
	}

	if len(skills) < 2 {
		t.Fatalf("got %d skills, want at least 2", len(skills))
	}

	scopes := map[agent.Scope]int{}
	for _, s := range skills {
		scopes[s.Scope]++
	}
	if scopes[agent.ScopePersonal] < 1 {
		t.Errorf("personal skills = %d, want >= 1", scopes[agent.ScopePersonal])
	}
	if scopes[agent.ScopePlugin] < 1 {
		t.Errorf("plugin skills = %d, want >= 1", scopes[agent.ScopePlugin])
	}
}

func TestGetSkill(t *testing.T) {
	a := claudecode.NewAgent(claudecode.Paths{
		ConfigFile:   "testdata/claude.json",
		SettingsFile: "testdata/settings.json",
		MCPFile:      "testdata/mcp.json",
		HomeDir:      "testdata",
		SkillsDir:    "testdata/skills",
		PluginsDir:   "testdata/plugins",
		ProjectDir:   "testdata/project",
	})

	skill, err := a.GetSkill("test-skill")
	if err != nil {
		t.Fatalf("GetSkill: %v", err)
	}
	if skill.Name != "test-skill" {
		t.Errorf("name = %q, want %q", skill.Name, "test-skill")
	}
	if skill.Description != "A test skill for unit tests" {
		t.Errorf("description = %q, want %q", skill.Description, "A test skill for unit tests")
	}

	_, err = a.GetSkill("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent skill")
	}
}
