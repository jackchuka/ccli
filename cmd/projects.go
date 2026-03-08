package cmd

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackchuka/ccli/internal/agent"
	"github.com/jackchuka/ccli/internal/output"
	"github.com/spf13/cobra"
)

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "Inspect known projects",
}

var projectsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all known projects with usage stats",
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := getAgent()
		if err != nil {
			return err
		}
		projects, err := a.ListProjects()
		if err != nil {
			return err
		}
		p, err := getPrinter()
		if err != nil {
			return err
		}
		if p.Format() != output.FormatText {
			return p.Print(projects)
		}
		return renderProjectList(p, projects)
	},
}

var projectsGetCmd = &cobra.Command{
	Use:   "get <name-or-path>",
	Short: "Show details for a specific project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := getAgent()
		if err != nil {
			return err
		}
		project, err := a.GetProject(args[0])
		if err != nil {
			return err
		}
		p, err := getPrinter()
		if err != nil {
			return err
		}
		if p.Format() != output.FormatText {
			return p.Print(project)
		}
		return renderProjectGet(p, project)
	},
}

func renderProjectList(p *output.Printer, projects []agent.Project) error {
	noColor := p.NoColor()
	p.PrintText(output.RenderDivider("Projects", noColor))

	sort.Slice(projects, func(i, j int) bool {
		return projects[i].LastCost > projects[j].LastCost
	})

	displayNames := disambiguateNames(projects)

	nameWidth := len("NAME")
	for _, name := range displayNames {
		if len(name) > nameWidth {
			nameWidth = len(name)
		}
	}

	fmtStr := fmt.Sprintf("  %%-%ds %%10s %%8s %%12s", nameWidth)
	p.PrintText(output.RenderHeader(fmt.Sprintf(fmtStr, "NAME", "LAST COST", "SESSIONS", "LINES (+/-)"), noColor))

	for i, proj := range projects {
		cost := ""
		if proj.LastCost > 0 {
			cost = fmt.Sprintf("$%.2f", proj.LastCost)
		}
		sessions := ""
		if proj.SessionCount > 0 {
			sessions = fmt.Sprintf("%d", proj.SessionCount)
		}
		lines := ""
		if proj.LinesAdded > 0 || proj.LinesRemoved > 0 {
			lines = fmt.Sprintf("+%d/-%d", proj.LinesAdded, proj.LinesRemoved)
		}
		p.PrintText(fmt.Sprintf(fmtStr, displayNames[i], cost, sessions, lines))
	}
	p.PrintText(output.RenderDim(fmt.Sprintf("  %d projects", len(projects)), noColor))
	return nil
}

// disambiguateNames builds display names for projects, adding parent path
// segments only when the base name collides with another project.
func disambiguateNames(projects []agent.Project) []string {
	names := make([]string, len(projects))

	// Group indices by base name to find duplicates.
	groups := make(map[string][]int)
	for i, proj := range projects {
		groups[proj.Name] = append(groups[proj.Name], i)
	}

	for _, indices := range groups {
		if len(indices) == 1 {
			names[indices[0]] = projects[indices[0]].Name
			continue
		}
		// Split each path into segments and walk backwards until unique.
		paths := make([][]string, len(indices))
		for j, idx := range indices {
			paths[j] = splitPath(projects[idx].Path)
		}
		// Start with 1 segment (the base name) and add parents until unique.
		depth := 2
		for {
			seen := make(map[string]int) // display name -> count
			for j := range indices {
				seen[suffixName(paths[j], depth)]++
			}
			allUnique := true
			for _, c := range seen {
				if c > 1 {
					allUnique = false
					break
				}
			}
			if allUnique || depth > 5 {
				break
			}
			depth++
		}
		for j, idx := range indices {
			names[idx] = suffixName(paths[j], depth)
		}
	}
	return names
}

func splitPath(p string) []string {
	p = filepath.Clean(p)
	var parts []string
	for p != "/" && p != "." && p != "" {
		parts = append([]string{filepath.Base(p)}, parts...)
		p = filepath.Dir(p)
	}
	return parts
}

func suffixName(parts []string, depth int) string {
	if depth > len(parts) {
		depth = len(parts)
	}
	return strings.Join(parts[len(parts)-depth:], "/")
}

func renderProjectGet(p *output.Printer, proj *agent.Project) error {
	noColor := p.NoColor()
	p.PrintText(output.RenderDivider(proj.Name, noColor))
	p.PrintText(fmt.Sprintf("  Path         %s", proj.Path))
	p.PrintText(fmt.Sprintf("  Trusted      %v", proj.Trusted))
	p.PrintText(fmt.Sprintf("  Sessions     %d", proj.SessionCount))

	if proj.LastCost > 0 {
		p.PrintText("")
		p.PrintText(output.RenderDivider("Last Session", noColor))
		p.PrintText(fmt.Sprintf("  Cost         $%.2f", proj.LastCost))
		p.PrintText(fmt.Sprintf("  Tokens       %s in / %s out", output.FormatCount(proj.InputTokens), output.FormatCount(proj.OutputTokens)))
		p.PrintText(fmt.Sprintf("  Lines        +%d / -%d", proj.LinesAdded, proj.LinesRemoved))
	}

	if len(proj.ModelUsage) > 0 {
		p.PrintText("")
		p.PrintText(output.RenderDivider("Model Costs", noColor))
		type mc struct {
			model string
			cost  float64
		}
		var models []mc
		for m, c := range proj.ModelUsage {
			models = append(models, mc{m, c})
		}
		sort.Slice(models, func(i, j int) bool {
			return models[i].cost > models[j].cost
		})
		for _, m := range models {
			name := output.ShortModelName(m.model)
			p.PrintText(fmt.Sprintf("  %-22s $%.2f", name, m.cost))
		}
	}

	return nil
}

func init() {
	projectsCmd.AddCommand(projectsListCmd)
	projectsCmd.AddCommand(projectsGetCmd)
	rootCmd.AddCommand(projectsCmd)
}
