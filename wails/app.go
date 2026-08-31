package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"draftline/internal/ai"
	"draftline/internal/backup"
	"draftline/internal/book"
	"draftline/internal/export"
	"draftline/internal/fsutil"
	"draftline/internal/indexing"
	"draftline/internal/logging"
	"draftline/internal/platform"
	"draftline/internal/types"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/zalando/go-keyring"
)

// ── Backup System ───────────────────────────────────────────────────────────
// Thin wrappers around backup package. See internal/backup/ for implementation.

// ListBackups returns available backups for the current file.
func (a *App) ListBackups() []types.BackupInfo {
	return backup.List(a.getCurrentFile())
}

// RestoreBackup restores a backup by number (1 = most recent, 5 = oldest).
func (a *App) RestoreBackup(number int) types.SaveResult {
	return backup.Restore(a.getCurrentFile(), number)
}

// AppVersion Format: MAJOR.MINOR.BUILD - Example: 0.8.02313 → 0.8.02314 (bug fix) → 0.9.02315 (new feature set)
const AppVersion = "0.16.02449"

// maxAIResponseBytes caps how much of a provider HTTP response body we will
// read into memory. It sits comfortably above any plausible max-output-tokens
// payload while preventing a hostile or malfunctioning endpoint from exhausting
// memory via an unbounded body (audit Sol SEC-007).
const maxAIResponseBytes = 16 << 20 // 16 MB

// readAIResponseBody reads a provider response body up to maxAIResponseBytes.
// If the body exceeds the cap it returns a clear "response too large" error
// instead of buffering it all.
func readAIResponseBody(r io.Reader) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(r, maxAIResponseBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxAIResponseBytes {
		return nil, fmt.Errorf("response too large (exceeds %d bytes)", maxAIResponseBytes)
	}
	return body, nil
}

type aiRequestProfile struct {
	lightweight bool
}

var standardAIRequest = aiRequestProfile{}

func rewriteRequestProfile(mode string) aiRequestProfile {
	return aiRequestProfile{lightweight: mode == "line_edit" || mode == "copy_edit"}
}

// App is the main application struct bound to the frontend.
type App struct {
	ctx context.Context

	// stateMu guards currentFile and settings, which are read and written from
	// multiple Wails-bound goroutines (open/save/import paths, AI dispatch,
	// getters). Critical sections stay small: callers snapshot the values they
	// need under the lock and release it before doing any I/O or HTTP work.
	stateMu     sync.RWMutex
	currentFile string
	settings    types.AppSettings

	cancelMu      sync.Mutex
	cancelRewrite context.CancelFunc // non-nil while a rewrite is in progress
	aiBusy        bool               // true while the single AI-request slot is held
	aiGen         uint64             // generation counter; guards stale releases

	// API key state. The key lives in the OS keyring; legacyAPIKey holds a
	// plaintext key only on machines where no keyring is available, so users
	// there don't lose AI access.
	apiKeyMu     sync.Mutex
	apiKeyCache  string
	apiKeyLoaded bool
	legacyAPIKey string
}

// GetAppVersion returns the current application version string.
func (a *App) GetAppVersion() string {
	return AppVersion
}

// getSettings returns a consistent snapshot of the current settings. Callers
// that read several fields should snapshot once and read from the copy so a
// concurrent SaveSettings can't tear the read across fields.
func (a *App) getSettings() types.AppSettings {
	a.stateMu.RLock()
	defer a.stateMu.RUnlock()
	return a.settings
}

// setSettings replaces the in-memory settings under the state lock.
func (a *App) setSettings(s types.AppSettings) {
	a.stateMu.Lock()
	a.settings = s
	a.stateMu.Unlock()
}

// getCurrentFile returns the current file path under the state lock.
func (a *App) getCurrentFile() string {
	a.stateMu.RLock()
	defer a.stateMu.RUnlock()
	return a.currentFile
}

// setCurrentFile updates the current file path under the state lock.
func (a *App) setCurrentFile(path string) {
	a.stateMu.Lock()
	a.currentFile = path
	a.stateMu.Unlock()
}

// setLegacyAPIKey writes the plaintext fallback key under the API-key lock.
func (a *App) setLegacyAPIKey(key string) {
	a.apiKeyMu.Lock()
	a.legacyAPIKey = key
	a.apiKeyMu.Unlock()
}

// getLegacyAPIKey reads the plaintext fallback key under the API-key lock.
func (a *App) getLegacyAPIKey() string {
	a.apiKeyMu.Lock()
	defer a.apiKeyMu.Unlock()
	return a.legacyAPIKey
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

// ── Character Indexing ────────────────────────────────────────────────────────
// Thin wrappers around indexing package. See internal/indexing/ for implementation.

// IndexBook runs the two-phase character pipeline: mention extraction over
// every chapter, then entity resolution ("Ruiz", "Officer Ruiz", "Carlos Ruiz"
// become one entity). The result includes the updated book with characters
// and entity data populated.
func (a *App) IndexBook(book types.BookData) types.IndexResult {
	return indexing.IndexBook(&book)
}

// SplitEntity separates mentions from an entity into a new entity.
// Use this when auto-merging incorrectly combined two different people.
// Example: "Ruiz" (the cop) and "Ruiz" (the sister) got merged, call this to split.
func (a *App) SplitEntity(book types.BookData, entityID string, mentionIDs []string, newCanonical string) types.SplitEntityResult {
	err := indexing.SplitCharacterEntity(&book, entityID, mentionIDs, newCanonical)
	if err != nil {
		return types.SplitEntityResult{
			Success: false,
			Error:   err.Error(),
		}
	}
	return types.SplitEntityResult{
		Success:    true,
		Book:       book,
		Characters: book.StoryBible.Characters,
	}
}

// MergeEntities merges two or more detected characters into one. The first
// ID is the primary; canonical overrides the display name if non-empty.
// The merge is remembered by name and re-applied on every re-index.
func (a *App) MergeEntities(book types.BookData, entityIDs []string, canonical string) types.SplitEntityResult {
	err := indexing.MergeCharacterEntities(&book, entityIDs, canonical)
	if err != nil {
		return types.SplitEntityResult{
			Success: false,
			Error:   err.Error(),
		}
	}
	return types.SplitEntityResult{
		Success:    true,
		Book:       book,
		Characters: book.StoryBible.Characters,
	}
}

// ── Relationship Analysis ─────────────────────────────────────────────────────
// Analyzes character interactions and builds relationship graphs.

// AnalyzeRelationships detects character interactions and builds relationship data.
// This should be called after entity resolution has been run.
func (a *App) AnalyzeRelationships(book types.BookData) types.RelationshipAnalysisResult {
	analyzer := indexing.NewRelationshipAnalyzer()
	relData, err := analyzer.AnalyzeBook(book)
	if err != nil {
		return types.RelationshipAnalysisResult{
			Success: false,
			Error:   err.Error(),
		}
	}

	// Attach to book
	book.Analysis.Relationships = relData

	return types.RelationshipAnalysisResult{
		Success:            true,
		Book:               book,
		ScenesDetected:     len(relData.Scenes),
		InteractionsFound:  len(relData.Interactions),
		RelationshipsBuilt: len(relData.Relationships),
	}
}

// AnalyzeBook runs the bundled local analysis pipeline as one coherent job.
// The frontend receives progress events for the JetBrains-style status bar and
// one final BookData value, avoiding intermediate whole-book replacements.
func (a *App) AnalyzeBook(bookData types.BookData) types.FullAnalysisResult {
	chapterCount := len(indexing.AllChapters(&bookData))
	a.emitAnalysisProgress(types.StoryAnalysisProgress{
		Phase: "characters", Message: fmt.Sprintf("Analyzing characters across %d chapters", chapterCount),
		Current: 0, Total: chapterCount, Percent: 5,
	})

	indexResult := indexing.IndexBook(&bookData)
	if !indexResult.Success {
		return types.FullAnalysisResult{Success: false, Error: indexResult.Error}
	}
	bookData = indexResult.Book

	a.emitAnalysisProgress(types.StoryAnalysisProgress{
		Phase: "relationships", Message: "Mapping confirmed character relationships",
		Current: 1, Total: 1, Percent: 45,
	})
	analyzer := indexing.NewRelationshipAnalyzer()
	relationships, err := analyzer.AnalyzeBook(bookData)
	if err != nil {
		return types.FullAnalysisResult{Success: false, Error: err.Error()}
	}
	bookData.Analysis.Relationships = relationships

	bookData.Analysis.Story = indexing.AnalyzeStory(&bookData, a.emitAnalysisProgress)
	if bookData.Analysis.Version < 2 {
		bookData.Analysis.Version = 2
	}
	a.emitAnalysisProgress(types.StoryAnalysisProgress{
		Phase: "complete", Message: "Story analysis current", Current: chapterCount, Total: chapterCount, Percent: 100,
	})
	return types.FullAnalysisResult{Success: true, Book: bookData}
}

func (a *App) emitAnalysisProgress(progress types.StoryAnalysisProgress) {
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "analysis:progress", progress)
	}
}

// GetCharacterRelationships returns all relationships for a specific character.
func (a *App) GetCharacterRelationships(book types.BookData, characterID string) []types.RelationshipRecord {
	if book.Analysis.Relationships == nil {
		return []types.RelationshipRecord{}
	}
	return indexing.GetCharacterRelationships(characterID, book.Analysis.Relationships.Relationships)
}

// GetCharacterTimeline returns a timeline of events for a character.
func (a *App) GetCharacterTimeline(book types.BookData, characterID string) types.CharacterTimelineResult {
	if book.Analysis.EntityResolution == nil {
		return types.CharacterTimelineResult{
			Success: false,
			Error:   "No analysis data available",
		}
	}

	timeline := indexing.GetCharacterTimeline(characterID, book, book.Analysis.Relationships)
	return types.CharacterTimelineResult{
		Success:     true,
		CharacterID: characterID,
		Events:      timeline,
	}
}

// AddCharacterEvent adds a user-defined event to the relationship data.
func (a *App) AddCharacterEvent(book types.BookData, event types.CharacterEvent) types.BookData {
	if book.Analysis.Relationships == nil {
		book.Analysis.Relationships = &types.RelationshipData{}
	}

	// Generate ID if not provided
	if event.ID == "" {
		event.ID = fmt.Sprintf("evt-%d", time.Now().UnixNano())
	}
	event.IsAutoDetected = false

	book.Analysis.Relationships.Events = append(book.Analysis.Relationships.Events, event)
	return book
}

// DeleteCharacterEvent removes an event from the relationship data.
func (a *App) DeleteCharacterEvent(book types.BookData, eventID string) types.BookData {
	if book.Analysis.Relationships == nil {
		return book
	}

	events := book.Analysis.Relationships.Events
	filtered := make([]types.CharacterEvent, 0, len(events))
	for _, evt := range events {
		if evt.ID != eventID {
			filtered = append(filtered, evt)
		}
	}
	book.Analysis.Relationships.Events = filtered
	return book
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	raw := a.loadSettingsFromDisk()
	// One-time migration: move a plaintext key from settings.json into the OS
	// keyring. The plaintext copy is only removed after the keyring accepts
	// the key; if no keyring is available the key stays put so it isn't lost.
	if raw.AIAPIKey != "" {
		if err := keyring.Set(keyringService, keyringAccount, raw.AIAPIKey); err == nil {
			stripped := raw
			stripped.AIAPIKey = ""
			if werr := a.writeSettingsFile(stripped); werr != nil {
				log.Printf("failed to strip migrated API key from settings.json: %v", werr)
			}
		} else {
			a.setLegacyAPIKey(raw.AIAPIKey)
		}
	}
	raw.AIAPIKey = ""
	a.setSettings(raw)
	logging.SetEnabled(raw.AIDebugLogging)
}

// NewBook returns an empty types.BookData struct with defaults.
func (a *App) NewBook() types.BookData {
	now := time.Now().Format(time.RFC3339)
	a.setCurrentFile("")
	newBook := types.BookData{
		Version: "2.2",
		Metadata: types.Metadata{
			Title:    "Untitled",
			Created:  now,
			Modified: now,
		},
		Copyright:   "",
		FrontMatter: []types.ChapterItem{},
		Body: []types.ChapterItem{
			{Title: "Chapter 1", Type: "Chapter", Content: "<p></p>"},
		},
		BackMatter: []types.ChapterItem{},
		StoryBible: types.StoryBible{Characters: []types.Character{}},
	}
	book.EnsureBookChapterIDs(&newBook)
	return newBook
}

// OpenBookDialog shows the native file picker and opens the selected file.
func (a *App) OpenBookDialog() (types.BookData, error) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "Open Draftline Project",
		DefaultDirectory: a.getSettings().DefaultSaveDir,
		Filters: []runtime.FileFilter{
			{DisplayName: "Draftline Files (*.draftline)", Pattern: "*.draftline"},
		},
	})
	if err != nil || path == "" {
		return types.BookData{}, nil
	}
	return a.openBook(path)
}

func (a *App) openBook(path string) (types.BookData, error) {
	b, err := book.Open(path)
	if err != nil {
		return types.BookData{}, err
	}
	a.setCurrentFile(path)
	return b, nil
}

// SaveBook saves to the current file path, or invokes SaveBookAs if unsaved.
func (a *App) SaveBook(book types.BookData) types.SaveResult {
	current := a.getCurrentFile()
	if current == "" {
		return a.SaveBookAs(book)
	}
	return a.writeBook(book, current)
}

// SaveBookSnapshots atomically saves the current book and appends deduplicated
// chapter snapshots to the version history embedded in its archive.
func (a *App) SaveBookSnapshots(b types.BookData, snapshots []types.ChapterSnapshotRequest) types.SaveResult {
	current := a.getCurrentFile()
	if current == "" {
		return types.SaveResult{Success: false, Error: "save the project before creating version history"}
	}
	result := book.WriteWithSnapshots(current, b, AppVersion, snapshots)
	if result.Success {
		a.setCurrentFile(current)
	}
	return result
}

// ListChapterHistory returns snapshot metadata for one stable chapter ID.
func (a *App) ListChapterHistory(chapterID string) ([]types.ChapterHistoryEntry, error) {
	return book.ListChapterHistory(a.getCurrentFile(), chapterID)
}

// GetChapterHistory returns one snapshot after resolving it through the
// archive index; callers cannot use the ID as an arbitrary archive path.
func (a *App) GetChapterHistory(snapshotID string) (types.ChapterHistorySnapshot, error) {
	return book.GetChapterHistorySnapshot(a.getCurrentFile(), snapshotID)
}

// SaveBookAs shows the native save dialog.
func (a *App) SaveBookAs(book types.BookData) types.SaveResult {
	defaultName := book.Metadata.Title
	if strings.TrimSpace(defaultName) == "" {
		defaultName = "Untitled"
	}
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:            "Save Draftline Project",
		DefaultFilename:  defaultName + ".draftline",
		DefaultDirectory: a.getSettings().DefaultSaveDir,
		Filters: []runtime.FileFilter{
			{DisplayName: "Draftline Files (*.draftline)", Pattern: "*.draftline"},
		},
	})
	if err != nil || path == "" {
		return types.SaveResult{Success: false, Error: "cancelled"}
	}
	if !strings.HasSuffix(strings.ToLower(path), ".draftline") {
		path += ".draftline"
	}
	return a.writeBook(book, path)
}

// GetCurrentFile returns the path of the currently open file.
func (a *App) GetCurrentFile() string {
	return a.getCurrentFile()
}

// ── Export Functions ─────────────────────────────────────────────────────────
// Thin wrappers around export package. See internal/export/ for implementation.

// ExportEPUB exports the book to EPUB format.
func (a *App) ExportEPUB(book types.BookData, options types.ExportOptions) types.ExportResult {
	defaultName := book.Metadata.Title
	if strings.TrimSpace(defaultName) == "" {
		defaultName = "Untitled"
	}

	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Export as EPUB",
		DefaultFilename: defaultName + ".epub",
		Filters: []runtime.FileFilter{
			{DisplayName: "EPUB Files (*.epub)", Pattern: "*.epub"},
		},
	})
	if err != nil || path == "" {
		return types.ExportResult{Success: false, Error: "cancelled"}
	}

	if !strings.HasSuffix(strings.ToLower(path), ".epub") {
		path += ".epub"
	}

	return export.EPUB(path, book, options)
}

// ExportDOCX exports the book to DOCX format.
func (a *App) ExportDOCX(book types.BookData, options types.ExportOptions) types.ExportResult {
	defaultName := book.Metadata.Title
	if strings.TrimSpace(defaultName) == "" {
		defaultName = "Untitled"
	}

	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Export as DOCX",
		DefaultFilename: defaultName + ".docx",
		Filters: []runtime.FileFilter{
			{DisplayName: "Word Documents (*.docx)", Pattern: "*.docx"},
		},
	})
	if err != nil || path == "" {
		return types.ExportResult{Success: false, Error: "cancelled"}
	}

	if !strings.HasSuffix(strings.ToLower(path), ".docx") {
		path += ".docx"
	}

	return export.DOCX(path, book, options)
}

// ExportPDF exports the book to PDF format.
func (a *App) ExportPDF(book types.BookData, options types.PDFOptions) types.ExportResult {
	defaultName := book.Metadata.Title
	if strings.TrimSpace(defaultName) == "" {
		defaultName = "Untitled"
	}

	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Export as PDF",
		DefaultFilename: defaultName + ".pdf",
		Filters: []runtime.FileFilter{
			{DisplayName: "PDF Files (*.pdf)", Pattern: "*.pdf"},
		},
	})
	if err != nil || path == "" {
		return types.ExportResult{Success: false, Error: "cancelled"}
	}

	if !strings.HasSuffix(strings.ToLower(path), ".pdf") {
		path += ".pdf"
	}

	return export.PDF(path, book, options)
}

// ExportPrintPDF exports the book to print-ready PDF format.
func (a *App) ExportPrintPDF(book types.BookData, options types.PrintPDFOptions) types.ExportResult {
	defaultName := book.Metadata.Title
	if strings.TrimSpace(defaultName) == "" {
		defaultName = "Untitled"
	}

	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Export as Print-Ready PDF",
		DefaultFilename: defaultName + "_print.pdf",
		Filters: []runtime.FileFilter{
			{DisplayName: "PDF Files (*.pdf)", Pattern: "*.pdf"},
		},
	})
	if err != nil || path == "" {
		return types.ExportResult{Success: false, Error: "cancelled"}
	}

	if !strings.HasSuffix(strings.ToLower(path), ".pdf") {
		path += ".pdf"
	}

	return export.PrintPDF(path, book, options)
}

// ── Settings ─────────────────────────────────────────────────────────────────

func (a *App) settingsPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = "."
	}
	dir := filepath.Join(configDir, "draftline")
	_ = os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "settings.json")
}

// loadSettingsFromDisk reads settings.json verbatim (including a legacy
// plaintext API key, which only startup's migration may see).
func (a *App) loadSettingsFromDisk() types.AppSettings {
	defaults := types.AppSettings{
		AIEnabled:               false,
		DarkMode:                true,
		ThemeMode:               "dark",
		AutoThemeUseManual:      true,
		AutoThemeDawn:           "06:30",
		AutoThemeDusk:           "19:00",
		ActivityAutoSaveEnabled: true,
		CustomDictionary:        []string{},
		SpellCheckEnabled:       true,
		GrammarCheckEnabled:     true,
		CastEnabled:             true,
		StoryBibleEnabled:       true,
		PlotWalkerEnabled:       true,
		AnalysisEnabled:         true,
		CharactersLaneView:      "grid",
		SidebarPanelWidth:       350,
		// Open on the Writing Dashboard by default; "" means closed.
		SidebarActiveSection: "dashboard",
	}
	data, err := os.ReadFile(a.settingsPath())
	if err != nil {
		return defaults
	}
	// Decode over defaults so settings written by older versions inherit newly
	// introduced feature flags instead of silently disabling them.
	s := defaults
	if err := json.Unmarshal(data, &s); err != nil {
		return defaults
	}
	return s
}

// LoadSettings returns the settings for the frontend. The API key itself
// never crosses the Wails bridge — only the has_api_key flag does.
func (a *App) LoadSettings() types.AppSettings {
	s := a.loadSettingsFromDisk()
	s.AIAPIKey = ""
	s.HasAPIKey = a.HasAPIKey()
	return s
}

func (a *App) writeSettingsFile(settings types.AppSettings) error {
	settings.HasAPIKey = false // derived at load time, not persisted
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(a.settingsPath(), data, 0600)
}

func (a *App) SaveSettings(settings types.AppSettings) error {
	// Never trust a key from the frontend; keys arrive via SetAPIKey only.
	settings.AIAPIKey = ""
	a.setSettings(settings)
	logging.SetEnabled(settings.AIDebugLogging)

	onDisk := settings
	if legacy := a.getLegacyAPIKey(); legacy != "" {
		// Keyring unavailable on this machine: keep the stored key so a
		// settings round-trip doesn't wipe it.
		onDisk.AIAPIKey = legacy
	}
	return a.writeSettingsFile(onDisk)
}

// ── API key (OS keyring) ─────────────────────────────────────────────────────

const (
	keyringService = "draftline"
	keyringAccount = "ai_api_key"
)

// SetAPIKey stores the key in the OS keyring. If no keyring is available it
// falls back to settings.json (user-only permissions) rather than losing it.
func (a *App) SetAPIKey(key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("API key is empty")
	}
	if err := keyring.Set(keyringService, keyringAccount, key); err != nil {
		log.Printf("keyring unavailable (%v); storing key in settings.json", err)
		a.setLegacyAPIKey(key)
		a.setCachedKey(key)
		onDisk := a.getSettings()
		onDisk.AIAPIKey = key
		return a.writeSettingsFile(onDisk)
	}
	a.setLegacyAPIKey("")
	a.setCachedKey(key)
	// Make sure no plaintext copy lingers on disk.
	onDisk := a.getSettings()
	onDisk.AIAPIKey = ""
	return a.writeSettingsFile(onDisk)
}

// HasAPIKey reports whether a key is stored, without revealing it.
func (a *App) HasAPIKey() bool {
	return a.getAPIKey() != ""
}

// ClearAPIKey removes the stored key from the keyring and settings.json.
// Local copies (legacy plaintext, cache) are cleared even if the keyring
// operation fails — on keyring-less machines settings.json is the only place
// the key lives, and clearing must not be blocked by an "unavailable" error.
func (a *App) ClearAPIKey() error {
	delErr := keyring.Delete(keyringService, keyringAccount)
	if delErr != nil && errors.Is(delErr, keyring.ErrNotFound) {
		delErr = nil
	}
	a.setLegacyAPIKey("")
	a.setCachedKey("")
	onDisk := a.getSettings()
	onDisk.AIAPIKey = ""
	if err := a.writeSettingsFile(onDisk); err != nil {
		return err
	}
	return delErr
}

func (a *App) setCachedKey(key string) {
	a.apiKeyMu.Lock()
	a.apiKeyCache = key
	a.apiKeyLoaded = true
	a.apiKeyMu.Unlock()
}

// getAPIKey returns the stored key (keyring first, then legacy fallback),
// caching the keyring read.
func (a *App) getAPIKey() string {
	a.apiKeyMu.Lock()
	defer a.apiKeyMu.Unlock()
	if a.apiKeyLoaded {
		return a.apiKeyCache
	}
	v, err := keyring.Get(keyringService, keyringAccount)
	switch {
	case err == nil:
		a.apiKeyCache = v
	case errors.Is(err, keyring.ErrNotFound):
		a.apiKeyCache = a.legacyAPIKey
	default:
		// Transient keyring failure: fall back without caching so the next
		// call retries the keyring.
		return a.legacyAPIKey
	}
	a.apiKeyLoaded = true
	return a.apiKeyCache
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

func (a *App) recentProjectsPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = "."
	}
	dir := filepath.Join(configDir, "draftline")
	_ = os.MkdirAll(dir, 0700)
	// Best-effort: tighten an existing directory created by an older version.
	_ = os.Chmod(dir, 0700)
	path := filepath.Join(dir, "recent_projects.json")
	// Best-effort: tighten an existing recent-projects file written with
	// looser permissions by an older version. Ignore errors.
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

// TestLocalAI makes a quick connectivity check to a local OpenAI-compatible endpoint.
func (a *App) TestLocalAI(endpoint string) types.AIRewriteResult {
	if endpoint == "" {
		return types.AIRewriteResult{Error: "No endpoint URL configured"}
	}
	url := strings.TrimRight(endpoint, "/") + "/models"
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return types.AIRewriteResult{Error: "Could not connect: " + err.Error()}
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		return types.AIRewriteResult{Error: fmt.Sprintf("Server returned %d", resp.StatusCode)}
	}
	return types.AIRewriteResult{Result: "Connected"}
}

// RewriteText sends the HTML chapter content to the configured AI provider.
// mode: "line_edit" | "copy_edit" | "expand" | "smooth"
// styleOptionsJson: JSON string of types.WritingStyleOptions (for expand/smooth modes)
func (a *App) RewriteText(html string, mode string, styleOptionsJson string) types.AIRewriteResult {
	s := a.getSettings()
	logging.AI("========== RewriteText START ==========")
	logging.AI("mode=%s ai_mode=%s provider=%s", mode, s.AIMode, s.AIProvider)
	logging.AIContent("INPUT_HTML", html)

	if !s.AIEnabled {
		logging.AI("ERROR: AI features disabled")
		return types.AIRewriteResult{Error: "AI features are disabled — enable them in App Settings"}
	}
	if mode == "" {
		mode = "line_edit"
	}

	// Parse style options if provided
	var styleOpts *types.WritingStyleOptions
	if styleOptionsJson != "" {
		styleOpts = &types.WritingStyleOptions{}
		if err := json.Unmarshal([]byte(styleOptionsJson), styleOpts); err != nil {
			styleOpts = nil // Ignore invalid JSON, use defaults
		}
	}

	// Diff format: for targeted per-paragraph edits, ask the model to return ONLY
	// changed paragraphs. This cuts output tokens by ~80% for typical chapters.
	useDiffFormat := mode == "line_edit" || mode == "copy_edit" || mode == "smooth"

	system := ai.BuildSystemPrompt(mode, s.ProseGuide, styleOpts)
	var userMsg string

	switch {
	case useDiffFormat:
		// Append diff-format output instruction
		system += "\n\nCRITICAL OUTPUT FORMAT: Each input paragraph is prefixed §N§ where N is its 1-based index.\nReturn ONLY paragraphs you change, one per line:\n§N§<p>revised text</p>\nOmit unchanged paragraphs entirely. If nothing needs changing: §NONE§"
		userMsg = ai.BuildDiffUserMsg(html)
	default:
		userMsg = "Rewrite the following, returning only the rewritten HTML paragraphs:\n\n" + html
	}

	res := a.dispatchAI(system, userMsg, rewriteRequestProfile(mode))

	// If the model returned diff format, reconstruct full HTML before handing back.
	if res.Error == "" && useDiffFormat {
		res.Result = ai.ApplyDiffResponse(html, res.Result)
	}

	// Log the result
	if res.Error != "" {
		logging.AI("ERROR: %s", res.Error)
	} else {
		logging.AIContent("OUTPUT_RESULT", res.Result)
	}
	logging.AI("========== RewriteText END ==========")
	return res
}

// RewriteTextCustom applies a custom user prompt to rewrite text.
func (a *App) RewriteTextCustom(text string, prompt string) types.AIRewriteResult {
	s := a.getSettings()
	logging.AI("========== RewriteTextCustom START ==========")
	logging.AI("prompt=%s ai_mode=%s provider=%s", prompt, s.AIMode, s.AIProvider)
	logging.AIContent("INPUT_TEXT", text)

	if !s.AIEnabled {
		logging.AI("ERROR: AI features disabled")
		return types.AIRewriteResult{Error: "AI features are disabled — enable them in App Settings"}
	}
	if prompt == "" {
		logging.AI("ERROR: no prompt provided")
		return types.AIRewriteResult{Error: "no prompt provided"}
	}

	system := `You are a skilled fiction editor helping an author revise their manuscript.
Follow the user's instruction precisely. Preserve the author's voice and style.
If the input is HTML, preserve the HTML structure (<p>, <em>, <strong>, etc).

ABSOLUTELY FORBIDDEN — AI TELL CONSTRUCTIONS:
1. Em-dash appositive definitions: NEVER write "[quality] — that [particular/specific/certain] [noun] of someone who [explanation]". Show the quality, never name and define it in the same breath.
2. Gerund-plus-abstract-noun behavior labeling: NEVER write "performing normalcy", "performing grief", "performing calm", or any "[gerund] + [abstract social/emotional noun]" construction. Let behavior speak for itself.
3. Clinical precision words that no narrator actually thinks in: "over-relaxation", "micro-expression", "hyperawareness", "hypervigilance". Replace with visceral physical observation.
4. Meta-pattern references: NEVER write "the thing it did", "the way she always", "that look he had". Show the specific instance, not the pattern.
5. Narrator taxonomy and cataloguing: NEVER have the narrator classify, catalogue, or taxonomize behavior with fake academic precision. BANNED: "a particular subspecies", "catalogued privately", "a specific category of". Narrators notice things, they do not file them.
6. Triple synonym stacking: NEVER stack near-synonyms in twos or threes for emphasis. BANNED: "simply, entirely, thoroughly", "wordless and mutual and instinctive". Pick the single strongest word and trust it.
7. Similes that overstay: End comparisons when the image lands. NEVER extend a simile past the point where the meaning is clear. If you are still explaining the comparison after the first clause, cut it.
8. Announcing literary references as shortcuts: BANNED: "contained multitudes", "the whole of her", "more than she let on". Show the contradiction directly, never name it.
9. The indifferent world pan-out: NEVER end a scene or paragraph by pulling back to an outside world that is unaware of or indifferent to the characters. This is an AI default scene-closing move and is always cut.

CRITICAL: Return ONLY the revised text content. Do not include any instructions, explanations, system prompts, or meta-commentary. Output the prose only.`

	userMsg := fmt.Sprintf("Instruction: %s\n\nText to revise:\n%s", prompt, text)

	res := a.dispatchAI(system, userMsg, standardAIRequest)

	// Log the result
	if res.Error != "" {
		logging.AI("ERROR: %s", res.Error)
	} else {
		logging.AIContent("OUTPUT_RESULT", res.Result)
	}
	logging.AI("========== RewriteTextCustom END ==========")
	return res
}

// GenerateInlineContent generates new content based on context and instruction
func (a *App) GenerateInlineContent(req types.InlineGenerateRequest) types.AIRewriteResult {
	s := a.getSettings()
	if !s.AIEnabled {
		return types.AIRewriteResult{Error: "AI features are disabled — enable them in App Settings"}
	}
	if req.Instruction == "" {
		return types.AIRewriteResult{Error: "no instruction provided"}
	}

	// Build character context
	charContext := ""
	if len(req.Characters) > 0 {
		charContext = fmt.Sprintf("\n\ntypes.Characters in this story: %s", strings.Join(req.Characters, ", "))
	}

	// Build style guide section if configured
	styleBlock := ""
	if s.ProseGuide != "" {
		styleBlock = "\n\nSTYLE GUIDE — match the rhythm, vocabulary, and voice of these examples:\n---\n" + s.ProseGuide + "\n---"
	}

	banned := "BANNED words and phrases: tapestry, testament, navigate, delve, underscore, myriad, realm, crucial, pivotal, journey, beacon, vibrant, game-changer"

	system := fmt.Sprintf(`You are a skilled fiction author helping write a manuscript.
Generate new content that seamlessly fits between the existing prose.
Match the voice, style, tense, and POV of the surrounding text.
Return ONLY the new content as HTML paragraphs (<p>...</p>).
Do not include any explanations, just the prose.

%s

ABSOLUTELY FORBIDDEN — AI TELL CONSTRUCTIONS:
1. Em-dash appositive definitions: NEVER write "[quality] — that [particular/specific/certain] [noun] of someone who [explanation]". Show the quality, never name and define it in the same breath.
2. Gerund-plus-abstract-noun behavior labeling: NEVER write "performing normalcy", "performing grief", "performing calm", or any "[gerund] + [abstract social/emotional noun]" construction. Let behavior speak for itself.
3. Clinical precision words that no narrator actually thinks in: "over-relaxation", "micro-expression", "hyperawareness", "hypervigilance". Replace with visceral physical observation.
4. Meta-pattern references: NEVER write "the thing it did", "the way she always", "that look he had". Show the specific instance, not the pattern.
5. Narrator taxonomy and cataloguing: NEVER have the narrator classify, catalogue, or taxonomize behavior with fake academic precision. BANNED: "a particular subspecies", "catalogued privately", "a specific category of". Narrators notice things, they do not file them.
6. Triple synonym stacking: NEVER stack near-synonyms in twos or threes for emphasis. BANNED: "simply, entirely, thoroughly", "wordless and mutual and instinctive". Pick the single strongest word and trust it.
7. Similes that overstay: End comparisons when the image lands. NEVER extend a simile past the point where the meaning is clear. If you are still explaining the comparison after the first clause, cut it.
8. Announcing literary references as shortcuts: BANNED: "contained multitudes", "the whole of her", "more than she let on". Show the contradiction directly, never name it.
9. The indifferent world pan-out: NEVER end a scene or paragraph by pulling back to an outside world that is unaware of or indifferent to the characters. This is an AI default scene-closing move and is always cut.

STYLE RULE:
11. Break grammar rules intentionally where rhythm demands it. Fragments are allowed. Sentences can start with And or But. Comma splices are permitted for pacing. Grammatical correctness is not the goal. The sentence is the goal.
%s
CRITICAL: Return ONLY the new prose content as HTML paragraphs. Do not include any instructions, explanations, system prompts, or meta-commentary. Output raw HTML only.%s`, banned, styleBlock, charContext)

	// Build the user message with context
	var contextParts []string
	if req.ChapterTitle != "" {
		contextParts = append(contextParts, fmt.Sprintf("Chapter: %s", req.ChapterTitle))
	}
	if req.BeforeContext != "" {
		contextParts = append(contextParts, fmt.Sprintf("TEXT BEFORE:\n%s", req.BeforeContext))
	}
	if req.AfterContext != "" {
		contextParts = append(contextParts, fmt.Sprintf("TEXT AFTER:\n%s", req.AfterContext))
	}
	contextParts = append(contextParts, fmt.Sprintf("INSTRUCTION: %s", req.Instruction))
	contextParts = append(contextParts, "Generate the new content to insert between the before and after text:")

	userMsg := strings.Join(contextParts, "\n\n")

	return a.dispatchAI(system, userMsg, standardAIRequest)
}

// CheckClaudeCode checks whether the claude CLI is installed and authenticated.
// It checks both the system PATH and Draftline's bundled install location.
func (a *App) CheckClaudeCode() types.ClaudeCodeStatus {
	npmAvailable := resolveNpmBin() != ""

	claudePath := resolveClaudeBin()
	if claudePath == "" {
		return types.ClaudeCodeStatus{NpmAvailable: npmAvailable}
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
	return types.ClaudeCodeStatus{Installed: true, Authenticated: authenticated, NpmAvailable: npmAvailable, Version: version}
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
		platform.HideWindow(cmd)
		_ = cmd.Start()
		_ = cmd.Wait()
		runtime.EventsEmit(a.ctx, "claude:auth_complete", nil)
	}()
}

// CheckCodexCLI checks whether the OpenAI Codex CLI is installed and
// authenticated (ChatGPT-account mode). Reuses the ClaudeCodeStatus shape.
func (a *App) CheckCodexCLI() types.ClaudeCodeStatus {
	npmAvailable := resolveNpmBin() != ""

	codexPath := resolveCodexBin()
	if codexPath == "" {
		return types.ClaudeCodeStatus{NpmAvailable: npmAvailable}
	}
	out, err := codexExec(context.Background(), codexPath, "--version").Output()
	version := ""
	if err == nil {
		version = strings.TrimSpace(string(out))
	}
	home, _ := os.UserHomeDir()
	authenticated := false
	if _, err := os.Stat(filepath.Join(home, ".codex", "auth.json")); err == nil {
		authenticated = true
	}
	return types.ClaudeCodeStatus{Installed: true, Authenticated: authenticated, NpmAvailable: npmAvailable, Version: version}
}

// OpenCodexAuth runs "codex login" as a hidden background process. It opens
// the user's browser for the ChatGPT OAuth flow — the in-app intercept for
// what would otherwise require running /login in a terminal. When the process
// exits it emits "codex:auth_complete".
func (a *App) OpenCodexAuth() {
	codexPath := resolveCodexBin()
	if codexPath == "" {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		cmd := codexExec(ctx, codexPath, "login")
		platform.HideWindow(cmd)
		_ = cmd.Start()
		_ = cmd.Wait()
		runtime.EventsEmit(a.ctx, "codex:auth_complete", nil)
	}()
}

// codexExecArgs builds the constant argv for a non-interactive codex run.
// The prompt is NEVER an argv element — it is piped via stdin ("-"), the same
// injection-safe pattern as the Claude CLI. lastMessageFile receives the final
// assistant message (--output-last-message) so the result needn't be parsed
// out of the progress stream.
func codexExecArgs(model, lastMessageFile string, lightweight bool) []string {
	args := []string{
		"exec",
		"--skip-git-repo-check", // temp workdir is not a git repo
		"--ignore-user-config",
		"--ignore-rules",
		"--strict-config",
		"--sandbox", "read-only",
		"--ask-for-approval", "never",
		"--config", `shell_environment_policy.inherit="none"`,
		"--ephemeral",
		"--color", "never",
		"--json",
		"--output-last-message", lastMessageFile,
	}
	// Draftline needs text transformation, not an agent. Disable every stable
	// capability-bearing feature exposed by the pinned Codex CLI so manuscript
	// content cannot induce shell, file, browser, app, or sub-agent activity.
	for _, feature := range []string{
		"shell_tool", "unified_exec", "view_image", "apps", "browser_use",
		"computer_use", "image_generation", "multi_agent", "skill_search", "hooks",
	} {
		args = append(args, "--disable", feature)
	}
	if lightweight {
		args = append(args, "--config", `model_reasoning_effort="low"`)
	}
	if model != "" {
		args = append(args, "-m", model)
	}
	return append(args, "-")
}

func claudeCodeExecArgs(model string, lightweight bool) []string {
	args := []string{
		"-p",
		"--output-format", "stream-json",
		"--verbose",
		"--max-turns", "1",
		"--model", model,
		"--safe-mode",
		"--disable-slash-commands",
		"--strict-mcp-config",
		"--mcp-config", `{}`,
		"--tools", "",
		"--permission-mode", "dontAsk",
		"--no-session-persistence",
	}
	if lightweight {
		args = append(args, "--effort", "low")
	}
	return args
}

func claudeFailureMessage(stderr string, runErr error) string {
	lower := strings.ToLower(stderr)
	switch {
	case strings.Contains(lower, "401"), strings.Contains(lower, "403"), strings.Contains(lower, "auth"):
		return "Claude authentication failed. Sign in again from Settings › AI Studio."
	case strings.Contains(lower, "rate limit"), strings.Contains(lower, "429"):
		return "Claude rate limit reached. Wait a moment and try again."
	case strings.Contains(lower, "model") && (strings.Contains(lower, "not found") || strings.Contains(lower, "not supported")):
		return "Claude rejected the selected model. Check the model setting and try again."
	case runErr != nil:
		return "Claude request failed (" + runErr.Error() + "). Check your Claude sign-in and connection."
	default:
		return "Claude request failed. Check your Claude sign-in and connection."
	}
}

// codexModelOverride prevents the shared legacy ai_model setting from leaking
// a provider-specific model into Codex when the AI Studio quick-switcher is
// used. Unknown values are left alone so future OpenAI model IDs still work.
func codexModelOverride(model string) string {
	trimmed := strings.TrimSpace(model)
	lower := strings.ToLower(trimmed)
	for _, prefix := range []string{"claude", "gemini", "grok", "llama", "mistral"} {
		if strings.HasPrefix(lower, prefix) {
			return ""
		}
	}
	return trimmed
}

// resolveCodexLightweightModel uses the CLI's own cached availability list so
// fast editing never hard-fails on installations that predate a model name.
// With no known lightweight model, an empty override safely uses the CLI's
// account default (still at low reasoning effort).
func resolveCodexLightweightModel(home string) string {
	data, err := os.ReadFile(filepath.Join(home, ".codex", "models_cache.json"))
	if err != nil {
		return ""
	}
	return selectCodexLightweightModel(data)
}

func selectCodexLightweightModel(data []byte) string {
	var cache struct {
		Models []struct {
			Slug string `json:"slug"`
		} `json:"models"`
	}
	if json.Unmarshal(data, &cache) != nil {
		return ""
	}
	available := make(map[string]bool, len(cache.Models))
	for _, model := range cache.Models {
		available[model.Slug] = true
	}
	for _, preferred := range []string{"gpt-5.6-luna", "gpt-5.4-mini", "gpt-reserve"} {
		if available[preferred] {
			return preferred
		}
	}
	return ""
}

func codexFailureMessage(stderr string, runErr error) string {
	lines := strings.Split(stderr, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		// API failures commonly end in a JSON object containing detail/message.
		if start := strings.Index(line, "{"); start >= 0 {
			var payload map[string]any
			if json.Unmarshal([]byte(line[start:]), &payload) == nil {
				for _, key := range []string{"detail", "message", "error"} {
					if message, ok := payload[key].(string); ok && strings.TrimSpace(message) != "" {
						return "Codex request failed: " + truncateCodexError(message)
					}
				}
			}
		}

		lower := strings.ToLower(line)
		actionable := strings.HasPrefix(lower, "error:") || strings.HasPrefix(lower, "error ") ||
			strings.Contains(lower, "unexpected status") || strings.Contains(lower, "not supported") ||
			strings.Contains(lower, "invalid model") || strings.Contains(lower, "authentication failed") ||
			strings.Contains(lower, "rate limit")
		if !actionable {
			continue
		}
		if idx := strings.Index(lower, "error:"); idx >= 0 {
			line = strings.TrimSpace(line[idx+len("error:"):])
		}
		return "Codex request failed: " + truncateCodexError(line)
	}

	if runErr != nil {
		return "Codex request failed (" + runErr.Error() + "). Check your Codex model setting and ChatGPT sign-in."
	}
	return "Codex request failed. Check your Codex model setting and ChatGPT sign-in."
}

func truncateCodexError(message string) string {
	message = strings.TrimSpace(message)
	if len(message) > 500 {
		return message[:500] + "…"
	}
	return message
}

// callCodexCLI handles the "codex" AI mode: prose rewrites through the OpenAI
// Codex CLI using the user's ChatGPT account.
func (a *App) callCodexCLI(ctx context.Context, system, userMsg string, profile aiRequestProfile) types.AIRewriteResult {
	path := resolveCodexBin()
	if path == "" {
		return types.AIRewriteResult{Error: "Codex CLI is not installed — open Settings › AI Studio to set it up"}
	}

	// Isolated home: real auth.json so the ChatGPT session works, but none of
	// the user's config.toml (MCP servers etc. could hang the subprocess).
	tempHome, err := os.MkdirTemp("", "draftline-codex-*")
	if err != nil {
		return types.AIRewriteResult{Error: "cannot create temp dir: " + err.Error()}
	}
	defer func() { _ = os.RemoveAll(tempHome) }()

	realHome, _ := os.UserHomeDir()
	codexDir := filepath.Join(tempHome, ".codex")
	_ = os.MkdirAll(codexDir, 0755)
	for _, cachedFile := range []string{"auth.json", "models_cache.json"} {
		if data, e2 := os.ReadFile(filepath.Join(realHome, ".codex", cachedFile)); e2 == nil {
			_ = os.WriteFile(filepath.Join(codexDir, cachedFile), data, 0600)
		}
	}

	lastMsg := filepath.Join(tempHome, "last-message.txt")
	model := codexModelOverride(a.getSettings().AIModel)
	if profile.lightweight {
		model = resolveCodexLightweightModel(realHome)
		if model != "" {
			runtime.EventsEmit(a.ctx, "ai:log", "Fast edit model: "+model)
		}
	}
	// Lightweight editing prefers a model advertised by this CLI installation;
	// otherwise no hardcoded override is used, avoiding model-churn failures.
	cmd := codexExec(ctx, path, codexExecArgs(model, lastMsg, profile.lightweight)...)
	cmd.Stdin = strings.NewReader(system + "\n\n" + userMsg)
	cmd.Dir = tempHome

	nodeDir := nodeInstallBinDir()
	baseEnv := os.Environ()
	filteredEnv := make([]string, 0, len(baseEnv)+5)
	for _, e := range baseEnv {
		key, _, _ := strings.Cut(e, "=")
		switch strings.ToUpper(key) {
		case "PATH", "HOME", "USERPROFILE", "HOMEDRIVE", "HOMEPATH", "CODEX_HOME":
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
		"CODEX_HOME="+codexDir,
	)
	platform.HideWindow(cmd)

	stdoutPipe, stdoutPipeErr := cmd.StdoutPipe()
	stderrPipe, stderrPipeErr := cmd.StderrPipe()

	runtime.EventsEmit(a.ctx, "ai:log", "Starting: "+filepath.Base(cmd.Path))
	if err := cmd.Start(); err != nil {
		return types.AIRewriteResult{Error: err.Error()}
	}
	runtime.EventsEmit(a.ctx, "ai:log", "Process started…")

	var wg sync.WaitGroup
	var stderrBuf strings.Builder
	for _, p := range []struct {
		pipe io.ReadCloser
		err  error
		errs bool
	}{{stdoutPipe, stdoutPipeErr, false}, {stderrPipe, stderrPipeErr, true}} {
		if p.err != nil {
			continue
		}
		wg.Add(1)
		go func(r io.ReadCloser, isErr bool) {
			defer wg.Done()
			drainLines(r, func(line string) {
				if isErr {
					stderrBuf.WriteString(line + "\n")
				}
			})
		}(p.pipe, p.errs)
	}

	runErr := cmd.Wait()
	wg.Wait()

	if errors.Is(ctx.Err(), context.Canceled) {
		return types.AIRewriteResult{Error: "cancelled"}
	}
	if runErr != nil {
		return types.AIRewriteResult{Error: codexFailureMessage(stderrBuf.String(), runErr)}
	}

	result, err := os.ReadFile(lastMsg)
	if err != nil || len(strings.TrimSpace(string(result))) == 0 {
		return types.AIRewriteResult{Error: "Codex produced no output — check that you are signed in (Settings › AI Studio)"}
	}
	return types.AIRewriteResult{Result: strings.TrimSpace(string(result))}
}

// maxDrainLine caps how many bytes of a single output line are retained in
// memory for logging/preview. Longer lines are truncated to this size but the
// remainder is still consumed off the pipe.
const maxDrainLine = 2 * 1024 * 1024

// drainLines reads r to EOF, invoking onLine once per newline-delimited line
// (with the trailing CR/LF stripped, matching bufio.Scanner.Text semantics).
//
// Unlike bufio.Scanner, it never stops on an over-long line: the retained
// portion of each line is capped at maxDrainLine bytes and the rest of that
// line is consumed and discarded. This guarantees the reader is drained for
// the full lifetime of the pipe so the child process can never block writing
// to a full stdout/stderr pipe (which would otherwise deadlock cmd.Wait()).
func drainLines(r io.Reader, onLine func(string)) {
	br := bufio.NewReaderSize(r, 64*1024)
	var b strings.Builder
	pending := false // some bytes of the current line have been seen
	for {
		chunk, err := br.ReadString('\n')
		if len(chunk) > 0 {
			pending = true
			// Strip a single trailing LF (and preceding CR) to match Scanner.
			end := len(chunk)
			hasNL := chunk[end-1] == '\n'
			if hasNL {
				end--
				if end > 0 && chunk[end-1] == '\r' {
					end--
				}
			}
			if remaining := maxDrainLine - b.Len(); remaining > 0 {
				if end > remaining {
					b.WriteString(chunk[:remaining])
				} else {
					b.WriteString(chunk[:end])
				}
			}
			if hasNL {
				onLine(b.String())
				b.Reset()
				pending = false
			}
		}
		if err != nil {
			// EOF or read error: flush any trailing line without a newline.
			if pending {
				onLine(b.String())
			}
			return
		}
	}
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
		if errors.Is(ctx.Err(), context.Canceled) {
			return "", fmt.Errorf("cancelled")
		}
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		body, _ := readAIResponseBody(resp.Body)
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
			return "", fmt.Errorf("cancelled")
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
		if errors.Is(ctx.Err(), context.Canceled) {
			return "", fmt.Errorf("cancelled")
		}
		return "", err
	}

	return strings.TrimSpace(sb.String()), nil
}

// dispatchAI routes the AI request to the appropriate provider based on settings.
// This is the central dispatch point for all AI modes (claudecode, local, api)
// and the single place that claims the AI-request slot: concurrent requests
// get a clean "busy" error instead of corrupting each other's cancel handles.
func (a *App) dispatchAI(system, userMsg string, profile aiRequestProfile) types.AIRewriteResult {
	ctx, release, ok := a.acquireAI()
	if !ok {
		return types.AIRewriteResult{Error: "an AI request is already in progress — wait for it to finish or cancel it"}
	}
	defer release()

	s := a.getSettings()
	switch s.AIMode {
	case "claudecode":
		return a.callClaudeCode(ctx, system, userMsg, profile)
	case "codex":
		return a.callCodexCLI(ctx, system, userMsg, profile)
	case "local":
		if s.AILocalEndpoint == "" {
			return types.AIRewriteResult{Error: "no local endpoint configured — set it in App Settings"}
		}
		return a.callLocalAI(ctx, system, userMsg)
	default: // "api"
		if s.AIProvider == "" || a.getAPIKey() == "" {
			return types.AIRewriteResult{Error: "no API provider configured — add your key in App Settings"}
		}
		switch s.AIProvider {
		case "claude":
			return a.callClaude(ctx, system, userMsg, profile)
		case "openai":
			return a.callOpenAI(ctx, system, userMsg, profile)
		case "gemini":
			return a.callGemini(ctx, system, userMsg, profile)
		case "grok":
			return a.callGrok(ctx, system, userMsg, profile)
		default:
			return types.AIRewriteResult{Error: "unknown provider: " + s.AIProvider}
		}
	}
}

// acquireAI claims the single AI-request slot. On success it returns a
// 180-second timeout context, an idempotent release function that must be
// deferred by the caller, and ok=true. If another AI request is already in
// flight it returns ok=false — callers surface a "busy" error instead of
// interleaving cancel handles and token streams.
//
// CancelRewrite cancels the live context but does NOT free the slot; only the
// owning request's release does. release is guarded by a generation counter so
// a stale (double) release can never free or disturb a successor's slot.
func (a *App) acquireAI() (ctx context.Context, release func(), ok bool) {
	a.cancelMu.Lock()
	defer a.cancelMu.Unlock()
	if a.aiBusy {
		return nil, nil, false
	}
	a.aiBusy = true
	a.aiGen++
	gen := a.aiGen

	parent := a.ctx
	if parent == nil {
		parent = context.Background() // tests construct App without a Wails context
	}
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(parent, 180*time.Second)
	a.cancelRewrite = cancel

	release = func() {
		cancel() // idempotent; only affects this request's own context
		a.cancelMu.Lock()
		defer a.cancelMu.Unlock()
		if a.aiGen == gen {
			a.aiBusy = false
			a.cancelRewrite = nil
		}
	}
	return ctx, release, true
}

// resolveAIModel returns the configured model name, falling back to the
// provider's default when none is set.
func (a *App) resolveAIModel(defaultModel string) string {
	if model := a.getSettings().AIModel; model != "" {
		return model
	}
	return defaultModel
}

func (a *App) resolveRequestModel(defaultModel, lightweightModel string, profile aiRequestProfile) string {
	if profile.lightweight && lightweightModel != "" {
		return lightweightModel
	}
	return a.resolveAIModel(defaultModel)
}

// callClaudeCode handles the "claudecode" AI mode. If the credentials file
// contains a direct API key it calls the Anthropic API with streaming directly —
// faster and more reliable than the CLI subprocess. Falls back to the CLI when
// only OAuth credentials are present.
func (a *App) callClaudeCode(ctx context.Context, system, userMsg string, profile aiRequestProfile) types.AIRewriteResult {
	if apiKey := readClaudeAPIKey(); apiKey != "" {
		model := a.resolveRequestModel("claude-sonnet-4-6", "claude-haiku-4-5-20251001", profile)
		if profile.lightweight {
			runtime.EventsEmit(a.ctx, "ai:log", "Fast edit model: Claude Haiku")
		}
		runtime.EventsEmit(a.ctx, "ai:log", "Connecting to Anthropic API…")
		result, err := a.streamAnthropic(ctx, apiKey, model, system, userMsg)
		if err != nil {
			return types.AIRewriteResult{Error: err.Error()}
		}
		return types.AIRewriteResult{Result: result}
	}
	return a.callClaudeCodeCLI(ctx, system, userMsg, profile)
}

// callClaudeCodeCLI invokes the Claude Code CLI as a subprocess.
// Used as a fallback when only OAuth credentials are available (no raw API key).
func (a *App) callClaudeCodeCLI(ctx context.Context, system, userMsg string, profile aiRequestProfile) types.AIRewriteResult {
	path := resolveClaudeBin()
	if path == "" {
		return types.AIRewriteResult{Error: "Claude Code is not installed — open Settings › AI Studio to set it up"}
	}
	model := a.resolveRequestModel("claude-sonnet-4-6", "claude-haiku-4-5-20251001", profile)
	if profile.lightweight {
		runtime.EventsEmit(a.ctx, "ai:log", "Fast edit model: Claude Haiku")
	}

	// Create an isolated home directory: real credentials so auth works, but no
	// MCP server config. MCP servers are started between init and the first API
	// call — if any hang, the subprocess hangs silently for the full timeout.
	tempHome, err := os.MkdirTemp("", "draftline-claude-*")
	if err != nil {
		return types.AIRewriteResult{Error: "cannot create temp dir: " + err.Error()}
	}
	defer func() { _ = os.RemoveAll(tempHome) }()

	realHome, _ := os.UserHomeDir()
	claudeDir := filepath.Join(tempHome, ".claude")
	_ = os.MkdirAll(claudeDir, 0755)
	if data, e2 := os.ReadFile(filepath.Join(realHome, ".claude", ".credentials.json")); e2 == nil {
		_ = os.WriteFile(filepath.Join(claudeDir, ".credentials.json"), data, 0600)
	}
	_ = os.WriteFile(filepath.Join(claudeDir, "settings.json"),
		[]byte(`{"mcpServers":{}}`), 0644)

	// The prompt is piped via stdin rather than passed as an argv element: in
	// print mode (-p) the CLI reads piped stdin as the prompt. This keeps
	// manuscript-derived content out of every exec path (notably the cmd.exe
	// .cmd-shim fallback, where argv metacharacters would be interpreted) and
	// sidesteps the ~32K Windows command-line length limit.
	fullPrompt := system + "\n\n" + userMsg
	claudeArgs := claudeCodeExecArgs(model, profile.lightweight)
	cmd := claudeExec(ctx, path, claudeArgs...)
	cmd.Stdin = strings.NewReader(fullPrompt)
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
		return types.AIRewriteResult{Error: err.Error()}
	}
	runtime.EventsEmit(a.ctx, "ai:log", "Process started…")

	var wg sync.WaitGroup
	var resultText string
	var stderrBuf strings.Builder

	if stdoutPipeErr == nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			streamed := false
			drainLines(stdoutPipe, func(line string) {
				// Raw stream-json lines contain prompt/manuscript-derived
				// content. Keep them only in the opt-in debug log; never emit
				// them to the ai:log runtime event (which can appear in
				// screenshots). Parity with the Codex path.
				logging.AIContent("CLAUDE_STREAM", line)
				var obj map[string]any
				if json.Unmarshal([]byte(line), &obj) == nil {
					switch obj["type"] {
					case "assistant", "content_block_delta", "message_delta":
						if !streamed {
							streamed = true
							runtime.EventsEmit(a.ctx, "ai:log", "Streaming…")
						}
					case "result":
						if r, ok := obj["result"].(string); ok {
							resultText = r
						}
						if isErr, _ := obj["is_error"].(bool); isErr {
							if msg, ok := obj["result"].(string); ok {
								stderrBuf.WriteString(msg)
							}
							runtime.EventsEmit(a.ctx, "ai:log", "Error")
						} else {
							runtime.EventsEmit(a.ctx, "ai:log", "Done")
						}
					}
				}
			})
		}()
	}
	if stderrPipeErr == nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			drainLines(stderrPipe, func(line string) {
				stderrBuf.WriteString(line + "\n")
				// stderr may echo prompt/manuscript-derived content; keep it in
				// the opt-in debug log only, not the screenshot-visible ai:log.
				logging.AIContent("CLAUDE_STDERR", line)
			})
		}()
	}

	runErr := cmd.Wait()
	wg.Wait()

	if runErr != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			return types.AIRewriteResult{Error: "cancelled"}
		}
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return types.AIRewriteResult{Error: "timed out — try a shorter chapter or check your Claude Code connection"}
		}
		return types.AIRewriteResult{Error: claudeFailureMessage(stderrBuf.String(), runErr)}
	}
	if strings.TrimSpace(resultText) == "" {
		return types.AIRewriteResult{Error: "Claude produced no output — check that you are signed in (Settings › AI Studio)"}
	}
	return types.AIRewriteResult{Result: strings.TrimSpace(resultText)}
}

// extractHTMLParagraphs returns each <p>...</p> element from HTML as a full tag string.

func (a *App) callLocalAI(ctx context.Context, system, userMsg string) types.AIRewriteResult {
	s := a.getSettings()
	model := s.AILocalModel
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
	endpoint := strings.TrimRight(s.AILocalEndpoint, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return types.AIRewriteResult{Error: err.Error()}
	}
	req.Header.Set("Content-Type", "application/json")
	// No client timeout: the request context's 180-second deadline governs,
	// and a shorter competing timeout would mask cancellation.
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			return types.AIRewriteResult{Error: "cancelled"}
		}
		return types.AIRewriteResult{Error: "local AI request failed: " + err.Error()}
	}
	defer func() { _ = resp.Body.Close() }()
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error struct{ Message string } `json:"error"`
	}
	body, err := readAIResponseBody(resp.Body)
	if err != nil {
		return types.AIRewriteResult{Error: err.Error()}
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return types.AIRewriteResult{Error: "failed to parse response: " + err.Error()}
	}
	if result.Error.Message != "" {
		return types.AIRewriteResult{Error: result.Error.Message}
	}
	if len(result.Choices) == 0 {
		return types.AIRewriteResult{Error: "empty response from local AI"}
	}
	return types.AIRewriteResult{Result: strings.TrimSpace(result.Choices[0].Message.Content)}
}

func (a *App) callClaude(ctx context.Context, system, userMsg string, profile aiRequestProfile) types.AIRewriteResult {
	model := a.resolveRequestModel("claude-sonnet-4-6", "claude-haiku-4-5-20251001", profile)
	runtime.EventsEmit(a.ctx, "ai:log", "Connecting to Claude API…")
	result, err := a.streamAnthropic(ctx, a.getAPIKey(), model, system, userMsg)
	if err != nil {
		return types.AIRewriteResult{Error: err.Error()}
	}
	return types.AIRewriteResult{Result: result}
}

func (a *App) callOpenAI(ctx context.Context, system, userMsg string, profile aiRequestProfile) types.AIRewriteResult {
	model := a.resolveRequestModel("gpt-4o", "gpt-4o-mini", profile)
	reqBody, _ := json.Marshal(map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": userMsg},
		},
	})
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(reqBody))
	if err != nil {
		return types.AIRewriteResult{Error: err.Error()}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.getAPIKey())

	// No client timeout: the request context's 180-second deadline governs.
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			return types.AIRewriteResult{Error: "cancelled"}
		}
		return types.AIRewriteResult{Error: err.Error()}
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readAIResponseBody(resp.Body)
	if err != nil {
		return types.AIRewriteResult{Error: err.Error()}
	}

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
		return types.AIRewriteResult{Error: "failed to parse OpenAI response"}
	}
	if result.Error != nil {
		return types.AIRewriteResult{Error: result.Error.Message}
	}
	if len(result.Choices) == 0 {
		return types.AIRewriteResult{Error: "empty response from OpenAI"}
	}
	return types.AIRewriteResult{Result: strings.TrimSpace(result.Choices[0].Message.Content)}
}

func (a *App) callGemini(ctx context.Context, system, userMsg string, profile aiRequestProfile) types.AIRewriteResult {
	model := a.resolveRequestModel("gemini-1.5-pro", "gemini-1.5-flash", profile)

	// Gemini API uses a different format
	reqBody, _ := json.Marshal(map[string]any{
		"contents": []map[string]any{
			{
				"role":  "user",
				"parts": []map[string]string{{"text": userMsg}},
			},
		},
		"systemInstruction": map[string]any{
			"parts": []map[string]string{{"text": system}},
		},
		"generationConfig": map[string]any{
			"temperature":     0.7,
			"maxOutputTokens": 8192,
		},
	})

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent", model)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return types.AIRewriteResult{Error: err.Error()}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", a.getAPIKey())

	// No client timeout: the request context's 180-second deadline governs.
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			return types.AIRewriteResult{Error: "cancelled"}
		}
		return types.AIRewriteResult{Error: "Failed to connect to Gemini API: " + err.Error()}
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readAIResponseBody(resp.Body)
	if err != nil {
		return types.AIRewriteResult{Error: err.Error()}
	}

	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return types.AIRewriteResult{Error: "Failed to parse Gemini response"}
	}
	if result.Error != nil {
		return types.AIRewriteResult{Error: result.Error.Message}
	}
	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return types.AIRewriteResult{Error: "Empty response from Gemini"}
	}
	return types.AIRewriteResult{Result: strings.TrimSpace(result.Candidates[0].Content.Parts[0].Text)}
}

func (a *App) callGrok(ctx context.Context, system, userMsg string, profile aiRequestProfile) types.AIRewriteResult {
	model := a.resolveRequestModel("grok-2", "", profile)

	// Grok uses OpenAI-compatible API format
	reqBody, _ := json.Marshal(map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": userMsg},
		},
	})

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.x.ai/v1/chat/completions", bytes.NewBuffer(reqBody))
	if err != nil {
		return types.AIRewriteResult{Error: err.Error()}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.getAPIKey())

	// No client timeout: the request context's 180-second deadline governs.
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			return types.AIRewriteResult{Error: "cancelled"}
		}
		return types.AIRewriteResult{Error: "Failed to connect to Grok API: " + err.Error()}
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readAIResponseBody(resp.Body)
	if err != nil {
		return types.AIRewriteResult{Error: err.Error()}
	}

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
		return types.AIRewriteResult{Error: "Failed to parse Grok response"}
	}
	if result.Error != nil {
		return types.AIRewriteResult{Error: result.Error.Message}
	}
	if len(result.Choices) == 0 {
		return types.AIRewriteResult{Error: "Empty response from Grok"}
	}
	return types.AIRewriteResult{Result: strings.TrimSpace(result.Choices[0].Message.Content)}
}

func (a *App) writeBook(b types.BookData, path string) types.SaveResult {
	result := book.Write(path, b, AppVersion)
	if result.Success {
		a.setCurrentFile(path)
	}
	return result
}
