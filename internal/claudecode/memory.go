package claudecode

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jackchuka/ccli/internal/agent"
)

// Thresholds Claude Code documents for memory files. Lines and bytes are the
// units the docs use, so they are the units this audit reports.
const (
	memoryAdvisoryLines = 200       // per CLAUDE.md: longer files reduce adherence
	memoryHardByteLimit = 4 << 20   // 4 MiB: Claude Code skips a larger CLAUDE.md
	autoIndexMaxLines   = 200       // MEMORY.md: only the first 200 lines load
	autoIndexMaxBytes   = 25 * 1024 // MEMORY.md: or the first 25KB, whichever comes first
	maxImportDepth      = 4         // @path imports expand at most four hops
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
		)...)
	}

	// 4. The working directory: ./CLAUDE.md, ./.claude/CLAUDE.md, ./CLAUDE.local.md.
	if a.paths.CWD != "" {
		files = append(files,
			readMemoryFile(filepath.Join(a.paths.CWD, "CLAUDE.md"), agent.ScopeProject, agent.MemoryKindClaudeMD, agent.MemoryTierLaunch),
			readMemoryFile(filepath.Join(a.paths.CWD, ".claude", "CLAUDE.md"), agent.ScopeProject, agent.MemoryKindClaudeMD, agent.MemoryTierLaunch),
			readMemoryFile(filepath.Join(a.paths.CWD, "CLAUDE.local.md"), agent.ScopeLocal, agent.MemoryKindClaudeLocalMD, agent.MemoryTierLaunch),
		)
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

// LaunchTierFiles exposes launchTierFiles for tests, loading settings itself.
func (a *Agent) LaunchTierFiles() []agent.Memory {
	return a.launchTierFiles(LoadMergedSettings(a.paths))
}

// ReadMemoryFile exposes readMemoryFile for tests.
func ReadMemoryFile(path string, scope agent.Scope, kind agent.MemoryKind, tier agent.MemoryTier) agent.Memory {
	return readMemoryFile(path, scope, kind, tier)
}

// GlobMatch exposes globMatch for tests.
func GlobMatch(pattern, path string) bool { return globMatch(pattern, path) }

// ExpandTilde exposes expandTilde for tests.
func ExpandTilde(path, home string) string { return expandTilde(path, home) }
