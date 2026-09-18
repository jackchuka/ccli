package claudecode

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jackchuka/ccli/internal/agent"
)

// Thresholds Claude Code documents for memory files. Lines and bytes are the
// units the docs use, so they are the units this audit reports.
const (
	memoryAdvisoryLines = 200       // per CLAUDE.md: longer files reduce adherence
	memoryHardByteLimit = 4 << 20   // 4 MiB: Claude Code skips a larger CLAUDE.md
	autoIndexMaxLines   = 200       // MEMORY.md: only the first 200 lines load
	autoIndexMaxBytes   = 25 * 1024 // MEMORY.md: or the first 25KB, whichever comes first
	maxImportDepth      = 4         // @path imports expand at most four hops
)

// ListMemory audits every memory file Claude Code would load for the current
// working directory: the launch tier in load order with imports nested
// underneath, then the on-demand tier, with totals and warnings.
func (a *Agent) ListMemory() (*agent.MemoryReport, error) {
	s := LoadMergedSettings(a.paths)

	autoDir := a.resolveAutoMemoryDir(s)
	autoLaunch, autoOnDemand := a.autoMemoryFiles(autoDir)

	launch := a.ExpandInto(append(a.launchTierFiles(s), autoLaunch...))

	onDemand := append(a.onDemandSubdirFiles(), autoOnDemand...)

	files := append(launch, onDemand...)
	applyExclusions(files, s.ClaudeMdExcludes, a.paths.UserHomeDir)
	if !s.AutoMemoryEnabled {
		// MEMORY.md is gathered above regardless of the setting, so its
		// non-load has to be recorded here rather than skipped upstream.
		markAutoMemoryDisabled(files)
	}
	for i := range files {
		annotateWarnings(&files[i])
	}

	report := &agent.MemoryReport{
		Files:             files,
		AutoMemoryEnabled: s.AutoMemoryEnabled,
		AutoMemoryDir:     autoDir,
	}
	summarize(report)
	return report, nil
}

// markAutoMemoryDisabled warns on the auto memory index when the setting
// turns auto memory off. summarize excludes MemoryKindAutoIndex and
// MemoryKindAutoTopic entries from the totals in that case, since Claude
// Code never loads either when the setting is disabled.
func markAutoMemoryDisabled(files []agent.Memory) {
	for i := range files {
		if files[i].Kind == agent.MemoryKindAutoIndex {
			files[i].Warnings = append(files[i].Warnings,
				"auto memory is disabled in settings, so it does not load")
		}
	}
}

// GetMemory resolves a scope name ("managed", "personal", "user", "project",
// "local", "auto") or a path suffix against the audit.
func (a *Agent) GetMemory(name string) (*agent.Memory, error) {
	if name == "" || name == "." {
		// Without this guard, the suffix match below treats an empty or "."
		// name as matching the managed inline entry, whose Path is "".
		return nil, fmt.Errorf("memory %q not found", name)
	}
	report, err := a.ListMemory()
	if err != nil {
		return nil, err
	}
	flat := flattenMemory(report.Files)

	scope := name
	if scope == "user" {
		scope = string(agent.ScopePersonal)
	}
	// Prefer a file that exists, so "project" does not resolve to an absent
	// location when a real one is present.
	for _, wantExists := range []bool{true, false} {
		for i := range flat {
			if string(flat[i].Scope) == scope && flat[i].Exists == wantExists {
				return &flat[i], nil
			}
		}
	}

	for i := range flat {
		p := flat[i].Path
		if p == name || filepath.Base(p) == name || strings.HasSuffix(p, string(filepath.Separator)+name) {
			return &flat[i], nil
		}
	}
	return nil, fmt.Errorf("memory %q not found", name)
}

// annotateWarnings adds the size warnings that apply to a file's kind,
// recursing into its imports.
func annotateWarnings(m *agent.Memory) {
	if m.Exists && !m.Excluded {
		switch m.Kind {
		case agent.MemoryKindAutoIndex:
			if m.Lines > autoIndexMaxLines {
				m.Warnings = append(m.Warnings, fmt.Sprintf(
					"%d lines: only the first %d load, the rest is dropped", m.Lines, autoIndexMaxLines))
			}
			if m.Bytes > autoIndexMaxBytes {
				m.Warnings = append(m.Warnings, fmt.Sprintf(
					"%d bytes: only the first %d load, the rest is dropped", m.Bytes, autoIndexMaxBytes))
			}
		default:
			if m.Bytes > memoryHardByteLimit {
				m.Warnings = append(m.Warnings, fmt.Sprintf(
					"%d bytes: over the 4 MiB limit, Claude Code skips this file entirely", m.Bytes))
			} else if m.Lines > memoryAdvisoryLines {
				m.Warnings = append(m.Warnings, fmt.Sprintf(
					"%d lines: over the %d-line guideline, which reduces adherence", m.Lines, memoryAdvisoryLines))
			}
		}
	}
	for i := range m.Imports {
		annotateWarnings(&m.Imports[i])
	}
}

// summarize fills the report's counts and hoists every file warning, labeled
// with the file it came from.
func summarize(report *agent.MemoryReport) {
	for _, m := range flattenMemory(report.Files) {
		autoDisabled := !report.AutoMemoryEnabled &&
			(m.Kind == agent.MemoryKindAutoIndex || m.Kind == agent.MemoryKindAutoTopic)
		switch {
		case m.Excluded, !m.Exists, autoDisabled:
			// Counted nowhere: it does not load.
		case m.Tier == agent.MemoryTierLaunch:
			report.LaunchFiles++
			report.LaunchLines += m.Lines
			report.LaunchBytes += m.Bytes
		case m.Tier == agent.MemoryTierOnDemand:
			report.OnDemandFiles++
		}

		label := m.Path
		if label == "" {
			label = "managed claudeMd (inline)"
		}
		for _, w := range m.Warnings {
			report.Warnings = append(report.Warnings, label+": "+w)
		}
	}
}

// flattenMemory returns every file in the tree, parents before their imports.
func flattenMemory(files []agent.Memory) []agent.Memory {
	var out []agent.Memory
	for _, f := range files {
		imports := f.Imports
		out = append(out, f)
		out = append(out, flattenMemory(imports)...)
	}
	return out
}

// FlattenMemory exposes flattenMemory for tests.
func FlattenMemory(files []agent.Memory) []agent.Memory { return flattenMemory(files) }

// AnnotateWarnings exposes annotateWarnings for tests.
func AnnotateWarnings(m *agent.Memory) { annotateWarnings(m) }
