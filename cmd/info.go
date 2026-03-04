package cmd

import (
	"fmt"

	"github.com/jackchuka/ccli/internal/agent"
	"github.com/jackchuka/ccli/internal/output"
	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show Claude Code installation summary",
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := getAgent()
		if err != nil {
			return err
		}
		info, err := a.Info()
		if err != nil {
			return err
		}
		p, err := getPrinter()
		if err != nil {
			return err
		}
		if p.Format() != output.FormatText {
			return p.Print(info)
		}
		return renderInfo(p, info)
	},
}

func renderInfo(p *output.Printer, info *agent.InstallInfo) error {
	noColor := p.NoColor()

	p.PrintText(output.RenderTitle("◆ Claude Code", noColor))
	p.PrintText(fmt.Sprintf("  Version   %s", info.Version))
	if info.AuthStatus != "" {
		p.PrintText(fmt.Sprintf("  Auth      ● %s", info.AuthStatus))
	}
	p.PrintText(fmt.Sprintf("  Model     %s", info.Model))
	p.PrintText("")
	p.PrintText(output.RenderDivider("Paths", noColor))
	p.PrintText(fmt.Sprintf("  Config     %s", info.ConfigPath))
	p.PrintText(fmt.Sprintf("  Settings   %s", info.SettingsPath))
	historyStr := info.HistoryPath
	if info.HistoryCount > 0 {
		historyStr += fmt.Sprintf(" (%d entries)", info.HistoryCount)
	}
	p.PrintText(fmt.Sprintf("  History    %s", historyStr))
	p.PrintText("")
	p.PrintText(output.RenderDivider("Stats", noColor))
	p.PrintText(fmt.Sprintf("  Sessions   %d", info.SessionCount))
	p.PrintText(fmt.Sprintf("  Projects   %d", info.ProjectCount))
	p.PrintText(fmt.Sprintf("  Storage    %s", output.FormatBytes(info.StorageBytes)))
	p.PrintText(output.RenderDim(fmt.Sprintf("  ◈ %d MCP servers   ◈ %d Skills   ◈ %d Plugins",
		info.MCPCount, info.SkillCount, info.PluginCount), noColor))

	return nil
}

func init() {
	rootCmd.AddCommand(infoCmd)
}
