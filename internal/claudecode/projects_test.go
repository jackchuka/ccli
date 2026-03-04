package claudecode_test

import (
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
