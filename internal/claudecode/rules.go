package claudecode

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jackchuka/ccli/internal/agent"
)

// ListRules returns rule files from global and project rules directories.
func (a *Agent) ListRules() ([]agent.Rule, error) {
	var rules []agent.Rule

	// Global: ~/.claude/rules/
	global, err := readRulesDir(a.paths.RulesDir, agent.ScopeGlobal, "~/.claude/rules")
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("global rules: %w", err)
	}
	rules = append(rules, global...)

	// Project: .claude/rules/ (if in a project with .claude/)
	if a.paths.ProjectDir != "" {
		projectDir := filepath.Join(a.paths.ProjectDir, "rules")
		project, err := readRulesDir(projectDir, agent.ScopeProject, ".claude/rules")
		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("project rules: %w", err)
		}
		rules = append(rules, project...)
	}

	return rules, nil
}

// GetRule finds a rule by name across all scopes.
func (a *Agent) GetRule(name string) (*agent.Rule, error) {
	rules, err := a.ListRules()
	if err != nil {
		return nil, err
	}
	return findByName(rules, name, func(r agent.Rule) string { return r.Name }, "rule")
}

func readRulesDir(dir string, scope agent.Scope, source string) ([]agent.Rule, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var rules []agent.Rule
	for _, e := range entries {
		if e.IsDir() || e.Name()[0] == '.' {
			continue
		}
		paths := parseRulePaths(filepath.Join(dir, e.Name()))
		rules = append(rules, agent.Rule{
			Name:   e.Name(),
			Scope:  scope,
			Source: source,
			Paths:  paths,
		})
	}
	return rules, nil
}

// parseRulePaths extracts the paths list from YAML frontmatter.
func parseRulePaths(filePath string) []string {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}
	block := extractFrontmatterBlock(string(data))
	if block == "" {
		return nil
	}
	var paths []string
	inPaths := false
	for _, line := range strings.Split(block, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "paths:" {
			inPaths = true
			continue
		}
		if inPaths {
			if strings.HasPrefix(trimmed, "- ") {
				val := strings.TrimPrefix(trimmed, "- ")
				val = strings.Trim(val, "\"'")
				paths = append(paths, val)
			} else {
				break
			}
		}
	}
	return paths
}
