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
	Long:  "A unified CLI for inspecting Claude Code — MCP servers, skills, and metadata.",
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
