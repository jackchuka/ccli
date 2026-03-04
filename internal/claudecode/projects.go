package claudecode

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jackchuka/ccli/internal/agent"
)

// ListProjects returns all projects from ~/.claude.json with session counts.
func (a *Agent) ListProjects() ([]agent.Project, error) {
	cfg, err := LoadConfig(a.paths.ConfigFile)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	if cfg == nil || len(cfg.Projects) == 0 {
		return nil, nil
	}

	sessionCounts := buildSessionCountMap(filepath.Join(a.paths.HomeDir, "projects"))

	var projects []agent.Project
	for path, pc := range cfg.Projects {
		modelUsage := make(map[string]float64)
		for model, usage := range pc.LastModelUsage {
			modelUsage[model] = usage.CostUSD
		}

		projects = append(projects, agent.Project{
			Path:         path,
			Name:         filepath.Base(path),
			LastCost:     pc.LastCost,
			LinesAdded:   pc.LastLinesAdded,
			LinesRemoved: pc.LastLinesRemoved,
			InputTokens:  pc.LastTotalInputTokens,
			OutputTokens: pc.LastTotalOutputTokens,
			SessionCount: sessionCounts[encodeProjectPath(path)],
			Trusted:      pc.HasTrustDialogAccepted,
			ModelUsage:   modelUsage,
		})
	}

	return projects, nil
}

// GetProject finds a project by path (exact or suffix match).
func (a *Agent) GetProject(query string) (*agent.Project, error) {
	cfg, err := LoadConfig(a.paths.ConfigFile)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	if cfg == nil {
		return nil, fmt.Errorf("project %q not found", query)
	}
	for path, pc := range cfg.Projects {
		name := filepath.Base(path)
		if path != query && name != query && !strings.HasSuffix(path, "/"+query) {
			continue
		}
		sessionCount := countSessionsInDir(
			filepath.Join(a.paths.HomeDir, "projects", encodeProjectPath(path)),
		)
		modelUsage := make(map[string]float64)
		for model, usage := range pc.LastModelUsage {
			modelUsage[model] = usage.CostUSD
		}
		return &agent.Project{
			Path:         path,
			Name:         name,
			LastCost:     pc.LastCost,
			LinesAdded:   pc.LastLinesAdded,
			LinesRemoved: pc.LastLinesRemoved,
			InputTokens:  pc.LastTotalInputTokens,
			OutputTokens: pc.LastTotalOutputTokens,
			SessionCount: sessionCount,
			Trusted:      pc.HasTrustDialogAccepted,
			ModelUsage:   modelUsage,
		}, nil
	}
	return nil, fmt.Errorf("project %q not found", query)
}

// buildSessionCountMap reads all project session directories once and returns
// a map from encoded project path to session count.
func buildSessionCountMap(projectsDir string) map[string]int {
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return nil
	}
	counts := make(map[string]int, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			counts[e.Name()] = countSessionsInDir(filepath.Join(projectsDir, e.Name()))
		}
	}
	return counts
}

// encodeProjectPath encodes a project path the way Claude Code stores it.
// Every non-alphanumeric character is replaced with a single "-".
func encodeProjectPath(path string) string {
	var b strings.Builder
	for _, c := range path {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			b.WriteRune(c)
		} else {
			b.WriteByte('-')
		}
	}
	return b.String()
}
