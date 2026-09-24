package claudecode

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jackchuka/ccli/internal/agent"
)

// InstructionMode selects which instruction files Claude Code loads. It is
// configured per install under the built-in agents-md plugin, and it changes
// what the whole audit means: the same repository reports differently under
// different modes.
type InstructionMode string

const (
	// ModeClaudeMDOrAgentsMD reads AGENTS.md only when no CLAUDE.md-family
	// file exists in the working directory or above it. Claude Code's default.
	ModeClaudeMDOrAgentsMD InstructionMode = "claude-md-or-agents-md"
	// ModeClaudeMDAndAgentsMD reads both, each directory's CLAUDE.md files
	// before its AGENTS.md.
	ModeClaudeMDAndAgentsMD InstructionMode = "claude-md-and-agents-md"
	// ModeClaudeMDOnly ignores AGENTS.md entirely.
	ModeClaudeMDOnly InstructionMode = "claude-md"
	// ModeManagedOnly loads only the organization's managed CLAUDE.md and auto
	// memory at launch.
	ModeManagedOnly InstructionMode = "managed-only"
)

// ParseInstructionMode resolves a settings value to a mode. An empty or
// unrecognized value falls back to the default rather than failing: a
// configuration this tool does not recognize should still produce an audit.
func ParseInstructionMode(v string) InstructionMode {
	switch m := InstructionMode(v); m {
	case ModeClaudeMDOrAgentsMD, ModeClaudeMDAndAgentsMD, ModeClaudeMDOnly, ModeManagedOnly:
		return m
	default:
		return ModeClaudeMDOrAgentsMD
	}
}

// shadowingClaudeMD returns the CLAUDE.md-family file that stops Claude Code
// reading AGENTS.md, or "" when none exists. It walks the ancestor chain
// itself rather than scanning the discovered tree, because the files that
// count here are not the files that load: an ancestor's .claude/CLAUDE.md
// shadows AGENTS.md although it is never loaded. It searches nearest-first so
// the warning names the file a user would go and look at.
//
// personalDir is the personal config directory (Paths.HomeDir, ~/.claude).
// The personal and managed CLAUDE.md are excluded from shadowing by scope,
// but this predicate has no scope to read, only paths to stat, so the one
// path that is both an ancestor candidate and the personal file —
// <homeDir>/.claude/CLAUDE.md, when the working directory is under $HOME —
// must be skipped explicitly. The managed CLAUDE.md needs no equivalent
// guard: its OS-specific path is never an ancestor of a working directory.
func shadowingClaudeMD(cwd, personalDir string) string {
	chain := ancestorDirs(cwd)
	for i := len(chain) - 1; i >= 0; i-- {
		if p := claudeMDInExceptPersonal(chain[i], personalDir); p != "" {
			return p
		}
	}
	return ""
}

// claudeMDInExceptPersonal mirrors claudeMDIn's candidate list and priority
// order, but skips the .claude/CLAUDE.md candidate when dir's .claude is the
// personal config directory itself — that candidate is the personal memory
// file, not a project one, and must not shadow.
func claudeMDInExceptPersonal(dir, personalDir string) string {
	for _, name := range []string{"CLAUDE.md", filepath.Join(".claude", "CLAUDE.md"), "CLAUDE.local.md"} {
		if personalDir != "" && name == filepath.Join(".claude", "CLAUDE.md") && filepath.Join(dir, ".claude") == personalDir {
			continue
		}
		p := filepath.Join(dir, name)
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return p
		}
	}
	return ""
}

// applyInstructionMode records which discovered files the active mode does not
// read. It marks rather than filters, so a shadowed file still appears in the
// audit with an explanation — a user who keeps both a CLAUDE.md and an
// AGENTS.md most needs to know that one of them is doing nothing. The strings
// it returns are report-level warnings with no file to hang from.
func applyInstructionMode(files []agent.Memory, mode InstructionMode, cwd, personalDir string) []string {
	switch mode {
	case ModeClaudeMDOnly:
		markAgentsMD(files, fmt.Sprintf("not read in %s mode", ModeClaudeMDOnly))
	case ModeManagedOnly:
		markManagedOnly(files)
		return []string{fmt.Sprintf(
			"%s mode also excludes .claude/rules/, which this audit does not enumerate; run ccli rules to see them",
			ModeManagedOnly)}
	case ModeClaudeMDAndAgentsMD:
		markAlreadyImportedAgentsMD(files)
	default:
		if shadow := shadowingClaudeMD(cwd, personalDir); shadow != "" {
			markAgentsMD(files, fmt.Sprintf("shadowed by %s; not read in %s mode", shadow, ModeClaudeMDOrAgentsMD))
		}
	}
	return nil
}

// markAgentsMD marks each top-level AGENTS.md as not loading. It does not
// recurse into Imports: an import node is always constructed with
// MemoryKindImport regardless of what file it targets, so it can never match
// MemoryKindAgentsMD — an explicit @AGENTS.md import is read whatever the
// mode, since the mode governs discovery, not an explicit request to include
// a file.
func markAgentsMD(files []agent.Memory, reason string) {
	for i := range files {
		if files[i].Kind == agent.MemoryKindAgentsMD {
			files[i].NotLoaded = true
			files[i].Warnings = append(files[i].Warnings, reason)
		}
	}
}

// markManagedOnly leaves the organization's managed CLAUDE.md and auto memory
// loading and marks the rest of the launch tier. On-demand entries are
// untouched: a subdirectory's CLAUDE.md still loads when Claude reads a file
// there, even in this mode.
func markManagedOnly(files []agent.Memory) {
	reason := fmt.Sprintf("not read in %s mode", ModeManagedOnly)
	for i := range files {
		if files[i].Tier != agent.MemoryTierLaunch {
			continue
		}
		if files[i].Scope == agent.ScopeManaged || files[i].Kind == agent.MemoryKindAutoIndex {
			continue
		}
		files[i].NotLoaded = true
		files[i].Warnings = append(files[i].Warnings, reason)
		markSubtreeNotLoaded(files[i].Imports, reason)
	}
}

// markSubtreeNotLoaded cascades a refusal onto imports, which cannot load when
// the file importing them does not.
func markSubtreeNotLoaded(files []agent.Memory, reason string) {
	for i := range files {
		files[i].NotLoaded = true
		files[i].Warnings = append(files[i].Warnings, reason)
		markSubtreeNotLoaded(files[i].Imports, reason)
	}
}

// markAlreadyImportedAgentsMD marks an AGENTS.md that some other file already
// pulls in. Claude Code skips one it has already loaded, so counting the
// top-level entry as well would double its contribution to the totals.
func markAlreadyImportedAgentsMD(files []agent.Memory) {
	loaded := map[string]bool{}
	var collect func(ms []agent.Memory, nested bool)
	collect = func(ms []agent.Memory, nested bool) {
		for _, m := range ms {
			// A node past the hop limit, or one that cycles back to its own
			// ancestor, is recorded but never loaded; if it happens to name
			// an AGENTS.md, that copy contributes nothing, so it must not
			// suppress the top-level AGENTS.md as "already loaded".
			if nested && m.Path != "" && m.Loads() {
				loaded[m.Path] = true
			}
			if m.LinkTarget != "" {
				loaded[m.LinkTarget] = true
			}
			collect(m.Imports, true)
		}
	}
	collect(files, false)

	for i := range files {
		if files[i].Kind == agent.MemoryKindAgentsMD && loaded[files[i].Path] {
			files[i].NotLoaded = true
			files[i].Warnings = append(files[i].Warnings,
				"already loaded through an import, so Claude Code does not read it twice")
		}
	}
}
