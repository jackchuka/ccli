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

// RemoveProject removes a project entry from the config file.
// It preserves all other fields by doing a partial read-modify-write.
func RemoveProject(configPath, projectPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	projectsRaw, ok := raw["projects"]
	if !ok {
		return nil
	}
	var projects map[string]json.RawMessage
	if err := json.Unmarshal(projectsRaw, &projects); err != nil {
		return err
	}
	delete(projects, projectPath)
	updated, err := json.Marshal(projects)
	if err != nil {
		return err
	}
	raw["projects"] = updated
	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, out, 0o644)
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
