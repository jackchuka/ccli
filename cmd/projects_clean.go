package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackchuka/ccli/internal/claudecode"
	"github.com/jackchuka/ccli/internal/output"
	"github.com/spf13/cobra"
)

var (
	cleanOlderThan string
	cleanDryRun    bool
)

var projectsCleanCmd = &cobra.Command{
	Use:   "clean [project]",
	Short: "Delete old session data and associated artifacts",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if cleanOlderThan == "" {
			return fmt.Errorf("--older-than is required (e.g. 7d, 30d, 90d)")
		}

		dur, err := parseDayDuration(cleanOlderThan)
		if err != nil {
			return err
		}

		a, err := getClaudeAgent()
		if err != nil {
			return err
		}

		opts := claudecode.CleanOptions{
			OlderThan: dur,
			DryRun:    cleanDryRun,
		}
		if len(args) == 1 {
			opts.Project = args[0]
		}

		result, err := a.CleanProjects(opts)
		if err != nil {
			return err
		}

		p, err := getPrinter()
		if err != nil {
			return err
		}

		if p.Format() != output.FormatText {
			return p.Print(result)
		}
		return renderCleanResult(p, result, cleanDryRun)
	},
}

func getClaudeAgent() (*claudecode.Agent, error) {
	paths, err := claudecode.DefaultPaths()
	if err != nil {
		return nil, err
	}
	return claudecode.NewAgent(paths), nil
}

func parseDayDuration(s string) (time.Duration, error) {
	if !strings.HasSuffix(s, "d") {
		return 0, fmt.Errorf("invalid duration %q: must end with 'd' (e.g. 7d, 30d)", s)
	}
	n, err := strconv.Atoi(strings.TrimSuffix(s, "d"))
	if err != nil {
		return 0, fmt.Errorf("invalid duration %q: %w", s, err)
	}
	if n <= 0 {
		return 0, fmt.Errorf("invalid duration %q: must be a positive number of days", s)
	}
	return time.Duration(n) * 24 * time.Hour, nil
}

func renderCleanResult(p *output.Printer, r *claudecode.CleanResult, dryRun bool) error {
	total := r.Sessions.Count
	if total == 0 {
		return p.PrintText("No sessions found matching criteria")
	}

	noColor := p.NoColor()

	verb := "Cleaned"
	if dryRun {
		verb = "Would clean"
	}
	p.PrintText(fmt.Sprintf("%s %d sessions (%s freed)", verb, total, output.FormatBytes(r.TotalBytes)))

	type category struct {
		name   string
		result claudecode.CleanCategoryResult
	}
	categories := []category{
		{"sessions", r.Sessions},
		{"debug", r.Debug},
		{"telemetry", r.Telemetry},
		{"todos", r.Todos},
		{"tasks", r.Tasks},
		{"file history", r.FileHistory},
		{"session env", r.SessionEnv},
	}
	for _, c := range categories {
		if c.result.Count == 0 {
			continue
		}
		line := fmt.Sprintf("  %s: %d (%s)", c.name, c.result.Count, output.FormatBytes(c.result.Bytes))
		p.PrintText(output.RenderDim(line, noColor))
	}

	return nil
}

func init() {
	projectsCleanCmd.Flags().StringVar(&cleanOlderThan, "older-than", "", "Remove sessions older than this duration (e.g. 7d, 30d, 90d)")
	projectsCleanCmd.Flags().BoolVar(&cleanDryRun, "dry-run", false, "Show what would be deleted without deleting")
	projectsCmd.AddCommand(projectsCleanCmd)
}
