package claudecode_test

import (
	"os"
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

func TestManagedPaths(t *testing.T) {
	tests := []struct {
		goos         string
		wantPolicy   string
		wantSettings string
	}{
		{"darwin", "/Library/Application Support/ClaudeCode/CLAUDE.md", "/Library/Application Support/ClaudeCode/managed-settings.json"},
		{"linux", "/etc/claude-code/CLAUDE.md", "/etc/claude-code/managed-settings.json"},
		{"windows", `C:\Program Files\ClaudeCode\CLAUDE.md`, `C:\Program Files\ClaudeCode\managed-settings.json`},
	}
	for _, tt := range tests {
		t.Run(tt.goos, func(t *testing.T) {
			policy, settings := claudecode.ManagedPaths(tt.goos)
			if policy != tt.wantPolicy {
				t.Errorf("policy = %q, want %q", policy, tt.wantPolicy)
			}
			if settings != tt.wantSettings {
				t.Errorf("settings = %q, want %q", settings, tt.wantSettings)
			}
		})
	}
}

func TestLoadMergedSettingsPrecedence(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) string {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		return path
	}

	user := write("user.json", `{"claudeMdExcludes":["**/user.md"],"autoMemoryDirectory":"~/user-mem"}`)
	project := write("project.json", `{"claudeMdExcludes":["**/project.md"],"autoMemoryDirectory":"~/project-mem"}`)
	local := write("local.json", `{"claudeMdExcludes":["**/local.md"],"autoMemoryEnabled":false}`)
	managed := write("managed.json", `{"claudeMd":"Always run make lint."}`)

	got := claudecode.LoadMergedSettings(claudecode.Paths{
		SettingsFile:        user,
		ProjectSettingsFile: project,
		LocalSettingsFile:   local,
		ManagedSettingsFile: managed,
	})

	// Arrays merge across layers.
	if len(got.ClaudeMdExcludes) != 3 {
		t.Errorf("excludes = %v, want 3 entries", got.ClaudeMdExcludes)
	}
	// Scalars: local beats project beats user.
	if got.AutoMemoryDirectory != "~/project-mem" {
		t.Errorf("autoMemoryDirectory = %q, want %q", got.AutoMemoryDirectory, "~/project-mem")
	}
	// Explicit false is honored; absent means the documented default of true.
	if got.AutoMemoryEnabled {
		t.Error("autoMemoryEnabled = true, want false")
	}
	if got.ClaudeMd != "Always run make lint." {
		t.Errorf("claudeMd = %q", got.ClaudeMd)
	}
}

func TestLoadMergedSettingsClaudeMdManagedOnly(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) string {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		return path
	}

	t.Run("managed wins over user", func(t *testing.T) {
		user := write("user.json", `{"claudeMd":"User instruction, never loaded."}`)
		managed := write("managed.json", `{"claudeMd":"Managed instruction."}`)

		got := claudecode.LoadMergedSettings(claudecode.Paths{
			SettingsFile:        user,
			ManagedSettingsFile: managed,
		})
		if got.ClaudeMd != "Managed instruction." {
			t.Errorf("claudeMd = %q, want the managed value", got.ClaudeMd)
		}
	})

	t.Run("user-only claudeMd has no effect", func(t *testing.T) {
		user := write("user-only.json", `{"claudeMd":"User instruction, never loaded."}`)

		got := claudecode.LoadMergedSettings(claudecode.Paths{
			SettingsFile: user,
		})
		if got.ClaudeMd != "" {
			t.Errorf("claudeMd = %q, want empty since only user settings set it", got.ClaudeMd)
		}
	})
}

func TestLoadMergedSettingsDefaults(t *testing.T) {
	got := claudecode.LoadMergedSettings(claudecode.Paths{})
	if !got.AutoMemoryEnabled {
		t.Error("autoMemoryEnabled = false, want true when no settings file sets it")
	}
	if len(got.ClaudeMdExcludes) != 0 {
		t.Errorf("excludes = %v, want empty", got.ClaudeMdExcludes)
	}
}
