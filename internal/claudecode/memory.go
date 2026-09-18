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
	memoryAdvisoryLines = 200          // per CLAUDE.md: longer files reduce adherence
	memoryHardByteLimit = 4 << 20      // 4 MiB: Claude Code skips a larger CLAUDE.md
	autoIndexMaxLines   = 200          // MEMORY.md: only the first 200 lines load
	autoIndexMaxBytes   = 25 * 1024    // MEMORY.md: or the first 25KB, whichever comes first
	maxImportDepth      = 4            // @path imports expand at most four hops
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

// ReadMemoryFile exposes readMemoryFile for tests.
func ReadMemoryFile(path string, scope agent.Scope, kind agent.MemoryKind, tier agent.MemoryTier) agent.Memory {
	return readMemoryFile(path, scope, kind, tier)
}

// GlobMatch exposes globMatch for tests.
func GlobMatch(pattern, path string) bool { return globMatch(pattern, path) }

// ExpandTilde exposes expandTilde for tests.
func ExpandTilde(path, home string) string { return expandTilde(path, home) }
