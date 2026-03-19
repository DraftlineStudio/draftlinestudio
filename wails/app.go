package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the main application struct bound to the frontend.
type App struct {
	ctx           context.Context
	currentFile   string
	settings      AppSettings
	cancelMu      sync.Mutex
	cancelRewrite context.CancelFunc // non-nil while a rewrite is in progress
}

// CancelRewrite aborts any in-progress AI rewrite call.
func (a *App) CancelRewrite() {
	a.cancelMu.Lock()
	defer a.cancelMu.Unlock()
	if a.cancelRewrite != nil {
		a.cancelRewrite()
		a.cancelRewrite = nil
	}
}

type Metadata struct {
	Title     string `json:"title"`
	Author    string `json:"author"`
	ISBN      string `json:"isbn"`
	Publisher string `json:"publisher"`
	Created   string `json:"created"`
	Modified  string `json:"modified"`
}

type ChapterItem struct {
	Title   string `json:"title"`
	Type    string `json:"type"`
	Content string `json:"content"`
}

type BookData struct {
	Version     string        `json:"version"`
	Metadata    Metadata      `json:"metadata"`
	Copyright   string        `json:"copyright"`
	FrontMatter []ChapterItem `json:"front_matter"`
	Body        []ChapterItem `json:"body"`
	BackMatter  []ChapterItem `json:"back_matter"`
	FilePath    string        `json:"file_path,omitempty"`
	StoryBible  StoryBible    `json:"story_bible,omitempty"`
}

type SaveResult struct {
	Success  bool   `json:"success"`
	FilePath string `json:"file_path"`
	Error    string `json:"error,omitempty"`
}

type Character struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Role        string `json:"role"` // protagonist | antagonist | supporting | minor | other
	Description string `json:"description"`
	Appearance  string `json:"appearance"`
	Personality string `json:"personality"`
	Motivation  string `json:"motivation"`
	Notes       string `json:"notes"`
}

type StoryBible struct {
	Characters []Character `json:"characters"`
	PlotNotes  string      `json:"plot_notes"`
	Timeline   string      `json:"timeline"`
}

type AppSettings struct {
	// Application
	DefaultAuthor       string `json:"default_author"`
	DefaultPublisher    string `json:"default_publisher"`
	DefaultCopyright    string `json:"default_copyright"`
	DefaultSaveDir      string `json:"default_save_dir"`
	DarkMode            bool   `json:"dark_mode"`
	ThemeMode           string `json:"theme_mode"`             // "light" | "dark" | "auto"
	AutoThemeUseManual  bool   `json:"auto_theme_use_manual"`
	AutoThemeDawn       string `json:"auto_theme_dawn"`        // "HH:MM" format
	AutoThemeDusk       string `json:"auto_theme_dusk"`        // "HH:MM" format
	// AI
	AIEnabled       bool   `json:"ai_enabled"`
	AIMode          string `json:"ai_mode"`           // "claudecode" | "api" | "local"
	AIProvider      string `json:"ai_provider"`       // "claude" | "openai" | ""
	AIAPIKey        string `json:"ai_api_key"`
	AIModel         string `json:"ai_model"`
	AILocalEndpoint string `json:"ai_local_endpoint"` // e.g. http://localhost:11434/v1
	AILocalModel    string `json:"ai_local_model"`
	ProseGuide      string `json:"prose_guide"`
	// Book defaults
	BookFont        string `json:"book_font"`
	BookFontSize    int    `json:"book_font_size"`
	BookLineSpacing string `json:"book_line_spacing"` // "1.0"|"1.25"|"1.5"|"2.0"
	BookDropCaps    bool   `json:"book_drop_caps"`
	BookTrimSize    string `json:"book_trim_size"` // "6x9"|"5.5x8.5"|"5x8"|"7x10"|"A5"
}

type ClaudeCodeStatus struct {
	Installed     bool   `json:"installed"`
	Authenticated bool   `json:"authenticated"`
	NpmAvailable  bool   `json:"npm_available"`
	Version       string `json:"version"`
	Error         string `json:"error,omitempty"`
}

type AIRewriteResult struct {
	Result string `json:"result"`
	Error  string `json:"error,omitempty"`
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.settings = a.LoadSettings()
}

// NewBook returns an empty BookData struct with defaults.
func (a *App) NewBook() BookData {
	now := time.Now().Format(time.RFC3339)
	a.currentFile = ""
	return BookData{
		Version: "2.0",
		Metadata: Metadata{
			Title:    "Untitled",
			Created:  now,
			Modified: now,
		},
		Copyright:   "",
		FrontMatter: []ChapterItem{},
		Body: []ChapterItem{
			{Title: "Chapter 1", Type: "Chapter", Content: "<p></p>"},
		},
		BackMatter: []ChapterItem{},
		StoryBible: StoryBible{Characters: []Character{}},
	}
}

// OpenBookDialog shows the native file picker and opens the selected file.
func (a *App) OpenBookDialog() (BookData, error) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "Open Draftline Project",
		DefaultDirectory: a.settings.DefaultSaveDir,
		Filters: []runtime.FileFilter{
			{DisplayName: "Draftline Files (*.draftline)", Pattern: "*.draftline"},
		},
	})
	if err != nil || path == "" {
		return BookData{}, nil
	}
	return a.openBook(path)
}

func (a *App) openBook(path string) (BookData, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return BookData{}, fmt.Errorf("failed to open file: %w", err)
	}
	defer r.Close()

	manifestData, err := readZipEntry(r, "manifest.json")
	if err != nil {
		return BookData{}, fmt.Errorf("invalid .draftline file: missing manifest.json")
	}

	var raw struct {
		Version  string   `json:"version"`
		Metadata Metadata `json:"metadata"`
		Chapters []struct {
			Title string `json:"title"`
			File  string `json:"file"`
		} `json:"chapters,omitempty"`
		FrontMatter []struct {
			Title string `json:"title"`
			Type  string `json:"type"`
			File  string `json:"file"`
		} `json:"front_matter,omitempty"`
		Body []struct {
			Title string `json:"title"`
			Type  string `json:"type"`
			File  string `json:"file"`
		} `json:"body,omitempty"`
		BackMatter []struct {
			Title string `json:"title"`
			Type  string `json:"type"`
			File  string `json:"file"`
		} `json:"back_matter,omitempty"`
	}

	if err := json.Unmarshal(manifestData, &raw); err != nil {
		return BookData{}, fmt.Errorf("failed to parse manifest: %w", err)
	}

	book := BookData{
		Version:     "2.0",
		Metadata:    raw.Metadata,
		FilePath:    path,
		FrontMatter: []ChapterItem{},
		Body:        []ChapterItem{},
		BackMatter:  []ChapterItem{},
	}

	// v1.0 migration: chapters/ -> body/
	if raw.Version == "1.0" {
		for _, ch := range raw.Chapters {
			content, _ := readZipEntry(r, ch.File)
			book.Body = append(book.Body, ChapterItem{
				Title:   ch.Title,
				Type:    "Chapter",
				Content: string(content),
			})
		}
		a.currentFile = path
		return book, nil
	}

	// v2.0
	copyright, _ := readZipEntry(r, "copyright.html")
	book.Copyright = string(copyright)

	for _, item := range raw.FrontMatter {
		content, _ := readZipEntry(r, item.File)
		book.FrontMatter = append(book.FrontMatter, ChapterItem{
			Title: item.Title, Type: item.Type, Content: string(content),
		})
	}
	for _, item := range raw.Body {
		content, _ := readZipEntry(r, item.File)
		book.Body = append(book.Body, ChapterItem{
			Title: item.Title, Type: item.Type, Content: string(content),
		})
	}
	for _, item := range raw.BackMatter {
		content, _ := readZipEntry(r, item.File)
		book.BackMatter = append(book.BackMatter, ChapterItem{
			Title: item.Title, Type: item.Type, Content: string(content),
		})
	}

	// story_bible.json (optional — not present in older files)
	if bibleData, err := readZipEntry(r, "story_bible.json"); err == nil {
		var bible StoryBible
		if json.Unmarshal(bibleData, &bible) == nil {
			if bible.Characters == nil {
				bible.Characters = []Character{}
			}
			book.StoryBible = bible
		}
	}
	if book.StoryBible.Characters == nil {
		book.StoryBible.Characters = []Character{}
	}

	a.currentFile = path
	return book, nil
}

// SaveBook saves to the current file path, or invokes SaveBookAs if unsaved.
func (a *App) SaveBook(book BookData) SaveResult {
	if a.currentFile == "" {
		return a.SaveBookAs(book)
	}
	return a.writeBook(book, a.currentFile)
}

// SaveBookAs shows the native save dialog.
func (a *App) SaveBookAs(book BookData) SaveResult {
	defaultName := book.Metadata.Title
	if strings.TrimSpace(defaultName) == "" {
		defaultName = "Untitled"
	}
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:            "Save Draftline Project",
		DefaultFilename:  defaultName + ".draftline",
		DefaultDirectory: a.settings.DefaultSaveDir,
		Filters: []runtime.FileFilter{
			{DisplayName: "Draftline Files (*.draftline)", Pattern: "*.draftline"},
		},
	})
	if err != nil || path == "" {
		return SaveResult{Success: false, Error: "cancelled"}
	}
	if !strings.HasSuffix(strings.ToLower(path), ".draftline") {
		path += ".draftline"
	}
	return a.writeBook(book, path)
}

// GetCurrentFile returns the path of the currently open file.
func (a *App) GetCurrentFile() string {
	return a.currentFile
}

func (a *App) settingsPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = "."
	}
	dir := filepath.Join(configDir, "draftline")
	_ = os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "settings.json")
}

func (a *App) LoadSettings() AppSettings {
	defaults := AppSettings{
		DarkMode:      true,
		ThemeMode:     "dark",
		AutoThemeDawn: "06:30",
		AutoThemeDusk: "19:00",
	}
	data, err := os.ReadFile(a.settingsPath())
	if err != nil {
		return defaults
	}
	var s AppSettings
	if err := json.Unmarshal(data, &s); err != nil {
		return defaults
	}
	return s
}

func (a *App) SaveSettings(settings AppSettings) error {
	a.settings = settings
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(a.settingsPath(), data, 0644)
}

func (a *App) BrowseForDirectory() string {
	path, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Default Save Location",
	})
	if err != nil {
		return ""
	}
	return path
}

// ── Recent Projects ─────────────────────────────────────────────────────────

type RecentProject struct {
	Type       string              `json:"type"` // "book" | "universe"
	Path       string              `json:"path"`
	Name       string              `json:"name"`
	LastOpened string              `json:"lastOpened"` // ISO 8601 timestamp
	Stats      RecentProjectStats  `json:"stats"`
}

type RecentProjectStats struct {
	Books    *int `json:"books,omitempty"`
	Chapters int  `json:"chapters"`
	Words    int  `json:"words"`
}

func (a *App) recentProjectsPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = "."
	}
	dir := filepath.Join(configDir, "draftline")
	_ = os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "recent_projects.json")
}

// GetRecentProjects returns the list of recently opened projects.
func (a *App) GetRecentProjects() []RecentProject {
	data, err := os.ReadFile(a.recentProjectsPath())
	if err != nil {
		return []RecentProject{}
	}
	var projects []RecentProject
	if err := json.Unmarshal(data, &projects); err != nil {
		return []RecentProject{}
	}
	return projects
}

// AddRecentProject adds or updates a project in the recent list.
func (a *App) AddRecentProject(project RecentProject) error {
	projects := a.GetRecentProjects()

	// Remove existing entry with same path
	filtered := make([]RecentProject, 0, len(projects))
	for _, p := range projects {
		if p.Path != project.Path {
			filtered = append(filtered, p)
		}
	}

	// Add new project at the front
	project.LastOpened = time.Now().Format(time.RFC3339)
	projects = append([]RecentProject{project}, filtered...)

	// Keep only the most recent 20
	if len(projects) > 20 {
		projects = projects[:20]
	}

	data, err := json.MarshalIndent(projects, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(a.recentProjectsPath(), data, 0644)
}

// RemoveRecentProject removes a project from the recent list by path.
func (a *App) RemoveRecentProject(path string) error {
	projects := a.GetRecentProjects()
	filtered := make([]RecentProject, 0, len(projects))
	for _, p := range projects {
		if p.Path != path {
			filtered = append(filtered, p)
		}
	}
	data, err := json.MarshalIndent(filtered, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(a.recentProjectsPath(), data, 0644)
}

// ClearRecentProjects removes all projects from the recent list.
func (a *App) ClearRecentProjects() error {
	return os.WriteFile(a.recentProjectsPath(), []byte("[]"), 0644)
}

// OpenRecentProject opens a project from the recent list by path.
func (a *App) OpenRecentProject(path string) (BookData, error) {
	return a.openBook(path)
}

// TestLocalAI makes a quick connectivity check to a local OpenAI-compatible endpoint.
func (a *App) TestLocalAI(endpoint string) AIRewriteResult {
	if endpoint == "" {
		return AIRewriteResult{Error: "No endpoint URL configured"}
	}
	url := strings.TrimRight(endpoint, "/") + "/models"
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return AIRewriteResult{Error: "Could not connect: " + err.Error()}
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return AIRewriteResult{Error: fmt.Sprintf("Server returned %d", resp.StatusCode)}
	}
	return AIRewriteResult{Result: "Connected"}
}

// RewriteText sends the HTML chapter content to the configured AI provider.
// mode: "line_edit" | "copy_edit" | "dev_edit" | "expand" | "smooth" | "voice_check"
func (a *App) RewriteText(html string, mode string) AIRewriteResult {
	if !a.settings.AIEnabled {
		return AIRewriteResult{Error: "AI features are disabled — enable them in App Settings"}
	}
	if mode == "" {
		mode = "line_edit"
	}

	// Diff format: for targeted per-paragraph edits, ask the model to return ONLY
	// changed paragraphs. This cuts output tokens by ~80% for typical chapters.
	useDiffFormat := mode == "line_edit" || mode == "copy_edit" || mode == "smooth"

	system := buildSystemPrompt(mode, a.settings.ProseGuide)
	var userMsg string

	switch {
	case mode == "voice_check":
		userMsg = "Analyse the following chapter for POV, tense, and narrative voice consistency. Return your findings as a brief bulleted list. If there are no issues, respond with \"No issues found.\"\n\n" + html
	case useDiffFormat:
		// Append diff-format output instruction
		system += "\n\nCRITICAL OUTPUT FORMAT: Each input paragraph is prefixed §N§ where N is its 1-based index.\nReturn ONLY paragraphs you change, one per line:\n§N§<p>revised text</p>\nOmit unchanged paragraphs entirely. If nothing needs changing: §NONE§"
		userMsg = buildDiffUserMsg(html)
	default:
		userMsg = "Rewrite the following, returning only the rewritten HTML paragraphs:\n\n" + html
	}

	var res AIRewriteResult
	switch a.settings.AIMode {
	case "claudecode":
		res = a.callClaudeCode(system, userMsg)
	case "local":
		if a.settings.AILocalEndpoint == "" {
			return AIRewriteResult{Error: "no local endpoint configured — set it in App Settings"}
		}
		res = a.callLocalAI(system, userMsg)
	default: // "api"
		if a.settings.AIProvider == "" || a.settings.AIAPIKey == "" {
			return AIRewriteResult{Error: "no API provider configured — add your key in App Settings"}
		}
		switch a.settings.AIProvider {
		case "claude":
			res = a.callClaude(system, userMsg)
		case "openai":
			res = a.callOpenAI(system, userMsg)
		default:
			return AIRewriteResult{Error: "unknown provider: " + a.settings.AIProvider}
		}
	}

	// If the model returned diff format, reconstruct full HTML before handing back.
	if res.Error == "" && useDiffFormat {
		res.Result = applyDiffResponse(html, res.Result)
	}
	return res
}

// CheckClaudeCode checks whether the claude CLI is installed and authenticated.
// It checks both the system PATH and Draftline's bundled install location.
func (a *App) CheckClaudeCode() ClaudeCodeStatus {
	npmAvailable := resolveNpmBin() != ""

	claudePath := resolveClaudeBin()
	if claudePath == "" {
		return ClaudeCodeStatus{NpmAvailable: npmAvailable}
	}
	out, err := claudeExec(context.Background(), claudePath, "--version").Output()
	version := ""
	if err == nil {
		version = strings.TrimSpace(string(out))
	}
	home, _ := os.UserHomeDir()
	credPaths := []string{
		filepath.Join(home, ".claude", ".credentials.json"),
		filepath.Join(home, ".config", "claude", ".credentials.json"),
	}
	authenticated := false
	for _, p := range credPaths {
		if _, err := os.Stat(p); err == nil {
			authenticated = true
			break
		}
	}
	return ClaudeCodeStatus{Installed: true, Authenticated: authenticated, NpmAvailable: npmAvailable, Version: version}
}

// OpenClaudeAuth runs "claude auth login" as a hidden background process.
// It opens the user's browser to complete the OAuth flow. When the process
// exits (auth complete), it emits a "claude:auth_complete" event to the frontend.
func (a *App) OpenClaudeAuth() {
	claudePath := resolveClaudeBin()
	if claudePath == "" {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		cmd := claudeExec(ctx, claudePath, "auth", "login")
		hideWindow(cmd)
		_ = cmd.Start()
		_ = cmd.Wait()
		runtime.EventsEmit(a.ctx, "claude:auth_complete", nil)
	}()
}

// readClaudeAPIKey reads the Anthropic API key stored by the Claude Code CLI.
// Returns empty string if not found or if using OAuth (no direct API key).
func readClaudeAPIKey() string {
	home, _ := os.UserHomeDir()
	credPaths := []string{
		filepath.Join(home, ".claude", ".credentials.json"),
		filepath.Join(home, ".config", "claude", ".credentials.json"),
	}
	for _, p := range credPaths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var creds struct {
			APIKey string `json:"apiKey"`
		}
		if json.Unmarshal(data, &creds) == nil && creds.APIKey != "" {
			return creds.APIKey
		}
	}
	return ""
}

// streamAnthropic calls the Anthropic Messages API with SSE streaming.
// It emits ai:token events for each text delta so the frontend can show
// tokens as they arrive. Returns the full accumulated text.
func (a *App) streamAnthropic(ctx context.Context, apiKey, model, system, userMsg string) (string, error) {
	reqBody, _ := json.Marshal(map[string]any{
		"model":      model,
		"max_tokens": 8192,
		"stream":     true,
		"system":     system,
		"messages":   []map[string]string{{"role": "user", "content": userMsg}},
	})

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() == context.Canceled {
			return "", fmt.Errorf("Cancelled")
		}
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		var apiErr struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(body, &apiErr) == nil && apiErr.Error.Message != "" {
			return "", fmt.Errorf("%s", apiErr.Error.Message)
		}
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var sb strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64*1024), 64*1024)

	for scanner.Scan() {
		if ctx.Err() != nil {
			return "", fmt.Errorf("Cancelled")
		}
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		var event struct {
			Type  string `json:"type"`
			Delta struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"delta"`
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal([]byte(data), &event) != nil {
			continue
		}
		switch event.Type {
		case "content_block_delta":
			if event.Delta.Type == "text_delta" {
				sb.WriteString(event.Delta.Text)
				runtime.EventsEmit(a.ctx, "ai:token", event.Delta.Text)
			}
		case "error":
			return "", fmt.Errorf("%s", event.Error.Message)
		}
	}

	if err := scanner.Err(); err != nil {
		if ctx.Err() == context.Canceled {
			return "", fmt.Errorf("Cancelled")
		}
		return "", err
	}

	return strings.TrimSpace(sb.String()), nil
}

// callClaudeCode handles the "claudecode" AI mode. If the credentials file
// contains a direct API key it calls the Anthropic API with streaming directly —
// faster and more reliable than the CLI subprocess. Falls back to the CLI when
// only OAuth credentials are present.
func (a *App) callClaudeCode(system, userMsg string) AIRewriteResult {
	if apiKey := readClaudeAPIKey(); apiKey != "" {
		model := a.settings.AIModel
		if model == "" {
			model = "claude-sonnet-4-6"
		}
		ctx, cancel := context.WithTimeout(a.ctx, 180*time.Second)
		a.cancelMu.Lock()
		a.cancelRewrite = cancel
		a.cancelMu.Unlock()
		defer func() {
			cancel()
			a.cancelMu.Lock()
			a.cancelRewrite = nil
			a.cancelMu.Unlock()
		}()
		runtime.EventsEmit(a.ctx, "ai:log", "Connecting to Anthropic API…")
		result, err := a.streamAnthropic(ctx, apiKey, model, system, userMsg)
		if err != nil {
			return AIRewriteResult{Error: err.Error()}
		}
		return AIRewriteResult{Result: result}
	}
	return a.callClaudeCodeCLI(system, userMsg)
}

// callClaudeCodeCLI invokes the Claude Code CLI as a subprocess.
// Used as a fallback when only OAuth credentials are available (no raw API key).
func (a *App) callClaudeCodeCLI(system, userMsg string) AIRewriteResult {
	path := resolveClaudeBin()
	if path == "" {
		return AIRewriteResult{Error: "Claude Code is not installed — open Settings › AI Studio to set it up"}
	}
	model := a.settings.AIModel
	if model == "" {
		model = "claude-sonnet-4-6"
	}
	ctx, cancel := context.WithTimeout(a.ctx, 180*time.Second)
	a.cancelMu.Lock()
	a.cancelRewrite = cancel
	a.cancelMu.Unlock()
	defer func() {
		cancel()
		a.cancelMu.Lock()
		a.cancelRewrite = nil
		a.cancelMu.Unlock()
	}()

	// Create an isolated home directory: real credentials so auth works, but no
	// MCP server config. MCP servers are started between init and the first API
	// call — if any hang, the subprocess hangs silently for the full timeout.
	tempHome, err := os.MkdirTemp("", "draftline-claude-*")
	if err != nil {
		return AIRewriteResult{Error: "cannot create temp dir: " + err.Error()}
	}
	defer os.RemoveAll(tempHome)

	realHome, _ := os.UserHomeDir()
	claudeDir := filepath.Join(tempHome, ".claude")
	_ = os.MkdirAll(claudeDir, 0755)
	if data, e2 := os.ReadFile(filepath.Join(realHome, ".claude", ".credentials.json")); e2 == nil {
		_ = os.WriteFile(filepath.Join(claudeDir, ".credentials.json"), data, 0600)
	}
	_ = os.WriteFile(filepath.Join(claudeDir, "settings.json"),
		[]byte(`{"mcpServers":{}}`), 0644)

	fullPrompt := system + "\n\n" + userMsg
	cmd := claudeExec(ctx, path,
		"-p", fullPrompt,
		"--output-format", "stream-json",
		"--verbose",
		"--max-turns", "1",
		"--model", model,
		"--dangerously-skip-permissions",
	)
	cmd.Dir = tempHome

	nodeDir := nodeInstallBinDir()
	baseEnv := os.Environ()
	filteredEnv := make([]string, 0, len(baseEnv)+4)
	for _, e := range baseEnv {
		key, _, _ := strings.Cut(e, "=")
		switch strings.ToUpper(key) {
		case "PATH", "HOME", "USERPROFILE", "HOMEDRIVE", "HOMEPATH":
			continue
		}
		filteredEnv = append(filteredEnv, e)
	}
	vol := filepath.VolumeName(tempHome)
	cmd.Env = append(filteredEnv,
		"PATH="+nodeDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"HOME="+tempHome,
		"USERPROFILE="+tempHome,
		"HOMEDRIVE="+vol,
		"HOMEPATH="+strings.TrimPrefix(tempHome, vol),
	)

	stdoutPipe, stdoutPipeErr := cmd.StdoutPipe()
	stderrPipe, stderrPipeErr := cmd.StderrPipe()

	runtime.EventsEmit(a.ctx, "ai:log", "Starting: "+filepath.Base(cmd.Path))
	if err := cmd.Start(); err != nil {
		return AIRewriteResult{Error: err.Error()}
	}
	runtime.EventsEmit(a.ctx, "ai:log", "Process started…")

	var wg sync.WaitGroup
	var resultText string
	var stderrBuf strings.Builder

	if stdoutPipeErr == nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			scanner := bufio.NewScanner(stdoutPipe)
			scanner.Buffer(make([]byte, 2*1024*1024), 2*1024*1024)
			var rawLines []string
			for scanner.Scan() {
				line := scanner.Text()
				rawLines = append(rawLines, line)
				preview := line
				if len(preview) > 100 {
					preview = preview[:100] + "…"
				}
				runtime.EventsEmit(a.ctx, "ai:log", "→ "+preview)
				var obj map[string]any
				if json.Unmarshal([]byte(line), &obj) == nil {
					if obj["type"] == "result" {
						if r, ok := obj["result"].(string); ok {
							resultText = r
						}
						if isErr, _ := obj["is_error"].(bool); isErr {
							if msg, ok := obj["result"].(string); ok {
								stderrBuf.WriteString(msg)
							}
						}
					}
				}
			}
			if resultText == "" {
				resultText = strings.Join(rawLines, "\n")
			}
		}()
	}
	if stderrPipeErr == nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			scanner := bufio.NewScanner(stderrPipe)
			for scanner.Scan() {
				line := scanner.Text()
				stderrBuf.WriteString(line + "\n")
				runtime.EventsEmit(a.ctx, "ai:log", line)
			}
		}()
	}

	runErr := cmd.Wait()
	wg.Wait()

	if runErr != nil {
		errMsg := strings.TrimSpace(stderrBuf.String())
		if errMsg == "" {
			errMsg = runErr.Error()
		}
		if ctx.Err() == context.Canceled {
			return AIRewriteResult{Error: "Cancelled"}
		}
		if ctx.Err() == context.DeadlineExceeded {
			return AIRewriteResult{Error: "Timed out — try a shorter chapter or check your Claude Code connection"}
		}
		return AIRewriteResult{Error: errMsg}
	}
	return AIRewriteResult{Result: strings.TrimSpace(resultText)}
}

// extractHTMLParagraphs returns each <p>...</p> element from HTML as a full tag string.
func extractHTMLParagraphs(html string) []string {
	re := regexp.MustCompile(`(?is)<p[^>]*>[\s\S]*?</p>`)
	paras := re.FindAllString(html, -1)
	if len(paras) == 0 && strings.TrimSpace(html) != "" {
		return []string{"<p>" + strings.TrimSpace(html) + "</p>"}
	}
	return paras
}

// buildDiffUserMsg prefixes each paragraph with §N§ so the model can return
// only changed paragraphs by index instead of the full chapter.
func buildDiffUserMsg(html string) string {
	paras := extractHTMLParagraphs(html)
	var sb strings.Builder
	for i, p := range paras {
		sb.WriteString(fmt.Sprintf("§%d§%s\n", i+1, p))
	}
	return sb.String()
}

// applyDiffResponse parses §N§<p>...</p> AI output and splices the changed
// paragraphs back into the original HTML, returning a complete HTML string.
func applyDiffResponse(originalHTML, aiResponse string) string {
	if strings.Contains(aiResponse, "§NONE§") || strings.TrimSpace(aiResponse) == "" {
		return originalHTML
	}
	paras := extractHTMLParagraphs(originalHTML)
	if len(paras) == 0 {
		return originalHTML
	}

	// Split on § — the response is "§N§content§N§content…" so splitting yields
	// ["", "1", "content", "5", "content", …]. Process odd/even index pairs.
	parts := strings.Split(aiResponse, "§")
	changes := make(map[int]string)
	for i := 1; i+1 < len(parts); i += 2 {
		idx, err := strconv.Atoi(strings.TrimSpace(parts[i]))
		if err != nil || idx < 1 || idx > len(paras) {
			continue
		}
		content := strings.TrimSpace(parts[i+1])
		if content != "" {
			changes[idx-1] = content
		}
	}

	var sb strings.Builder
	for i, p := range paras {
		if changed, ok := changes[i]; ok {
			sb.WriteString(changed)
		} else {
			sb.WriteString(p)
		}
		if i < len(paras)-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

func (a *App) callLocalAI(system, userMsg string) AIRewriteResult {
	model := a.settings.AILocalModel
	if model == "" {
		model = "llama3"
	}
	type msg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	reqBody, _ := json.Marshal(map[string]any{
		"model":    model,
		"messages": []msg{{Role: "system", Content: system}, {Role: "user", Content: userMsg}},
	})
	endpoint := strings.TrimRight(a.settings.AILocalEndpoint, "/") + "/chat/completions"
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Post(endpoint, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return AIRewriteResult{Error: "local AI request failed: " + err.Error()}
	}
	defer resp.Body.Close()
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error struct{ Message string } `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return AIRewriteResult{Error: "failed to parse response: " + err.Error()}
	}
	if result.Error.Message != "" {
		return AIRewriteResult{Error: result.Error.Message}
	}
	if len(result.Choices) == 0 {
		return AIRewriteResult{Error: "empty response from local AI"}
	}
	return AIRewriteResult{Result: strings.TrimSpace(result.Choices[0].Message.Content)}
}

func buildSystemPrompt(mode, proseGuide string) string {
	styleBlock := ""
	if proseGuide != "" {
		styleBlock = "\n\nSTYLE GUIDE — match the rhythm, vocabulary, and voice of these examples:\n---\n" + proseGuide + "\n---"
	}
	banned := "BANNED words and phrases: tapestry, testament, navigate, delve, underscore, myriad, realm, crucial, pivotal, journey, beacon, vibrant, game-changer"

	switch mode {
	case "copy_edit":
		return `You are a meticulous copy editor. Correct grammar, punctuation, spelling, and style inconsistencies in the provided HTML text.

Rules:
- Fix only errors — do NOT rewrite prose or change author's voice
- Correct subject-verb agreement, tense consistency errors, and punctuation
- Fix repeated words used awkwardly in the same sentence
- Preserve paragraph breaks — return one <p> element per original paragraph
- ` + banned + `

Return ONLY the corrected HTML using <p> tags. No explanations.`

	case "dev_edit":
		return `You are a developmental editor. Improve pacing and scene structure in the provided HTML text.

Rules:
- Cut slow, redundant passages — every sentence must earn its place
- Strengthen scene transitions and cause-and-effect clarity
- Heighten tension where the pacing drags
- Preserve all story facts, characters, and dialogue content
- Write in the same tense and POV as the original
- Preserve paragraph breaks — return one <p> element per original paragraph
- ` + banned + styleBlock + `

Return ONLY the rewritten HTML using <p> tags. No explanations.`

	case "expand":
		return `You are a literary prose writer. Expand and enrich the provided HTML text with sensory detail, atmosphere, and texture.

Rules:
- Flesh out thin paragraphs — add physical sensation, setting detail, internal thought
- Show don't tell: replace summary with scene
- Match the existing POV depth, tense, and voice exactly
- Do not introduce new plot events or characters
- Preserve paragraph breaks — return one <p> element per original paragraph (may be longer)
- ` + banned + styleBlock + `

Return ONLY the rewritten HTML using <p> tags. No explanations.`

	case "smooth":
		return `You are a line editor focused on flow and rhythm. Smooth the provided HTML text.

Rules:
- Eliminate word repetition within paragraphs (same word used 2+ times nearby)
- Improve sentence-to-sentence transitions
- Vary sentence openings — avoid starting consecutive sentences the same way
- Minimal changes — improve flow without changing meaning or voice
- Preserve paragraph breaks — return one <p> element per original paragraph
- ` + banned + styleBlock + `

Return ONLY the rewritten HTML using <p> tags. No explanations.`

	case "voice_check":
		return `You are a developmental editor reviewing for POV, tense, and voice consistency.

Analyse the provided chapter and report:
- POV violations (head-hopping, unearned omniscience)
- Tense inconsistencies (unexpected shifts)
- Narrative voice breaks (narrator suddenly sounds different)
- Overuse of filter words (saw, heard, felt, noticed)

Format each issue as: • [Type]: description (approximate paragraph or quote)
If no issues are found, respond with exactly: No issues found.

Do NOT rewrite any text. Return findings only.`

	default: // "line_edit"
		base := `You are a skilled literary prose editor. Rewrite the provided HTML text, preserving all narrative content, characters, events, and dialogue meaning exactly.

Rewriting rules:
- Vary sentence rhythm: mix short, punchy sentences with longer, flowing ones
- Use strong, precise, concrete words — avoid vague abstractions
- Write in the same tense and POV as the original
- Preserve paragraph breaks — return one <p> element per original paragraph
- ` + banned
		if proseGuide != "" {
			base = `You are a skilled literary prose editor. Rewrite the provided HTML text to match the style shown below, while preserving all narrative content exactly.` + styleBlock + `

Rewriting rules:
- Match the rhythm, cadence, and sentence variety of the style examples above
- Vary sentence length as in the examples
- Preserve all story facts: names, places, events, exact dialogue content
- Preserve paragraph breaks — return one <p> element per original paragraph
- ` + banned
		}
		return base + `

Return ONLY the rewritten HTML using <p> tags. No explanations, no headings, no extra text.`
	}
}

func (a *App) callClaude(system, userMsg string) AIRewriteResult {
	model := a.settings.AIModel
	if model == "" {
		model = "claude-sonnet-4-6"
	}
	ctx, cancel := context.WithTimeout(a.ctx, 180*time.Second)
	a.cancelMu.Lock()
	a.cancelRewrite = cancel
	a.cancelMu.Unlock()
	defer func() {
		cancel()
		a.cancelMu.Lock()
		a.cancelRewrite = nil
		a.cancelMu.Unlock()
	}()
	runtime.EventsEmit(a.ctx, "ai:log", "Connecting to Claude API…")
	result, err := a.streamAnthropic(ctx, a.settings.AIAPIKey, model, system, userMsg)
	if err != nil {
		return AIRewriteResult{Error: err.Error()}
	}
	return AIRewriteResult{Result: result}
}

func (a *App) callOpenAI(system, userMsg string) AIRewriteResult {
	model := a.settings.AIModel
	if model == "" {
		model = "gpt-4o"
	}
	reqBody, _ := json.Marshal(map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": userMsg},
		},
	})
	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(reqBody))
	if err != nil {
		return AIRewriteResult{Error: err.Error()}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.settings.AIAPIKey)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return AIRewriteResult{Error: err.Error()}
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return AIRewriteResult{Error: "failed to parse OpenAI response"}
	}
	if result.Error != nil {
		return AIRewriteResult{Error: result.Error.Message}
	}
	if len(result.Choices) == 0 {
		return AIRewriteResult{Error: "empty response from OpenAI"}
	}
	return AIRewriteResult{Result: strings.TrimSpace(result.Choices[0].Message.Content)}
}

func (a *App) writeBook(book BookData, path string) SaveResult {
	book.Metadata.Modified = time.Now().Format(time.RFC3339)

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	type entry struct {
		Title string `json:"title"`
		Type  string `json:"type"`
		File  string `json:"file"`
	}
	type manifest struct {
		Version     string   `json:"version"`
		Metadata    Metadata `json:"metadata"`
		FrontMatter []entry  `json:"front_matter"`
		Body        []entry  `json:"body"`
		BackMatter  []entry  `json:"back_matter"`
	}

	mf := manifest{
		Version:     "2.0",
		Metadata:    book.Metadata,
		FrontMatter: []entry{},
		Body:        []entry{},
		BackMatter:  []entry{},
	}

	addEntry := func(name, content string) error {
		f, err := w.Create(name)
		if err != nil {
			return err
		}
		_, err = f.Write([]byte(content))
		return err
	}

	if err := addEntry("copyright.html", book.Copyright); err != nil {
		return SaveResult{Success: false, Error: err.Error()}
	}
	if book.StoryBible.Characters == nil {
		book.StoryBible.Characters = []Character{}
	}
	bibleJSON, _ := json.MarshalIndent(book.StoryBible, "", "  ")
	if err := addEntry("story_bible.json", string(bibleJSON)); err != nil {
		return SaveResult{Success: false, Error: err.Error()}
	}
	for i, item := range book.FrontMatter {
		file := fmt.Sprintf("front_matter/%03d.html", i)
		if err := addEntry(file, item.Content); err != nil {
			return SaveResult{Success: false, Error: err.Error()}
		}
		mf.FrontMatter = append(mf.FrontMatter, entry{item.Title, item.Type, file})
	}
	for i, item := range book.Body {
		file := fmt.Sprintf("body/%03d.html", i)
		if err := addEntry(file, item.Content); err != nil {
			return SaveResult{Success: false, Error: err.Error()}
		}
		mf.Body = append(mf.Body, entry{item.Title, item.Type, file})
	}
	for i, item := range book.BackMatter {
		file := fmt.Sprintf("back_matter/%03d.html", i)
		if err := addEntry(file, item.Content); err != nil {
			return SaveResult{Success: false, Error: err.Error()}
		}
		mf.BackMatter = append(mf.BackMatter, entry{item.Title, item.Type, file})
	}

	manifestBytes, _ := json.MarshalIndent(mf, "", "  ")
	if err := addEntry("manifest.json", string(manifestBytes)); err != nil {
		return SaveResult{Success: false, Error: err.Error()}
	}

	w.Close()

	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		return SaveResult{Success: false, Error: err.Error()}
	}

	a.currentFile = path
	return SaveResult{Success: true, FilePath: path}
}

func readZipEntry(r *zip.ReadCloser, name string) ([]byte, error) {
	for _, f := range r.File {
		if f.Name == name {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return nil, fmt.Errorf("entry %q not found", name)
}
