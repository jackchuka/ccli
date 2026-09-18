package cmd

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/jackchuka/ccli/internal/agent"
	"github.com/jackchuka/ccli/internal/output"
)

func TestMemoryDisplayPath(t *testing.T) {
	tests := []struct {
		path, home, cwd, want string
	}{
		{"/home/u/repo/CLAUDE.md", "/home/u", "/home/u/repo", "./CLAUDE.md"},
		{"/home/u/repo/docs/a.md", "/home/u", "/home/u/repo", "./docs/a.md"},
		{"/home/u/.claude/CLAUDE.md", "/home/u", "/home/u/repo", "~/.claude/CLAUDE.md"},
		{"/etc/claude-code/CLAUDE.md", "/home/u", "/home/u/repo", "/etc/claude-code/CLAUDE.md"},
		{"", "/home/u", "/home/u/repo", "(inline)"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := memoryDisplayPath(tt.path, tt.home, tt.cwd); got != tt.want {
				t.Errorf("memoryDisplayPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestRenderMemoryList(t *testing.T) {
	report := &agent.MemoryReport{
		Files: []agent.Memory{
			{Path: "/home/u/.claude/CLAUDE.md", Scope: agent.ScopePersonal, Kind: agent.MemoryKindClaudeMD,
				Tier: agent.MemoryTierLaunch, Exists: true, Lines: 31, Bytes: 1200, LinkTarget: "/dotfiles/CLAUDE.md"},
			{Path: "/home/u/repo/CLAUDE.md", Scope: agent.ScopeProject, Kind: agent.MemoryKindClaudeMD,
				Tier: agent.MemoryTierLaunch, Exists: true, Lines: 88, Bytes: 3400,
				Imports: []agent.Memory{
					{Path: "/home/u/repo/docs/architecture.md", Scope: agent.ScopeProject, Kind: agent.MemoryKindImport,
						Tier: agent.MemoryTierLaunch, Exists: true, Lines: 140, Bytes: 6100, Depth: 1},
				}},
			{Path: "/home/u/repo/CLAUDE.local.md", Scope: agent.ScopeLocal, Kind: agent.MemoryKindClaudeLocalMD,
				Tier: agent.MemoryTierLaunch, Exists: false},
			{Path: "/home/u/repo/pkg/CLAUDE.md", Scope: agent.ScopeProject, Kind: agent.MemoryKindClaudeMD,
				Tier: agent.MemoryTierOnDemand, Exists: true, Lines: 10, Bytes: 200},
		},
		LaunchFiles:   3,
		LaunchLines:   259,
		LaunchBytes:   10700,
		OnDemandFiles: 1,
		Warnings:      []string{"/home/u/repo/CLAUDE.md: 88 lines: fine, this is just a test warning"},
	}

	t.Run("default output", func(t *testing.T) {
		var buf bytes.Buffer
		p := output.NewPrinter(&buf, output.FormatText, true)
		if err := renderMemoryList(p, report, "/home/u", "/home/u/repo", false); err != nil {
			t.Fatalf("renderMemoryList: %v", err)
		}
		got := buf.String()

		for _, want := range []string{
			"Memory",
			"~/.claude/CLAUDE.md",
			"./CLAUDE.md",
			"└─ @docs/architecture.md",
			"(absent)",
			"loaded at launch: 3 files",
			"259L",
			"on demand:",
			"⇢ /dotfiles/CLAUDE.md",
			"this is just a test warning",
		} {
			if !strings.Contains(got, want) {
				t.Errorf("output missing %q:\n%s", want, got)
			}
		}
		// On-demand files are summarized, not listed, without the flag.
		if strings.Contains(got, "./pkg/CLAUDE.md") {
			t.Errorf("on-demand file listed without --on-demand:\n%s", got)
		}
		// The personal file loads before the project file.
		if strings.Index(got, "~/.claude/CLAUDE.md") > strings.Index(got, "./CLAUDE.md") {
			t.Error("load order not preserved in output")
		}
	})

	t.Run("--on-demand lists them", func(t *testing.T) {
		var buf bytes.Buffer
		p := output.NewPrinter(&buf, output.FormatText, true)
		if err := renderMemoryList(p, report, "/home/u", "/home/u/repo", true); err != nil {
			t.Fatalf("renderMemoryList: %v", err)
		}
		got := buf.String()
		if !strings.Contains(got, "On demand") || !strings.Contains(got, "./pkg/CLAUDE.md") {
			t.Errorf("--on-demand did not list on-demand files:\n%s", got)
		}
	})

	t.Run("--on-demand with none prints a placeholder", func(t *testing.T) {
		empty := &agent.MemoryReport{
			Files: []agent.Memory{
				{Path: "/home/u/repo/CLAUDE.md", Scope: agent.ScopeProject, Kind: agent.MemoryKindClaudeMD,
					Tier: agent.MemoryTierLaunch, Exists: true, Lines: 1, Bytes: 10},
			},
			LaunchFiles: 1,
			LaunchLines: 1,
			LaunchBytes: 10,
		}
		var buf bytes.Buffer
		p := output.NewPrinter(&buf, output.FormatText, true)
		if err := renderMemoryList(p, empty, "/home/u", "/home/u/repo", true); err != nil {
			t.Fatalf("renderMemoryList: %v", err)
		}
		got := buf.String()
		if !strings.Contains(got, "On demand") || !strings.Contains(got, "(none)") {
			t.Errorf("empty --on-demand section did not print a placeholder:\n%s", got)
		}
		// Singular counts must not read "1 files".
		if !strings.Contains(got, "1 file ") {
			t.Errorf("singular count not pluralized correctly:\n%s", got)
		}
	})
}

// TestMemoryColumnAlignment renders a short path, a long path that would
// overflow the fixed-width name column unelided, and a nested import, and
// asserts the size column begins at the same rune index on every row. A
// strings.Contains check cannot catch a column that has drifted right.
func TestMemoryColumnAlignment(t *testing.T) {
	longPath := "/home/u/repo/" + strings.Repeat("very-long-directory-name/", 4) + "CLAUDE.md"
	report := &agent.MemoryReport{
		Files: []agent.Memory{
			{Path: "/home/u/repo/CLAUDE.md", Scope: agent.ScopeProject, Kind: agent.MemoryKindClaudeMD,
				Tier: agent.MemoryTierLaunch, Exists: true, Lines: 88, Bytes: 3400},
			{Path: longPath, Scope: agent.ScopeProject, Kind: agent.MemoryKindClaudeMD,
				Tier: agent.MemoryTierLaunch, Exists: true, Lines: 5, Bytes: 100,
				Imports: []agent.Memory{
					{Path: "/home/u/repo/" + strings.Repeat("nested-import-dir/", 3) + "deep.md",
						Scope: agent.ScopeProject, Kind: agent.MemoryKindImport,
						Tier: agent.MemoryTierLaunch, Exists: true, Lines: 10, Bytes: 50, Depth: 1},
				}},
		},
		LaunchFiles: 2,
	}

	var buf bytes.Buffer
	p := output.NewPrinter(&buf, output.FormatText, true)
	if err := renderMemoryList(p, report, "/home/u", "/home/u/repo", false); err != nil {
		t.Fatalf("renderMemoryList: %v", err)
	}

	// Mirrors renderMemoryNode's format string: 2 leading spaces + a
	// 1-rune bullet + 1 space + the 9-wide scope field + 1 space + the
	// name column + 1 space, before the size text begins.
	const sizeCol = 2 + 1 + 1 + 9 + 1 + nameColWidth + 1

	wants := map[string]string{
		"./CLAUDE.md":       fmt.Sprintf("%5dL %10s", 88, output.FormatBytes(3400)),
		"very-long":         fmt.Sprintf("%5dL %10s", 5, output.FormatBytes(100)),
		"nested-import-dir": fmt.Sprintf("%5dL %10s", 10, output.FormatBytes(50)),
	}
	found := map[string]bool{}
	for _, line := range strings.Split(buf.String(), "\n") {
		runes := []rune(line)
		for marker, want := range wants {
			if !strings.Contains(line, marker) {
				continue
			}
			found[marker] = true
			if len(runes) < sizeCol+len([]rune(want)) {
				t.Errorf("line for %q too short to hold the size column at %d:\n%q", marker, sizeCol, line)
				continue
			}
			got := string(runes[sizeCol : sizeCol+len([]rune(want))])
			if got != want {
				t.Errorf("size column misaligned for %q: got %q at rune %d, want %q\nfull line: %q", marker, got, sizeCol, want, line)
			}
		}
	}
	for marker := range wants {
		if !found[marker] {
			t.Fatalf("expected a rendered row containing %q, got:\n%s", marker, buf.String())
		}
	}
}

func TestRenderMemoryGet(t *testing.T) {
	var buf bytes.Buffer
	p := output.NewPrinter(&buf, output.FormatText, true)
	m := &agent.Memory{
		Path: "/home/u/repo/CLAUDE.md", Scope: agent.ScopeProject, Kind: agent.MemoryKindClaudeMD,
		Tier: agent.MemoryTierLaunch, Exists: true, Lines: 88, Bytes: 3400,
		Excluded: true, ExcludedBy: "**/repo/CLAUDE.md",
		Imports: []agent.Memory{{Path: "/home/u/repo/docs/a.md", Exists: true}},
	}
	if err := renderMemoryGet(p, m, "/home/u", "/home/u/repo"); err != nil {
		t.Fatalf("renderMemoryGet: %v", err)
	}
	got := buf.String()
	for _, want := range []string{"Path", "./CLAUDE.md", "Scope", "project", "Kind", "claude-md",
		"Tier", "launch", "Lines", "88", "Excluded", "**/repo/CLAUDE.md", "Imports", "./docs/a.md"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
}

func TestRenderMemoryGetWarnings(t *testing.T) {
	var buf bytes.Buffer
	p := output.NewPrinter(&buf, output.FormatText, true)
	m := &agent.Memory{
		Path: "/home/u/repo/CLAUDE.md", Scope: agent.ScopeProject, Kind: agent.MemoryKindClaudeMD,
		Tier: agent.MemoryTierLaunch, Exists: true, Lines: 900, Bytes: 40000,
		Warnings: []string{"/home/u/repo/CLAUDE.md: 900 lines: large enough to crowd out context"},
	}
	if err := renderMemoryGet(p, m, "/home/u", "/home/u/repo"); err != nil {
		t.Fatalf("renderMemoryGet: %v", err)
	}
	got := buf.String()
	if !strings.Contains(got, "! /home/u/repo/CLAUDE.md: 900 lines") {
		t.Errorf("output missing rendered warning:\n%s", got)
	}
}
