package main

// The recent-projects list.
//
// A recent project is a path and a little bookkeeping: enough for the start
// screen to show a book before it is open. The list is capped, most-recent
// first, and the cover key is stamped on read rather than stored so it can
// never disagree with the path the record actually holds.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"draftline/internal/fsutil"
	"draftline/internal/types"
)

// ── Recent Projects ─────────────────────────────────────────────────────────

func (a *App) recentProjectsPath() string {
	path := filepath.Join(configDir(), "recent_projects.json")
	// Best-effort: tighten a file written with looser permissions by an older
	// version. Ignore errors.
	_ = os.Chmod(path, 0600)
	return path
}

// GetRecentProjects returns the list of recently opened projects.
func (a *App) GetRecentProjects() []types.RecentProject {
	data, err := os.ReadFile(a.recentProjectsPath())
	if err != nil {
		return []types.RecentProject{}
	}
	var projects []types.RecentProject
	if err := json.Unmarshal(data, &projects); err != nil {
		return []types.RecentProject{}
	}
	// Stamped here rather than stored, so it is always right for the path the
	// record actually holds and a hand-edited recents file cannot disagree
	// with itself.
	for i := range projects {
		projects[i].CoverKey = recentCoverKey(projects[i].Path)
	}
	return projects
}

// AddRecentProject adds or updates a project in the recent list.
func (a *App) AddRecentProject(project types.RecentProject) error {
	projects := a.GetRecentProjects()

	// Remove existing entry with same path
	filtered := make([]types.RecentProject, 0, len(projects))
	for _, p := range projects {
		if p.Path != project.Path {
			filtered = append(filtered, p)
		}
	}

	// Add new project at the front
	project.LastOpened = time.Now().Format(time.RFC3339)
	projects = append([]types.RecentProject{project}, filtered...)

	// Keep only the most recent 20
	if len(projects) > 20 {
		projects = projects[:20]
	}

	data, err := json.MarshalIndent(projects, "", "  ")
	if err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(a.recentProjectsPath(), data, 0600)
}

// RemoveRecentProject removes a project from the recent list by path.
func (a *App) RemoveRecentProject(path string) error {
	projects := a.GetRecentProjects()
	filtered := make([]types.RecentProject, 0, len(projects))
	for _, p := range projects {
		if p.Path != path {
			filtered = append(filtered, p)
		}
	}
	data, err := json.MarshalIndent(filtered, "", "  ")
	if err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(a.recentProjectsPath(), data, 0600)
}

// ClearRecentProjects removes all projects from the recent list.
func (a *App) ClearRecentProjects() error {
	return fsutil.WriteFileAtomic(a.recentProjectsPath(), []byte("[]"), 0600)
}

// OpenRecentProject opens a project from the recent list by path.
func (a *App) OpenRecentProject(path string) (types.BookData, error) {
	return a.openBook(path)
}
