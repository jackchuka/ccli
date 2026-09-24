package agent

// Scope indicates where a resource was discovered.
type Scope string

const (
	ScopeGlobal   Scope = "global"
	ScopeProject  Scope = "project"
	ScopePersonal Scope = "personal"
	ScopePlugin   Scope = "plugin"
	ScopeManaged  Scope = "managed"
	ScopeLocal    Scope = "local"
	ScopeAuto     Scope = "auto"
)

// MCPServer represents an MCP server configuration.
type MCPServer struct {
	Name    string            `json:"name" yaml:"name"`
	Scope   Scope             `json:"scope" yaml:"scope"`
	Type    string            `json:"type" yaml:"type"`
	Command string            `json:"command,omitempty" yaml:"command,omitempty"`
	Args    []string          `json:"args,omitempty" yaml:"args,omitempty"`
	URL     string            `json:"url,omitempty" yaml:"url,omitempty"`
	Env     map[string]string `json:"env,omitempty" yaml:"env,omitempty"`
}

// Skill represents a Claude Code skill.
type Skill struct {
	Name        string `json:"name" yaml:"name"`
	Scope       Scope  `json:"scope" yaml:"scope"`
	Source      string `json:"source" yaml:"source"`
	Path        string `json:"path" yaml:"path"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	LinkTarget  string `json:"linkTarget,omitempty" yaml:"linkTarget,omitempty"`
}

// Project represents a project known to the agent.
type Project struct {
	Path         string             `json:"path" yaml:"path"`
	Name         string             `json:"name" yaml:"name"`
	LastCost     float64            `json:"lastCost,omitempty" yaml:"lastCost,omitempty"`
	LinesAdded   int                `json:"linesAdded,omitempty" yaml:"linesAdded,omitempty"`
	LinesRemoved int                `json:"linesRemoved,omitempty" yaml:"linesRemoved,omitempty"`
	InputTokens  int                `json:"inputTokens,omitempty" yaml:"inputTokens,omitempty"`
	OutputTokens int                `json:"outputTokens,omitempty" yaml:"outputTokens,omitempty"`
	SessionCount int                `json:"sessionCount,omitempty" yaml:"sessionCount,omitempty"`
	Trusted      bool               `json:"trusted" yaml:"trusted"`
	ModelUsage   map[string]float64 `json:"modelUsage,omitempty" yaml:"modelUsage,omitempty"`
}

// Rule represents a rule file from a rules directory.
type Rule struct {
	Name   string   `json:"name" yaml:"name"`
	Scope  Scope    `json:"scope" yaml:"scope"`
	Source string   `json:"source" yaml:"source"`
	Paths  []string `json:"paths,omitempty" yaml:"paths,omitempty"`
}

// MemoryKind distinguishes the sorts of memory files Claude Code loads.
type MemoryKind string

const (
	MemoryKindClaudeMD      MemoryKind = "claude-md"
	MemoryKindClaudeLocalMD MemoryKind = "claude-local-md"
	MemoryKindAgentsMD      MemoryKind = "agents-md"
	MemoryKindManagedInline MemoryKind = "managed-inline"
	MemoryKindImport        MemoryKind = "import"
	MemoryKindAutoIndex     MemoryKind = "auto-index"
	MemoryKindAutoTopic     MemoryKind = "auto-topic"
)

// MemoryTier records when Claude Code loads a memory file: at session start,
// or on demand when it reads a file the memory applies to.
type MemoryTier string

const (
	MemoryTierLaunch   MemoryTier = "launch"
	MemoryTierOnDemand MemoryTier = "on-demand"
)

// Memory is a single memory file, with the files it imports nested underneath.
type Memory struct {
	Path       string     `json:"path" yaml:"path"`
	Scope      Scope      `json:"scope" yaml:"scope"`
	Kind       MemoryKind `json:"kind" yaml:"kind"`
	Tier       MemoryTier `json:"tier" yaml:"tier"`
	Exists     bool       `json:"exists" yaml:"exists"`
	Excluded   bool       `json:"excluded,omitempty" yaml:"excluded,omitempty"`
	ExcludedBy string     `json:"excludedBy,omitempty" yaml:"excludedBy,omitempty"`
	// NotLoaded records a refusal by Claude Code that no other field
	// captures: an import past the hop limit, an import that is one of its
	// own ancestors, or auto memory switched off in settings. Exists and
	// Excluded carry the remaining non-load reasons; Loads combines them all.
	NotLoaded  bool     `json:"notLoaded,omitempty" yaml:"notLoaded,omitempty"`
	LinkTarget string   `json:"linkTarget,omitempty" yaml:"linkTarget,omitempty"`
	External   bool     `json:"external,omitempty" yaml:"external,omitempty"`
	Lines      int      `json:"lines,omitempty" yaml:"lines,omitempty"`
	Bytes      int64    `json:"bytes,omitempty" yaml:"bytes,omitempty"`
	Type       string   `json:"type,omitempty" yaml:"type,omitempty"`
	Modified   string   `json:"modified,omitempty" yaml:"modified,omitempty"`
	Depth      int      `json:"depth,omitempty" yaml:"depth,omitempty"`
	Imports    []Memory `json:"imports,omitempty" yaml:"imports,omitempty"`
	Warnings   []string `json:"warnings,omitempty" yaml:"warnings,omitempty"`
}

// Loads reports whether Claude Code actually reads this file's content into
// the session. It is the single answer the launch totals, the size warnings,
// and the tree renderer all ask, so a newly discovered way for a file not to
// load is honored everywhere by setting NotLoaded once, rather than by
// repeating a boolean expression in three places.
//
// An External import still counts as loading: Claude Code gates those behind
// a one-time approval that ccli cannot observe, and an over-estimate the
// warnings make visible beats an under-estimate that hides content.
func (m Memory) Loads() bool {
	return m.Exists && !m.Excluded && !m.NotLoaded
}

// MemoryReport is the full memory audit: the launch tier in load order
// followed by the on-demand tier, with precomputed totals.
type MemoryReport struct {
	Files             []Memory `json:"files" yaml:"files"`
	LaunchFiles       int      `json:"launchFiles" yaml:"launchFiles"`
	LaunchLines       int      `json:"launchLines" yaml:"launchLines"`
	LaunchBytes       int64    `json:"launchBytes" yaml:"launchBytes"`
	OnDemandFiles     int      `json:"onDemandFiles" yaml:"onDemandFiles"`
	AutoMemoryEnabled bool     `json:"autoMemoryEnabled" yaml:"autoMemoryEnabled"`
	AutoMemoryDir     string   `json:"autoMemoryDir" yaml:"autoMemoryDir"`
	Warnings          []string `json:"warnings,omitempty" yaml:"warnings,omitempty"`
}

// InstallInfo holds comprehensive installation metadata.
type InstallInfo struct {
	Version      string `json:"version" yaml:"version"`
	AuthStatus   string `json:"authStatus" yaml:"authStatus"`
	Model        string `json:"model" yaml:"model"`
	ConfigPath   string `json:"configPath" yaml:"configPath"`
	SettingsPath string `json:"settingsPath" yaml:"settingsPath"`
	HistoryPath  string `json:"historyPath" yaml:"historyPath"`
	HistoryCount int    `json:"historyCount" yaml:"historyCount"`
	SessionCount int    `json:"sessionCount" yaml:"sessionCount"`
	ProjectCount int    `json:"projectCount" yaml:"projectCount"`
	StorageBytes int64  `json:"storageBytes" yaml:"storageBytes"`
	MCPCount     int    `json:"mcpCount" yaml:"mcpCount"`
	SkillCount   int    `json:"skillCount" yaml:"skillCount"`
	PluginCount  int    `json:"pluginCount" yaml:"pluginCount"`
	MemoryCount  int    `json:"memoryCount" yaml:"memoryCount"`
}
