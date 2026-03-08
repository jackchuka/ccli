package claudecode

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jackchuka/ccli/internal/agent"
)

// skillFrontmatter holds the parsed frontmatter from a SKILL.md file.
type skillFrontmatter struct {
	Name        string
	Description string
}

// ListSkills discovers skills from personal, project, and plugin directories.
func (a *Agent) ListSkills() ([]agent.Skill, error) {
	var skills []agent.Skill

	// Personal skills: ~/.claude/skills/*/
	if a.paths.SkillsDir != "" {
		personal, err := discoverSkills(a.paths.SkillsDir, agent.ScopePersonal, "~/.claude/skills")
		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("personal skills: %w", err)
		}
		skills = append(skills, personal...)
	}

	// Project skills: .claude/skills/*/
	if a.paths.ProjectDir != "" {
		projectSkillsDir := filepath.Join(a.paths.ProjectDir, "skills")
		project, err := discoverSkills(projectSkillsDir, agent.ScopeProject, ".claude/skills")
		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("project skills: %w", err)
		}
		skills = append(skills, project...)
	}

	// Plugin skills: ~/.claude/plugins/cache/*/*/*/skills/*/
	if a.paths.PluginsDir != "" {
		plugin, err := discoverPluginSkills(a.paths.PluginsDir)
		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("plugin skills: %w", err)
		}
		skills = append(skills, plugin...)
	}

	return skills, nil
}

// GetSkill finds a skill by name across all scopes.
func (a *Agent) GetSkill(name string) (*agent.Skill, error) {
	skills, err := a.ListSkills()
	if err != nil {
		return nil, err
	}
	return findByName(skills, name, func(s agent.Skill) string { return s.Name }, "skill")
}

func discoverSkills(dir string, scope agent.Scope, displaySource string) ([]agent.Skill, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var skills []agent.Skill
	for _, e := range entries {
		fullPath := filepath.Join(dir, e.Name())
		info, err := os.Stat(fullPath)
		if err != nil || !info.IsDir() {
			continue
		}
		skillFile := findSkillFile(fullPath)
		if skillFile == "" {
			continue
		}
		fm, err := loadFrontmatter(skillFile)
		if err != nil {
			continue
		}
		name := fm.Name
		if name == "" {
			name = e.Name()
		}
		skills = append(skills, agent.Skill{
			Name:        name,
			Scope:       scope,
			Source:      displaySource,
			Path:        skillFile,
			Description: fm.Description,
		})
	}
	return skills, nil
}

func discoverPluginSkills(pluginsDir string) ([]agent.Skill, error) {
	cacheDir := filepath.Join(pluginsDir, "cache")
	matches, err := filepath.Glob(filepath.Join(cacheDir, "*", "*", "*", "skills", "*"))
	if err != nil {
		return nil, err
	}
	var skills []agent.Skill
	seen := map[string]bool{}
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil || !info.IsDir() {
			continue
		}
		skillFile := findSkillFile(match)
		if skillFile == "" {
			continue
		}
		fm, err := loadFrontmatter(skillFile)
		if err != nil {
			continue
		}
		name := fm.Name
		if name == "" {
			name = filepath.Base(match)
		}
		if seen[name] {
			continue
		}
		seen[name] = true

		rel, _ := filepath.Rel(cacheDir, match)
		parts := strings.Split(rel, string(filepath.Separator))
		source := parts[1] // plugin name

		skills = append(skills, agent.Skill{
			Name:        name,
			Scope:       agent.ScopePlugin,
			Source:      source,
			Path:        skillFile,
			Description: fm.Description,
		})
	}
	return skills, nil
}

func findSkillFile(dir string) string {
	skillPath := filepath.Join(dir, "SKILL.md")
	if _, err := os.Stat(skillPath); err == nil {
		return skillPath
	}
	return ""
}

func loadFrontmatter(path string) (*skillFrontmatter, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseFrontmatter(string(data))
}

// parseFrontmatter extracts YAML frontmatter from markdown content.
func parseFrontmatter(content string) (*skillFrontmatter, error) {
	block := extractFrontmatterBlock(content)
	if block == "" {
		return &skillFrontmatter{}, nil
	}
	var fm skillFrontmatter
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(line)
		if idx := strings.Index(line, ": "); idx > 0 {
			key := line[:idx]
			val := line[idx+2:]
			switch key {
			case "name":
				fm.Name = val
			case "description":
				fm.Description = val
			}
		}
	}
	return &fm, nil
}
