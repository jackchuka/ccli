package claudecode

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jackchuka/ccli/internal/agent"
)

// readMemoryFile stats and measures one memory file. A missing file is not an
// error: the audit reports expected-but-absent locations the way /memory does.
func readMemoryFile(path string, scope agent.Scope, kind agent.MemoryKind, tier agent.MemoryTier) agent.Memory {
	m := agent.Memory{Path: path, Scope: scope, Kind: kind, Tier: tier}

	if target, err := os.Readlink(path); err == nil {
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(path), target)
		}
		m.LinkTarget = target
	}

	info, err := os.Stat(path) // Stat, not Lstat: a symlinked CLAUDE.md is loaded by its target.
	if err != nil || info.IsDir() {
		return m
	}
	m.Exists = true
	m.Bytes = info.Size()
	m.Lines = countLines(path)
	return m
}

// expandTilde resolves a leading "~/" against home. Any other value,
// including a bare "~user" prefix, is returned unchanged.
func expandTilde(path, home string) string {
	if path == "~" {
		return home
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:])
	}
	return path
}

// globMatch matches a claudeMdExcludes pattern against an absolute path.
// filepath.Match has no "**", so patterns are compiled to a regexp where
// "**" crosses separators, "*" and "?" do not.
func globMatch(pattern, path string) bool {
	var b strings.Builder
	b.WriteString("^")
	for i := 0; i < len(pattern); i++ {
		switch c := pattern[i]; c {
		case '*':
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				b.WriteString(".*")
				i++
			} else {
				b.WriteString("[^/]*")
			}
		case '?':
			b.WriteString("[^/]")
		default:
			b.WriteString(regexp.QuoteMeta(string(c)))
		}
	}
	b.WriteString("$")

	re, err := regexp.Compile(b.String())
	if err != nil {
		return false
	}
	return re.MatchString(path)
}

// launchTierFiles returns every memory file Claude Code loads at session
// start, in the order it concatenates them: broadest scope first, so the most
// specific instruction is read last.
func (a *Agent) launchTierFiles(s *MergedSettings) []agent.Memory {
	var files []agent.Memory

	// 1. Managed policy CLAUDE.md, then the inline claudeMd string.
	if a.paths.ManagedPolicyFile != "" {
		if m := readMemoryFile(a.paths.ManagedPolicyFile, agent.ScopeManaged, agent.MemoryKindClaudeMD, agent.MemoryTierLaunch); m.Exists {
			files = append(files, m)
		}
	}
	if s.ClaudeMd != "" {
		files = append(files, agent.Memory{
			Scope:  agent.ScopeManaged,
			Kind:   agent.MemoryKindManagedInline,
			Tier:   agent.MemoryTierLaunch,
			Exists: true,
			Lines:  countLinesIn(strings.NewReader(s.ClaudeMd)),
			Bytes:  int64(len(s.ClaudeMd)),
		})
	}

	// 2. User CLAUDE.md. Reported even when absent: /memory lists the location.
	if a.paths.HomeDir != "" {
		files = append(files, readMemoryFile(
			filepath.Join(a.paths.HomeDir, "CLAUDE.md"),
			agent.ScopePersonal, agent.MemoryKindClaudeMD, agent.MemoryTierLaunch))
	}

	// 3. Ancestor directories, filesystem root down to the working directory,
	// excluding the working directory itself, which step 4 handles.
	ancestors := ancestorDirs(a.paths.CWD)
	for _, dir := range ancestors[:max(0, len(ancestors)-1)] {
		files = append(files, existingOnly(
			readMemoryFile(filepath.Join(dir, "CLAUDE.md"), agent.ScopeProject, agent.MemoryKindClaudeMD, agent.MemoryTierLaunch),
			readMemoryFile(filepath.Join(dir, "CLAUDE.local.md"), agent.ScopeLocal, agent.MemoryKindClaudeLocalMD, agent.MemoryTierLaunch),
			readMemoryFile(filepath.Join(dir, "AGENTS.md"), agent.ScopeProject, agent.MemoryKindAgentsMD, agent.MemoryTierLaunch),
			readMemoryFile(filepath.Join(dir, ".claude", "AGENTS.md"), agent.ScopeProject, agent.MemoryKindAgentsMD, agent.MemoryTierLaunch),
		)...)
	}

	// 4. The working directory: ./CLAUDE.md, ./.claude/CLAUDE.md, ./CLAUDE.local.md.
	if a.paths.CWD != "" {
		files = append(files,
			readMemoryFile(filepath.Join(a.paths.CWD, "CLAUDE.md"), agent.ScopeProject, agent.MemoryKindClaudeMD, agent.MemoryTierLaunch),
			readMemoryFile(filepath.Join(a.paths.CWD, ".claude", "CLAUDE.md"), agent.ScopeProject, agent.MemoryKindClaudeMD, agent.MemoryTierLaunch),
			readMemoryFile(filepath.Join(a.paths.CWD, "CLAUDE.local.md"), agent.ScopeLocal, agent.MemoryKindClaudeLocalMD, agent.MemoryTierLaunch),
		)

		// AGENTS.md entries are reported only when present. The three CLAUDE.md
		// locations above are listed even when absent because /memory lists
		// them; AGENTS.md has no such listing, and an absent row in every
		// CLAUDE.md repository would be noise.
		files = append(files, existingOnly(
			readMemoryFile(filepath.Join(a.paths.CWD, "AGENTS.md"), agent.ScopeProject, agent.MemoryKindAgentsMD, agent.MemoryTierLaunch),
			readMemoryFile(filepath.Join(a.paths.CWD, ".claude", "AGENTS.md"), agent.ScopeProject, agent.MemoryKindAgentsMD, agent.MemoryTierLaunch),
		)...)
	}

	return files
}

// ancestorDirs returns dir and every directory above it, ordered from the
// filesystem root down to dir.
func ancestorDirs(dir string) []string {
	if dir == "" {
		return nil
	}
	var chain []string
	for {
		chain = append(chain, dir)
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	// Reverse: root first.
	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
	}
	return chain
}

// existingOnly filters out absent files. Ancestor and subdirectory locations
// are only interesting when a file is actually there, unlike the user,
// project, and local paths, which are reported either way.
func existingOnly(files ...agent.Memory) []agent.Memory {
	var out []agent.Memory
	for _, f := range files {
		if f.Exists {
			out = append(out, f)
		}
	}
	return out
}

var (
	// importRe finds "@path" tokens at a line start or after whitespace.
	importRe = regexp.MustCompile(`(^|\s)@(\S+)`)
	// fencedRe matches fenced code blocks, whose contents are not imports.
	fencedRe = regexp.MustCompile("(?s)```.*?```")
	// spanRe matches inline code spans, whose contents are not imports.
	spanRe = regexp.MustCompile("`[^`\n]*`")
)

// parseImports returns the @path imports declared in a memory file's content.
// Import parsing skips code spans and fenced blocks, so a backticked
// `@README` stays literal.
func parseImports(content string) []string {
	stripped := spanRe.ReplaceAllString(fencedRe.ReplaceAllString(content, ""), "")

	var out []string
	for _, m := range importRe.FindAllStringSubmatch(stripped, -1) {
		p := strings.TrimRight(m[2], ".,;:!?)")
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// expandImports fills parent.Imports recursively. depth is the hop count of
// the children being added, starting at 1. rootScope is the scope of the
// top-level file that began the chain: project-scope chains get their
// outside-the-working-directory imports flagged, because Claude Code gates
// those behind a one-time approval dialog, while user-scope files are trusted.
// seen holds only this branch's ancestors, not the whole tree, so a diamond
// (two branches importing the same file) is not mistaken for a cycle.
func (a *Agent) expandImports(parent *agent.Memory, rootScope agent.Scope, depth int, seen map[string]bool) {
	if parent.Path == "" || !parent.Exists {
		return
	}

	data, err := os.ReadFile(parent.Path)
	if err != nil {
		return
	}

	for _, raw := range parseImports(string(data)) {
		resolved := expandTilde(raw, a.paths.UserHomeDir)
		if !filepath.IsAbs(resolved) {
			resolved = filepath.Join(filepath.Dir(parent.Path), resolved)
		}
		resolved = filepath.Clean(resolved)

		child := readMemoryFile(resolved, parent.Scope, agent.MemoryKindImport, parent.Tier)
		child.Depth = depth

		if !child.Exists {
			child.Warnings = append(child.Warnings, "import target does not exist")
		}
		// Deliberately not NotLoaded: the approval may well have been granted,
		// and ccli cannot tell. Counting it keeps the total an over-estimate
		// the warning explains, rather than an under-estimate that silently
		// hides content the session is probably carrying.
		if (rootScope == agent.ScopeProject || rootScope == agent.ScopeLocal) && !withinDir(resolved, a.paths.CWD) {
			child.External = true
			child.Warnings = append(child.Warnings,
				"import resolves outside the working directory and needs one-time approval")
		}

		switch {
		case depth > maxImportDepth:
			// The parent itself loaded; this is the import that would exceed the
			// hop limit, so it is recorded but never read or recursed into.
			child.NotLoaded = true
			child.Warnings = append(child.Warnings,
				fmt.Sprintf("import exceeds the %d-hop depth limit and is not loaded", maxImportDepth))
		case seen[resolved]:
			// A cycle is Claude Code refusing a second inclusion, the opposite
			// of a diamond, where two branches each genuinely expand the same
			// file and both occurrences cost context.
			child.NotLoaded = true
			child.Warnings = append(child.Warnings, "import is one of its own ancestors, forming a cycle")
		default:
			next := make(map[string]bool, len(seen)+1)
			for k := range seen {
				next[k] = true
			}
			next[resolved] = true
			a.expandImports(&child, rootScope, depth+1, next)
		}

		parent.Imports = append(parent.Imports, child)
	}
}

// withinDir reports whether path is dir or sits underneath it.
func withinDir(path, dir string) bool {
	if dir == "" {
		return false
	}
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// ExpandInto expands the imports of each file in place and returns the slice.
func (a *Agent) ExpandInto(files []agent.Memory) []agent.Memory {
	for i := range files {
		seen := map[string]bool{files[i].Path: true}
		a.expandImports(&files[i], files[i].Scope, 1, seen)
	}
	return files
}

// applyExclusions marks files matched by claudeMdExcludes. Managed policy
// files are exempt: Claude Code does not let individual settings exclude them.
// A file's own exclusion is decided before its imports are considered, so an
// excluded parent can cascade the exclusion onto its whole import subtree
// instead of recursing into imports that would never actually load.
func applyExclusions(files []agent.Memory, patterns []string, home string) {
	if len(patterns) == 0 {
		return
	}
	for i := range files {
		f := &files[i]

		if f.Scope != agent.ScopeManaged && f.Path != "" {
			for _, raw := range patterns {
				pattern := expandTilde(raw, home)
				if globMatch(pattern, f.Path) || (f.LinkTarget != "" && globMatch(pattern, f.LinkTarget)) {
					f.Excluded = true
					f.ExcludedBy = raw
					f.Warnings = append(f.Warnings, "excluded by claudeMdExcludes, so it does not load")
					break
				}
			}
		}

		if f.Excluded {
			excludeImportSubtree(f.Imports, f.Path, f.ExcludedBy)
		} else {
			applyExclusions(f.Imports, patterns, home)
		}
	}
}

// excludeImportSubtree cascades an exclusion onto every import beneath an
// excluded file: Claude Code never reads an excluded parent, so its imports
// never load either, regardless of whether they individually match a
// pattern. ExcludedBy carries the pattern that caused the cascade, while the
// warning names the direct importer, not the original excluded file, so a
// depth-2 import's warning correctly names the file that imported it.
func excludeImportSubtree(imports []agent.Memory, importerPath, importerPattern string) {
	for i := range imports {
		imp := &imports[i]
		imp.Excluded = true
		imp.ExcludedBy = importerPattern
		imp.Warnings = append(imp.Warnings,
			fmt.Sprintf("not loaded: imported by %s, which is excluded", importerPath))
		excludeImportSubtree(imp.Imports, imp.Path, importerPattern)
	}
}

// resolveAutoMemoryDir finds the auto memory directory, mirroring Claude
// Code's resolution: an explicit autoMemoryDirectory setting wins; then a
// CLAUDE_CONFIG_DIR plus CLAUDE_CODE_PROJECT_DIR_NAME pair; otherwise the
// per-repository default under ~/.claude/projects/.
func (a *Agent) resolveAutoMemoryDir(s *MergedSettings) string {
	if s.AutoMemoryDirectory != "" {
		return expandTilde(s.AutoMemoryDirectory, a.paths.UserHomeDir)
	}
	if configDir, name := os.Getenv("CLAUDE_CONFIG_DIR"), os.Getenv("CLAUDE_CODE_PROJECT_DIR_NAME"); configDir != "" && name != "" {
		return filepath.Join(configDir, "projects", name, "memory")
	}
	// Auto memory is per repository, so all worktrees and subdirectories of
	// one repo share a directory keyed on the repository root.
	return filepath.Join(a.paths.HomeDir, "projects", encodeProjectPath(gitRoot(a.paths.CWD)), "memory")
}

// gitRoot returns the nearest ancestor of dir containing a .git entry,
// falling back to dir itself when there is none. The fallback matters
// because a working directory outside any git repository must still resolve
// to a stable auto memory location instead of failing.
func gitRoot(dir string) string {
	for cur := dir; cur != ""; {
		if _, err := os.Stat(filepath.Join(cur, ".git")); err == nil {
			return cur
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}
	return dir
}

// autoMemoryFiles splits an auto memory directory into the MEMORY.md index,
// which loads at launch, and the topic files, which Claude reads on demand.
func (a *Agent) autoMemoryFiles(dir string) ([]agent.Memory, []agent.Memory) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil
	}

	var launch, onDemand []agent.Memory
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		if e.Name() == "MEMORY.md" {
			launch = append(launch, readMemoryFile(path, agent.ScopeAuto, agent.MemoryKindAutoIndex, agent.MemoryTierLaunch))
			continue
		}
		m := readMemoryFile(path, agent.ScopeAuto, agent.MemoryKindAutoTopic, agent.MemoryTierOnDemand)
		if data, err := os.ReadFile(path); err == nil {
			block := extractFrontmatterBlock(string(data))
			m.Type = frontmatterValue(block, "type")
			m.Modified = frontmatterValue(block, "modified")
		}
		onDemand = append(onDemand, m)
	}
	return launch, onDemand
}

// frontmatterValue reads a scalar key from a YAML frontmatter block. It
// hand-scans lines rather than parsing YAML because this project takes on no
// YAML dependency, and the frontmatter here is always flat scalars.
func frontmatterValue(block, key string) string {
	for _, line := range strings.Split(block, "\n") {
		name, value, ok := strings.Cut(strings.TrimSpace(line), ":")
		if !ok || name != key {
			continue
		}
		return strings.Trim(strings.TrimSpace(value), `"'`)
	}
	return ""
}

// skippedWalkDirs are directories never worth walking for memory files.
var skippedWalkDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"vendor":       true,
}

// claudeMDIn returns the path of the first CLAUDE.md-family file in dir, or
// "" when the directory has none. Claude Code treats any of the three as a
// reason to read CLAUDE.md instead of AGENTS.md.
func claudeMDIn(dir string) string {
	for _, name := range []string{"CLAUDE.md", filepath.Join(".claude", "CLAUDE.md"), "CLAUDE.local.md"} {
		p := filepath.Join(dir, name)
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return p
		}
	}
	return ""
}

// onDemandSubdirFiles finds CLAUDE.md and CLAUDE.local.md below the working
// directory. Claude Code loads these only when it reads files in those
// directories, so they are reported separately from the launch tier.
func (a *Agent) onDemandSubdirFiles() []agent.Memory {
	if a.paths.CWD == "" {
		return nil
	}

	var files []agent.Memory
	_ = filepath.WalkDir(a.paths.CWD, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil //nolint:nilerr // an unreadable directory should not fail the audit
		}
		if d.IsDir() {
			name := d.Name()
			if path != a.paths.CWD && (skippedWalkDirs[name] || strings.HasPrefix(name, ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Dir(path) == a.paths.CWD {
			return nil // the working directory's own files are launch tier
		}
		switch d.Name() {
		case "CLAUDE.md":
			files = append(files, readMemoryFile(path, agent.ScopeProject, agent.MemoryKindClaudeMD, agent.MemoryTierOnDemand))
		case "CLAUDE.local.md":
			files = append(files, readMemoryFile(path, agent.ScopeLocal, agent.MemoryKindClaudeLocalMD, agent.MemoryTierOnDemand))
		case "AGENTS.md":
			// Per-directory test, not the global one: a subdirectory's own
			// CLAUDE.md stops Claude Code reading its AGENTS.md.
			if claudeMDIn(filepath.Dir(path)) != "" {
				return nil
			}
			files = append(files, readMemoryFile(path, agent.ScopeProject, agent.MemoryKindAgentsMD, agent.MemoryTierOnDemand))
		}
		return nil
	})
	return files
}
