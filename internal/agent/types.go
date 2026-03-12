package agent

// Scope indicates where a resource was discovered.
type Scope string

const (
	ScopeGlobal   Scope = "global"
	ScopeProject  Scope = "project"
	ScopePersonal Scope = "personal"
	ScopePlugin   Scope = "plugin"
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
}
