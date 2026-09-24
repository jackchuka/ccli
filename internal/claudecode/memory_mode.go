package claudecode

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
