package cmd

import (
	"bytes"
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
			"└─ @./docs/architecture.md",
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
