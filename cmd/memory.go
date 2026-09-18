package cmd

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf8"

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
	if err := p.PrintText(output.RenderDim(fmt.Sprintf("  loaded at launch: %d %s · %dL · %s",
		report.LaunchFiles, pluralize(report.LaunchFiles, "file"), report.LaunchLines, output.FormatBytes(report.LaunchBytes)), noColor)); err != nil {
		return err
	}
	if err := p.PrintText(output.RenderDim(fmt.Sprintf("  on demand:        %d %s",
		report.OnDemandFiles, pluralize(report.OnDemandFiles, "file")), noColor)); err != nil {
		return err
	}

	if showOnDemand {
		if err := p.PrintText(""); err != nil {
			return err
		}
		if err := p.PrintText(output.RenderDivider("On demand", noColor)); err != nil {
			return err
		}
		found := false
		for _, f := range report.Files {
			if f.Tier != agent.MemoryTierOnDemand {
				continue
			}
			found = true
			if err := renderMemoryNode(p, f, home, cwd, 0); err != nil {
				return err
			}
		}
		if !found {
			if err := p.PrintText(output.RenderDim("  (none)", noColor)); err != nil {
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

// nameColWidth is the fixed rune width of the name column. Both top-level
// and nested rows pad or elide into this width so the size columns that
// follow line up regardless of how deep or long a path is.
const nameColWidth = 44

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
	display := memoryDisplayPath(m.Path, home, cwd)
	marker := ""
	if depth == 0 {
		bullet = output.RenderScopeBullet(string(m.Scope), noColor)
		scope = string(m.Scope)
	} else {
		// The tree structure already signals nesting, so the leading "./"
		// that top-level rows show would be redundant here.
		display = strings.TrimPrefix(display, "./")
		marker = strings.Repeat("   ", depth+1) + "└─ @"
	}
	budget := nameColWidth - utf8.RuneCountInString(marker)
	name := marker + elideMiddlePath(display, budget)

	line := fmt.Sprintf("  %s %-9s %-*s %s", bullet, scope, nameColWidth, name, size)
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

// pluralize returns singular unmodified for a count of 1, and with a
// trailing "s" otherwise, for count labels like "3 files".
func pluralize(n int, singular string) string {
	if n == 1 {
		return singular
	}
	return singular + "s"
}

// elideMiddlePath shortens a display path to fit within width runes,
// keeping the leading "./" or "~/" marker and the full basename intact —
// the basename is what a reader scans for, and the fixed-width name column
// is what lets this command render a tree instead of a table. When there
// is no directory component to elide, the path is returned unshortened
// rather than cutting into the basename.
func elideMiddlePath(path string, width int) string {
	if utf8.RuneCountInString(path) <= width {
		return path
	}

	prefix := ""
	rest := path
	if strings.HasPrefix(path, "./") || strings.HasPrefix(path, "~/") {
		prefix, rest = path[:2], path[2:]
	}

	dir, base := filepath.Split(rest)
	if dir == "" {
		return path
	}

	const ellipsis = "…"
	tail := ellipsis + "/" + base
	minimal := prefix + tail
	minimalLen := utf8.RuneCountInString(minimal)
	if minimalLen >= width {
		return minimal
	}

	headBudget := width - minimalLen
	dirRunes := []rune(dir)
	if headBudget > len(dirRunes) {
		headBudget = len(dirRunes)
	}
	head := strings.TrimRight(string(dirRunes[:headBudget]), "/")
	return prefix + head + tail
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
