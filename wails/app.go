package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"draftline/internal/ai"
	"draftline/internal/backup"
	"draftline/internal/export"
	"draftline/internal/indexing"
	"draftline/internal/logging"
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
const AppVersion = "0.21.02735"

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

	// analysisMu is the backend single-flight boundary shared by automatic
	// analysis and manual character/relationship rebuilds. Frontend guards are
	// insufficient because these are separate Wails entry points.
	analysisMu sync.Mutex

	// plugins is the plugin-platform host: discovered installs plus the
	// sidecar supervisor (see plugins.go).
	plugins *pluginHost

	// bookLock holds the per-book instance lock (see locking.go).
	bookLock bookLockState
	// device holds the cross-device claim (see devicelock.go).
	device deviceClaim

	// covers holds edition cover art (see cover.go). Image bytes are here and
	// not on types.BookData, which crosses the Wails bridge as JSON on every
	// autosave. Usable as a zero value.
	covers coverCache

	// recentThumbs holds start-screen cover art already read out of an
	// archive (see recentcover.go). Usable as a zero value.
	recentThumbs recentCoverCache

	// snapshots holds frozen manuscripts that have not reached the project
	// file yet (see snapshot.go). Frozen text is a whole novel and stays off
	// types.BookData for the same reason cover art does. Usable as a zero
	// value.
	snapshots snapshotCache

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

// getSettings returns a snapshot of the current settings that the caller owns
// outright. Cloned, not merely copied: a value copy of AppSettings shares its
// slice and map fields, and callers do edit those in place.
func (a *App) getSettings() types.AppSettings {
	a.stateMu.RLock()
	defer a.stateMu.RUnlock()
	return a.settings.Clone()
}

// setSettings replaces the in-memory settings under the state lock, taking its
// own copy so the caller cannot keep editing what is now shared state.
func (a *App) setSettings(s types.AppSettings) {
	a.stateMu.Lock()
	a.settings = s.Clone()
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

// leaveOpenProject is what every path that stops working on the open book has
// to do: forget the save target, and forget the bytes held for it.
//
// They have to move together. Cover art and frozen manuscripts are the two
// things the backend holds that are not on BookData, so they do not travel
// with the book across the bridge and nothing on screen would show that they
// are still there. A cover attached but not yet saved, or a manuscript frozen
// for an edition and not yet written, left in a cache while the author imports
// a DOCX or goes back to the launch screen, would be written into whatever
// book is saved next - another book's artwork, or another book's words, inside
// a project file, invisible, and carried forward by the editions passthrough
// on every save after that.
func (a *App) leaveOpenProject() {
	a.setCurrentFile("")
	a.covers.reset()
	a.snapshots.reset()
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
	budget, done, ok := a.beginAnalysis(book)
	if !ok {
		return types.IndexResult{Success: false, Error: analysisBusyMessage}
	}
	defer done()
	return indexing.IndexBookWithOptions(&book, indexing.AnalysisPoolOptions{Workers: budget.Workers, MaxInFlightBytes: budget.MaxInFlightBytes, MaxBatchBytes: budget.MaxBatchBytes})
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
	_, done, ok := a.beginAnalysis(book)
	if !ok {
		return types.RelationshipAnalysisResult{Success: false, Error: analysisBusyMessage}
	}
	defer done()
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
	// After setSettings: moving the legacy key reads and rewrites settings.
	a.setSettings(a.migrateAIProvidersOnce(raw))
	logging.SetEnabled(raw.AIDebugLogging)

	a.initPlugins(pluginDevDir(os.Args[1:]))

	// Best-effort per-user file associations (Windows HKCU; no-op elsewhere).
	// Off the startup path — registry writes must never delay first paint.
	go func() {
		if err := registerFileAssociations(); err != nil {
			log.Printf("file association registration failed: %v", err)
		}
	}()
}

// ── Export Functions ─────────────────────────────────────────────────────────
// Thin wrappers around export package. See internal/export/ for implementation.

// ExportEPUB exports the book to EPUB format.
func (a *App) ExportEPUB(book types.BookData, options types.EPUBOptions) types.ExportResult {
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

	source, err := a.exportSource(book, options.ExportOptions)
	if err != nil {
		return types.ExportResult{Success: false, Error: err.Error()}
	}
	return export.EPUB(path, source, options, a.exportCover(book, options.ExportOptions))
}

// ExportDOCX exports the book to DOCX format.
func (a *App) ExportDOCX(book types.BookData, options types.DOCXOptions) types.ExportResult {
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

	source, err := a.exportSource(book, options.ExportOptions)
	if err != nil {
		return types.ExportResult{Success: false, Error: err.Error()}
	}
	return export.DOCX(path, source, options)
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

	source, err := a.exportSource(book, options.ExportOptions)
	if err != nil {
		return types.ExportResult{Success: false, Error: err.Error()}
	}
	return export.PDF(path, source, options, a.exportCover(book, options.ExportOptions))
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

	source, err := a.exportSource(book, options.ExportOptions)
	if err != nil {
		return types.ExportResult{Success: false, Error: err.Error()}
	}
	return export.PrintPDF(path, source, options)
}

// RewriteText sends the HTML chapter content to the configured AI provider.
// mode: "line_edit" | "copy_edit" | "expand" | "smooth"
// styleOptionsJson: JSON string of types.WritingStyleOptions (for expand/smooth modes)
func (a *App) RewriteText(html string, mode string, styleOptionsJson string, providerMode string) types.AIRewriteResult {
	s := a.getSettings()
	logging.AI("========== RewriteText START ==========")
	logging.AI("mode=%s ai_mode=%s task_provider=%s provider=%s", mode, s.AIMode, providerMode, s.AIProvider)
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
		system += "\n\nCRITICAL OUTPUT FORMAT: Each input block is prefixed §N§ where N is its 1-based index.\nReturn ONLY blocks you change, one per line:\n§N§<p>revised text</p>\nOmit unchanged blocks entirely. Blocks that are not <p> (scene breaks <hr>, block quotes, code blocks, headings, lists) are structure: omit them, or return them with their exact wrapper tags — never as <p>. If nothing needs changing: §NONE§"
		userMsg = ai.BuildDiffUserMsg(html)
	default:
		userMsg = "Rewrite the following, returning only the rewritten HTML paragraphs:\n\n" + html
	}

	res := a.dispatchAIWithMode(system, userMsg, rewriteRequestProfile(mode), providerMode)

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
func (a *App) RewriteTextCustom(text string, prompt string, providerMode string) types.AIRewriteResult {
	s := a.getSettings()
	logging.AI("========== RewriteTextCustom START ==========")
	logging.AI("prompt=%s ai_mode=%s task_provider=%s provider=%s", prompt, s.AIMode, providerMode, s.AIProvider)
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

	res := a.dispatchAIWithMode(system, userMsg, standardAIRequest, providerMode)

	// Log the result
	if res.Error != "" {
		logging.AI("ERROR: %s", res.Error)
	} else {
		logging.AIContent("OUTPUT_RESULT", res.Result)
	}
	logging.AI("========== RewriteTextCustom END ==========")
	return res
}

// aiEmit forwards a provider event to the Wails runtime. It is injected into
// providers.Request as the Emit closure so the providers package never imports
// the Wails runtime.
func (a *App) aiEmit(event string, data any) {
	runtime.EventsEmit(a.ctx, event, data)
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
	if model := a.getSettings().AIModel; model != "" && modelMatchesProvider(model, defaultModel) {
		return model
	}
	return defaultModel
}
