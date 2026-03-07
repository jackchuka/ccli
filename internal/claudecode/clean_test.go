package claudecode_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackchuka/ccli/internal/claudecode"
)

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// writeFile creates a file (and parent dirs) with the given content.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	must(t, os.MkdirAll(filepath.Dir(path), 0o755))
	must(t, os.WriteFile(path, []byte(content), 0o644))
}

// setModTime sets the modification time of a file.
func setModTime(t *testing.T, path string, mod time.Time) {
	t.Helper()
	must(t, os.Chtimes(path, mod, mod))
}

func TestCleanProjects_DryRun(t *testing.T) {
	tmp := t.TempDir()
	projDir := filepath.Join(tmp, "projects", "-Users-test-projA")
	must(t, os.MkdirAll(projDir, 0o755))

	oldFile := filepath.Join(projDir, "aaaa-bbbb-cccc-dddd.jsonl")
	newFile := filepath.Join(projDir, "eeee-ffff-1111-2222.jsonl")
	writeFile(t, oldFile, `{"msg":"old"}`)
	writeFile(t, newFile, `{"msg":"new"}`)

	setModTime(t, oldFile, time.Now().Add(-60*24*time.Hour))

	a := claudecode.NewAgent(claudecode.Paths{HomeDir: tmp})
	result, err := a.CleanProjects(claudecode.CleanOptions{
		OlderThan: 30 * 24 * time.Hour,
		DryRun:    true,
	})
	if err != nil {
		t.Fatalf("CleanProjects: %v", err)
	}

	if result.Sessions.Count != 1 {
		t.Errorf("sessions count = %d, want 1", result.Sessions.Count)
	}

	// File should still exist because dry-run
	if _, err := os.Stat(oldFile); os.IsNotExist(err) {
		t.Error("old file was deleted during dry run")
	}
}

func TestCleanProjects_DeletesOldSessions(t *testing.T) {
	tmp := t.TempDir()
	projDir := filepath.Join(tmp, "projects", "-Users-test-projA")
	oldUUID := "aaaa-bbbb-cccc-dddd"
	newUUID := "eeee-ffff-1111-2222"

	// Old session: .jsonl file and matching directory with subagents
	oldJSONL := filepath.Join(projDir, oldUUID+".jsonl")
	writeFile(t, oldJSONL, `{"msg":"old"}`)
	setModTime(t, oldJSONL, time.Now().Add(-60*24*time.Hour))

	writeFile(t, filepath.Join(projDir, oldUUID, "subagents", "sub.jsonl"), "data")

	// New session: should survive
	newJSONL := filepath.Join(projDir, newUUID+".jsonl")
	writeFile(t, newJSONL, `{"msg":"new"}`)

	// Artifact directories
	writeFile(t, filepath.Join(tmp, "debug", oldUUID+".txt"), "debug log")
	writeFile(t, filepath.Join(tmp, "telemetry", "1p_failed_events."+oldUUID+".batch1.json"), "telemetry")
	writeFile(t, filepath.Join(tmp, "todos", oldUUID+"-agent-"+oldUUID+".json"), "todo")
	writeFile(t, filepath.Join(tmp, "tasks", oldUUID, "task.json"), "task")
	writeFile(t, filepath.Join(tmp, "file-history", oldUUID, "history.json"), "history")
	writeFile(t, filepath.Join(tmp, "session-env", oldUUID, "env.json"), "env")

	a := claudecode.NewAgent(claudecode.Paths{HomeDir: tmp})
	result, err := a.CleanProjects(claudecode.CleanOptions{
		OlderThan: 30 * 24 * time.Hour,
		DryRun:    false,
	})
	if err != nil {
		t.Fatalf("CleanProjects: %v", err)
	}

	// Old session file deleted
	if _, err := os.Stat(oldJSONL); !os.IsNotExist(err) {
		t.Error("old .jsonl file should be deleted")
	}
	// Old session directory deleted
	if _, err := os.Stat(filepath.Join(projDir, oldUUID)); !os.IsNotExist(err) {
		t.Error("old session directory should be deleted")
	}
	// New session survives
	if _, err := os.Stat(newJSONL); os.IsNotExist(err) {
		t.Error("new .jsonl file should survive")
	}

	// Artifact directories cleaned
	if _, err := os.Stat(filepath.Join(tmp, "debug", oldUUID+".txt")); !os.IsNotExist(err) {
		t.Error("debug artifact should be deleted")
	}
	if _, err := os.Stat(filepath.Join(tmp, "telemetry", "1p_failed_events."+oldUUID+".batch1.json")); !os.IsNotExist(err) {
		t.Error("telemetry artifact should be deleted")
	}
	if _, err := os.Stat(filepath.Join(tmp, "todos", oldUUID+"-agent-"+oldUUID+".json")); !os.IsNotExist(err) {
		t.Error("todos artifact should be deleted")
	}
	if _, err := os.Stat(filepath.Join(tmp, "tasks", oldUUID)); !os.IsNotExist(err) {
		t.Error("tasks artifact should be deleted")
	}
	if _, err := os.Stat(filepath.Join(tmp, "file-history", oldUUID)); !os.IsNotExist(err) {
		t.Error("file-history artifact should be deleted")
	}
	if _, err := os.Stat(filepath.Join(tmp, "session-env", oldUUID)); !os.IsNotExist(err) {
		t.Error("session-env artifact should be deleted")
	}

	// Verify category counts
	if result.Sessions.Count != 1 {
		t.Errorf("sessions count = %d, want 1", result.Sessions.Count)
	}
	if result.Debug.Count != 1 {
		t.Errorf("debug count = %d, want 1", result.Debug.Count)
	}
	if result.Telemetry.Count != 1 {
		t.Errorf("telemetry count = %d, want 1", result.Telemetry.Count)
	}
	if result.Todos.Count != 1 {
		t.Errorf("todos count = %d, want 1", result.Todos.Count)
	}
	if result.Tasks.Count != 1 {
		t.Errorf("tasks count = %d, want 1", result.Tasks.Count)
	}
	if result.FileHistory.Count != 1 {
		t.Errorf("fileHistory count = %d, want 1", result.FileHistory.Count)
	}
	if result.SessionEnv.Count != 1 {
		t.Errorf("sessionEnv count = %d, want 1", result.SessionEnv.Count)
	}
	if result.TotalBytes <= 0 {
		t.Errorf("totalBytes = %d, want > 0", result.TotalBytes)
	}
}

func TestCleanProjects_SpecificProject(t *testing.T) {
	tmp := t.TempDir()

	// Two projects with old sessions
	projAEncoded := encodeTestPath("/Users/test/projA")
	projBEncoded := encodeTestPath("/Users/test/projB")
	projADir := filepath.Join(tmp, "projects", projAEncoded)
	projBDir := filepath.Join(tmp, "projects", projBEncoded)

	oldUUID := "aaaa-bbbb-cccc-dddd"

	oldA := filepath.Join(projADir, oldUUID+".jsonl")
	writeFile(t, oldA, `{"msg":"old-a"}`)
	setModTime(t, oldA, time.Now().Add(-60*24*time.Hour))

	oldB := filepath.Join(projBDir, oldUUID+".jsonl")
	writeFile(t, oldB, `{"msg":"old-b"}`)
	setModTime(t, oldB, time.Now().Add(-60*24*time.Hour))

	// Config file so resolveProjectDir can find projects
	cfg := map[string]interface{}{
		"projects": map[string]interface{}{
			"/Users/test/projA": map[string]interface{}{
				"hasTrustDialogAccepted": true,
			},
			"/Users/test/projB": map[string]interface{}{
				"hasTrustDialogAccepted": true,
			},
		},
	}
	cfgPath := filepath.Join(tmp, "claude.json")
	cfgData, err := json.Marshal(cfg)
	must(t, err)
	must(t, os.WriteFile(cfgPath, cfgData, 0o644))

	a := claudecode.NewAgent(claudecode.Paths{
		ConfigFile: cfgPath,
		HomeDir:    tmp,
	})
	_, err = a.CleanProjects(claudecode.CleanOptions{
		Project:   "projA",
		OlderThan: 30 * 24 * time.Hour,
		DryRun:    false,
	})
	if err != nil {
		t.Fatalf("CleanProjects: %v", err)
	}

	// projA session deleted
	if _, err := os.Stat(oldA); !os.IsNotExist(err) {
		t.Error("projA old session should be deleted")
	}
	// projB session untouched
	if _, err := os.Stat(oldB); os.IsNotExist(err) {
		t.Error("projB old session should be untouched")
	}
}

// encodeTestPath mirrors encodeProjectPath for test setup.
func encodeTestPath(path string) string {
	var b []byte
	for _, c := range path {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			b = append(b, byte(c))
		} else {
			b = append(b, '-')
		}
	}
	return string(b)
}
