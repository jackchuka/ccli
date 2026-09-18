package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jackchuka/ccli/internal/agent"
	"github.com/jackchuka/ccli/internal/output"
	"github.com/spf13/cobra"
)

var memoryOnDemand bool

var memoryCmd = &cobra.Command{
	Use:   "memory",
	Short: "Audit the memory files that load into a Claude Code session",
	Long: "Reports which CLAUDE.md files, imports, and auto memory load for the\n" +
		"current directory, in load order, with their context cost.\n\n" +
		"The audit reflects Claude Code's documented resolution rules. To confirm\n" +
		"what a live session actually loaded, run /context in that session.",
}

var memoryListCmd = &cobra.Command{
	Use:   "list",
	Short: "List memory files in load order with their context cost",
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := getAgent()
		if err != nil {
			return err
		}
		report, err := a.ListMemory()
		if err != nil {
			return err
		}
		p, err := getPrinter()
		if err != nil {
			return err
		}
		if p.Format() != output.FormatText {
			return p.Print(report)
		}
		paths, err := getPaths()
		if err != nil {
			return err
		}
		return renderMemoryList(p, report, paths.UserHomeDir, paths.CWD, memoryOnDemand)
	},
}

var memoryGetCmd = &cobra.Command{
	Use:   "get <name>",
	Short: "Show details for one memory file, by scope name or path",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := getAgent()
		if err != nil {
			return err
		}
		m, err := a.GetMemory(args[0])
		if err != nil {
			return err
		}
		p, err := getPrinter()
		if err != nil {
			return err
		}
		if p.Format() != output.FormatText {
			return p.Print(m)
		}
		paths, err := getPaths()
		if err != nil {
			return err
		}
		return renderMemoryGet(p, m, paths.UserHomeDir, paths.CWD)
	},
}

func renderMemoryList(p *output.Printer, report *agent.MemoryReport, home, cwd string, showOnDemand bool) error {
	noColor := p.NoColor()
	if err := p.PrintText(output.RenderDivider("Memory", noColor)); err != nil {
		return err
	}
	for _, f := range report.Files {
		if f.Tier != agent.MemoryTierLaunch {
			continue
		}
		if err := renderMemoryNode(p, f, home, cwd, 0); err != nil {
			return err
		}
	}

	if err := p.PrintText(""); err != nil {
		return err
	}
	if err := p.PrintText(output.RenderDim(fmt.Sprintf("  loaded at launch: %d files · %dL · %s",
		report.LaunchFiles, report.LaunchLines, output.FormatBytes(report.LaunchBytes)), noColor)); err != nil {
		return err
	}
	if err := p.PrintText(output.RenderDim(fmt.Sprintf("  on demand:        %d files",
		report.OnDemandFiles), noColor)); err != nil {
		return err
	}

	if showOnDemand {
		if err := p.PrintText(""); err != nil {
			return err
		}
		if err := p.PrintText(output.RenderDivider("On demand", noColor)); err != nil {
			return err
		}
		for _, f := range report.Files {
			if f.Tier != agent.MemoryTierOnDemand {
				continue
			}
			if err := renderMemoryNode(p, f, home, cwd, 0); err != nil {
				return err
			}
		}
	}

	if len(report.Warnings) > 0 {
		if err := p.PrintText(""); err != nil {
			return err
		}
		if err := p.PrintText(output.RenderDivider("Warnings", noColor)); err != nil {
			return err
		}
		for _, w := range report.Warnings {
			if err := p.PrintText("  ! " + w); err != nil {
				return err
			}
		}
	}
	return nil
}

// renderMemoryNode prints one file and, indented beneath it, the files it
// imports. Import rows leave the bullet and scope columns blank so the size
// columns stay aligned with their parent.
func renderMemoryNode(p *output.Printer, m agent.Memory, home, cwd string, depth int) error {
	noColor := p.NoColor()

	size := fmt.Sprintf("%5dL %10s", m.Lines, output.FormatBytes(m.Bytes))
	if !m.Exists {
		size = fmt.Sprintf("%5s %10s", "——", "(absent)")
	}

	bullet, scope := " ", ""
	name := memoryDisplayPath(m.Path, home, cwd)
	if depth == 0 {
		bullet = output.RenderScopeBullet(string(m.Scope), noColor)
		scope = string(m.Scope)
	} else {
		name = strings.Repeat("   ", depth+1) + "└─ @" + name
	}

	line := fmt.Sprintf("  %s %-9s %-44s %s", bullet, scope, name, size)
	if m.LinkTarget != "" {
		line += output.RenderDim("  ⇢ "+memoryDisplayPath(m.LinkTarget, home, cwd), noColor)
	}
	if m.External {
		line += output.RenderDim("  ⚠ external", noColor)
	}
	if m.Excluded {
		line += output.RenderDim("  excluded: "+m.ExcludedBy, noColor)
	}
	if err := p.PrintText(line); err != nil {
		return err
	}

	for _, im := range m.Imports {
		if err := renderMemoryNode(p, im, home, cwd, depth+1); err != nil {
			return err
		}
	}
	return nil
}

func renderMemoryGet(p *output.Printer, m *agent.Memory, home, cwd string) error {
	rows := [][2]string{
		{"Path", memoryDisplayPath(m.Path, home, cwd)},
		{"Scope", string(m.Scope)},
		{"Kind", string(m.Kind)},
		{"Tier", string(m.Tier)},
		{"Exists", fmt.Sprintf("%t", m.Exists)},
		{"Lines", fmt.Sprintf("%d", m.Lines)},
		{"Size", output.FormatBytes(m.Bytes)},
	}
	if m.LinkTarget != "" {
		rows = append(rows, [2]string{"Link target", memoryDisplayPath(m.LinkTarget, home, cwd)})
	}
	if m.Excluded {
		rows = append(rows, [2]string{"Excluded by", m.ExcludedBy})
	}
	if m.Type != "" {
		rows = append(rows, [2]string{"Type", m.Type})
	}
	if m.Modified != "" {
		rows = append(rows, [2]string{"Modified", m.Modified})
	}
	for _, r := range rows {
		if err := p.PrintText(fmt.Sprintf("  %-12s %s", r[0], r[1])); err != nil {
			return err
		}
	}

	if len(m.Imports) > 0 {
		var names []string
		for _, im := range m.Imports {
			names = append(names, memoryDisplayPath(im.Path, home, cwd))
		}
		if err := p.PrintText(fmt.Sprintf("  %-12s %s", "Imports", strings.Join(names, ", "))); err != nil {
			return err
		}
	}
	if len(m.Warnings) > 0 {
		for _, w := range m.Warnings {
			if err := p.PrintText("  ! " + w); err != nil {
				return err
			}
		}
	}
	return nil
}

// memoryDisplayPath shortens a path for display: inside the working
// directory it becomes ./-relative, inside the home directory it gets a ~.
func memoryDisplayPath(path, home, cwd string) string {
	if path == "" {
		return "(inline)"
	}
	if cwd != "" {
		if rel, err := filepath.Rel(cwd, path); err == nil && !strings.HasPrefix(rel, "..") {
			return "./" + rel
		}
	}
	if home != "" && strings.HasPrefix(path, home+string(filepath.Separator)) {
		return "~" + path[len(home):]
	}
	return path
}

func init() {
	memoryListCmd.Flags().BoolVar(&memoryOnDemand, "on-demand", false,
		"also list the files Claude Code loads on demand rather than at launch")
	memoryCmd.AddCommand(memoryListCmd)
	memoryCmd.AddCommand(memoryGetCmd)
	rootCmd.AddCommand(memoryCmd)
}
