package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jackchuka/ccli/internal/agent"
	"github.com/jackchuka/ccli/internal/output"
)

func TestRenderSkillGetMatchesOtherGetCommands(t *testing.T) {
	var buf bytes.Buffer
	p := output.NewPrinter(&buf, output.FormatText, true)
	s := &agent.Skill{Name: "commit", Scope: agent.ScopePersonal, Path: "/s/commit/SKILL.md", Description: "Write a commit"}
	if err := renderSkillGet(p, s); err != nil {
		t.Fatal(err)
	}
	want := "  Name         commit\n" +
		"  Scope        personal\n" +
		"  Path         /s/commit/SKILL.md\n" +
		"  Description  Write a commit\n"
	if got := buf.String(); got != want {
		t.Errorf("renderSkillGet =\n%s\nwant\n%s", got, want)
	}
}

func TestRootHelpNamesEveryCommand(t *testing.T) {
	for _, c := range rootCmd.Commands() {
		name := c.Name()
		if c.Hidden || name == "help" || name == "completion" || name == "version" {
			continue
		}
		if !strings.Contains(rootCmd.Long, name) {
			t.Errorf("root help %q does not mention %q", rootCmd.Long, name)
		}
	}
}
