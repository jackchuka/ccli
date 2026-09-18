package claudecode_test

import (
	"os"
	"path/filepath"
	"strings"
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

// memoryFixture builds a working tree with memory files at several levels and
// returns the root plus a Paths pointing at a nested working directory.
func memoryFixture(t *testing.T) (string, claudecode.Paths) {
	t.Helper()
	root := t.TempDir()

	home := filepath.Join(root, "home")
	claudeHome := filepath.Join(home, ".claude")
	repo := filepath.Join(root, "repo")
	sub := filepath.Join(repo, "pkg", "api")
	for _, d := range []string{claudeHome, filepath.Join(repo, ".claude"), sub} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	write := func(path, body string) {
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(claudeHome, "CLAUDE.md"), "user rules\n")
	write(filepath.Join(repo, "CLAUDE.md"), "repo rules\n")
	write(filepath.Join(repo, "CLAUDE.local.md"), "repo local\n")
	write(filepath.Join(repo, ".claude", "CLAUDE.md"), "dot claude rules\n")
	write(filepath.Join(sub, "CLAUDE.md"), "api rules\n")

	return root, claudecode.Paths{
		SettingsFile: filepath.Join(claudeHome, "settings.json"),
		HomeDir:      claudeHome,
		UserHomeDir:  home,
		CWD:          sub,
		ProjectDir:   filepath.Join(sub, ".claude"),
	}
}

func TestLaunchTierOrdering(t *testing.T) {
	_, paths := memoryFixture(t)
	a := claudecode.NewAgent(paths)

	files := a.LaunchTierFiles()

	var present []string
	for _, f := range files {
		if f.Exists {
			present = append(present, f.Path)
		}
	}
	if len(present) < 4 {
		t.Fatalf("expected at least 4 existing launch files, got %v", present)
	}

	// Personal comes before anything in the repo.
	if !strings.Contains(present[0], filepath.Join(".claude", "CLAUDE.md")) {
		t.Errorf("first launch file = %q, want the user CLAUDE.md", present[0])
	}
	// Ancestors are ordered root -> cwd, so repo/CLAUDE.md precedes pkg/api/CLAUDE.md.
	repoIdx, apiIdx := -1, -1
	for i, p := range present {
		if strings.HasSuffix(p, filepath.Join("repo", "CLAUDE.md")) {
			repoIdx = i
		}
		if strings.HasSuffix(p, filepath.Join("api", "CLAUDE.md")) {
			apiIdx = i
		}
	}
	if repoIdx < 0 || apiIdx < 0 || repoIdx > apiIdx {
		t.Errorf("ancestor order wrong: repo at %d, api at %d", repoIdx, apiIdx)
	}
}

func TestLaunchTierLocalAfterMain(t *testing.T) {
	_, paths := memoryFixture(t)
	a := claudecode.NewAgent(paths)

	mainIdx, localIdx := -1, -1
	for i, f := range a.LaunchTierFiles() {
		if strings.HasSuffix(f.Path, filepath.Join("repo", "CLAUDE.md")) {
			mainIdx = i
		}
		if strings.HasSuffix(f.Path, "CLAUDE.local.md") && f.Exists {
			localIdx = i
		}
	}
	if mainIdx < 0 || localIdx < 0 || mainIdx > localIdx {
		t.Errorf("CLAUDE.local.md must follow CLAUDE.md in the same directory: %d vs %d", mainIdx, localIdx)
	}
}

func TestLaunchTierFindsBothProjectLocations(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "CLAUDE.md"), []byte("root\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".claude", "CLAUDE.md"), []byte("dot claude\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	a := claudecode.NewAgent(claudecode.Paths{UserHomeDir: root, CWD: root})

	var found []string
	for _, f := range a.LaunchTierFiles() {
		if f.Exists {
			found = append(found, f.Path)
		}
	}
	if len(found) != 2 {
		t.Fatalf("got %v, want both ./CLAUDE.md and ./.claude/CLAUDE.md", found)
	}
	// ./CLAUDE.md is listed before ./.claude/CLAUDE.md.
	if filepath.Dir(found[0]) != root {
		t.Errorf("first = %q, want the root CLAUDE.md", found[0])
	}
}

func TestLaunchTierReportsAbsentExpectedPaths(t *testing.T) {
	root := t.TempDir()
	a := claudecode.NewAgent(claudecode.Paths{
		HomeDir:     filepath.Join(root, "home", ".claude"),
		UserHomeDir: filepath.Join(root, "home"),
		CWD:         root,
	})
	var absent int
	for _, f := range a.LaunchTierFiles() {
		if !f.Exists {
			absent++
		}
	}
	if absent == 0 {
		t.Error("expected absent user/project/local paths to be reported")
	}
}

func TestLaunchTierManagedInline(t *testing.T) {
	dir := t.TempDir()
	managed := filepath.Join(dir, "managed-settings.json")
	if err := os.WriteFile(managed, []byte(`{"claudeMd":"Never push to main."}`), 0o644); err != nil {
		t.Fatal(err)
	}
	a := claudecode.NewAgent(claudecode.Paths{
		ManagedSettingsFile: managed,
		UserHomeDir:         dir,
		CWD:                 dir,
	})
	found := false
	for _, f := range a.LaunchTierFiles() {
		if f.Kind == agent.MemoryKindManagedInline {
			found = true
			if f.Scope != agent.ScopeManaged || !f.Exists || f.Lines != 1 {
				t.Errorf("managed inline entry wrong: %+v", f)
			}
		}
	}
	if !found {
		t.Error("managed claudeMd string not reported")
	}
}

func TestParseImports(t *testing.T) {
	content := "See @README for the overview.\n" +
		"- git workflow @docs/git.md\n" +
		"Mention `@notanimport` stays literal.\n" +
		"```\n@fenced/also-literal.md\n```\n" +
		"@~/.claude/shared.md\n"

	got := claudecode.ParseImports(content)
	want := []string{"README", "docs/git.md", "~/.claude/shared.md"}

	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("import %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestExpandImportsNestsAndLimitsDepth(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// A five-deep chain: root imports 1, 1 imports 2, ... 4 imports 5.
	write("CLAUDE.md", "@one.md\n")
	write("one.md", "@two.md\n")
	write("two.md", "@three.md\n")
	write("three.md", "@four.md\n")
	write("four.md", "@five.md\n")
	// five.md's own import must never be read: expansion stops at five.
	write("five.md", "@six.md\n")

	a := claudecode.NewAgent(claudecode.Paths{CWD: dir, UserHomeDir: dir})
	root := claudecode.ReadMemoryFile(filepath.Join(dir, "CLAUDE.md"), agent.ScopeProject, agent.MemoryKindClaudeMD, agent.MemoryTierLaunch)
	expanded := a.ExpandInto([]agent.Memory{root})

	// Walk the chain, pinning Depth at every hop so an off-by-one at any
	// level other than the deepest one would be caught.
	node := expanded[0]
	for i := 1; i <= claudecode.MaxImportDepth(); i++ {
		if len(node.Imports) != 1 {
			t.Fatalf("hop %d: want 1 import, got %d: %+v", i, len(node.Imports), node)
		}
		node = node.Imports[0]
		if node.Depth != i {
			t.Errorf("node at hop %d has Depth = %d, want %d", i, node.Depth, i)
		}
	}

	// node is now the file at maxImportDepth (four.md). Its import (five.md)
	// is recorded, past the limit, and not expanded.
	if len(node.Imports) != 1 {
		t.Fatalf("want the depth-limit file to record its dropped import, got %d imports", len(node.Imports))
	}
	dropped := node.Imports[0]
	if dropped.Depth != claudecode.MaxImportDepth()+1 {
		t.Errorf("dropped import Depth = %d, want %d", dropped.Depth, claudecode.MaxImportDepth()+1)
	}
	if len(dropped.Warnings) == 0 {
		t.Errorf("dropped import should warn that the depth limit is exceeded: %+v", dropped)
	}
	if len(dropped.Imports) != 0 {
		t.Errorf("dropped import's own @six.md must never be read: %+v", dropped)
	}
}

func TestExpandImportsDiamondIsNotACycle(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// Two sibling branches import the same file: a diamond, not a cycle.
	write("CLAUDE.md", "@left.md\n@right.md\n")
	write("left.md", "@shared.md\n")
	write("right.md", "@shared.md\n")
	write("shared.md", "shared content\n")

	a := claudecode.NewAgent(claudecode.Paths{CWD: dir, UserHomeDir: dir})
	root := claudecode.ReadMemoryFile(filepath.Join(dir, "CLAUDE.md"), agent.ScopeProject, agent.MemoryKindClaudeMD, agent.MemoryTierLaunch)
	expanded := a.ExpandInto([]agent.Memory{root})

	if len(expanded[0].Imports) != 2 {
		t.Fatalf("want 2 imports, got %d", len(expanded[0].Imports))
	}
	for _, branch := range expanded[0].Imports {
		if len(branch.Imports) != 1 {
			t.Fatalf("branch %q should import shared.md once, got %d", branch.Path, len(branch.Imports))
		}
		shared := branch.Imports[0]
		if !shared.Exists || len(shared.Imports) != 0 {
			t.Errorf("shared.md under %q should be expanded normally: %+v", branch.Path, shared)
		}
		if len(shared.Warnings) != 0 {
			t.Errorf("shared.md reached via two siblings is a diamond, not a cycle, and must not warn: %+v", shared)
		}
	}
}

func TestExpandImportsResolvesRelativeToImportingFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "docs", "adr"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("@docs/architecture.md\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// The nested import is relative to docs/, not to the working directory.
	if err := os.WriteFile(filepath.Join(dir, "docs", "architecture.md"), []byte("@adr/0003.md\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "docs", "adr", "0003.md"), []byte("decision\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	a := claudecode.NewAgent(claudecode.Paths{CWD: dir, UserHomeDir: dir})
	root := claudecode.ReadMemoryFile(filepath.Join(dir, "CLAUDE.md"), agent.ScopeProject, agent.MemoryKindClaudeMD, agent.MemoryTierLaunch)
	expanded := a.ExpandInto([]agent.Memory{root})

	if len(expanded[0].Imports) != 1 {
		t.Fatalf("want 1 import, got %d", len(expanded[0].Imports))
	}
	nested := expanded[0].Imports[0]
	if len(nested.Imports) != 1 || !nested.Imports[0].Exists {
		t.Fatalf("nested import not resolved relative to its importer: %+v", nested)
	}
	if nested.Imports[0].Depth != 2 {
		t.Errorf("nested Depth = %d, want 2", nested.Imports[0].Depth)
	}
}

func TestExpandImportsCycleAndMissing(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("@loop.md\n@gone.md\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "loop.md"), []byte("@CLAUDE.md\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	a := claudecode.NewAgent(claudecode.Paths{CWD: dir, UserHomeDir: dir})
	root := claudecode.ReadMemoryFile(filepath.Join(dir, "CLAUDE.md"), agent.ScopeProject, agent.MemoryKindClaudeMD, agent.MemoryTierLaunch)
	expanded := a.ExpandInto([]agent.Memory{root}) // must terminate

	if len(expanded[0].Imports) != 2 {
		t.Fatalf("want 2 imports, got %d", len(expanded[0].Imports))
	}
	loop, gone := expanded[0].Imports[0], expanded[0].Imports[1]
	if len(loop.Imports) != 1 || len(loop.Imports[0].Warnings) == 0 {
		t.Errorf("cycle should be recorded once with a warning: %+v", loop)
	}
	if gone.Exists || len(gone.Warnings) == 0 {
		t.Errorf("missing import should be absent and warned: %+v", gone)
	}
}

func TestExpandImportsFlagsExternal(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	outside := filepath.Join(root, "outside")
	for _, d := range []string{repo, outside} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(outside, "shared.md"), []byte("shared\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "CLAUDE.md"), []byte("@"+filepath.Join(outside, "shared.md")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	a := claudecode.NewAgent(claudecode.Paths{CWD: repo, UserHomeDir: root})
	rootFile := claudecode.ReadMemoryFile(filepath.Join(repo, "CLAUDE.md"), agent.ScopeProject, agent.MemoryKindClaudeMD, agent.MemoryTierLaunch)
	expanded := a.ExpandInto([]agent.Memory{rootFile})

	imp := expanded[0].Imports[0]
	if !imp.External || len(imp.Warnings) == 0 {
		t.Errorf("import outside the working directory should be External and warned: %+v", imp)
	}

	// A personal-scope file's imports are trusted, so they are not flagged.
	personal := claudecode.ReadMemoryFile(filepath.Join(repo, "CLAUDE.md"), agent.ScopePersonal, agent.MemoryKindClaudeMD, agent.MemoryTierLaunch)
	personalExpanded := a.ExpandInto([]agent.Memory{personal})
	if personalExpanded[0].Imports[0].External {
		t.Error("personal-scope imports must not be flagged external")
	}
}

func TestApplyExclusions(t *testing.T) {
	files := []agent.Memory{
		{Path: "/repo/CLAUDE.md", Scope: agent.ScopeProject, Exists: true, Imports: []agent.Memory{
			{Path: "/repo/docs/nested.md", Scope: agent.ScopeProject, Exists: true},
		}},
		{Path: "/repo/other/CLAUDE.md", Scope: agent.ScopeProject, Exists: true},
		{Path: "/Library/Application Support/ClaudeCode/CLAUDE.md", Scope: agent.ScopeManaged, Exists: true},
		{Path: "/repo/.claude/rules/link.md", Scope: agent.ScopeProject, Exists: true, LinkTarget: "/shared/security.md"},
	}

	claudecode.ApplyExclusions(files, []string{
		"**/other/CLAUDE.md",
		"/repo/docs/**",
		"**/ClaudeCode/CLAUDE.md",
		"/shared/**",
	}, "/home/u")

	if files[0].Excluded {
		t.Error("/repo/CLAUDE.md should not be excluded")
	}
	if !files[0].Imports[0].Excluded {
		t.Error("nested import matching /repo/docs/** should be excluded")
	}
	if !files[1].Excluded || files[1].ExcludedBy != "**/other/CLAUDE.md" {
		t.Errorf("expected exclusion by pattern, got %+v", files[1])
	}
	if len(files[1].Warnings) == 0 {
		t.Error("excluded file should carry a warning")
	}
	if files[2].Excluded {
		t.Error("managed policy CLAUDE.md cannot be excluded")
	}
	if !files[3].Excluded {
		t.Error("a pattern matching the symlink target should exclude the file")
	}
}

func TestApplyExclusionsExpandsTilde(t *testing.T) {
	files := []agent.Memory{{Path: "/home/u/.claude/CLAUDE.md", Scope: agent.ScopePersonal, Exists: true}}
	claudecode.ApplyExclusions(files, []string{"~/.claude/CLAUDE.md"}, "/home/u")
	if !files[0].Excluded {
		t.Error("a ~/-prefixed pattern should match the expanded path")
	}
}

func TestResolveAutoMemoryDirFromSettings(t *testing.T) {
	dir := t.TempDir()
	settings := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(settings, []byte(`{"autoMemoryDirectory":"~/custom-mem"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	a := claudecode.NewAgent(claudecode.Paths{
		SettingsFile: settings,
		UserHomeDir:  "/home/u",
		HomeDir:      "/home/u/.claude",
		CWD:          dir,
	})
	if got := a.ResolveAutoMemoryDir(); got != "/home/u/custom-mem" {
		t.Errorf("ResolveAutoMemoryDir() = %q, want %q", got, "/home/u/custom-mem")
	}
}

func TestResolveAutoMemoryDirUsesGitRootSlug(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "myrepo")
	sub := filepath.Join(repo, "pkg", "api")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	a := claudecode.NewAgent(claudecode.Paths{
		UserHomeDir: root,
		HomeDir:     filepath.Join(root, ".claude"),
		CWD:         sub, // a subdirectory: the slug must still be the repo root
	})

	got := a.ResolveAutoMemoryDir()
	if !strings.HasSuffix(got, filepath.Join("memory")) {
		t.Fatalf("got %q, want a path ending in memory", got)
	}
	if strings.Contains(got, "pkg") || strings.Contains(got, "api") {
		t.Errorf("slug derived from the working directory, not the git root: %q", got)
	}
	if !strings.Contains(got, "myrepo") {
		t.Errorf("slug does not contain the repo name: %q", got)
	}
}

func TestAutoMemoryFiles(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("MEMORY.md", "- prefers pnpm\n- API tests need redis\n")
	write("user_role.md", "---\ntype: user\nmodified: 2026-09-01T10:00:00Z\n---\n\nStaff engineer\n")
	write("feedback_testing.md", "---\ntype: feedback\n---\n\nUse table tests\n")
	write("notes.txt", "ignored\n")

	a := claudecode.NewAgent(claudecode.Paths{UserHomeDir: dir, CWD: dir})
	launch, onDemand := a.AutoMemoryFiles(dir)

	if len(launch) != 1 || launch[0].Kind != agent.MemoryKindAutoIndex || launch[0].Tier != agent.MemoryTierLaunch {
		t.Fatalf("launch tier = %+v, want one auto-index entry", launch)
	}
	if launch[0].Scope != agent.ScopeAuto {
		t.Errorf("scope = %q, want %q", launch[0].Scope, agent.ScopeAuto)
	}
	if len(onDemand) != 2 {
		t.Fatalf("on-demand = %d files, want 2 (.txt must be ignored)", len(onDemand))
	}
	byName := map[string]agent.Memory{}
	for _, f := range onDemand {
		byName[filepath.Base(f.Path)] = f
	}
	if got := byName["user_role.md"]; got.Type != "user" || got.Modified != "2026-09-01T10:00:00Z" {
		t.Errorf("user_role.md frontmatter = type %q modified %q", got.Type, got.Modified)
	}
	if got := byName["feedback_testing.md"]; got.Type != "feedback" || got.Modified != "" {
		t.Errorf("feedback_testing.md frontmatter = type %q modified %q", got.Type, got.Modified)
	}
	if byName["user_role.md"].Tier != agent.MemoryTierOnDemand {
		t.Error("topic files must be on-demand tier")
	}
}

func TestAutoMemoryFilesMissingDir(t *testing.T) {
	a := claudecode.NewAgent(claudecode.Paths{})
	launch, onDemand := a.AutoMemoryFiles(filepath.Join(t.TempDir(), "nope"))
	if len(launch) != 0 || len(onDemand) != 0 {
		t.Errorf("missing memory directory should yield nothing, got %d/%d", len(launch), len(onDemand))
	}
}
