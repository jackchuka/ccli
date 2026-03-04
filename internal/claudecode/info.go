package claudecode

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/jackchuka/ccli/internal/agent"
)

// Info gathers comprehensive installation metadata.
func (a *Agent) Info() (*agent.InstallInfo, error) {
	info := &agent.InstallInfo{
		ConfigPath:  a.paths.ConfigFile,
		HistoryPath: filepath.Join(a.paths.HomeDir, "history.jsonl"),
	}

	// Settings
	if a.paths.SettingsFile != "" {
		info.SettingsPath = a.paths.SettingsFile
		settings, err := LoadSettings(a.paths.SettingsFile)
		if err == nil {
			info.Model = settings.Model
			info.PluginCount = countEnabledPlugins(settings.EnabledPlugins)
		}
	}

	// Version and auth status: run concurrently since both spawn external processes
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		if out, err := exec.Command("claude", "--version").Output(); err == nil {
			info.Version = strings.TrimSpace(string(out))
		}
	}()
	go func() {
		defer wg.Done()
		if out, err := exec.Command("claude", "auth", "status").Output(); err == nil {
			var authData struct {
				LoggedIn   bool   `json:"loggedIn"`
				AuthMethod string `json:"authMethod"`
			}
			if json.Unmarshal(out, &authData) == nil {
				if authData.LoggedIn {
					info.AuthStatus = fmt.Sprintf("authenticated (%s)", authData.AuthMethod)
				} else {
					info.AuthStatus = "not authenticated"
				}
			}
		}
	}()
	wg.Wait()

	// History count
	info.HistoryCount = countLines(info.HistoryPath)

	// Session and project counts
	projectsDir := filepath.Join(a.paths.HomeDir, "projects")
	info.ProjectCount = countDirs(projectsDir)
	info.SessionCount = countSessionFiles(projectsDir)

	// Storage size
	info.StorageBytes = dirSize(a.paths.HomeDir)

	// MCP count
	servers, err := a.ListMCPServers()
	if err == nil {
		info.MCPCount = len(servers)
	}

	// Skill count
	skills, err := a.ListSkills()
	if err == nil {
		info.SkillCount = len(skills)
	}

	return info, nil
}

func countEnabledPlugins(plugins map[string]bool) int {
	n := 0
	for _, enabled := range plugins {
		if enabled {
			n++
		}
	}
	return n
}

func countLines(path string) int {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close() //nolint:errcheck // read-only file
	scanner := bufio.NewScanner(f)
	n := 0
	for scanner.Scan() {
		n++
	}
	return n
}

func countDirs(path string) int {
	entries, err := os.ReadDir(path)
	if err != nil {
		return 0
	}
	n := 0
	for _, e := range entries {
		if e.IsDir() {
			n++
		}
	}
	return n
}

func countSessionFiles(projectsDir string) int {
	n := 0
	projects, err := os.ReadDir(projectsDir)
	if err != nil {
		return 0
	}
	for _, p := range projects {
		if !p.IsDir() {
			continue
		}
		n += countSessionsInDir(filepath.Join(projectsDir, p.Name()))
	}
	return n
}

// countSessionsInDir counts unique sessions in a project directory.
// Sessions may exist as .jsonl files and/or directories sharing the same UUID.
func countSessionsInDir(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	seen := map[string]bool{}
	for _, e := range entries {
		id := strings.TrimSuffix(e.Name(), ".jsonl")
		seen[id] = true
	}
	return len(seen)
}

func dirSize(path string) int64 {
	var size int64
	_ = filepath.WalkDir(path, func(_ string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if info, err := d.Info(); err == nil {
			size += info.Size()
		}
		return nil
	})
	return size
}
