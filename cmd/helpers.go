package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/jackchuka/ccli/internal/agent"
	"github.com/jackchuka/ccli/internal/claudecode"
	"github.com/jackchuka/ccli/internal/output"
)

func getAgent() (agent.Agent, error) {
	paths, err := claudecode.DefaultPaths()
	if err != nil {
		return nil, err
	}
	return claudecode.NewAgent(paths), nil
}

func getPrinter() (*output.Printer, error) {
	f, err := output.ParseFormat(format)
	if err != nil {
		return nil, fmt.Errorf("invalid --format value: %w", err)
	}
	return output.NewPrinter(os.Stdout, f, noColor), nil
}

// renderScopeSummary renders a scope-breakdown footer line like "  5 items  ● 3 global  ○ 2 project".
func renderScopeSummary(p *output.Printer, label string, counts map[agent.Scope]int, total int, scopes []agent.Scope) error {
	noColor := p.NoColor()
	var summary strings.Builder
	summary.WriteString(output.RenderDim(fmt.Sprintf("  %d %s", total, label), noColor))
	for _, scope := range scopes {
		if n, ok := counts[scope]; ok {
			bullet := output.RenderScopeBullet(string(scope), noColor)
			fmt.Fprintf(&summary, "  %s %s", bullet, output.RenderDim(fmt.Sprintf("%d %s", n, scope), noColor))
		}
	}
	return p.PrintText(summary.String())
}
