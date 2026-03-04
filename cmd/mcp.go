package cmd

import (
	"fmt"

	"github.com/jackchuka/ccli/internal/agent"
	"github.com/jackchuka/ccli/internal/output"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Inspect MCP servers",
}

var mcpListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all MCP servers across all scopes",
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := getAgent()
		if err != nil {
			return err
		}
		servers, err := a.ListMCPServers()
		if err != nil {
			return err
		}
		p, err := getPrinter()
		if err != nil {
			return err
		}
		if p.Format() != output.FormatText {
			return p.Print(servers)
		}
		return renderMCPList(p, servers)
	},
}

var mcpGetCmd = &cobra.Command{
	Use:   "get <name>",
	Short: "Show details for a specific MCP server",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := getAgent()
		if err != nil {
			return err
		}
		server, err := a.GetMCPServer(args[0])
		if err != nil {
			return err
		}
		p, err := getPrinter()
		if err != nil {
			return err
		}
		if p.Format() != output.FormatText {
			return p.Print(server)
		}
		return renderMCPGet(p, server)
	},
}

func renderMCPList(p *output.Printer, servers []agent.MCPServer) error {
	noColor := p.NoColor()
	p.PrintText(output.RenderDivider("MCP Servers", noColor))
	p.PrintText(output.RenderHeader(fmt.Sprintf("    %-22s %-10s %s", "NAME", "SCOPE", "TYPE"), noColor))
	scopeCounts := map[agent.Scope]int{}
	for _, s := range servers {
		scopeCounts[s.Scope]++
		bullet := output.RenderScopeBullet(string(s.Scope), noColor)
		p.PrintText(fmt.Sprintf("  %s %-22s %-10s %s", bullet, s.Name, s.Scope, s.Type))
	}
	return renderScopeSummary(p, "servers", scopeCounts, len(servers), []agent.Scope{agent.ScopeGlobal, agent.ScopeProject})
}

func renderMCPGet(p *output.Printer, s *agent.MCPServer) error {
	noColor := p.NoColor()
	bullet := output.RenderScopeBullet(string(s.Scope), noColor)
	p.PrintText(fmt.Sprintf("  %s %-36s %s", bullet, s.Name, s.Scope))
	p.PrintText("")

	if s.Type != "" {
		p.PrintText(fmt.Sprintf("  Type       %s", s.Type))
	}
	if s.Command != "" {
		p.PrintText(fmt.Sprintf("  Command    %s", s.Command))
	}
	if s.URL != "" {
		p.PrintText(fmt.Sprintf("  URL        %s", s.URL))
	}

	if len(s.Env) > 0 {
		p.PrintText("")
		p.PrintText(output.RenderDivider("Environment", noColor))
		for k, v := range s.Env {
			p.PrintText(fmt.Sprintf("  %-18s %s", k, output.MaskEnvValue(k, v)))
		}
	}
	return nil
}

func init() {
	mcpCmd.AddCommand(mcpListCmd)
	mcpCmd.AddCommand(mcpGetCmd)
	rootCmd.AddCommand(mcpCmd)
}
