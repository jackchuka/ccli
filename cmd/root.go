package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	format  string
	noColor bool
)

var rootCmd = &cobra.Command{
	Use:   "ccli",
	Short: "Inspect your Claude Code installation",
	Long: `Inspect your Claude Code installation: an installation summary (info),
MCP servers (mcp), skills, rules, the memory files that load into a session
(memory), and per-project usage and session cleanup (projects).`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&format, "format", "f", "text", "Output format: text, json, yaml")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colored output")
}
