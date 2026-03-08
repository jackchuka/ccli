package claudecode

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

type CleanOptions struct {
	Project   string
	OlderThan time.Duration
	DryRun    bool
}

type CleanCategoryResult struct {
	Count int   `json:"count" yaml:"count"`
	Bytes int64 `json:"bytes" yaml:"bytes"`
}

type CleanResult struct {
	Sessions      CleanCategoryResult `json:"sessions" yaml:"sessions"`
	Debug         CleanCategoryResult `json:"debug" yaml:"debug"`
	Telemetry     CleanCategoryResult `json:"telemetry" yaml:"telemetry"`
	Todos         CleanCategoryResult `json:"todos" yaml:"todos"`
	Tasks         CleanCategoryResult `json:"tasks" yaml:"tasks"`
	FileHistory   CleanCategoryResult `json:"fileHistory" yaml:"fileHistory"`
	SessionEnv    CleanCategoryResult `json:"sessionEnv" yaml:"sessionEnv"`
	TotalBytes    int64               `json:"totalBytes" yaml:"totalBytes"`
	ConfigRemoved bool                `json:"configRemoved" yaml:"configRemoved"`
}

func (a *Agent) CleanProjects(opts CleanOptions) (*CleanResult, error) {
	projectDirs, err := a.resolveCleanTargets(opts.Project)
	if err != nil {
		return nil, err
	}

	var cutoff time.Time
	if opts.OlderThan > 0 {
		cutoff = time.Now().Add(-opts.OlderThan)
	}
	result := &CleanResult{}

	var expiredUUIDs []string
	for _, projDir := range projectDirs {
		uuids, sessResult := a.findExpiredSessions(projDir, cutoff, opts.DryRun)
		result.Sessions.Count += sessResult.Count
		result.Sessions.Bytes += sessResult.Bytes
		expiredUUIDs = append(expiredUUIDs, uuids...)
	}

	for _, uuid := range expiredUUIDs {
		a.cleanDebug(uuid, opts.DryRun, result)
		a.cleanTelemetry(uuid, opts.DryRun, result)
		a.cleanTodos(uuid, opts.DryRun, result)
		a.cleanUUIDDirs(uuid, opts.DryRun, result)
	}

	// Remove config entry when cleaning all sessions for a specific project
	if opts.Project != "" && opts.OlderThan == 0 {
		configPath, _ := a.resolveProjectConfigPath(opts.Project)
		if configPath != "" {
			if !opts.DryRun {
				_ = RemoveProject(a.paths.ConfigFile, configPath)
			}
			result.ConfigRemoved = true
		}
	}

	result.TotalBytes = result.Sessions.Bytes +
		result.Debug.Bytes +
		result.Telemetry.Bytes +
		result.Todos.Bytes +
		result.Tasks.Bytes +
		result.FileHistory.Bytes +
		result.SessionEnv.Bytes

	return result, nil
}

// resolveCleanTargets returns the project directories to scan.
func (a *Agent) resolveCleanTargets(project string) ([]string, error) {
	projectsRoot := filepath.Join(a.paths.HomeDir, "projects")

	if project == "" {
		entries, err := os.ReadDir(projectsRoot)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, nil
			}
			return nil, err
		}
		var dirs []string
		for _, e := range entries {
			if e.IsDir() {
				dirs = append(dirs, filepath.Join(projectsRoot, e.Name()))
			}
		}
		return dirs, nil
	}

	// Resolve project name to encoded directory using config (same logic as GetProject)
	dir, err := a.resolveProjectDir(project)
	if err != nil {
		return nil, err
	}
	return []string{dir}, nil
}

// resolveProjectDir finds the encoded project directory matching a query.
func (a *Agent) resolveProjectDir(query string) (string, error) {
	configPath, err := a.resolveProjectConfigPath(query)
	if err != nil {
		return "", err
	}
	return filepath.Join(a.paths.HomeDir, "projects", encodeProjectPath(configPath)), nil
}

// resolveProjectConfigPath returns the full config key for a project query.
func (a *Agent) resolveProjectConfigPath(query string) (string, error) {
	cfg, err := LoadConfig(a.paths.ConfigFile)
	if err != nil {
		return "", err
	}
	for path := range cfg.Projects {
		name := filepath.Base(path)
		if path == query || name == query || strings.HasSuffix(path, "/"+query) {
			return path, nil
		}
	}
	return "", &projectNotFoundError{query}
}

type projectNotFoundError struct{ query string }

func (e *projectNotFoundError) Error() string {
	return "project " + e.query + " not found"
}

// findExpiredSessions scans a project dir for .jsonl files older than cutoff.
// Returns the UUIDs found and session-level cleanup stats. On non-dry-run,
// it deletes the .jsonl file and matching UUID subdirectory.
func (a *Agent) findExpiredSessions(projDir string, cutoff time.Time, dryRun bool) ([]string, CleanCategoryResult) {
	var result CleanCategoryResult
	var uuids []string

	entries, err := os.ReadDir(projDir)
	if err != nil {
		return nil, result
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if !cutoff.IsZero() && info.ModTime().After(cutoff) {
			continue
		}

		uuid := strings.TrimSuffix(e.Name(), ".jsonl")
		uuids = append(uuids, uuid)

		jsonlPath := filepath.Join(projDir, e.Name())
		result.Count++
		result.Bytes += info.Size()

		// Also account for the UUID subdirectory
		subDir := filepath.Join(projDir, uuid)
		result.Bytes += dirSize(subDir)

		if !dryRun {
			_ = os.Remove(jsonlPath)
			_ = os.RemoveAll(subDir)
		}
	}
	return uuids, result
}

func (a *Agent) cleanDebug(uuid string, dryRun bool, result *CleanResult) {
	debugDir := filepath.Join(a.paths.HomeDir, "debug")
	target := filepath.Join(debugDir, uuid+".txt")
	info, err := os.Stat(target)
	if err != nil {
		return
	}
	result.Debug.Count++
	result.Debug.Bytes += info.Size()
	if !dryRun {
		_ = os.Remove(target)
	}
}

func (a *Agent) cleanTelemetry(uuid string, dryRun bool, result *CleanResult) {
	telDir := filepath.Join(a.paths.HomeDir, "telemetry")
	entries, err := os.ReadDir(telDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || !strings.Contains(e.Name(), uuid) {
			continue
		}
		result.Telemetry.Count++
		if info, err := e.Info(); err == nil {
			result.Telemetry.Bytes += info.Size()
		}
		if !dryRun {
			_ = os.Remove(filepath.Join(telDir, e.Name()))
		}
	}
}

func (a *Agent) cleanTodos(uuid string, dryRun bool, result *CleanResult) {
	todosDir := filepath.Join(a.paths.HomeDir, "todos")
	entries, err := os.ReadDir(todosDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), uuid) {
			continue
		}
		result.Todos.Count++
		if info, err := e.Info(); err == nil {
			result.Todos.Bytes += info.Size()
		}
		if !dryRun {
			_ = os.Remove(filepath.Join(todosDir, e.Name()))
		}
	}
}

func (a *Agent) cleanUUIDDirs(uuid string, dryRun bool, result *CleanResult) {
	type dirCategory struct {
		subdir string
		cat    *CleanCategoryResult
	}
	categories := []dirCategory{
		{"tasks", &result.Tasks},
		{"file-history", &result.FileHistory},
		{"session-env", &result.SessionEnv},
	}
	for _, dc := range categories {
		target := filepath.Join(a.paths.HomeDir, dc.subdir, uuid)
		info, err := os.Stat(target)
		if err != nil {
			continue
		}
		if !info.IsDir() {
			continue
		}
		dc.cat.Count++
		dc.cat.Bytes += dirSize(target)
		if !dryRun {
			_ = os.RemoveAll(target)
		}
	}
}
