package cmd

import (
	"fmt"
	"sort"

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
	p.PrintText(output.RenderHeader(fmt.Sprintf("  %-30s %10s %8s %12s", "NAME", "LAST COST", "SESSIONS", "LINES (+/-)"), noColor))

	sort.Slice(projects, func(i, j int) bool {
		return projects[i].LastCost > projects[j].LastCost
	})

	for _, proj := range projects {
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
		name := proj.Name
		if len(name) > 30 {
			name = name[:27] + "..."
		}
		p.PrintText(fmt.Sprintf("  %-30s %10s %8s %12s", name, cost, sessions, lines))
	}
	p.PrintText(output.RenderDim(fmt.Sprintf("  %d projects", len(projects)), noColor))
	return nil
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
