package claudecode_test

import (
	"os"
	"path/filepath"
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

func TestListSkills_SymlinkDetection(t *testing.T) {
	tmpDir := t.TempDir()
	skillsDir := filepath.Join(tmpDir, "skills")

	// Create a real skill directory
	realDir := filepath.Join(tmpDir, "real-skill")
	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(realDir, "SKILL.md"), []byte("---\nname: linked-skill\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create skills dir with a symlink pointing to the real skill
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realDir, filepath.Join(skillsDir, "linked-skill")); err != nil {
		t.Fatal(err)
	}

	a := claudecode.NewAgent(claudecode.Paths{
		ConfigFile:   "testdata/claude.json",
		SettingsFile: "testdata/settings.json",
		MCPFile:      "testdata/mcp.json",
		HomeDir:      "testdata",
		SkillsDir:    skillsDir,
	})

	skills, err := a.ListSkills()
	if err != nil {
		t.Fatalf("ListSkills: %v", err)
	}

	var found bool
	for _, s := range skills {
		if s.Name == "linked-skill" {
			found = true
			if s.LinkTarget == "" {
				t.Error("expected LinkTarget to be set for symlinked skill")
			}
			if s.LinkTarget != realDir {
				t.Errorf("LinkTarget = %q, want %q", s.LinkTarget, realDir)
			}
		}
	}
	if !found {
		t.Error("symlinked skill not discovered")
	}
}
