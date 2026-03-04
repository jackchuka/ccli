package claudecode_test

import (
	"path/filepath"
	"testing"

	"github.com/jackchuka/ccli/internal/claudecode"
)

func TestLoadConfig(t *testing.T) {
	path := filepath.Join("testdata", "claude.json")
	cfg, err := claudecode.LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.NumStartups != 705 {
		t.Errorf("numStartups = %d, want 705", cfg.NumStartups)
	}
	if len(cfg.Projects) != 1 {
		t.Errorf("projects count = %d, want 1", len(cfg.Projects))
	}
}

func TestLoadSettings(t *testing.T) {
	path := filepath.Join("testdata", "settings.json")
	settings, err := claudecode.LoadSettings(path)
	if err != nil {
		t.Fatalf("LoadSettings: %v", err)
	}
	if settings.Model != "opus" {
		t.Errorf("model = %q, want %q", settings.Model, "opus")
	}
	if len(settings.EnabledPlugins) != 2 {
		t.Errorf("enabledPlugins count = %d, want 2", len(settings.EnabledPlugins))
	}
}
