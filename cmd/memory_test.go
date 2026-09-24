package cmd

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

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
				Warnings: []string{"fine, this is just a test warning"},
				Imports: []agent.Memory{
					{Path: "/home/u/repo/docs/architecture.md", Scope: agent.ScopeProject, Kind: agent.MemoryKindImport,
						Tier: agent.MemoryTierLaunch, Exists: true, Lines: 140, Bytes: 6100, Depth: 1,
						NotLoaded: true,
						Warnings:  []string{"import is one of its own ancestors, forming a cycle"}},
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

	// The tree is the point of this command, so a file's findings have to be
	// readable against the row they belong to, not only in a trailing list
	// keyed by a path the reader must match back by hand.
	t.Run("warnings render inline under their file", func(t *testing.T) {
		var buf bytes.Buffer
		p := output.NewPrinter(&buf, output.FormatText, true)
		if err := renderMemoryList(p, report, "/home/u", "/home/u/repo", false); err != nil {
			t.Fatalf("renderMemoryList: %v", err)
		}
		got := buf.String()

		// Inline means before the import row that follows the warned file,
		// which a bottom-only summary could never satisfy.
		inline := strings.Index(got, "this is just a test warning")
		importRow := strings.Index(got, "└─ @docs/architecture.md")
		if inline < 0 || importRow < 0 || inline > importRow {
			t.Errorf("warning not rendered under its own row (warning at %d, next row at %d):\n%s", inline, importRow, got)
		}
		// The nested node's own warning belongs under the nested row.
		cycle := strings.Index(got, "forming a cycle")
		if cycle < importRow {
			t.Errorf("import warning not rendered under the import row:\n%s", got)
		}
		// Warning rows indent to the name column so they read as belonging
		// to the row above rather than as another file.
		for _, line := range strings.Split(got, "\n") {
			// The first match is the inline row; the trailing summary repeats
			// the same text at its own indentation.
			if strings.Contains(line, "this is just a test warning") {
				if !strings.HasPrefix(line, strings.Repeat(" ", nameColStart)+"! ") {
					t.Errorf("warning row not indented to the name column: %q", line)
				}
				break
			}
		}
		// The summary names files the way the rows do, not by absolute path.
		if !strings.Contains(got, "./CLAUDE.md: fine, this is just a test warning") {
			t.Errorf("warning summary does not use the display path used by the rows:\n%s", got)
		}
		if strings.Contains(got, "/home/u/repo/CLAUDE.md:") {
			t.Errorf("warning summary still uses raw absolute paths:\n%s", got)
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

// TestMemoryColumnAlignment renders a short top-level path, a long
// top-level path, and long imports at depth 1 and depth 2, then locates
// each row's size field independently (by the exact Lines/Bytes text it
// must contain) and asserts all four start at the same offset. The
// imported basenames are long enough that even the ellipsis-plus-basename
// skeleton doesn't fit a nested row's shrunken budget, which is what
// actually overflowed the name column before elideMiddlePath bounded its
// output — a strings.Contains check, or a fixture whose long segment is
// all in the directory portion, would not have caught that.
//
// Offsets are measured in runes, not bytes: rows contain multi-byte scope
// bullets, box-drawing characters, and the ellipsis, so a byte offset
// would not agree with a rune offset even on correctly aligned output.
func TestMemoryColumnAlignment(t *testing.T) {
	report := &agent.MemoryReport{
		Files: []agent.Memory{
			{Path: "/home/u/repo/CLAUDE.md", Scope: agent.ScopeProject, Kind: agent.MemoryKindClaudeMD,
				Tier: agent.MemoryTierLaunch, Exists: true, Lines: 2, Bytes: 107},
			{Path: "/home/u/repo/0001-an-extremely-long-top-level-decision-record-filename-example.md",
				Scope: agent.ScopeProject, Kind: agent.MemoryKindClaudeMD,
				Tier: agent.MemoryTierLaunch, Exists: true, Lines: 3, Bytes: 210,
				Imports: []agent.Memory{
					{Path: "/home/u/repo/0007-a-lengthy-decision-record-filename-for-testing-alignment.md",
						Scope: agent.ScopeProject, Kind: agent.MemoryKindImport,
						Tier: agent.MemoryTierLaunch, Exists: true, Lines: 4, Bytes: 320, Depth: 1,
						Imports: []agent.Memory{
							{Path: "/home/u/repo/0012-another-lengthy-decision-record-filename-example-long.md",
								Scope: agent.ScopeProject, Kind: agent.MemoryKindImport,
								Tier: agent.MemoryTierLaunch, Exists: true, Lines: 5, Bytes: 430, Depth: 2},
						}},
				}},
		},
		LaunchFiles: 4,
	}

	var buf bytes.Buffer
	p := output.NewPrinter(&buf, output.FormatText, true)
	if err := renderMemoryList(p, report, "/home/u", "/home/u/repo", false); err != nil {
		t.Fatalf("renderMemoryList: %v", err)
	}
	lines := strings.Split(buf.String(), "\n")

	rows := []struct {
		name  string
		lines int
		bytes int64
	}{
		{"short top-level", 2, 107},
		{"long top-level", 3, 210},
		{"long import depth 1", 4, 320},
		{"long import depth 2", 5, 430},
	}

	var want int
	for i, r := range rows {
		sizeText := fmt.Sprintf("%5dL %10s", r.lines, output.FormatBytes(r.bytes))
		offset := -1
		for _, line := range lines {
			if idx := strings.Index(line, sizeText); idx >= 0 {
				offset = utf8.RuneCountInString(line[:idx]) // rune offset, not byte offset
				break
			}
		}
		if offset < 0 {
			t.Fatalf("%s: could not find its size field %q in output:\n%s", r.name, sizeText, buf.String())
		}
		if i == 0 {
			want = offset
		} else if offset != want {
			t.Errorf("%s: size column starts at rune %d, want %d (same as %s)", r.name, offset, want, rows[0].name)
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

func TestRenderMemoryListShowsInstructionMode(t *testing.T) {
	var buf bytes.Buffer
	p := output.NewPrinter(&buf, output.FormatText, true)
	report := &agent.MemoryReport{
		Files:            []agent.Memory{},
		LaunchFiles:      0,
		InstructionFiles: "claude-md-and-agents-md",
	}
	if err := renderMemoryList(p, report, "/home/u", "/home/u/repo", false); err != nil {
		t.Fatalf("renderMemoryList: %v", err)
	}
	if !strings.Contains(buf.String(), "instruction files: claude-md-and-agents-md") {
		t.Errorf("output should name the active mode:\n%s", buf.String())
	}
}
