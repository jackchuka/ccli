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
	// Neutralize the auto-memory env pair so this fixture never resolves to
	// a real Claude Code memory directory on the machine running the test.
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	t.Setenv("CLAUDE_CODE_PROJECT_DIR_NAME", "")
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
	// Neutralize the auto-memory env pair: this test pins the git-root
	// fallback, which only runs when the pair is unset.
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	t.Setenv("CLAUDE_CODE_PROJECT_DIR_NAME", "")
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

func TestResolveAutoMemoryDirSettingOverridesEnv(t *testing.T) {
	dir := t.TempDir()
	settings := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(settings, []byte(`{"autoMemoryDirectory":"~/custom-mem"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CLAUDE_CONFIG_DIR", "/env/config")
	t.Setenv("CLAUDE_CODE_PROJECT_DIR_NAME", "env-project")

	a := claudecode.NewAgent(claudecode.Paths{
		SettingsFile: settings,
		UserHomeDir:  "/home/u",
		HomeDir:      "/home/u/.claude",
		CWD:          dir,
	})
	if got := a.ResolveAutoMemoryDir(); got != "/home/u/custom-mem" {
		t.Errorf("ResolveAutoMemoryDir() = %q, want the explicit setting %q to win over the env pair", got, "/home/u/custom-mem")
	}
}

func TestResolveAutoMemoryDirFromEnvPair(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", "/env/config")
	t.Setenv("CLAUDE_CODE_PROJECT_DIR_NAME", "env-project")

	a := claudecode.NewAgent(claudecode.Paths{
		UserHomeDir: "/home/u",
		HomeDir:     "/home/u/.claude",
		CWD:         dir,
	})
	want := filepath.Join("/env/config", "projects", "env-project", "memory")
	if got := a.ResolveAutoMemoryDir(); got != want {
		t.Errorf("ResolveAutoMemoryDir() = %q, want %q", got, want)
	}
}

func TestResolveAutoMemoryDirGitRootFromWorktreeFile(t *testing.T) {
	// Neutralize the auto-memory env pair: this test pins the git-root
	// fallback, which only runs when the pair is unset.
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	t.Setenv("CLAUDE_CODE_PROJECT_DIR_NAME", "")
	root := t.TempDir()
	repo := filepath.Join(root, "myrepo")
	sub := filepath.Join(repo, "pkg", "api")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	// A worktree or submodule's ".git" is a file pointing elsewhere, not a
	// directory, so the root detection must accept either.
	if err := os.WriteFile(filepath.Join(repo, ".git"), []byte("gitdir: ../main/.git/worktrees/myrepo\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	a := claudecode.NewAgent(claudecode.Paths{
		UserHomeDir: root,
		HomeDir:     filepath.Join(root, ".claude"),
		CWD:         sub,
	})

	got := a.ResolveAutoMemoryDir()
	if strings.Contains(got, "pkg") || strings.Contains(got, "api") {
		t.Errorf("slug derived from the working directory, not the worktree's .git file: %q", got)
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

func TestOnDemandSubdirFiles(t *testing.T) {
	root := t.TempDir()
	mk := func(parts ...string) string {
		p := filepath.Join(append([]string{root}, parts...)...)
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
		return p
	}
	write := func(dir, name string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write(root, "CLAUDE.md") // launch tier, must not appear here
	write(mk("pkg", "api"), "CLAUDE.md")
	write(mk("pkg", "db"), "CLAUDE.local.md")
	write(mk("node_modules", "dep"), "CLAUDE.md")
	write(mk("vendor", "lib"), "CLAUDE.md")
	write(mk(".git", "hooks"), "CLAUDE.md")
	write(mk("pkg", "api"), "README.md") // not a memory file

	a := claudecode.NewAgent(claudecode.Paths{CWD: root})
	got := a.OnDemandSubdirFiles()

	if len(got) != 2 {
		var paths []string
		for _, f := range got {
			paths = append(paths, f.Path)
		}
		t.Fatalf("got %d files (%v), want 2", len(got), paths)
	}
	for _, f := range got {
		if f.Tier != agent.MemoryTierOnDemand {
			t.Errorf("%s tier = %q, want on-demand", f.Path, f.Tier)
		}
		if !f.Exists {
			t.Errorf("%s should exist", f.Path)
		}
	}
	if filepath.Base(got[0].Path) != "CLAUDE.md" {
		t.Errorf("first file = %q", got[0].Path)
	}
	if got[0].Kind != agent.MemoryKindClaudeMD {
		t.Errorf("kind = %q, want %q", got[0].Kind, agent.MemoryKindClaudeMD)
	}
	if got[1].Kind != agent.MemoryKindClaudeLocalMD || got[1].Scope != agent.ScopeLocal {
		t.Errorf("CLAUDE.local.md should be local-scope claude-local-md: %+v", got[1])
	}
}

func TestListMemoryReport(t *testing.T) {
	_, paths := memoryFixture(t)
	a := claudecode.NewAgent(paths)

	report, err := a.ListMemory()
	if err != nil {
		t.Fatalf("ListMemory: %v", err)
	}
	if report.LaunchFiles == 0 {
		t.Error("LaunchFiles = 0")
	}
	if report.LaunchLines == 0 || report.LaunchBytes == 0 {
		t.Errorf("totals not summed: %dL %dB", report.LaunchLines, report.LaunchBytes)
	}
	if !report.AutoMemoryEnabled {
		t.Error("AutoMemoryEnabled should default to true")
	}
	if report.AutoMemoryDir == "" {
		t.Error("AutoMemoryDir should be resolved even when the directory is absent")
	}

	// The fixture has a subdirectory CLAUDE.md at pkg/api, but the working
	// directory IS pkg/api, so it is launch tier, not on-demand.
	for _, f := range report.Files {
		if f.Tier == agent.MemoryTierOnDemand && f.Exists {
			t.Errorf("unexpected on-demand file %q", f.Path)
		}
	}
}

func TestListMemoryTotalsSkipExcludedAndOnDemand(t *testing.T) {
	// Neutralize the auto-memory env pair so this test never resolves to a
	// real Claude Code memory directory on the machine running the test.
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	t.Setenv("CLAUDE_CODE_PROJECT_DIR_NAME", "")
	root := t.TempDir()
	claudeHome := filepath.Join(root, "home", ".claude")
	repo := filepath.Join(root, "repo")
	if err := os.MkdirAll(claudeHome, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repo, "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}

	settings := filepath.Join(claudeHome, "settings.json")
	if err := os.WriteFile(settings, []byte(`{"claudeMdExcludes":["**/repo/CLAUDE.md"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "CLAUDE.md"), []byte("excluded\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "pkg", "CLAUDE.md"), []byte("on demand\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	a := claudecode.NewAgent(claudecode.Paths{
		SettingsFile: settings,
		HomeDir:      claudeHome,
		UserHomeDir:  filepath.Join(root, "home"),
		CWD:          repo,
	})
	report, err := a.ListMemory()
	if err != nil {
		t.Fatalf("ListMemory: %v", err)
	}

	if report.LaunchFiles != 0 {
		t.Errorf("LaunchFiles = %d, want 0: the only launch file is excluded", report.LaunchFiles)
	}
	if report.OnDemandFiles != 1 {
		t.Errorf("OnDemandFiles = %d, want 1", report.OnDemandFiles)
	}
	if len(report.Warnings) == 0 {
		t.Error("the exclusion should surface as a report warning")
	}
}

func TestListMemoryWarnsOnOversizedFiles(t *testing.T) {
	root := t.TempDir()
	claudeHome := filepath.Join(root, "home", ".claude")
	memDir := filepath.Join(root, "mem")
	if err := os.MkdirAll(claudeHome, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(memDir, 0o755); err != nil {
		t.Fatal(err)
	}

	settings := filepath.Join(claudeHome, "settings.json")
	if err := os.WriteFile(settings, []byte(`{"autoMemoryDirectory":"`+memDir+`"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	// 201 lines: one past the MEMORY.md load cutoff.
	if err := os.WriteFile(filepath.Join(memDir, "MEMORY.md"), []byte(strings.Repeat("entry\n", 201)), 0o644); err != nil {
		t.Fatal(err)
	}
	// 250 lines: past the CLAUDE.md advisory limit.
	if err := os.WriteFile(filepath.Join(claudeHome, "CLAUDE.md"), []byte(strings.Repeat("rule\n", 250)), 0o644); err != nil {
		t.Fatal(err)
	}

	a := claudecode.NewAgent(claudecode.Paths{
		SettingsFile: settings,
		HomeDir:      claudeHome,
		UserHomeDir:  filepath.Join(root, "home"),
		CWD:          root,
	})
	report, err := a.ListMemory()
	if err != nil {
		t.Fatalf("ListMemory: %v", err)
	}

	var sawIndex, sawAdvisory bool
	for _, f := range claudecode.FlattenMemory(report.Files) {
		for _, w := range f.Warnings {
			// Match the distinctive phrase, not a shared "200" substring, so
			// the two branches in annotateWarnings are pinned separately.
			if f.Kind == agent.MemoryKindAutoIndex && strings.Contains(w, "only the first 200 load") {
				sawIndex = true
			}
			if f.Kind == agent.MemoryKindClaudeMD && strings.Contains(w, "200-line guideline") {
				sawAdvisory = true
			}
		}
	}
	if !sawIndex {
		t.Error("MEMORY.md over 200 lines did not warn")
	}
	if !sawAdvisory {
		t.Error("CLAUDE.md over 200 lines did not warn")
	}
}

func TestListMemoryExcludedFileImportsDoNotCount(t *testing.T) {
	// Neutralize the auto-memory env pair so this test never resolves to a
	// real Claude Code memory directory on the machine running the test.
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	t.Setenv("CLAUDE_CODE_PROJECT_DIR_NAME", "")
	root := t.TempDir()
	claudeHome := filepath.Join(root, "home", ".claude")
	repo := filepath.Join(root, "repo")
	if err := os.MkdirAll(claudeHome, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}

	settings := filepath.Join(claudeHome, "settings.json")
	if err := os.WriteFile(settings, []byte(`{"claudeMdExcludes":["**/repo/CLAUDE.md"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "CLAUDE.md"), []byte("@notes.md\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// 300 lines: large enough that, if counted, it could not be mistaken
	// for anything else contributing to the totals.
	if err := os.WriteFile(filepath.Join(repo, "notes.md"), []byte(strings.Repeat("line\n", 300)), 0o644); err != nil {
		t.Fatal(err)
	}

	a := claudecode.NewAgent(claudecode.Paths{
		SettingsFile: settings,
		HomeDir:      claudeHome,
		UserHomeDir:  filepath.Join(root, "home"),
		CWD:          repo,
	})
	report, err := a.ListMemory()
	if err != nil {
		t.Fatalf("ListMemory: %v", err)
	}

	if report.LaunchFiles != 0 {
		t.Errorf("LaunchFiles = %d, want 0: the only launch-tier file is excluded", report.LaunchFiles)
	}
	if report.LaunchLines != 0 {
		t.Errorf("LaunchLines = %d, want 0: an excluded file's import never loads either", report.LaunchLines)
	}

	var importExcluded bool
	for _, f := range claudecode.FlattenMemory(report.Files) {
		if filepath.Base(f.Path) == "notes.md" {
			importExcluded = f.Excluded
		}
	}
	if !importExcluded {
		t.Error("notes.md, imported by an excluded file, should itself be marked excluded")
	}
}

func TestListMemoryAutoMemoryDisabledDoesNotCount(t *testing.T) {
	// Neutralize the auto-memory env pair so this test never resolves to a
	// real Claude Code memory directory on the machine running the test.
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	t.Setenv("CLAUDE_CODE_PROJECT_DIR_NAME", "")
	root := t.TempDir()
	claudeHome := filepath.Join(root, "home", ".claude")
	memDir := filepath.Join(root, "mem")
	if err := os.MkdirAll(claudeHome, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(memDir, 0o755); err != nil {
		t.Fatal(err)
	}

	settings := filepath.Join(claudeHome, "settings.json")
	if err := os.WriteFile(settings, []byte(`{"autoMemoryEnabled":false,"autoMemoryDirectory":"`+memDir+`"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(claudeHome, "CLAUDE.md"), []byte(strings.Repeat("rule\n", 250)), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(memDir, "MEMORY.md"), []byte("a\nb\nc\nd\ne\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	a := claudecode.NewAgent(claudecode.Paths{
		SettingsFile: settings,
		HomeDir:      claudeHome,
		UserHomeDir:  filepath.Join(root, "home"),
		CWD:          root,
	})
	report, err := a.ListMemory()
	if err != nil {
		t.Fatalf("ListMemory: %v", err)
	}

	if report.LaunchFiles != 1 {
		t.Errorf("LaunchFiles = %d, want 1: MEMORY.md must not count when auto memory is disabled", report.LaunchFiles)
	}
	if report.LaunchLines != 250 {
		t.Errorf("LaunchLines = %d, want 250: MEMORY.md's lines must not count when auto memory is disabled", report.LaunchLines)
	}

	var sawDisabledWarning bool
	for _, f := range claudecode.FlattenMemory(report.Files) {
		if f.Kind == agent.MemoryKindAutoIndex {
			for _, w := range f.Warnings {
				if strings.Contains(w, "disabled") {
					sawDisabledWarning = true
				}
			}
		}
	}
	if !sawDisabledWarning {
		t.Error("MEMORY.md should warn that auto memory is disabled")
	}
}

func TestListMemoryAutoMemoryDisabledImportContributesZero(t *testing.T) {
	// Neutralize the auto-memory env pair so this test never resolves to a
	// real Claude Code memory directory on the machine running the test.
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	t.Setenv("CLAUDE_CODE_PROJECT_DIR_NAME", "")
	root := t.TempDir()
	claudeHome := filepath.Join(root, "home", ".claude")
	memDir := filepath.Join(root, "mem")
	if err := os.MkdirAll(claudeHome, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(memDir, 0o755); err != nil {
		t.Fatal(err)
	}

	settings := filepath.Join(claudeHome, "settings.json")
	if err := os.WriteFile(settings, []byte(`{"autoMemoryEnabled":false,"autoMemoryDirectory":"`+memDir+`"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(claudeHome, "CLAUDE.md"), []byte(strings.Repeat("rule\n", 250)), 0o644); err != nil {
		t.Fatal(err)
	}
	// MEMORY.md imports a large file. Disabling auto memory has to stop the
	// import from loading too, not just the index itself: an import
	// inherits the index's launch tier, not its disabled state, unless the
	// counting walk carries that state down explicitly.
	if err := os.WriteFile(filepath.Join(memDir, "MEMORY.md"), []byte("@extra.md\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(memDir, "extra.md"), []byte(strings.Repeat("line\n", 300)), 0o644); err != nil {
		t.Fatal(err)
	}

	a := claudecode.NewAgent(claudecode.Paths{
		SettingsFile: settings,
		HomeDir:      claudeHome,
		UserHomeDir:  filepath.Join(root, "home"),
		CWD:          root,
	})
	report, err := a.ListMemory()
	if err != nil {
		t.Fatalf("ListMemory: %v", err)
	}

	if report.LaunchFiles != 1 {
		t.Errorf("LaunchFiles = %d, want 1: extra.md must not count when its importer's auto memory is disabled", report.LaunchFiles)
	}
	if report.LaunchLines != 250 {
		t.Errorf("LaunchLines = %d, want 250: extra.md's 300 lines must not count", report.LaunchLines)
	}

	var extraWarned bool
	for _, f := range claudecode.FlattenMemory(report.Files) {
		if filepath.Base(f.Path) == "extra.md" {
			for _, w := range f.Warnings {
				if strings.Contains(w, "disabled") {
					extraWarned = true
				}
			}
		}
	}
	if !extraWarned {
		t.Error("extra.md should warn that it does not load because its importer's auto memory is disabled")
	}
}

func TestListMemoryTotalsExactArithmetic(t *testing.T) {
	// Neutralize the auto-memory env pair so this test never resolves to a
	// real Claude Code memory directory on the machine running the test.
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	t.Setenv("CLAUDE_CODE_PROJECT_DIR_NAME", "")
	root := t.TempDir()
	claudeHome := filepath.Join(root, "home", ".claude")
	repo := filepath.Join(root, "repo")
	pkg := filepath.Join(repo, "pkg")
	for _, d := range []string{claudeHome, pkg} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	settings := filepath.Join(claudeHome, "settings.json")
	if err := os.WriteFile(settings, []byte(`{"claudeMdExcludes":["**/repo/CLAUDE.local.md"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	write := func(path, body string) {
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(claudeHome, "CLAUDE.md"), "personal one\n") // 1 line, 13 bytes
	write(filepath.Join(repo, "CLAUDE.md"), "@imported.md\n")       // 1 line, 13 bytes; imports imported.md
	write(filepath.Join(repo, "imported.md"), "one\ntwo\nthree\n")  // 3 lines, 14 bytes
	write(filepath.Join(repo, "CLAUDE.local.md"), "excluded, must not count\n")
	write(filepath.Join(pkg, "CLAUDE.md"), "on demand only\n")

	a := claudecode.NewAgent(claudecode.Paths{
		SettingsFile: settings,
		HomeDir:      claudeHome,
		UserHomeDir:  filepath.Join(root, "home"),
		CWD:          repo,
	})
	report, err := a.ListMemory()
	if err != nil {
		t.Fatalf("ListMemory: %v", err)
	}

	if report.LaunchFiles != 3 {
		t.Fatalf("LaunchFiles = %d, want 3 (personal, project CLAUDE.md, its import)", report.LaunchFiles)
	}
	if report.LaunchLines != 5 {
		t.Errorf("LaunchLines = %d, want 5 (1 + 1 + 3)", report.LaunchLines)
	}
	if report.LaunchBytes != 40 {
		t.Errorf("LaunchBytes = %d, want 40 (13 + 13 + 14)", report.LaunchBytes)
	}
	if report.OnDemandFiles != 1 {
		t.Errorf("OnDemandFiles = %d, want 1", report.OnDemandFiles)
	}
}

func TestAnnotateWarningsSizeLimits(t *testing.T) {
	tests := []struct {
		name     string
		memory   agent.Memory
		wantSnip string
	}{
		{
			name:     "MEMORY.md past the 25KB cutoff",
			memory:   agent.Memory{Kind: agent.MemoryKindAutoIndex, Exists: true, Lines: 10, Bytes: 25*1024 + 1},
			wantSnip: "25600",
		},
		{
			name:     "CLAUDE.md over 4 MiB is skipped entirely",
			memory:   agent.Memory{Kind: agent.MemoryKindClaudeMD, Exists: true, Lines: 10, Bytes: 4<<20 + 1},
			wantSnip: "4 MiB",
		},
		{
			name:     "an excluded file gets no size warning",
			memory:   agent.Memory{Kind: agent.MemoryKindClaudeMD, Exists: true, Lines: 900, Excluded: true},
			wantSnip: "",
		},
		// At-limit cases: the limits are exclusive, so a file exactly at the
		// threshold must stay silent. These catch a ">" flipped to ">=".
		{
			name:     "MEMORY.md at exactly the 25KB cutoff does not warn",
			memory:   agent.Memory{Kind: agent.MemoryKindAutoIndex, Exists: true, Lines: 10, Bytes: 25 * 1024},
			wantSnip: "",
		},
		{
			name:     "CLAUDE.md at exactly 4 MiB does not warn",
			memory:   agent.Memory{Kind: agent.MemoryKindClaudeMD, Exists: true, Lines: 10, Bytes: 4 << 20},
			wantSnip: "",
		},
		{
			name:     "CLAUDE.md at exactly 200 lines does not warn",
			memory:   agent.Memory{Kind: agent.MemoryKindClaudeMD, Exists: true, Lines: 200, Bytes: 10},
			wantSnip: "",
		},
		{
			name:     "MEMORY.md at exactly 200 lines does not warn",
			memory:   agent.Memory{Kind: agent.MemoryKindAutoIndex, Exists: true, Lines: 200, Bytes: 10},
			wantSnip: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.memory
			claudecode.AnnotateWarnings(&m)
			joined := strings.Join(m.Warnings, " | ")
			if tt.wantSnip == "" {
				if joined != "" {
					t.Errorf("want no warnings, got %q", joined)
				}
				return
			}
			if !strings.Contains(joined, tt.wantSnip) {
				t.Errorf("warnings %q missing %q", joined, tt.wantSnip)
			}
		})
	}
}

func TestGetMemory(t *testing.T) {
	_, paths := memoryFixture(t)
	a := claudecode.NewAgent(paths)

	t.Run("by scope name", func(t *testing.T) {
		m, err := a.GetMemory("personal")
		if err != nil {
			t.Fatalf("GetMemory: %v", err)
		}
		if m.Scope != agent.ScopePersonal {
			t.Errorf("scope = %q", m.Scope)
		}
	})

	t.Run("user is an alias for personal", func(t *testing.T) {
		m, err := a.GetMemory("user")
		if err != nil {
			t.Fatalf("GetMemory: %v", err)
		}
		if m.Scope != agent.ScopePersonal {
			t.Errorf("scope = %q, want %q", m.Scope, agent.ScopePersonal)
		}
	})

	t.Run("by path suffix", func(t *testing.T) {
		m, err := a.GetMemory("CLAUDE.local.md")
		if err != nil {
			t.Fatalf("GetMemory: %v", err)
		}
		if filepath.Base(m.Path) != "CLAUDE.local.md" {
			t.Errorf("path = %q", m.Path)
		}
	})

	t.Run("not found", func(t *testing.T) {
		if _, err := a.GetMemory("nope.md"); err == nil {
			t.Fatal("expected an error")
		}
	})

	t.Run("rejects an empty name", func(t *testing.T) {
		// Without a guard this matches the managed inline entry, whose
		// Path is "".
		if _, err := a.GetMemory(""); err == nil {
			t.Fatal("expected an error")
		}
	})

	t.Run("rejects a dot", func(t *testing.T) {
		// filepath.Base("") == ".", so this would also match the managed
		// inline entry without a guard.
		if _, err := a.GetMemory("."); err == nil {
			t.Fatal("expected an error")
		}
	})
}

// memoryTestPaths builds a home/repo pair with the auto-memory env pair
// neutralized, so no test resolves to a real Claude Code memory directory on
// the machine running it.
func memoryTestPaths(t *testing.T, settingsJSON string) (string, claudecode.Paths) {
	t.Helper()
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	t.Setenv("CLAUDE_CODE_PROJECT_DIR_NAME", "")

	root := t.TempDir()
	claudeHome := filepath.Join(root, "home", ".claude")
	repo := filepath.Join(root, "repo")
	for _, d := range []string{claudeHome, repo} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	settings := filepath.Join(claudeHome, "settings.json")
	if err := os.WriteFile(settings, []byte(settingsJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	return repo, claudecode.Paths{
		SettingsFile: settings,
		HomeDir:      claudeHome,
		UserHomeDir:  filepath.Join(root, "home"),
		CWD:          repo,
	}
}

func writeMemoryFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestListMemoryDepthLimitAndCycleDoNotCount pins the two structural
// non-load conditions that carry no Excluded or Exists flag of their own.
// Both files are on disk and readable, so only the audit's own knowledge
// that Claude Code refuses them keeps them out of the totals.
func TestListMemoryDepthLimitAndCycleDoNotCount(t *testing.T) {
	repo, paths := memoryTestPaths(t, `{}`)

	writeMemoryFile(t, filepath.Join(repo, "CLAUDE.md"), "@one.md\n@loop.md\n")
	writeMemoryFile(t, filepath.Join(repo, "one.md"), "@two.md\n")
	writeMemoryFile(t, filepath.Join(repo, "two.md"), "@three.md\n")
	writeMemoryFile(t, filepath.Join(repo, "three.md"), "@four.md\n")
	writeMemoryFile(t, filepath.Join(repo, "four.md"), "@five.md\n")
	// five.md sits one hop past the limit. Its 99 lines are large enough
	// that counting it could not be mistaken for any other contribution.
	writeMemoryFile(t, filepath.Join(repo, "five.md"), strings.Repeat("dropped\n", 99))
	// loop.md imports the file that imported it: a cycle, which is Claude
	// Code refusing a second inclusion rather than a second inclusion.
	writeMemoryFile(t, filepath.Join(repo, "loop.md"), "@CLAUDE.md\n")

	report, err := claudecode.NewAgent(paths).ListMemory()
	if err != nil {
		t.Fatalf("ListMemory: %v", err)
	}

	// CLAUDE.md, one, two, three, four, loop: six files that load.
	if report.LaunchFiles != 6 {
		t.Errorf("LaunchFiles = %d, want 6 (the over-depth import and the cycle node load nothing)", report.LaunchFiles)
	}
	if report.LaunchLines != 7 {
		t.Errorf("LaunchLines = %d, want 7 (2 + 1 + 1 + 1 + 1 + 1)", report.LaunchLines)
	}

	var sawDepth, sawCycle bool
	for _, f := range claudecode.FlattenMemory(report.Files) {
		switch {
		case filepath.Base(f.Path) == "five.md":
			sawDepth = true
			if f.Loads() || !f.NotLoaded {
				t.Errorf("five.md is past the depth limit and must not load: %+v", f)
			}
		case f.Kind == agent.MemoryKindImport && filepath.Base(f.Path) == "CLAUDE.md":
			sawCycle = true
			if f.Loads() || !f.NotLoaded {
				t.Errorf("the cycle node must not load: %+v", f)
			}
		}
	}
	if !sawDepth || !sawCycle {
		t.Fatalf("fixture did not produce both nodes (depth %v, cycle %v)", sawDepth, sawCycle)
	}
}

// TestListMemoryDiamondCountsTwice is the counterpart to the cycle case: two
// parents importing one file is Claude Code expanding it textually twice, so
// it genuinely occupies context twice and must not be deduplicated by path.
func TestListMemoryDiamondCountsTwice(t *testing.T) {
	repo, paths := memoryTestPaths(t, `{}`)

	writeMemoryFile(t, filepath.Join(repo, "CLAUDE.md"), "@left.md\n@right.md\n")
	writeMemoryFile(t, filepath.Join(repo, "left.md"), "@shared.md\n")
	writeMemoryFile(t, filepath.Join(repo, "right.md"), "@shared.md\n")
	writeMemoryFile(t, filepath.Join(repo, "shared.md"), strings.Repeat("shared\n", 10))

	report, err := claudecode.NewAgent(paths).ListMemory()
	if err != nil {
		t.Fatalf("ListMemory: %v", err)
	}

	if report.LaunchFiles != 5 {
		t.Errorf("LaunchFiles = %d, want 5 (root, two branches, shared.md under each)", report.LaunchFiles)
	}
	if report.LaunchLines != 24 {
		t.Errorf("LaunchLines = %d, want 24 (2 + 1 + 1 + 10 + 10)", report.LaunchLines)
	}
}

// TestListMemoryOnDemandCountMatchesTheList pins the count to the rows the
// renderer prints. On-demand files never load at launch, so filtering them
// by whether they load would zero out a count whose job is discovery.
func TestListMemoryOnDemandCountMatchesTheList(t *testing.T) {
	root := t.TempDir()
	memDir := filepath.Join(root, "mem")
	if err := os.MkdirAll(memDir, 0o755); err != nil {
		t.Fatal(err)
	}
	repo, paths := memoryTestPaths(t,
		`{"autoMemoryEnabled":false,"autoMemoryDirectory":"`+memDir+`"}`)

	if err := os.MkdirAll(filepath.Join(repo, "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeMemoryFile(t, filepath.Join(repo, "pkg", "CLAUDE.md"), "on demand\n")
	writeMemoryFile(t, filepath.Join(memDir, "MEMORY.md"), "index\n")
	writeMemoryFile(t, filepath.Join(memDir, "topic_a.md"), "a\n")
	writeMemoryFile(t, filepath.Join(memDir, "topic_b.md"), "b\n")

	report, err := claudecode.NewAgent(paths).ListMemory()
	if err != nil {
		t.Fatalf("ListMemory: %v", err)
	}

	listed := 0
	for _, f := range claudecode.FlattenMemory(report.Files) {
		if f.Tier == agent.MemoryTierOnDemand {
			listed++
		}
	}
	if listed != 3 {
		t.Fatalf("fixture discovered %d on-demand files, want 3", listed)
	}
	if report.OnDemandFiles != listed {
		t.Errorf("OnDemandFiles = %d but %d on-demand files are listed; the count and the list must agree",
			report.OnDemandFiles, listed)
	}

	// The topic files do not load when read either, and say so, which is
	// what lets the count stay a discovery count without misleading anyone.
	for _, f := range claudecode.FlattenMemory(report.Files) {
		if f.Kind == agent.MemoryKindAutoTopic && len(f.Warnings) == 0 {
			t.Errorf("topic file %q should explain that auto memory is disabled", f.Path)
		}
	}
}

// TestListMemoryNoContradictoryWarnings pins that a file which does not load
// gets no size warning: "only the first 200 lines load" and "it does not
// load" cannot both be true, and a reader handed both can act on neither.
func TestListMemoryNoContradictoryWarnings(t *testing.T) {
	root := t.TempDir()
	memDir := filepath.Join(root, "mem")
	if err := os.MkdirAll(memDir, 0o755); err != nil {
		t.Fatal(err)
	}
	_, paths := memoryTestPaths(t,
		`{"autoMemoryEnabled":false,"autoMemoryDirectory":"`+memDir+`"}`)

	writeMemoryFile(t, filepath.Join(memDir, "MEMORY.md"), strings.Repeat("entry\n", 201))

	report, err := claudecode.NewAgent(paths).ListMemory()
	if err != nil {
		t.Fatalf("ListMemory: %v", err)
	}

	var index *agent.Memory
	for _, f := range claudecode.FlattenMemory(report.Files) {
		if f.Kind == agent.MemoryKindAutoIndex {
			index = &f
		}
	}
	if index == nil {
		t.Fatal("fixture did not produce a MEMORY.md")
	}

	var sawDisabled bool
	for _, w := range index.Warnings {
		if strings.Contains(w, "disabled") {
			sawDisabled = true
		}
		if strings.Contains(w, "only the first") {
			t.Errorf("a file that does not load must not also be told part of it loads: %q", w)
		}
	}
	if !sawDisabled {
		t.Errorf("MEMORY.md should warn that auto memory is disabled, got %v", index.Warnings)
	}
}
