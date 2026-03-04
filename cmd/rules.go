package cmd

import (
	"fmt"
	"strings"

	"github.com/jackchuka/ccli/internal/agent"
	"github.com/jackchuka/ccli/internal/output"
	"github.com/spf13/cobra"
)

var rulesCmd = &cobra.Command{
	Use:   "rules",
	Short: "Inspect rules files",
}

var rulesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all rules across global and project scopes",
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := getAgent()
		if err != nil {
			return err
		}
		rules, err := a.ListRules()
		if err != nil {
			return err
		}
		p, err := getPrinter()
		if err != nil {
			return err
		}
		if p.Format() != output.FormatText {
			return p.Print(rules)
		}
		return renderRuleList(p, rules)
	},
}

func renderRuleList(p *output.Printer, rules []agent.Rule) error {
	noColor := p.NoColor()
	p.PrintText(output.RenderDivider("Rules", noColor))
	p.PrintText(output.RenderHeader(fmt.Sprintf("    %-30s %-10s %s", "NAME", "SCOPE", "SOURCE"), noColor))
	scopeCounts := map[agent.Scope]int{}
	for _, r := range rules {
		scopeCounts[r.Scope]++
		bullet := output.RenderScopeBullet(string(r.Scope), noColor)
		p.PrintText(fmt.Sprintf("  %s %-30s %-10s %s", bullet, r.Name, r.Scope, r.Source))
	}
	return renderScopeSummary(p, "rules", scopeCounts, len(rules), []agent.Scope{agent.ScopeGlobal, agent.ScopeProject})
}

var rulesGetCmd = &cobra.Command{
	Use:   "get <name>",
	Short: "Show details for a specific rule",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := getAgent()
		if err != nil {
			return err
		}
		rule, err := a.GetRule(args[0])
		if err != nil {
			return err
		}
		p, err := getPrinter()
		if err != nil {
			return err
		}
		if p.Format() != output.FormatText {
			return p.Print(rule)
		}
		return renderRuleGet(p, rule)
	},
}

func renderRuleGet(p *output.Printer, r *agent.Rule) error {
	p.PrintText(fmt.Sprintf("  Name         %s", r.Name))
	p.PrintText(fmt.Sprintf("  Scope        %s", r.Scope))
	p.PrintText(fmt.Sprintf("  Source       %s", r.Source))

	if len(r.Paths) > 0 {
		p.PrintText(fmt.Sprintf("  Paths        %s", strings.Join(r.Paths, ", ")))
	}

	return nil
}

func init() {
	rulesCmd.AddCommand(rulesListCmd)
	rulesCmd.AddCommand(rulesGetCmd)
	rootCmd.AddCommand(rulesCmd)
}
