package claudecode_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jackchuka/ccli/internal/claudecode"
)

func TestListProjects(t *testing.T) {
	a := claudecode.NewAgent(claudecode.Paths{
		ConfigFile: "testdata/claude.json",
		HomeDir:    "testdata",
	})

	projects, err := a.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}

	if len(projects) != 1 {
		t.Fatalf("got %d projects, want 1", len(projects))
	}

	p := projects[0]
	if p.Name != "project" {
		t.Errorf("name = %q, want %q", p.Name, "project")
	}
	if p.Path != "/Users/test/project" {
		t.Errorf("path = %q, want %q", p.Path, "/Users/test/project")
	}
	if !p.Trusted {
		t.Error("expected project to be trusted")
	}
}

func TestGetProject(t *testing.T) {
	a := claudecode.NewAgent(claudecode.Paths{
		ConfigFile: "testdata/claude.json",
		HomeDir:    "testdata",
	})

	t.Run("by full path", func(t *testing.T) {
		p, err := a.GetProject("/Users/test/project")
		if err != nil {
			t.Fatalf("GetProject: %v", err)
		}
		if p.Name != "project" {
			t.Errorf("name = %q, want %q", p.Name, "project")
		}
	})

	t.Run("by name", func(t *testing.T) {
		p, err := a.GetProject("project")
		if err != nil {
			t.Fatalf("GetProject: %v", err)
		}
		if p.Path != "/Users/test/project" {
			t.Errorf("path = %q, want %q", p.Path, "/Users/test/project")
		}
	})

	t.Run("by suffix", func(t *testing.T) {
		p, err := a.GetProject("test/project")
		if err != nil {
			t.Fatalf("GetProject: %v", err)
		}
		if p.Path != "/Users/test/project" {
			t.Errorf("path = %q, want %q", p.Path, "/Users/test/project")
		}
	})

	t.Run("nonexistent", func(t *testing.T) {
		_, err := a.GetProject("nonexistent")
		if err == nil {
			t.Fatal("expected error for nonexistent project")
		}
	})
}

func TestListProjectsNoConfig(t *testing.T) {
	a := claudecode.NewAgent(claudecode.Paths{
		ConfigFile: "testdata/nonexistent.json",
		HomeDir:    "testdata",
	})

	projects, err := a.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("got %d projects, want 0", len(projects))
	}
}

func TestSessionCountsSkipNonSessionDirs(t *testing.T) {
	root := t.TempDir()
	claudeHome := filepath.Join(root, ".claude")
	projectPath := "/work/app"
	projDir := filepath.Join(claudeHome, "projects", "-work-app")
	for _, dir := range []string{
		filepath.Join(projDir, "memory"),
		filepath.Join(projDir, "1b6f2a10-0000-4000-8000-000000000001", "subagents"),
		filepath.Join(projDir, "1b6f2a10-0000-4000-8000-000000000003", "tool-results"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range []string{
		"1b6f2a10-0000-4000-8000-000000000001.jsonl",
		"1b6f2a10-0000-4000-8000-000000000002.jsonl",
		"memory/MEMORY.md",
		"notes.txt",
	} {
		if err := os.WriteFile(filepath.Join(projDir, f), []byte("{}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cfg := filepath.Join(root, ".claude.json")
	if err := os.WriteFile(cfg, []byte(`{"projects": {"`+projectPath+`": {}}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	a := claudecode.NewAgent(claudecode.Paths{ConfigFile: cfg, HomeDir: claudeHome, UserHomeDir: root, CWD: root})
	const want = 3

	projects, err := a.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(projects) != 1 || projects[0].SessionCount != want {
		t.Errorf("ListProjects sessions = %+v, want one project with %d", projects, want)
	}
	p, err := a.GetProject("app")
	if err != nil {
		t.Fatalf("GetProject: %v", err)
	}
	if p.SessionCount != want {
		t.Errorf("GetProject sessions = %d, want %d", p.SessionCount, want)
	}
	info, err := a.Info()
	if err != nil {
		t.Fatalf("Info: %v", err)
	}
	if info.SessionCount != want {
		t.Errorf("Info sessions = %d, want %d", info.SessionCount, want)
	}
}
