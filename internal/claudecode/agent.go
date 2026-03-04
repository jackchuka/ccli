package claudecode

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Paths holds the file paths the Agent reads from.
type Paths struct {
	ConfigFile   string // ~/.claude.json
	SettingsFile string // ~/.claude/settings.json
	MCPFile      string // .mcp.json (project root)
	HomeDir      string // ~/.claude/
	PluginsDir   string // ~/.claude/plugins/
	SkillsDir    string // ~/.claude/skills/
	RulesDir     string // ~/.claude/rules/
	ProjectDir   string // current project .claude/
}

// DefaultPaths builds Paths from the user's home and current working directory.
func DefaultPaths() (Paths, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Paths{}, fmt.Errorf("determining home directory: %w", err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return Paths{}, fmt.Errorf("determining working directory: %w", err)
	}
	claudeHome := filepath.Join(home, ".claude")
	return Paths{
		ConfigFile:   filepath.Join(home, ".claude.json"),
		SettingsFile: filepath.Join(claudeHome, "settings.json"),
		MCPFile:      filepath.Join(cwd, ".mcp.json"),
		HomeDir:      claudeHome,
		PluginsDir:   filepath.Join(claudeHome, "plugins"),
		SkillsDir:    filepath.Join(claudeHome, "skills"),
		RulesDir:     filepath.Join(claudeHome, "rules"),
		ProjectDir:   filepath.Join(cwd, ".claude"),
	}, nil
}

// Agent implements the agent.Agent interface for Claude Code.
type Agent struct {
	paths Paths
}

// NewAgent creates a Claude Code agent with the given paths.
func NewAgent(p Paths) *Agent {
	return &Agent{paths: p}
}

func (a *Agent) Name() string { return "claude-code" }

// findByName searches a slice for an item where key(item) matches name.
func findByName[T any](items []T, name string, key func(T) string, label string) (*T, error) {
	for _, item := range items {
		if key(item) == name {
			return &item, nil
		}
	}
	return nil, fmt.Errorf("%s %q not found", label, name)
}

// extractFrontmatterBlock returns the raw YAML content between --- delimiters.
// Returns empty string if no valid frontmatter is found.
func extractFrontmatterBlock(content string) string {
	if !strings.HasPrefix(content, "---\n") {
		return ""
	}
	end := strings.Index(content[4:], "\n---")
	if end < 0 {
		return ""
	}
	return content[4 : 4+end]
}
