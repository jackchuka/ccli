package cmd

import (
	"fmt"
	"sort"

	"github.com/jackchuka/ccli/internal/agent"
	"github.com/jackchuka/ccli/internal/output"
	"github.com/spf13/cobra"
)

var skillsCmd = &cobra.Command{
	Use:   "skills",
	Short: "Inspect installed skills",
}

var skillsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all skills across all scopes",
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := getAgent()
		if err != nil {
			return err
		}
		skills, err := a.ListSkills()
		if err != nil {
			return err
		}
		p, err := getPrinter()
		if err != nil {
			return err
		}
		if p.Format() != output.FormatText {
			return p.Print(skills)
		}
		return renderSkillList(p, skills)
	},
}

var skillsGetCmd = &cobra.Command{
	Use:   "get <name>",
	Short: "Show details for a specific skill",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := getAgent()
		if err != nil {
			return err
		}
		skill, err := a.GetSkill(args[0])
		if err != nil {
			return err
		}
		p, err := getPrinter()
		if err != nil {
			return err
		}
		if p.Format() != output.FormatText {
			return p.Print(skill)
		}
		return renderSkillGet(p, skill)
	},
}

func renderSkillList(p *output.Printer, skills []agent.Skill) error {
	noColor := p.NoColor()
	p.PrintText(output.RenderDivider("Skills", noColor))
	hasLinks := false
	for _, s := range skills {
		if s.LinkTarget != "" {
			hasLinks = true
			break
		}
	}

	header := fmt.Sprintf("    %-30s %-10s %s", "NAME", "SCOPE", "SOURCE")
	if hasLinks {
		header = fmt.Sprintf("    %-30s %-10s %-20s %s", "NAME", "SCOPE", "SOURCE", "LINK")
	}
	p.PrintText(output.RenderHeader(header, noColor))

	sort.Slice(skills, func(i, j int) bool {
		oi, oj := scopeOrder(skills[i].Scope), scopeOrder(skills[j].Scope)
		if oi != oj {
			return oi < oj
		}
		return skills[i].Name < skills[j].Name
	})

	scopeCounts := map[agent.Scope]int{}
	for _, s := range skills {
		scopeCounts[s.Scope]++
		bullet := output.RenderScopeBullet(string(s.Scope), noColor)
		if hasLinks {
			link := ""
			if s.LinkTarget != "" {
				link = "→ " + s.LinkTarget
			}
			p.PrintText(fmt.Sprintf("  %s %-30s %-10s %-20s %s", bullet, s.Name, s.Scope, s.Source, link))
		} else {
			p.PrintText(fmt.Sprintf("  %s %-30s %-10s %s", bullet, s.Name, s.Scope, s.Source))
		}
	}
	return renderScopeSummary(p, "skills", scopeCounts, len(skills), []agent.Scope{agent.ScopePlugin, agent.ScopePersonal, agent.ScopeProject})
}

func renderSkillGet(p *output.Printer, s *agent.Skill) error {
	p.PrintText(fmt.Sprintf("  Name:        %s", s.Name))
	p.PrintText(fmt.Sprintf("  Scope:       %s", s.Scope))
	p.PrintText(fmt.Sprintf("  Path:        %s", s.Path))
	if s.Description != "" {
		p.PrintText(fmt.Sprintf("  Description: %s", s.Description))
	}
	return nil
}

func scopeOrder(s agent.Scope) int {
	switch s {
	case agent.ScopePlugin:
		return 0
	case agent.ScopePersonal:
		return 1
	case agent.ScopeProject:
		return 2
	default:
		return 3
	}
}

func init() {
	skillsCmd.AddCommand(skillsListCmd)
	skillsCmd.AddCommand(skillsGetCmd)
	rootCmd.AddCommand(skillsCmd)
}
