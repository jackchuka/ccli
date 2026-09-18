package claudecode

import "github.com/jackchuka/ccli/internal/agent"

// Unexported memory-audit internals, exposed to the package's external test
// package. They live here so the test binary can reach them without the
// production build carrying a second, exported copy of the package's API.

var (
	ReadMemoryFile   = readMemoryFile
	GlobMatch        = globMatch
	ExpandTilde      = expandTilde
	ParseImports     = parseImports
	ApplyExclusions  = applyExclusions
	FlattenMemory    = flattenMemory
	AnnotateWarnings = annotateWarnings
)

func MaxImportDepth() int { return maxImportDepth }

// LaunchTierFiles loads the settings itself, since the tests that call it
// care about the discovered files rather than the settings plumbing.
func (a *Agent) LaunchTierFiles() []agent.Memory {
	return a.launchTierFiles(LoadMergedSettings(a.paths))
}

func (a *Agent) ResolveAutoMemoryDir() string {
	return a.resolveAutoMemoryDir(LoadMergedSettings(a.paths))
}

func (a *Agent) AutoMemoryFiles(dir string) ([]agent.Memory, []agent.Memory) {
	return a.autoMemoryFiles(dir)
}

func (a *Agent) OnDemandSubdirFiles() []agent.Memory { return a.onDemandSubdirFiles() }

func (a *Agent) LaunchMemoryReport() *agent.MemoryReport { return a.launchMemoryReport() }
