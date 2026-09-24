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
	if err := p.PrintText(output.RenderDim(fmt.Sprintf("  instruction files: %s",
		report.InstructionFiles), noColor)); err != nil {
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

	if summary := memoryWarningSummary(report.Files, home, cwd); len(summary) > 0 {
		if err := p.PrintText(""); err != nil {
			return err
		}
		if err := p.PrintText(output.RenderDivider("Warnings", noColor)); err != nil {
			return err
		}
		for _, w := range summary {
			if err := p.PrintText("  ! " + w); err != nil {
				return err
			}
		}
	}
	return nil
}

// memoryWarningSummary repeats every warning in the tree as a flat list. It
// re-derives them from the files rather than reading report.Warnings, which
// labels each one with an absolute path: the rows above name their files
// ./-relative or with a ~, and a summary the reader has to map back by hand
// is worse than no summary.
func memoryWarningSummary(files []agent.Memory, home, cwd string) []string {
	var out []string
	for _, f := range files {
		label := memoryDisplayPath(f.Path, home, cwd)
		for _, w := range f.Warnings {
			out = append(out, label+": "+w)
		}
		out = append(out, memoryWarningSummary(f.Imports, home, cwd)...)
	}
	return out
}

// nameColWidth is the fixed rune width of the name column. Both top-level
// and nested rows pad or elide into this width so the size columns that
// follow line up regardless of how deep or long a path is.
const nameColWidth = 44

// nameColStart is the rune offset where the name column begins: the two
// leading spaces, the bullet, the nine-wide scope column, and the spaces
// between them in renderMemoryNode's format string. Warning rows indent from
// it so they line up under the name of the file they describe.
const nameColStart = 14

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

	// The tree is the reason this command is not a table, so a file's
	// findings belong under the file rather than in a flat list keyed by an
	// absolute path the reader has to match back to a row. The indent
	// follows the row's own name, not the top-level name column, so a
	// warning on a deeply nested import does not read as the root file's.
	warningIndent := strings.Repeat(" ", nameColStart+utf8.RuneCountInString(marker))
	for _, w := range m.Warnings {
		if err := p.PrintText(output.RenderDim(warningIndent+"! "+w, noColor)); err != nil {
			return err
		}
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

// elideMiddlePath shortens a display path to at most width runes, keeping
// the leading "./" or "~/" marker and as much of the basename as fits —
// the basename is what a reader scans for. The result never exceeds
// width: it is a column shared with sibling rows at other depths and
// budgets, so fitting it takes priority over showing the basename in
// full. When even the basename alone is wider than its budget, its tail
// (nearest the extension) is kept over its head, since that's the more
// identifying part.
func elideMiddlePath(path string, width int) string {
	if width <= 0 {
		return ""
	}
	if utf8.RuneCountInString(path) <= width {
		return path
	}

	prefix := ""
	rest := path
	if strings.HasPrefix(path, "./") || strings.HasPrefix(path, "~/") {
		prefix, rest = path[:2], path[2:]
	}
	base := filepath.Base(rest)
	baseRunes := []rune(base)
	const ellipsis = "…"

	// Reserve room for the prefix, the ellipsis, and the "/" before the
	// basename; whatever's left is the basename's budget.
	reserved := utf8.RuneCountInString(prefix) + utf8.RuneCountInString(ellipsis) + 1
	baseBudget := width - reserved
	if baseBudget <= 0 {
		full := []rune(path)
		return string(full[len(full)-width:])
	}
	if len(baseRunes) > baseBudget {
		return prefix + ellipsis + "/" + string(baseRunes[len(baseRunes)-baseBudget:])
	}

	headBudget := baseBudget - len(baseRunes)
	dirRunes := []rune(strings.TrimSuffix(rest, base))
	if headBudget > len(dirRunes) {
		headBudget = len(dirRunes)
	}
	head := strings.TrimRight(string(dirRunes[:headBudget]), "/")
	return prefix + head + ellipsis + "/" + string(baseRunes)
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
