package claudecode

import (
	"encoding/json"
	"os"
)

// Config represents the top-level ~/.claude.json structure.
type Config struct {
	NumStartups int                       `json:"numStartups"`
	Projects    map[string]ProjectConfig  `json:"projects"`
	MCPServers  map[string]MCPServerEntry `json:"mcpServers"`
}

// ProjectConfig represents a project entry in ~/.claude.json.
type ProjectConfig struct {
	MCPServers             map[string]MCPServerEntry `json:"mcpServers"`
	HasTrustDialogAccepted bool                      `json:"hasTrustDialogAccepted"`
	LastCost               float64                   `json:"lastCost"`
	LastLinesAdded         int                       `json:"lastLinesAdded"`
	LastLinesRemoved       int                       `json:"lastLinesRemoved"`
	LastTotalInputTokens   int                       `json:"lastTotalInputTokens"`
	LastTotalOutputTokens  int                       `json:"lastTotalOutputTokens"`
	LastModelUsage         map[string]ModelUsage     `json:"lastModelUsage"`
}

// ModelUsage represents per-model token and cost data.
type ModelUsage struct {
	InputTokens  int     `json:"inputTokens"`
	OutputTokens int     `json:"outputTokens"`
	CostUSD      float64 `json:"costUSD"`
}

// MCPServerEntry represents an MCP server in config JSON.
type MCPServerEntry struct {
	Type    string            `json:"type,omitempty"`
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	URL     string            `json:"url,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

// Settings represents ~/.claude/settings.json.
type Settings struct {
	Model          string          `json:"model"`
	EnabledPlugins map[string]bool `json:"enabledPlugins"`
	Permissions    struct {
		Allow []string `json:"allow"`
	} `json:"permissions"`
}

// LoadConfig reads and parses a claude.json file by full path.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// LoadSettings reads and parses a settings.json file.
func LoadSettings(path string) (*Settings, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}
