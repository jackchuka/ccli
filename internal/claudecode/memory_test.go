package claudecode_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jackchuka/ccli/internal/agent"
	"github.com/jackchuka/ccli/internal/claudecode"
)

func TestGlobMatch(t *testing.T) {
	tests := []struct {
		pattern string
		path    string
		want    bool
	}{
		{"**/monorepo/CLAUDE.md", "/home/u/monorepo/CLAUDE.md", true},
		{"**/monorepo/CLAUDE.md", "/home/u/monorepo/sub/CLAUDE.md", false},
		{"/home/u/other/.claude/rules/**", "/home/u/other/.claude/rules/a/b.md", true},
		{"/home/u/other/.claude/rules/**", "/home/u/other/.claude/CLAUDE.md", false},
		{"*.md", "/home/u/CLAUDE.md", false},
		{"/home/u/*.md", "/home/u/CLAUDE.md", true},
		{"/home/u/*.md", "/home/u/sub/CLAUDE.md", false},
		{"/home/u/CLAUDE.md", "/home/u/CLAUDE.md", true},
		{"/home/u/CLAUDE.?d", "/home/u/CLAUDE.md", true},
		{"/home/u/a+b/CLAUDE.md", "/home/u/a+b/CLAUDE.md", true},
	}
	for _, tt := range tests {
		t.Run(tt.pattern+" vs "+tt.path, func(t *testing.T) {
			if got := claudecode.GlobMatch(tt.pattern, tt.path); got != tt.want {
				t.Errorf("GlobMatch(%q, %q) = %v, want %v", tt.pattern, tt.path, got, tt.want)
			}
		})
	}
}

func TestExpandTilde(t *testing.T) {
	if got := claudecode.ExpandTilde("~/mem", "/home/u"); got != "/home/u/mem" {
		t.Errorf("got %q", got)
	}
	if got := claudecode.ExpandTilde("/abs/mem", "/home/u"); got != "/abs/mem" {
		t.Errorf("got %q", got)
	}
	if got := claudecode.ExpandTilde("~notme/mem", "/home/u"); got != "~notme/mem" {
		t.Errorf("got %q", got)
	}
}

func TestReadMemoryFile(t *testing.T) {
	dir := t.TempDir()

	present := filepath.Join(dir, "CLAUDE.md")
	if err := os.WriteFile(present, []byte("line one\nline two\nline three\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "linked.md")
	if err := os.Symlink(present, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	t.Run("existing file", func(t *testing.T) {
		m := claudecode.ReadMemoryFile(present, agent.ScopeProject, agent.MemoryKindClaudeMD, agent.MemoryTierLaunch)
		if !m.Exists {
			t.Fatal("Exists = false")
		}
		if m.Lines != 3 {
			t.Errorf("Lines = %d, want 3", m.Lines)
		}
		if m.Bytes != 29 {
			t.Errorf("Bytes = %d, want 29", m.Bytes)
		}
		if m.Scope != agent.ScopeProject || m.Kind != agent.MemoryKindClaudeMD || m.Tier != agent.MemoryTierLaunch {
			t.Errorf("metadata not carried through: %+v", m)
		}
	})

	t.Run("symlink records target", func(t *testing.T) {
		m := claudecode.ReadMemoryFile(link, agent.ScopePersonal, agent.MemoryKindClaudeMD, agent.MemoryTierLaunch)
		if m.LinkTarget != present {
			t.Errorf("LinkTarget = %q, want %q", m.LinkTarget, present)
		}
		if m.Lines != 3 {
			t.Errorf("Lines = %d, want 3 (symlink should be followed)", m.Lines)
		}
	})

	t.Run("absent file", func(t *testing.T) {
		m := claudecode.ReadMemoryFile(filepath.Join(dir, "nope.md"), agent.ScopeLocal, agent.MemoryKindClaudeLocalMD, agent.MemoryTierLaunch)
		if m.Exists {
			t.Error("Exists = true for a missing file")
		}
		if m.Lines != 0 || m.Bytes != 0 {
			t.Errorf("missing file should report zero size, got %dL %dB", m.Lines, m.Bytes)
		}
	})

	t.Run("file with no trailing newline counts its last line", func(t *testing.T) {
		p := filepath.Join(dir, "noeol.md")
		if err := os.WriteFile(p, []byte("a\nb"), 0o644); err != nil {
			t.Fatal(err)
		}
		m := claudecode.ReadMemoryFile(p, agent.ScopeProject, agent.MemoryKindClaudeMD, agent.MemoryTierLaunch)
		if m.Lines != 2 {
			t.Errorf("Lines = %d, want 2", m.Lines)
		}
	})
}
