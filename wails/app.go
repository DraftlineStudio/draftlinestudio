package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
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

// ── Debug Logging ────────────────────────────────────────────────────────────
// Logs AI operations to AppData/Draftline/logs/ for debugging

var debugLogger *log.Logger
var debugLogFile *os.File

func initDebugLog() {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return
	}
	logDir := filepath.Join(configDir, "draftline", "logs")
	_ = os.MkdirAll(logDir, 0755)

	// Rotate logs: keep only last 5
	files, _ := filepath.Glob(filepath.Join(logDir, "ai-*.log"))
	if len(files) >= 5 {
		for i := 0; i < len(files)-4; i++ {
			_ = os.Remove(files[i])
		}
	}

	logFile := filepath.Join(logDir, fmt.Sprintf("ai-%s.log", time.Now().Format("2006-01-02-150405")))
	debugLogFile, err = os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return
	}
	debugLogger = log.New(debugLogFile, "", log.LstdFlags)
	debugLogger.Println("=== Draftline AI Debug Log Started ===")
}

func logAI(format string, args ...interface{}) {
	if debugLogger == nil {
		initDebugLog()
	}
	if debugLogger != nil {
		debugLogger.Printf(format, args...)
	}
}

func logAIRequest(mode, provider string, systemLen, userLen int) {
	logAI("[REQUEST] mode=%s provider=%s system_len=%d user_len=%d", mode, provider, systemLen, userLen)
}

func logAIResponse(resultLen int, err string) {
	if err != "" {
		logAI("[RESPONSE] error=%s", err)
	} else {
		logAI("[RESPONSE] result_len=%d", resultLen)
	}
}

func logAIContent(label, content string) {
	if len(content) > 2000 {
		logAI("[%s] (truncated to 2000 chars)\n%s\n---END %s---", label, content[:2000], label)
	} else {
		logAI("[%s]\n%s\n---END %s---", label, content, label)
	}
}

// ── Backup System ───────────────────────────────────────────────────────────
// Rolling backups stored in AppData/Draftline/backups/<hash>/
// Keeps last 5 backups per project file.

const maxBackups = 5

// backupDir returns the backup directory for a given file path.
// Creates a subdirectory based on a hash of the file path.
func (a *App) backupDir(filePath string) string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = "."
	}
	// Create hash of the file path for unique folder name
	hash := sha256.Sum256([]byte(filepath.Clean(filePath)))
	hashStr := hex.EncodeToString(hash[:8]) // First 8 bytes = 16 hex chars
	dir := filepath.Join(configDir, "draftline", "backups", hashStr)
	_ = os.MkdirAll(dir, 0755)
	return dir
}

// createBackup copies the current file to the backup directory before saving.
// Rotates existing backups: backup.5 deleted, backup.4 → backup.5, etc.
func (a *App) createBackup(filePath string) error {
	// Only backup if the file exists (skip for new files)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil
	}

	backupDir := a.backupDir(filePath)

	// Rotate existing backups (delete oldest, shift others)
	for i := maxBackups; i >= 1; i-- {
		old := filepath.Join(backupDir, fmt.Sprintf("backup.%d.draftline", i))
		if i == maxBackups {
			// Delete the oldest backup
			_ = os.Remove(old)
		} else {
			// Rename backup.N to backup.N+1
			newName := filepath.Join(backupDir, fmt.Sprintf("backup.%d.draftline", i+1))
			_ = os.Rename(old, newName)
		}
	}

	// Copy current file to backup.1
	src, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	backupPath := filepath.Join(backupDir, "backup.1.draftline")
	if err := os.WriteFile(backupPath, src, 0644); err != nil {
		return err
	}

	// Also write a small metadata file so we know what this backup is
	metaPath := filepath.Join(backupDir, "info.json")
	meta := map[string]string{
		"original_path": filePath,
		"last_backup":   time.Now().Format(time.RFC3339),
	}
	metaBytes, _ := json.MarshalIndent(meta, "", "  ")
	_ = os.WriteFile(metaPath, metaBytes, 0644)

	return nil
}

// BackupInfo contains metadata about a backup file
type BackupInfo struct {
	Number   int    `json:"number"`
	Path     string `json:"path"`
	Modified string `json:"modified"`
	Size     int64  `json:"size"`
}

// ListBackups returns available backups for the current file
func (a *App) ListBackups() []BackupInfo {
	if a.currentFile == "" {
		return []BackupInfo{}
	}

	backupDir := a.backupDir(a.currentFile)
	var backups []BackupInfo

	for i := 1; i <= maxBackups; i++ {
		path := filepath.Join(backupDir, fmt.Sprintf("backup.%d.draftline", i))
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		backups = append(backups, BackupInfo{
			Number:   i,
			Path:     path,
			Modified: info.ModTime().Format(time.RFC3339),
			Size:     info.Size(),
		})
	}

	return backups
}

// RestoreBackup restores a backup by number (1 = most recent, 5 = oldest)
func (a *App) RestoreBackup(number int) SaveResult {
	if a.currentFile == "" {
		return SaveResult{Success: false, Error: "no file currently open"}
	}
	if number < 1 || number > maxBackups {
		return SaveResult{Success: false, Error: "invalid backup number"}
	}

	backupDir := a.backupDir(a.currentFile)
	backupPath := filepath.Join(backupDir, fmt.Sprintf("backup.%d.draftline", number))

	// Check backup exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return SaveResult{Success: false, Error: "backup not found"}
	}

	// Read backup
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return SaveResult{Success: false, Error: err.Error()}
	}

	// Before restoring, backup the current state (so restore is reversible)
	_ = a.createBackup(a.currentFile)

	// Write backup data to current file
	if err := os.WriteFile(a.currentFile, data, 0644); err != nil {
		return SaveResult{Success: false, Error: err.Error()}
	}

	return SaveResult{Success: true, FilePath: a.currentFile}
}

// ── App Version ──────────────────────────────────────────────────────────────
// Format: MAJOR.MINOR.BUILD
// - MAJOR: Large feature updates (1.x, 2.x, 3.x)
// - MINOR: Feature chunks within major (x.1, x.2, x.3)
// - BUILD: Always incrementing 5-digit build number (never resets)
// Example: 0.8.02313 → 0.8.02314 (bug fix) → 0.9.02315 (new feature set)
const (
	AppVersionMajor = 0
	AppVersionMinor = 12
	AppVersionBuild = 2322
	AppVersion      = "0.12.02322" +
		""
)

// App is the main application struct bound to the frontend.
type App struct {
	ctx           context.Context
	currentFile   string
	settings      AppSettings
	cancelMu      sync.Mutex
	cancelRewrite context.CancelFunc // non-nil while a rewrite is in progress
}

// GetAppVersion returns the current application version string.
func (a *App) GetAppVersion() string {
	return AppVersion
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
	Title    string `json:"title"`
	Subtitle string `json:"subtitle,omitempty"` // Optional chapter subheading
	Type     string `json:"type"`
	Content  string `json:"content"`
}

type WritingGoals struct {
	TargetWordCount int    `json:"target_word_count"`
	DailyWordGoal   int    `json:"daily_word_goal"`
	WordsToday      int    `json:"words_today"`
	LastWritingDate string `json:"last_writing_date"`
}

// WritingStyleOptions controls AI behavior for Expand/Smooth modes
// Each value: 0 = off, 1 = subtle, 2 = moderate, 3 = heavy
type WritingStyleOptions struct {
	Metaphors       int `json:"metaphors"`
	Similes         int `json:"similes"`
	SensoryDetail   int `json:"sensory_detail"`
	InternalThought int `json:"internal_thought"`
	Dialogue        int `json:"dialogue"`
	Action          int `json:"action"`
	Description     int `json:"description"`
	Pacing          int `json:"pacing"`
}

type BookData struct {
	Version             string              `json:"version"`
	Metadata            Metadata            `json:"metadata"`
	Copyright           string              `json:"copyright"`
	FrontMatter         []ChapterItem       `json:"front_matter"`
	Body                []ChapterItem       `json:"body"`
	BackMatter          []ChapterItem       `json:"back_matter"`
	FilePath            string              `json:"file_path,omitempty"`
	StoryBible          StoryBible          `json:"story_bible,omitempty"`
	WritingGoals        WritingGoals        `json:"writing_goals,omitempty"`
	StyleOptions        WritingStyleOptions `json:"style_options,omitempty"`
	IsIndexed           bool                `json:"is_indexed,omitempty"`
	LastIndexed         string              `json:"last_indexed,omitempty"`
	BeatSheet           BeatSheet           `json:"beat_sheet,omitempty"`
	ForeshadowingLedger ForeshadowingLedger `json:"foreshadowing,omitempty"`
	KnowledgeMatrix     KnowledgeMatrix     `json:"knowledge_matrix,omitempty"`
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
	// Auto-detection fields
	IsAutoDetected  bool              `json:"is_auto_detected,omitempty"`
	Aliases         []string          `json:"aliases,omitempty"`          // Alternative names/nicknames
	MentionCount    int               `json:"mention_count,omitempty"`    // Total mentions across all chapters
	FirstChapter    int               `json:"first_chapter,omitempty"`    // Chapter index where first mentioned
	ChapterMentions map[int]int       `json:"chapter_mentions,omitempty"` // Chapter index → mention count
	Attributes      map[string]string `json:"attributes,omitempty"`       // Extracted attributes (eye_color, hair_color, etc.)
}

// CharacterMention represents a detected character mention in text
type CharacterMention struct {
	CharacterID string `json:"character_id"`
	Name        string `json:"name"`    // The name as it appeared in text
	Chapter     int    `json:"chapter"` // Chapter index
	Section     string `json:"section"` // front_matter, body, back_matter
	Offset      int    `json:"offset"`  // Character offset in chapter content
	Context     string `json:"context"` // Surrounding text snippet
}

type StoryBible struct {
	Characters []Character `json:"characters"`
	PlotNotes  string      `json:"plot_notes"`
	Timeline   string      `json:"timeline"`
}

// Beat represents a story beat in the beat sheet
type Beat struct {
	ID           string `json:"id"`
	ChapterIndex int    `json:"chapter_index"`
	BeatType     string `json:"beat_type"` // opening_image, theme_stated, catalyst, midpoint, all_is_lost, finale, custom, etc.
	Description  string `json:"description"`
	Notes        string `json:"notes,omitempty"`
}

// BeatSheet contains all story beats for the book
type BeatSheet struct {
	Beats []Beat `json:"beats"`
}

// ForeshadowingItem tracks a foreshadowing element through plant → reinforce → payoff
type ForeshadowingItem struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Description       string `json:"description"`
	PlantChapter      int    `json:"plant_chapter"`
	ReinforceChapters []int  `json:"reinforce_chapters,omitempty"`
	PayoffChapter     *int   `json:"payoff_chapter,omitempty"` // Pointer for nullable
	Status            string `json:"status"`                   // planted, active, resolved
	Notes             string `json:"notes,omitempty"`
}

// ForeshadowingLedger contains all foreshadowing items for the book
type ForeshadowingLedger struct {
	Items []ForeshadowingItem `json:"items"`
}

// SecretInfo represents a piece of information/secret in the knowledge matrix
type SecretInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// KnowledgeEntry tracks when a character learns a secret
type KnowledgeEntry struct {
	SecretID         string `json:"secret_id"`
	CharacterID      string `json:"character_id"`
	LearnsChapter    *int   `json:"learns_chapter,omitempty"`    // When they definitively learn it
	SuspectedChapter *int   `json:"suspected_chapter,omitempty"` // When they start to suspect
}

// KnowledgeMatrix tracks which characters know which secrets at each chapter
type KnowledgeMatrix struct {
	Secrets []SecretInfo     `json:"secrets"`
	Entries []KnowledgeEntry `json:"entries"`
}

// IndexResult contains the results of character indexing
type IndexResult struct {
	Success           bool        `json:"success"`
	Error             string      `json:"error,omitempty"`
	CharactersFound   int         `json:"characters_found"`
	NewCharacters     int         `json:"new_characters"`
	UpdatedCharacters int         `json:"updated_characters"`
	Characters        []Character `json:"characters,omitempty"`
}

// Common words to exclude from character detection
var commonWords = map[string]bool{
	// Short words (1-2 chars) - filtered by regex but kept for safety
	"A": true, "I": true, "An": true, "He": true, "It": true, "We": true, "Or": true,
	"So": true, "No": true, "If": true, "My": true, "Me": true, "Us": true, "Up": true,
	// Pronouns and determiners (3+ chars)
	"The": true, "She": true, "You": true, "His": true, "Her": true, "Its": true,
	"Our": true, "They": true, "Your": true, "This": true, "That": true, "Him": true,
	"Their": true, "Mine": true, "Yours": true, "Ours": true, "Theirs": true,
	// Question words
	"Who": true, "What": true, "Where": true, "When": true, "Why": true, "How": true,
	// Common verbs and short words
	"Was": true, "Were": true, "Has": true, "Had": true, "Can": true, "Did": true,
	"Get": true, "Got": true, "Let": true, "See": true, "Say": true, "Use": true,
	// Conjunctions and prepositions
	"And": true, "But": true, "For": true, "Nor": true, "Yet": true, "Not": true,
	"All": true, "Any": true, "Out": true, "Now": true, "New": true, "Old": true,
	"Upon": true, "With": true, "From": true, "Over": true, "Under": true, "Along": true,
	// Adverbs and time words
	"There": true, "Here": true, "Then": true, "Just": true, "Only": true, "Even": true,
	"Still": true, "Already": true, "Very": true, "Really": true, "Quite": true, "Rather": true,
	"After": true, "Before": true, "During": true, "While": true, "About": true,
	"Against": true, "Between": true, "Into": true, "Through": true,
	"However": true, "Therefore": true, "Meanwhile": true, "Finally": true, "Suddenly": true,
	"Because": true, "Yes": true, "Maybe": true, "Perhaps": true,
	// Numbers
	"One": true, "Two": true, "Three": true, "Four": true, "Five": true,
	"First": true, "Second": true, "Third": true, "Last": true, "Next": true,
	"Few": true, "Way": true, "Day": true, "Man": true, "Boy": true, "Too": true, "Ago": true,
	// Titles (filtered but could appear in dialogue attribution)
	"Mr": true, "Mrs": true, "Ms": true, "Dr": true, "Sir": true, "Lord": true, "Lady": true,
	// Document/chapter words
	"Chapter": true, "Part": true, "Book": true, "Volume": true,
	// Days and months
	"Monday": true, "Tuesday": true, "Wednesday": true, "Thursday": true, "Friday": true, "Saturday": true, "Sunday": true,
	"January": true, "February": true, "March": true, "April": true, "May": true, "June": true,
	"July": true, "August": true, "September": true, "October": true, "November": true, "December": true,
	// Directions and places
	"North": true, "South": true, "East": true, "West": true,
	"God": true, "Earth": true, "Heaven": true, "Hell": true,
	// Indefinite pronouns
	"Something": true, "Nothing": true, "Everything": true, "Anything": true,
	"Someone": true, "Anyone": true, "Everyone": true,
}

// stripHTML removes HTML tags from content
func stripHTML(html string) string {
	// Simple regex to remove HTML tags
	re := regexp.MustCompile(`<[^>]*>`)
	text := re.ReplaceAllString(html, " ")
	// Clean up whitespace
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}

// detectCharacterNames extracts potential character names from text
// Uses quote-aware dialogue patterns to minimize false positives
// Errs on the side of false negatives - users can add missed characters manually
func detectCharacterNames(text string) map[string]int {
	mentions := make(map[string]int)

	// QUOTE-AWARE DIALOGUE PATTERNS
	// These are the most reliable indicators of character names in fiction
	dialogueVerbs := `said|asked|replied|answered|whispered|shouted|yelled|muttered|exclaimed|cried|called|screamed|murmured|snapped|growled|laughed|sighed|groaned|demanded|insisted|suggested|agreed|admitted|explained|continued|added|interrupted|announced|declared|observed|remarked|noted|commented|wondered|mused|thought|began|finished|concluded`

	// Pattern 1: Speaker AFTER closing quote: '," said John' or '." John said'
	// Matches: "Hello," said John. | "Hello." John replied.
	afterQuote := regexp.MustCompile(`[,.]"\s*(?i:` + dialogueVerbs + `)\s+([A-Z][a-z]{2,})\b`)
	afterQuote2 := regexp.MustCompile(`[,.]"\s*([A-Z][a-z]{2,})\s+(?i:` + dialogueVerbs + `)`)

	// Pattern 2: Speaker BEFORE opening quote: 'John said, "' or 'John said "'
	// Matches: John said, "Hello" | John replied "Hi"
	beforeQuote := regexp.MustCompile(`\b([A-Z][a-z]{2,})\s+(?i:` + dialogueVerbs + `)\s*,?\s*"`)

	// Pattern 3: Title + Name (Mr. Smith, Dr. Jones, Captain Kirk)
	// These are very reliable indicators
	afterTitle := regexp.MustCompile(`\b(Mr|Mrs|Ms|Miss|Dr|Prof|Professor|Captain|Colonel|General|Lieutenant|Sergeant|Officer|Detective|Agent|Lord|Lady|Sir|Dame|King|Queen|Prince|Princess|Senator|Governor|Mayor|Chief|Father|Mother|Sister|Brother|Uncle|Aunt)\.?\s+([A-Z][a-z]{2,})\b`)

	// Find speakers after closing quotes
	for _, match := range afterQuote.FindAllStringSubmatch(text, -1) {
		name := match[1]
		if !commonWords[name] && !looksLikeCommonWord(name) {
			mentions[name] += 3
		}
	}
	for _, match := range afterQuote2.FindAllStringSubmatch(text, -1) {
		name := match[1]
		if !commonWords[name] && !looksLikeCommonWord(name) {
			mentions[name] += 3
		}
	}

	// Find speakers before opening quotes
	for _, match := range beforeQuote.FindAllStringSubmatch(text, -1) {
		name := match[1]
		if !commonWords[name] && !looksLikeCommonWord(name) {
			mentions[name] += 3
		}
	}

	// Find names after titles
	for _, match := range afterTitle.FindAllStringSubmatch(text, -1) {
		name := match[2]
		if !looksLikeCommonWord(name) {
			mentions[name] += 2
		}
	}

	return mentions
}

// looksLikeCommonWord checks if a word has suffixes/patterns typical of
// common English words rather than names (adverbs, gerunds, etc.)
func looksLikeCommonWord(word string) bool {
	lower := strings.ToLower(word)

	// Common word suffixes that are almost never names
	suffixes := []string{
		"ly",    // adverbs: agreeably, quickly, suddenly
		"ing",   // gerunds: running, being, having (but not names like Ming)
		"tion",  // nouns: action, motion, station
		"sion",  // nouns: tension, mission
		"ness",  // nouns: darkness, happiness
		"ment",  // nouns: moment, movement
		"able",  // adjectives: capable, notable
		"ible",  // adjectives: possible, visible
		"ful",   // adjectives: beautiful, careful
		"less",  // adjectives: careless, hopeless
		"ous",   // adjectives: curious, nervous
		"ive",   // adjectives: active, creative
		"ward",  // directions: forward, backward
		"wards", // directions: afterwards, towards
		"wise",  // manner: otherwise, likewise
	}

	for _, suffix := range suffixes {
		if strings.HasSuffix(lower, suffix) && len(lower) > len(suffix)+2 {
			return true
		}
	}

	// Words that are clearly temporal/common
	temporalWords := map[string]bool{
		"today": true, "tomorrow": true, "yesterday": true,
		"morning": true, "evening": true, "afternoon": true, "tonight": true,
		"sometimes": true, "always": true, "never": true, "often": true,
		"perhaps": true, "maybe": true, "probably": true, "certainly": true,
		"however": true, "therefore": true, "although": true, "because": true,
		"before": true, "after": true, "during": true, "until": true,
		"inside": true, "outside": true, "between": true, "within": true,
		"around": true, "through": true, "without": true, "against": true,
	}

	return temporalWords[lower]
}

// extractAttributes looks for character attributes in surrounding context
func extractAttributes(text, name string) map[string]string {
	attrs := make(map[string]string)
	nameLower := strings.ToLower(name)
	textLower := strings.ToLower(text)

	// Patterns for attribute extraction
	// Eye color: "Kira's blue eyes", "her green eyes", "eyes were brown"
	eyeColors := []string{"blue", "green", "brown", "hazel", "gray", "grey", "black", "amber", "violet", "golden"}
	for _, color := range eyeColors {
		patterns := []string{
			nameLower + `'s\s+` + color + `\s+eyes?`,
			nameLower + `\s+.*\b` + color + `\s+eyes?`,
			`\b` + color + `\s+eyes?.*` + nameLower,
		}
		for _, pattern := range patterns {
			if matched, _ := regexp.MatchString(pattern, textLower); matched {
				attrs["eye_color"] = color
				break
			}
		}
	}

	// Hair color
	hairColors := []string{"blonde", "blond", "brunette", "brown", "black", "red", "auburn", "gray", "grey", "white", "silver", "golden", "dark", "light"}
	for _, color := range hairColors {
		patterns := []string{
			nameLower + `'s\s+` + color + `\s+hair`,
			nameLower + `\s+.*\b` + color + `\s+hair`,
			`\b` + color + `\s+hair.*` + nameLower,
		}
		for _, pattern := range patterns {
			if matched, _ := regexp.MatchString(pattern, textLower); matched {
				attrs["hair_color"] = color
				break
			}
		}
	}

	// Age patterns: "twenty-year-old", "aged 30", "30 years old"
	agePattern := regexp.MustCompile(`\b(\d{1,2})[\s-]?year[\s-]?old\b`)
	if matches := agePattern.FindStringSubmatch(textLower); len(matches) > 1 {
		attrs["age"] = matches[1]
	}

	return attrs
}

// IndexBook scans all chapters and extracts/updates character information
func (a *App) IndexBook(book BookData) IndexResult {
	if book.StoryBible.Characters == nil {
		book.StoryBible.Characters = []Character{}
	}

	// Build a map of existing characters by name (case-insensitive)
	existingChars := make(map[string]*Character)
	for i := range book.StoryBible.Characters {
		char := &book.StoryBible.Characters[i]
		existingChars[strings.ToLower(char.Name)] = char
		// Also index aliases
		for _, alias := range char.Aliases {
			existingChars[strings.ToLower(alias)] = char
		}
	}

	// Aggregate all detected names across all chapters
	allNames := make(map[string]int)                // name → total mentions
	chapterMentions := make(map[string]map[int]int) // name → (chapter → count)
	firstChapter := make(map[string]int)            // name → first chapter seen
	fullText := ""                                  // For attribute extraction

	// Process all sections
	sections := []struct {
		name  string
		items []ChapterItem
	}{
		{"front_matter", book.FrontMatter},
		{"body", book.Body},
		{"back_matter", book.BackMatter},
	}

	chapterIndex := 0
	for _, section := range sections {
		for _, chapter := range section.items {
			text := stripHTML(chapter.Content)
			fullText += " " + text
			names := detectCharacterNames(text)

			for name, count := range names {
				allNames[name] += count

				// Track chapter mentions
				if chapterMentions[name] == nil {
					chapterMentions[name] = make(map[int]int)
				}
				chapterMentions[name][chapterIndex] += count

				// Track first chapter
				if _, exists := firstChapter[name]; !exists {
					firstChapter[name] = chapterIndex
				}
			}
			chapterIndex++
		}
	}

	// Filter: only keep names with 3+ mentions (likely real characters)
	newChars := 0
	updatedChars := 0

	for name, count := range allNames {
		if count < 3 {
			continue // Skip names with too few mentions
		}

		nameLower := strings.ToLower(name)

		if existing, found := existingChars[nameLower]; found {
			// Update existing character
			existing.MentionCount = count
			existing.ChapterMentions = chapterMentions[name]
			if existing.FirstChapter == 0 {
				existing.FirstChapter = firstChapter[name]
			}
			// Try to extract more attributes
			newAttrs := extractAttributes(fullText, name)
			if existing.Attributes == nil {
				existing.Attributes = make(map[string]string)
			}
			for k, v := range newAttrs {
				if existing.Attributes[k] == "" {
					existing.Attributes[k] = v
				}
			}
			updatedChars++
		} else {
			// Create new character
			newChar := Character{
				ID:              fmt.Sprintf("char-%d", time.Now().UnixNano()),
				Name:            name,
				Role:            "minor", // Default role
				IsAutoDetected:  true,
				MentionCount:    count,
				FirstChapter:    firstChapter[name],
				ChapterMentions: chapterMentions[name],
				Attributes:      extractAttributes(fullText, name),
			}
			book.StoryBible.Characters = append(book.StoryBible.Characters, newChar)
			existingChars[nameLower] = &newChar
			newChars++
		}
	}

	return IndexResult{
		Success:           true,
		CharactersFound:   len(allNames),
		NewCharacters:     newChars,
		UpdatedCharacters: updatedChars,
		Characters:        book.StoryBible.Characters,
	}
}

// IndexChapter scans a single chapter for character mentions (incremental update)
func (a *App) IndexChapter(book BookData, section string, chapterIndex int) IndexResult {
	var content string
	switch section {
	case "front_matter":
		if chapterIndex < len(book.FrontMatter) {
			content = book.FrontMatter[chapterIndex].Content
		}
	case "body":
		if chapterIndex < len(book.Body) {
			content = book.Body[chapterIndex].Content
		}
	case "back_matter":
		if chapterIndex < len(book.BackMatter) {
			content = book.BackMatter[chapterIndex].Content
		}
	}

	if content == "" {
		return IndexResult{Success: true, CharactersFound: 0}
	}

	text := stripHTML(content)
	names := detectCharacterNames(text)

	// Build existing character map
	existingChars := make(map[string]*Character)
	for i := range book.StoryBible.Characters {
		char := &book.StoryBible.Characters[i]
		existingChars[strings.ToLower(char.Name)] = char
		for _, alias := range char.Aliases {
			existingChars[strings.ToLower(alias)] = char
		}
	}

	newChars := 0
	for name, count := range names {
		if count < 2 {
			continue
		}

		nameLower := strings.ToLower(name)
		if existing, found := existingChars[nameLower]; found {
			// Update mention count for this chapter
			if existing.ChapterMentions == nil {
				existing.ChapterMentions = make(map[int]int)
			}
			existing.ChapterMentions[chapterIndex] = count
			existing.MentionCount = 0
			for _, c := range existing.ChapterMentions {
				existing.MentionCount += c
			}
		} else {
			// New character found
			newChar := Character{
				ID:              fmt.Sprintf("char-%d", time.Now().UnixNano()),
				Name:            name,
				Role:            "minor",
				IsAutoDetected:  true,
				MentionCount:    count,
				FirstChapter:    chapterIndex,
				ChapterMentions: map[int]int{chapterIndex: count},
				Attributes:      extractAttributes(text, name),
			}
			book.StoryBible.Characters = append(book.StoryBible.Characters, newChar)
			newChars++
		}
	}

	return IndexResult{
		Success:         true,
		CharactersFound: len(names),
		NewCharacters:   newChars,
		Characters:      book.StoryBible.Characters,
	}
}

type AppSettings struct {
	// Application
	DefaultAuthor      string `json:"default_author"`
	DefaultPublisher   string `json:"default_publisher"`
	DefaultCopyright   string `json:"default_copyright"`
	DefaultSaveDir     string `json:"default_save_dir"`
	DarkMode           bool   `json:"dark_mode"`
	ThemeMode          string `json:"theme_mode"` // "light" | "dark" | "auto"
	AutoThemeUseManual bool   `json:"auto_theme_use_manual"`
	AutoThemeDawn      string `json:"auto_theme_dawn"` // "HH:MM" format
	AutoThemeDusk      string `json:"auto_theme_dusk"` // "HH:MM" format
	// AI
	AIEnabled       bool   `json:"ai_enabled"`
	AIMode          string `json:"ai_mode"`     // "claudecode" | "api" | "local"
	AIProvider      string `json:"ai_provider"` // "claude" | "openai" | ""
	AIAPIKey        string `json:"ai_api_key"`
	AIModel         string `json:"ai_model"`
	AILocalEndpoint string `json:"ai_local_endpoint"` // e.g. http://localhost:11434/v1
	AILocalModel    string `json:"ai_local_model"`
	ProseGuide      string `json:"prose_guide"`
	// Book defaults
	BookFont        string `json:"book_font"`
	EditorFontSize  string `json:"editor_font_size"`  // "small"|"normal"|"large" (12/14/16px)
	BookFontSize    int    `json:"book_font_size"`    // Export font size in points
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
			Title    string `json:"title"`
			Subtitle string `json:"subtitle,omitempty"`
			Type     string `json:"type"`
			File     string `json:"file"`
		} `json:"front_matter,omitempty"`
		Body []struct {
			Title    string `json:"title"`
			Subtitle string `json:"subtitle,omitempty"`
			Type     string `json:"type"`
			File     string `json:"file"`
		} `json:"body,omitempty"`
		BackMatter []struct {
			Title    string `json:"title"`
			Subtitle string `json:"subtitle,omitempty"`
			Type     string `json:"type"`
			File     string `json:"file"`
		} `json:"back_matter,omitempty"`
		WritingGoals WritingGoals        `json:"writing_goals,omitempty"`
		StyleOptions WritingStyleOptions `json:"style_options,omitempty"`
		IsIndexed    bool                `json:"is_indexed,omitempty"`
		LastIndexed  string              `json:"last_indexed,omitempty"`
	}

	if err := json.Unmarshal(manifestData, &raw); err != nil {
		return BookData{}, fmt.Errorf("failed to parse manifest: %w", err)
	}

	book := BookData{
		Version:      "2.0",
		Metadata:     raw.Metadata,
		FilePath:     path,
		FrontMatter:  []ChapterItem{},
		Body:         []ChapterItem{},
		BackMatter:   []ChapterItem{},
		WritingGoals: raw.WritingGoals,
		StyleOptions: raw.StyleOptions,
		IsIndexed:    raw.IsIndexed,
		LastIndexed:  raw.LastIndexed,
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
			Title: item.Title, Subtitle: item.Subtitle, Type: item.Type, Content: string(content),
		})
	}
	for _, item := range raw.Body {
		content, _ := readZipEntry(r, item.File)
		book.Body = append(book.Body, ChapterItem{
			Title: item.Title, Subtitle: item.Subtitle, Type: item.Type, Content: string(content),
		})
	}
	for _, item := range raw.BackMatter {
		content, _ := readZipEntry(r, item.File)
		book.BackMatter = append(book.BackMatter, ChapterItem{
			Title: item.Title, Subtitle: item.Subtitle, Type: item.Type, Content: string(content),
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

	// beat_sheet.json (optional)
	if beatData, err := readZipEntry(r, "beat_sheet.json"); err == nil {
		var beatSheet BeatSheet
		if json.Unmarshal(beatData, &beatSheet) == nil {
			book.BeatSheet = beatSheet
		}
	}

	// foreshadowing.json (optional)
	if foreshadowData, err := readZipEntry(r, "foreshadowing.json"); err == nil {
		var ledger ForeshadowingLedger
		if json.Unmarshal(foreshadowData, &ledger) == nil {
			book.ForeshadowingLedger = ledger
		}
	}

	// knowledge_matrix.json (optional)
	if matrixData, err := readZipEntry(r, "knowledge_matrix.json"); err == nil {
		var matrix KnowledgeMatrix
		if json.Unmarshal(matrixData, &matrix) == nil {
			book.KnowledgeMatrix = matrix
		}
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

// ── Export Functions ─────────────────────────────────────────────────────────

// ExportOptions contains common export settings
type ExportOptions struct {
	IncludeCopyright   bool `json:"includeCopyright"`
	IncludeFrontMatter bool `json:"includeFrontMatter"`
	IncludeBackMatter  bool `json:"includeBackMatter"`
}

// PDFOptions extends ExportOptions with PDF-specific settings
type PDFOptions struct {
	ExportOptions
	PageSize string `json:"pageSize"` // letter, a4, 6x9, 5x8
	FontSize int    `json:"fontSize"` // 11, 12, 14
}

// PrintPDFOptions extends PDFOptions with print-ready settings
type PrintPDFOptions struct {
	PDFOptions
	TrimSize         string `json:"trimSize"` // 5x8, 5.25x8, 5.5x8.5, 6x9, custom
	CustomWidth      string `json:"customWidth"`
	CustomHeight     string `json:"customHeight"`
	Bleed            string `json:"bleed"`
	GutterMargin     string `json:"gutterMargin"`
	OuterMargin      string `json:"outerMargin"`
	TopMargin        string `json:"topMargin"`
	BottomMargin     string `json:"bottomMargin"`
	IncludeCropMarks bool   `json:"includeCropMarks"`
	// Typography
	FontFamily      string  `json:"fontFamily"` // garamond, palatino, times, georgia
	LineHeight      float64 `json:"lineHeight"` // 1.3, 1.4, 1.5, 1.6
	ParagraphIndent string  `json:"paragraphIndent"`
	TextAlign       string  `json:"textAlign"` // justify, left
	// Chapter styling
	ChapterStartsRecto bool `json:"chapterStartsRecto"`
	DropCap            bool `json:"dropCap"`
	DropCapLines       int  `json:"dropCapLines"` // 2, 3, 4
	// Headers & footers
	RunningHeaders     bool   `json:"runningHeaders"`
	HeaderStyle        string `json:"headerStyle"`        // smallcaps, italic, normal
	PageNumberPosition string `json:"pageNumberPosition"` // bottom-center, bottom-outside, top-outside
	// Front matter
	GenerateHalfTitle bool `json:"generateHalfTitle"`
	GenerateTOC       bool `json:"generateTOC"`
	MirroredMargins   bool `json:"mirroredMargins"` // Critical: swap gutter/outer for odd/even pages
}

// ExportResult contains the result of an export operation
type ExportResult struct {
	Success  bool   `json:"success"`
	FilePath string `json:"file_path,omitempty"`
	Error    string `json:"error,omitempty"`
}

// htmlToPlainParagraphs converts HTML to plain text with paragraph breaks
func htmlToPlainParagraphs(html string) string {
	// Replace paragraph and heading tags with newlines
	text := regexp.MustCompile(`</?(p|h[1-6]|div|br)[^>]*>`).ReplaceAllString(html, "\n")
	// Remove remaining HTML tags
	text = regexp.MustCompile(`<[^>]*>`).ReplaceAllString(text, "")
	// Clean up multiple newlines
	text = regexp.MustCompile(`\n{3,}`).ReplaceAllString(text, "\n\n")
	return strings.TrimSpace(text)
}

// ExportEPUB exports the book to EPUB format
func (a *App) ExportEPUB(book BookData, options ExportOptions) ExportResult {
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
		return ExportResult{Success: false, Error: "cancelled"}
	}

	if !strings.HasSuffix(strings.ToLower(path), ".epub") {
		path += ".epub"
	}

	// Create EPUB file (it's a ZIP archive)
	file, err := os.Create(path)
	if err != nil {
		return ExportResult{Success: false, Error: fmt.Sprintf("failed to create file: %v", err)}
	}
	defer file.Close()

	zipWriter := zip.NewWriter(file)
	defer zipWriter.Close()

	// Write mimetype (must be first, uncompressed)
	mimetypeWriter, _ := zipWriter.CreateHeader(&zip.FileHeader{
		Name:   "mimetype",
		Method: zip.Store,
	})
	mimetypeWriter.Write([]byte("application/epub+zip"))

	// Write META-INF/container.xml
	containerWriter, _ := zipWriter.Create("META-INF/container.xml")
	containerWriter.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`))

	// Collect chapters to include
	var chapters []ChapterItem
	var chapterIDs []string

	if options.IncludeCopyright && book.Copyright != "" {
		chapters = append(chapters, ChapterItem{Title: "Copyright", Type: "Copyright", Content: book.Copyright})
		chapterIDs = append(chapterIDs, "copyright")
	}
	if options.IncludeFrontMatter {
		for i, ch := range book.FrontMatter {
			chapters = append(chapters, ch)
			chapterIDs = append(chapterIDs, fmt.Sprintf("front%d", i))
		}
	}
	for i, ch := range book.Body {
		chapters = append(chapters, ch)
		chapterIDs = append(chapterIDs, fmt.Sprintf("chapter%d", i))
	}
	if options.IncludeBackMatter {
		for i, ch := range book.BackMatter {
			chapters = append(chapters, ch)
			chapterIDs = append(chapterIDs, fmt.Sprintf("back%d", i))
		}
	}

	// Write content.opf
	var manifestItems, spineItems strings.Builder
	for i, id := range chapterIDs {
		manifestItems.WriteString(fmt.Sprintf(`    <item id="%s" href="%s.xhtml" media-type="application/xhtml+xml"/>
`, id, id))
		if i == 0 {
			spineItems.WriteString(fmt.Sprintf(`    <itemref idref="%s"/>
`, id))
		} else {
			spineItems.WriteString(fmt.Sprintf(`    <itemref idref="%s"/>
`, id))
		}
	}

	opfWriter, _ := zipWriter.Create("OEBPS/content.opf")
	opfWriter.Write([]byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="uid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:identifier id="uid">urn:uuid:%s</dc:identifier>
    <dc:title>%s</dc:title>
    <dc:creator>%s</dc:creator>
    <dc:publisher>%s</dc:publisher>
    <dc:language>en</dc:language>
    <meta property="dcterms:modified">%s</meta>
  </metadata>
  <manifest>
    <item id="nav" href="nav.xhtml" media-type="application/xhtml+xml" properties="nav"/>
%s  </manifest>
  <spine>
%s  </spine>
</package>`,
		generateUUID(),
		escapeXML(book.Metadata.Title),
		escapeXML(book.Metadata.Author),
		escapeXML(book.Metadata.Publisher),
		time.Now().Format("2006-01-02T15:04:05Z"),
		manifestItems.String(),
		spineItems.String())))

	// Write nav.xhtml (table of contents)
	var tocItems strings.Builder
	for i, ch := range chapters {
		tocItems.WriteString(fmt.Sprintf(`      <li><a href="%s.xhtml">%s</a></li>
`, chapterIDs[i], escapeXML(ch.Title)))
	}

	navWriter, _ := zipWriter.Create("OEBPS/nav.xhtml")
	navWriter.Write([]byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops">
<head>
  <title>Table of Contents</title>
</head>
<body>
  <nav epub:type="toc">
    <h1>Table of Contents</h1>
    <ol>
%s    </ol>
  </nav>
</body>
</html>`, tocItems.String())))

	// Write each chapter
	for i, ch := range chapters {
		chWriter, _ := zipWriter.Create(fmt.Sprintf("OEBPS/%s.xhtml", chapterIDs[i]))
		content := ch.Content
		if content == "" {
			content = "<p></p>"
		}
		chWriter.Write([]byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml">
<head>
  <title>%s</title>
</head>
<body>
  <h1>%s</h1>
  %s
</body>
</html>`, escapeXML(ch.Title), escapeXML(ch.Title), content)))
	}

	return ExportResult{Success: true, FilePath: path}
}

// ExportDOCX exports the book to DOCX format
func (a *App) ExportDOCX(book BookData, options ExportOptions) ExportResult {
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
		return ExportResult{Success: false, Error: "cancelled"}
	}

	if !strings.HasSuffix(strings.ToLower(path), ".docx") {
		path += ".docx"
	}

	// Create DOCX file (it's a ZIP archive with XML content)
	file, err := os.Create(path)
	if err != nil {
		return ExportResult{Success: false, Error: fmt.Sprintf("failed to create file: %v", err)}
	}
	defer file.Close()

	zipWriter := zip.NewWriter(file)
	defer zipWriter.Close()

	// Write [Content_Types].xml
	ctWriter, _ := zipWriter.Create("[Content_Types].xml")
	ctWriter.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
  <Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>
</Types>`))

	// Write _rels/.rels
	relsWriter, _ := zipWriter.Create("_rels/.rels")
	relsWriter.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`))

	// Write word/_rels/document.xml.rels
	docRelsWriter, _ := zipWriter.Create("word/_rels/document.xml.rels")
	docRelsWriter.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>
</Relationships>`))

	// Write word/styles.xml
	stylesWriter, _ := zipWriter.Create("word/styles.xml")
	stylesWriter.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:style w:type="paragraph" w:styleId="Heading1">
    <w:name w:val="Heading 1"/>
    <w:pPr><w:spacing w:before="480" w:after="240"/></w:pPr>
    <w:rPr><w:b/><w:sz w:val="48"/></w:rPr>
  </w:style>
  <w:style w:type="paragraph" w:styleId="Normal">
    <w:name w:val="Normal"/>
    <w:pPr><w:spacing w:after="200" w:line="276" w:lineRule="auto"/></w:pPr>
    <w:rPr><w:sz w:val="24"/></w:rPr>
  </w:style>
</w:styles>`))

	// Build document content
	var docContent strings.Builder
	docContent.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
`)

	// Title page
	docContent.WriteString(fmt.Sprintf(`    <w:p><w:pPr><w:jc w:val="center"/></w:pPr><w:r><w:rPr><w:b/><w:sz w:val="72"/></w:rPr><w:t>%s</w:t></w:r></w:p>
`, escapeXML(book.Metadata.Title)))
	if book.Metadata.Author != "" {
		docContent.WriteString(fmt.Sprintf(`    <w:p><w:pPr><w:jc w:val="center"/></w:pPr><w:r><w:rPr><w:sz w:val="36"/></w:rPr><w:t>by %s</w:t></w:r></w:p>
`, escapeXML(book.Metadata.Author)))
	}
	docContent.WriteString(`    <w:p><w:r><w:br w:type="page"/></w:r></w:p>
`)

	// Copyright
	if options.IncludeCopyright && book.Copyright != "" {
		text := htmlToPlainParagraphs(book.Copyright)
		for _, para := range strings.Split(text, "\n") {
			if strings.TrimSpace(para) != "" {
				docContent.WriteString(fmt.Sprintf(`    <w:p><w:r><w:t>%s</w:t></w:r></w:p>
`, escapeXML(para)))
			}
		}
		docContent.WriteString(`    <w:p><w:r><w:br w:type="page"/></w:r></w:p>
`)
	}

	// Front matter
	if options.IncludeFrontMatter {
		for _, ch := range book.FrontMatter {
			writeDocxChapter(&docContent, ch)
		}
	}

	// Body
	for _, ch := range book.Body {
		writeDocxChapter(&docContent, ch)
	}

	// Back matter
	if options.IncludeBackMatter {
		for _, ch := range book.BackMatter {
			writeDocxChapter(&docContent, ch)
		}
	}

	docContent.WriteString(`  </w:body>
</w:document>`)

	docWriter, _ := zipWriter.Create("word/document.xml")
	docWriter.Write([]byte(docContent.String()))

	return ExportResult{Success: true, FilePath: path}
}

func writeDocxChapter(sb *strings.Builder, ch ChapterItem) {
	// Chapter title
	sb.WriteString(fmt.Sprintf(`    <w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr><w:r><w:t>%s</w:t></w:r></w:p>
`, escapeXML(ch.Title)))

	// Subtitle if present
	if ch.Subtitle != "" {
		sb.WriteString(fmt.Sprintf(`    <w:p><w:pPr><w:jc w:val="center"/></w:pPr><w:r><w:rPr><w:i/></w:rPr><w:t>%s</w:t></w:r></w:p>
`, escapeXML(ch.Subtitle)))
	}

	// Content
	text := htmlToPlainParagraphs(ch.Content)
	for _, para := range strings.Split(text, "\n") {
		if strings.TrimSpace(para) != "" {
			sb.WriteString(fmt.Sprintf(`    <w:p><w:r><w:t>%s</w:t></w:r></w:p>
`, escapeXML(para)))
		}
	}

	// Page break after chapter
	sb.WriteString(`    <w:p><w:r><w:br w:type="page"/></w:r></w:p>
`)
}

// ExportPDF exports the book to PDF format
func (a *App) ExportPDF(book BookData, options PDFOptions) ExportResult {
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
		return ExportResult{Success: false, Error: "cancelled"}
	}

	if !strings.HasSuffix(strings.ToLower(path), ".pdf") {
		path += ".pdf"
	}

	// Generate PDF content
	pdf := a.generatePDF(book, options)

	err = os.WriteFile(path, pdf, 0644)
	if err != nil {
		return ExportResult{Success: false, Error: fmt.Sprintf("failed to write file: %v", err)}
	}

	return ExportResult{Success: true, FilePath: path}
}

// ExportPrintPDF exports the book to print-ready PDF format
func (a *App) ExportPrintPDF(book BookData, options PrintPDFOptions) ExportResult {
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
		return ExportResult{Success: false, Error: "cancelled"}
	}

	if !strings.HasSuffix(strings.ToLower(path), ".pdf") {
		path += ".pdf"
	}

	// Generate print-ready PDF with custom trim size
	pdf := a.generatePrintPDF(book, options)

	err = os.WriteFile(path, pdf, 0644)
	if err != nil {
		return ExportResult{Success: false, Error: fmt.Sprintf("failed to write file: %v", err)}
	}

	return ExportResult{Success: true, FilePath: path}
}

// pdfWriter handles multi-page PDF generation with proper text layout
type pdfWriter struct {
	pageWidth    float64
	pageHeight   float64
	marginLeft   float64
	marginRight  float64
	marginTop    float64
	marginBottom float64
	fontSize     float64
	lineHeight   float64
	currentY     float64
	currentPage  *strings.Builder
	pageContents []string
}

// printPDFWriter extends pdfWriter with professional print features
type printPDFWriter struct {
	pageWidth       float64
	pageHeight      float64
	gutterMargin    float64 // Inside margin (toward spine)
	outerMargin     float64 // Outside margin
	topMargin       float64
	bottomMargin    float64
	fontSize        float64
	lineHeight      float64
	paragraphIndent float64
	currentY        float64
	currentPage     *strings.Builder
	pageContents    []string
	pageNumber      int // 1-indexed current page number
	mirroredMargins bool
	// Book metadata for headers
	bookTitle      string
	currentChapter string
	// Options
	runningHeaders     bool
	headerStyle        string
	pageNumberPosition string
	dropCap            bool
	dropCapLines       int
	chapterStartsRecto bool
	// TOC tracking
	tocEntries []tocEntry
}

type tocEntry struct {
	title   string
	pageNum int
}

func newPrintPDFWriter(opts PrintPDFOptions, bookTitle string) *printPDFWriter {
	// Calculate page dimensions from trim size (72 points = 1 inch)
	var pageWidth, pageHeight float64

	switch opts.TrimSize {
	case "5x8":
		pageWidth = 5.0 * 72
		pageHeight = 8.0 * 72
	case "5.25x8":
		pageWidth = 5.25 * 72
		pageHeight = 8.0 * 72
	case "5.5x8.5":
		pageWidth = 5.5 * 72
		pageHeight = 8.5 * 72
	case "6x9":
		pageWidth = 6.0 * 72
		pageHeight = 9.0 * 72
	case "custom":
		w, _ := strconv.ParseFloat(opts.CustomWidth, 64)
		h, _ := strconv.ParseFloat(opts.CustomHeight, 64)
		if w <= 0 {
			w = 5.5
		}
		if h <= 0 {
			h = 8.5
		}
		pageWidth = w * 72
		pageHeight = h * 72
	default:
		pageWidth = 5.5 * 72
		pageHeight = 8.5 * 72
	}

	// Parse margins
	gutterMargin := 0.875
	if g, err := strconv.ParseFloat(opts.GutterMargin, 64); err == nil && g > 0 {
		gutterMargin = g
	}
	outerMargin := 0.625
	if o, err := strconv.ParseFloat(opts.OuterMargin, 64); err == nil && o > 0 {
		outerMargin = o
	}
	topMargin := 0.75
	if t, err := strconv.ParseFloat(opts.TopMargin, 64); err == nil && t > 0 {
		topMargin = t
	}
	bottomMargin := 0.625
	if b, err := strconv.ParseFloat(opts.BottomMargin, 64); err == nil && b > 0 {
		bottomMargin = b
	}
	paragraphIndent := 0.25
	if pi, err := strconv.ParseFloat(opts.ParagraphIndent, 64); err == nil && pi >= 0 {
		paragraphIndent = pi
	}

	lineHeight := opts.LineHeight
	if lineHeight <= 0 {
		lineHeight = 1.4
	}

	dropCapLines := opts.DropCapLines
	if dropCapLines <= 0 {
		dropCapLines = 3
	}

	return &printPDFWriter{
		pageWidth:          pageWidth,
		pageHeight:         pageHeight,
		gutterMargin:       gutterMargin * 72,
		outerMargin:        outerMargin * 72,
		topMargin:          topMargin * 72,
		bottomMargin:       bottomMargin * 72,
		fontSize:           float64(opts.FontSize),
		lineHeight:         float64(opts.FontSize) * lineHeight,
		paragraphIndent:    paragraphIndent * 72,
		currentY:           pageHeight - topMargin*72,
		currentPage:        &strings.Builder{},
		pageNumber:         0,
		mirroredMargins:    opts.MirroredMargins,
		bookTitle:          bookTitle,
		runningHeaders:     opts.RunningHeaders,
		headerStyle:        opts.HeaderStyle,
		pageNumberPosition: opts.PageNumberPosition,
		dropCap:            opts.DropCap,
		dropCapLines:       dropCapLines,
		chapterStartsRecto: opts.ChapterStartsRecto,
		tocEntries:         []tocEntry{},
	}
}

// getMargins returns left and right margins for the current page (handles mirroring)
func (p *printPDFWriter) getMargins() (left, right float64) {
	if !p.mirroredMargins {
		// No mirroring - use average
		avg := (p.gutterMargin + p.outerMargin) / 2
		return avg, avg
	}
	// Mirrored margins: odd pages have gutter on LEFT, even pages have gutter on RIGHT
	if p.pageNumber%2 == 1 {
		// Odd page (recto/right page) - gutter is on left (spine side)
		return p.gutterMargin, p.outerMargin
	}
	// Even page (verso/left page) - gutter is on right (spine side)
	return p.outerMargin, p.gutterMargin
}

func (p *printPDFWriter) textWidth() float64 {
	left, right := p.getMargins()
	return p.pageWidth - left - right
}

func (p *printPDFWriter) charsPerLine() int {
	charWidth := p.fontSize * 0.52
	return int(p.textWidth() / charWidth)
}

func (p *printPDFWriter) newPage() {
	if p.currentPage.Len() > 0 {
		p.pageContents = append(p.pageContents, p.currentPage.String())
	}
	p.currentPage = &strings.Builder{}
	p.pageNumber++
	p.currentY = p.pageHeight - p.topMargin
}

// ensureRectoPage ensures the next content starts on an odd (right-hand) page
func (p *printPDFWriter) ensureRectoPage() {
	// If we're on an even page, add a blank page
	if p.pageNumber > 0 && p.pageNumber%2 == 0 {
		p.newPage() // Add blank verso page
	}
}

func (p *printPDFWriter) writeLine(text string, fontSize float64, bold bool) {
	if p.currentY-p.lineHeight < p.bottomMargin {
		p.newPage()
	}

	escaped := escapePDFString(text)
	fontName := "/F1"
	if bold {
		fontName = "/F2"
	}

	left, _ := p.getMargins()
	p.currentPage.WriteString(fmt.Sprintf("BT\n%s %.1f Tf\n%.2f %.2f Td\n(%s) Tj\nET\n",
		fontName, fontSize, left, p.currentY, escaped))
	p.currentY -= p.lineHeight
}

func (p *printPDFWriter) writeLineCentered(text string, fontSize float64, bold bool) {
	if p.currentY-p.lineHeight < p.bottomMargin {
		p.newPage()
	}

	escaped := escapePDFString(text)
	fontName := "/F1"
	if bold {
		fontName = "/F2"
	}

	// Estimate text width and center it
	textWidthPts := float64(len(text)) * fontSize * 0.52
	left, right := p.getMargins()
	availWidth := p.pageWidth - left - right
	xPos := left + (availWidth-textWidthPts)/2
	if xPos < left {
		xPos = left
	}

	p.currentPage.WriteString(fmt.Sprintf("BT\n%s %.1f Tf\n%.2f %.2f Td\n(%s) Tj\nET\n",
		fontName, fontSize, xPos, p.currentY, escaped))
	p.currentY -= p.lineHeight
}

func (p *printPDFWriter) writeChapterTitle(title string) {
	// Position title roughly 1/3 down the page for chapter openings
	targetY := p.pageHeight * 0.7
	if p.currentY > targetY {
		p.currentY = targetY
	}

	titleSize := p.fontSize * 1.8
	p.writeLineCentered(title, titleSize, true)
	p.currentY -= p.lineHeight * 2 // Extra space after chapter title
}

func (p *printPDFWriter) writeParagraph(text string, isFirst bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}

	charsPerLine := p.charsPerLine()
	words := strings.Fields(text)
	var line strings.Builder
	lineNum := 0

	left, _ := p.getMargins()
	indent := p.paragraphIndent

	// First paragraph after chapter title: no indent (Reedsy style)
	if isFirst {
		indent = 0
	}

	for _, word := range words {
		if line.Len() == 0 {
			line.WriteString(word)
		} else if line.Len()+1+len(word) <= charsPerLine {
			line.WriteString(" ")
			line.WriteString(word)
		} else {
			// Write the line
			if p.currentY-p.lineHeight < p.bottomMargin {
				p.newPage()
				left, _ = p.getMargins()
			}
			xPos := left
			if lineNum == 0 {
				xPos += indent
			}
			escaped := escapePDFString(line.String())
			p.currentPage.WriteString(fmt.Sprintf("BT\n/F1 %.1f Tf\n%.2f %.2f Td\n(%s) Tj\nET\n",
				p.fontSize, xPos, p.currentY, escaped))
			p.currentY -= p.lineHeight
			lineNum++
			line.Reset()
			line.WriteString(word)
		}
	}

	if line.Len() > 0 {
		if p.currentY-p.lineHeight < p.bottomMargin {
			p.newPage()
			left, _ = p.getMargins()
		}
		xPos := left
		if lineNum == 0 {
			xPos += indent
		}
		escaped := escapePDFString(line.String())
		p.currentPage.WriteString(fmt.Sprintf("BT\n/F1 %.1f Tf\n%.2f %.2f Td\n(%s) Tj\nET\n",
			p.fontSize, xPos, p.currentY, escaped))
		p.currentY -= p.lineHeight
	}

	// Paragraph spacing - small gap
	p.currentY -= p.lineHeight * 0.3
}

func (p *printPDFWriter) writeHalfTitlePage(title, author string) {
	p.newPage()

	// Position title in upper third of page
	p.currentY = p.pageHeight * 0.65

	// Author name in small caps (simulated with smaller font)
	if author != "" {
		authorSize := p.fontSize * 1.1
		p.writeLineCentered(strings.ToUpper(author), authorSize, false)
		p.currentY -= p.lineHeight
	}

	// Book title
	titleSize := p.fontSize * 2.0
	p.writeLineCentered(title, titleSize, true)
}

func (p *printPDFWriter) writeCopyrightPage(copyright string) {
	p.newPage()

	// Copyright page content - positioned near top
	p.currentY = p.pageHeight - p.topMargin - p.lineHeight*2

	content := htmlToPlainParagraphs(copyright)
	paragraphs := strings.Split(content, "\n")
	for i, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para != "" {
			// Center copyright text
			p.writeLineCentered(para, p.fontSize*0.9, false)
			if i < len(paragraphs)-1 {
				p.currentY -= p.lineHeight * 0.5
			}
		}
	}
}

func (p *printPDFWriter) writeTOCPage() {
	if len(p.tocEntries) == 0 {
		return
	}

	p.newPage()

	// TOC header
	p.currentY = p.pageHeight * 0.75
	headerSize := p.fontSize * 1.5
	p.writeLineCentered("Contents", headerSize, true)
	p.currentY -= p.lineHeight * 2

	left, right := p.getMargins()
	availWidth := p.pageWidth - left - right

	for _, entry := range p.tocEntries {
		if p.currentY-p.lineHeight < p.bottomMargin {
			p.newPage()
			left, _ = p.getMargins()
		}

		// Write chapter title on left
		escaped := escapePDFString(entry.title)
		p.currentPage.WriteString(fmt.Sprintf("BT\n/F1 %.1f Tf\n%.2f %.2f Td\n(%s) Tj\nET\n",
			p.fontSize, left, p.currentY, escaped))

		// Write page number on right
		pageStr := fmt.Sprintf("%d", entry.pageNum)
		pageWidth := float64(len(pageStr)) * p.fontSize * 0.52
		xPos := left + availWidth - pageWidth
		escapedPage := escapePDFString(pageStr)
		p.currentPage.WriteString(fmt.Sprintf("BT\n/F1 %.1f Tf\n%.2f %.2f Td\n(%s) Tj\nET\n",
			p.fontSize, xPos, p.currentY, escapedPage))

		p.currentY -= p.lineHeight * 1.2
	}
}

func (p *printPDFWriter) writeChapter(ch ChapterItem, isFirstChapter bool) {
	// Record page for TOC (before potentially adding blank page)
	startPage := p.pageNumber + 1

	// Start chapter on new page
	if p.chapterStartsRecto {
		p.newPage()
		p.ensureRectoPage()
	} else {
		p.newPage()
	}

	// Update current chapter for running headers
	p.currentChapter = ch.Title

	// Add to TOC
	p.tocEntries = append(p.tocEntries, tocEntry{
		title:   ch.Title,
		pageNum: p.pageNumber,
	})
	_ = startPage // may use later for roman numerals

	// Chapter title
	p.writeChapterTitle(ch.Title)

	// Subtitle if present
	if ch.Subtitle != "" {
		subtitleSize := p.fontSize * 1.2
		p.writeLineCentered(ch.Subtitle, subtitleSize, false)
		p.currentY -= p.lineHeight
	}

	// Content - split into paragraphs
	content := htmlToPlainParagraphs(ch.Content)
	paragraphs := strings.Split(content, "\n")
	isFirst := true
	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para != "" {
			p.writeParagraph(para, isFirst)
			isFirst = false
		}
	}
}

func (p *printPDFWriter) addRunningHeadersAndPageNumbers() {
	// This processes all pages after content is generated
	// For now, page numbers and headers are added during build()
}

func (p *printPDFWriter) build() []byte {
	// Finalize current page
	if p.currentPage.Len() > 0 {
		p.pageContents = append(p.pageContents, p.currentPage.String())
	}

	if len(p.pageContents) == 0 {
		p.pageContents = append(p.pageContents, "")
	}

	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")

	numPages := len(p.pageContents)
	var pageObjIDs []int
	nextObjID := 5

	for i := 0; i < numPages; i++ {
		pageObjIDs = append(pageObjIDs, nextObjID)
		nextObjID += 2
	}

	// Object 1: Catalog
	buf.WriteString("1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")

	// Object 2: Pages
	buf.WriteString("2 0 obj\n<< /Type /Pages /Kids [")
	for i, id := range pageObjIDs {
		if i > 0 {
			buf.WriteString(" ")
		}
		buf.WriteString(fmt.Sprintf("%d 0 R", id))
	}
	buf.WriteString(fmt.Sprintf("] /Count %d >>\nendobj\n", numPages))

	// Object 3: Font (Helvetica)
	buf.WriteString("3 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>\nendobj\n")

	// Object 4: Bold Font (Helvetica-Bold)
	buf.WriteString("4 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold >>\nendobj\n")

	// Pages and content streams
	for i, content := range p.pageContents {
		pageObjID := pageObjIDs[i]
		contentObjID := pageObjID + 1
		pageNum := i + 1

		// Add running header and page number to content
		finalContent := content

		// Page number at bottom center
		if p.pageNumberPosition != "" {
			pageNumStr := fmt.Sprintf("%d", pageNum)
			pageNumEscaped := escapePDFString(pageNumStr)
			var pageNumX, pageNumY float64

			left, right := p.gutterMargin, p.outerMargin
			if p.mirroredMargins && pageNum%2 == 0 {
				left, right = p.outerMargin, p.gutterMargin
			}

			switch p.pageNumberPosition {
			case "bottom-center":
				pageNumX = p.pageWidth / 2
				pageNumY = p.bottomMargin / 2
			case "bottom-outside":
				if pageNum%2 == 1 {
					pageNumX = p.pageWidth - right - 10
				} else {
					pageNumX = left + 10
				}
				pageNumY = p.bottomMargin / 2
			case "top-outside":
				if pageNum%2 == 1 {
					pageNumX = p.pageWidth - right - 10
				} else {
					pageNumX = left + 10
				}
				pageNumY = p.pageHeight - p.topMargin/2
			default:
				pageNumX = p.pageWidth / 2
				pageNumY = p.bottomMargin / 2
			}

			finalContent += fmt.Sprintf("BT\n/F1 %.1f Tf\n%.2f %.2f Td\n(%s) Tj\nET\n",
				p.fontSize*0.9, pageNumX, pageNumY, pageNumEscaped)
		}

		// Running headers
		if p.runningHeaders && pageNum > 1 {
			headerY := p.pageHeight - p.topMargin/2
			headerFontSize := p.fontSize * 0.85

			var headerText string
			if pageNum%2 == 0 {
				// Even page (verso): book title
				headerText = p.bookTitle
			} else {
				// Odd page (recto): chapter title
				headerText = p.currentChapter
			}

			if p.headerStyle == "smallcaps" {
				headerText = strings.ToUpper(headerText)
				headerFontSize = p.fontSize * 0.75
			}

			headerEscaped := escapePDFString(headerText)
			headerWidth := float64(len(headerText)) * headerFontSize * 0.52
			headerX := (p.pageWidth - headerWidth) / 2

			fontName := "/F1"
			if p.headerStyle == "italic" {
				// Note: We'd need an italic font, using regular for now
				fontName = "/F1"
			}

			finalContent += fmt.Sprintf("BT\n%s %.1f Tf\n%.2f %.2f Td\n(%s) Tj\nET\n",
				fontName, headerFontSize, headerX, headerY, headerEscaped)
		}

		// Page object
		buf.WriteString(fmt.Sprintf("%d 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %.2f %.2f] /Contents %d 0 R /Resources << /Font << /F1 3 0 R /F2 4 0 R >> >> >>\nendobj\n",
			pageObjID, p.pageWidth, p.pageHeight, contentObjID))

		// Content stream
		buf.WriteString(fmt.Sprintf("%d 0 obj\n<< /Length %d >>\nstream\n%sendstream\nendobj\n",
			contentObjID, len(finalContent), finalContent))
	}

	// Xref and trailer
	buf.WriteString("xref\n")
	buf.WriteString(fmt.Sprintf("0 %d\n", nextObjID))
	buf.WriteString("0000000000 65535 f \n")
	for i := 1; i < nextObjID; i++ {
		buf.WriteString(fmt.Sprintf("%010d 00000 n \n", i*100))
	}
	buf.WriteString("trailer\n")
	buf.WriteString(fmt.Sprintf("<< /Size %d /Root 1 0 R >>\n", nextObjID))
	buf.WriteString("startxref\n")
	buf.WriteString(fmt.Sprintf("%d\n", buf.Len()-20))
	buf.WriteString("%%EOF\n")

	return buf.Bytes()
}

func newPDFWriter(width, height, marginLeft, marginRight, marginTop, marginBottom float64, fontSize int) *pdfWriter {
	return &pdfWriter{
		pageWidth:    width,
		pageHeight:   height,
		marginLeft:   marginLeft,
		marginRight:  marginRight,
		marginTop:    marginTop,
		marginBottom: marginBottom,
		fontSize:     float64(fontSize),
		lineHeight:   float64(fontSize) * 1.5,
		currentY:     height - marginTop,
		currentPage:  &strings.Builder{},
	}
}

func (p *pdfWriter) textWidth() float64 {
	return p.pageWidth - p.marginLeft - p.marginRight
}

func (p *pdfWriter) charsPerLine() int {
	// Approximate characters per line (Helvetica average char width ~ 0.52 * fontSize)
	charWidth := p.fontSize * 0.52
	return int(p.textWidth() / charWidth)
}

func (p *pdfWriter) newPage() {
	if p.currentPage.Len() > 0 {
		p.pageContents = append(p.pageContents, p.currentPage.String())
	}
	p.currentPage = &strings.Builder{}
	p.currentY = p.pageHeight - p.marginTop
}

func escapePDFString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "(", "\\(")
	s = strings.ReplaceAll(s, ")", "\\)")
	return s
}

func (p *pdfWriter) writeLine(text string, fontSize float64, bold bool) {
	if p.currentY-p.lineHeight < p.marginBottom {
		p.newPage()
	}

	escaped := escapePDFString(text)
	fontName := "/F1"
	if bold {
		fontName = "/F2"
	}

	p.currentPage.WriteString(fmt.Sprintf("BT\n%s %.1f Tf\n%.2f %.2f Td\n(%s) Tj\nET\n",
		fontName, fontSize, p.marginLeft, p.currentY, escaped))
	p.currentY -= p.lineHeight
}

func (p *pdfWriter) writeTitle(text string) {
	titleSize := p.fontSize * 1.8
	p.currentY -= p.lineHeight // Extra space before title
	p.writeLine(text, titleSize, true)
	p.currentY -= p.lineHeight * 0.5 // Extra space after title
}

func (p *pdfWriter) writeSubtitle(text string) {
	subtitleSize := p.fontSize * 1.2
	p.writeLine(text, subtitleSize, false)
	p.currentY -= p.lineHeight * 0.3
}

func (p *pdfWriter) writeParagraph(text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}

	// Word wrap
	charsPerLine := p.charsPerLine()
	words := strings.Fields(text)
	var line strings.Builder

	for _, word := range words {
		if line.Len() == 0 {
			line.WriteString(word)
		} else if line.Len()+1+len(word) <= charsPerLine {
			line.WriteString(" ")
			line.WriteString(word)
		} else {
			p.writeLine(line.String(), p.fontSize, false)
			line.Reset()
			line.WriteString(word)
		}
	}

	if line.Len() > 0 {
		p.writeLine(line.String(), p.fontSize, false)
	}

	// Paragraph spacing
	p.currentY -= p.lineHeight * 0.5
}

func (p *pdfWriter) writeChapter(ch ChapterItem) {
	// Start chapter on new page
	p.newPage()

	// Chapter title
	p.writeTitle(ch.Title)

	// Subtitle if present
	if ch.Subtitle != "" {
		p.writeSubtitle(ch.Subtitle)
	}

	p.currentY -= p.lineHeight // Space before content

	// Content - split into paragraphs
	content := htmlToPlainParagraphs(ch.Content)
	paragraphs := strings.Split(content, "\n")
	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para != "" {
			p.writeParagraph(para)
		}
	}
}

func (p *pdfWriter) build() []byte {
	// Finalize current page
	if p.currentPage.Len() > 0 {
		p.pageContents = append(p.pageContents, p.currentPage.String())
	}

	if len(p.pageContents) == 0 {
		p.pageContents = append(p.pageContents, "")
	}

	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")

	numPages := len(p.pageContents)
	var pageObjIDs []int
	nextObjID := 5

	for i := 0; i < numPages; i++ {
		pageObjIDs = append(pageObjIDs, nextObjID)
		nextObjID += 2
	}

	var kidsBuilder strings.Builder
	for i, id := range pageObjIDs {
		if i > 0 {
			kidsBuilder.WriteString(" ")
		}
		kidsBuilder.WriteString(fmt.Sprintf("%d 0 R", id))
	}

	var objects []string
	var offsets []int

	objects = append(objects, "1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")
	objects = append(objects, fmt.Sprintf("2 0 obj\n<< /Type /Pages /Kids [%s] /Count %d >>\nendobj\n",
		kidsBuilder.String(), numPages))
	objects = append(objects, "3 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>\nendobj\n")
	objects = append(objects, "4 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold /Encoding /WinAnsiEncoding >>\nendobj\n")

	for i, content := range p.pageContents {
		pageID := pageObjIDs[i]
		contentID := pageID + 1

		pageObj := fmt.Sprintf("%d 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %.2f %.2f] /Contents %d 0 R /Resources << /Font << /F1 3 0 R /F2 4 0 R >> >> >>\nendobj\n",
			pageID, p.pageWidth, p.pageHeight, contentID)
		objects = append(objects, pageObj)

		contentObj := fmt.Sprintf("%d 0 obj\n<< /Length %d >>\nstream\n%s\nendstream\nendobj\n",
			contentID, len(content), content)
		objects = append(objects, contentObj)
	}

	currentOffset := buf.Len()
	for _, obj := range objects {
		offsets = append(offsets, currentOffset)
		buf.WriteString(obj)
		currentOffset = buf.Len()
	}

	xrefOffset := buf.Len()
	buf.WriteString("xref\n")
	buf.WriteString(fmt.Sprintf("0 %d\n", len(objects)+1))
	buf.WriteString("0000000000 65535 f \n")
	for _, offset := range offsets {
		buf.WriteString(fmt.Sprintf("%010d 00000 n \n", offset))
	}

	buf.WriteString("trailer\n")
	buf.WriteString(fmt.Sprintf("<< /Size %d /Root 1 0 R >>\n", len(objects)+1))
	buf.WriteString("startxref\n")
	buf.WriteString(fmt.Sprintf("%d\n", xrefOffset))
	buf.WriteString("%%EOF\n")

	return buf.Bytes()
}

// generatePDF creates a properly formatted multi-page PDF (Letter size 8.5x11)
func (a *App) generatePDF(book BookData, options PDFOptions) []byte {
	// Letter size: 8.5 x 11 inches = 612 x 792 points
	pageWidth := 612.0
	pageHeight := 792.0

	// 1 inch margins
	marginLeft := 72.0
	marginRight := 72.0
	marginTop := 72.0
	marginBottom := 72.0

	pdf := newPDFWriter(pageWidth, pageHeight, marginLeft, marginRight, marginTop, marginBottom, options.FontSize)

	// Title page
	pdf.currentY = pageHeight/2 + 50
	titleSize := float64(options.FontSize) * 2.5
	pdf.writeLine(book.Metadata.Title, titleSize, true)
	pdf.currentY -= pdf.lineHeight * 2
	if book.Metadata.Author != "" {
		pdf.writeLine("by "+book.Metadata.Author, float64(options.FontSize)*1.3, false)
	}
	if book.Metadata.Publisher != "" {
		pdf.currentY -= pdf.lineHeight
		pdf.writeLine(book.Metadata.Publisher, float64(options.FontSize), false)
	}

	// Copyright page
	if options.IncludeCopyright && book.Copyright != "" {
		pdf.newPage()
		content := htmlToPlainParagraphs(book.Copyright)
		for _, para := range strings.Split(content, "\n") {
			para = strings.TrimSpace(para)
			if para != "" {
				pdf.writeParagraph(para)
			}
		}
	}

	// Front matter
	if options.IncludeFrontMatter {
		for _, ch := range book.FrontMatter {
			pdf.writeChapter(ch)
		}
	}

	// Body chapters
	for _, ch := range book.Body {
		pdf.writeChapter(ch)
	}

	// Back matter
	if options.IncludeBackMatter {
		for _, ch := range book.BackMatter {
			pdf.writeChapter(ch)
		}
	}

	return pdf.build()
}

// generatePrintPDF creates a print-ready PDF with custom trim size and professional formatting
func (a *App) generatePrintPDF(book BookData, options PrintPDFOptions) []byte {
	// Create print PDF writer with all options
	pdf := newPrintPDFWriter(options, book.Metadata.Title)

	// === FRONT MATTER ===

	// Half-title page (optional) - recto
	if options.GenerateHalfTitle {
		pdf.writeHalfTitlePage(book.Metadata.Title, book.Metadata.Author)
	}

	// Copyright page - verso (even page, back of half-title)
	if options.IncludeCopyright && book.Copyright != "" {
		pdf.writeCopyrightPage(book.Copyright)
	}

	// User's front matter chapters
	if options.IncludeFrontMatter {
		for i, ch := range book.FrontMatter {
			pdf.writeChapter(ch, i == 0)
		}
	}

	// === BODY CHAPTERS ===
	isFirstBody := true
	for _, ch := range book.Body {
		pdf.writeChapter(ch, isFirstBody)
		isFirstBody = false
	}

	// === BACK MATTER ===
	if options.IncludeBackMatter {
		for _, ch := range book.BackMatter {
			pdf.writeChapter(ch, false)
		}
	}

	// Build and return the PDF
	// Note: TOC generation would require a two-pass approach since we need page numbers
	// For now, TOC is tracked but not inserted (would require rewriting page order)
	return pdf.build()
}

func generateUUID() string {
	// Simple UUID-like string for EPUB
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		time.Now().UnixNano()&0xFFFFFFFF,
		time.Now().UnixNano()>>32&0xFFFF,
		0x4000|(time.Now().UnixNano()>>48&0x0FFF),
		0x8000|(time.Now().UnixNano()>>60&0x3FFF),
		time.Now().UnixNano()&0xFFFFFFFFFFFF)
}

func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
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
	Type       string             `json:"type"` // "book" | "universe"
	Path       string             `json:"path"`
	Name       string             `json:"name"`
	LastOpened string             `json:"lastOpened"` // ISO 8601 timestamp
	Stats      RecentProjectStats `json:"stats"`
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
// mode: "line_edit" | "expand" | "smooth"
// styleOptionsJson: JSON string of WritingStyleOptions (for expand/smooth modes)
func (a *App) RewriteText(html string, mode string, styleOptionsJson string) AIRewriteResult {
	logAI("========== RewriteText START ==========")
	logAI("mode=%s ai_mode=%s provider=%s", mode, a.settings.AIMode, a.settings.AIProvider)
	logAIContent("INPUT_HTML", html)

	if !a.settings.AIEnabled {
		logAI("ERROR: AI features disabled")
		return AIRewriteResult{Error: "AI features are disabled — enable them in App Settings"}
	}
	if mode == "" {
		mode = "line_edit"
	}

	// Parse style options if provided
	var styleOpts *WritingStyleOptions
	if styleOptionsJson != "" {
		styleOpts = &WritingStyleOptions{}
		if err := json.Unmarshal([]byte(styleOptionsJson), styleOpts); err != nil {
			styleOpts = nil // Ignore invalid JSON, use defaults
		}
	}

	// Diff format: for targeted per-paragraph edits, ask the model to return ONLY
	// changed paragraphs. This cuts output tokens by ~80% for typical chapters.
	useDiffFormat := mode == "line_edit" || mode == "smooth"

	system := buildSystemPrompt(mode, a.settings.ProseGuide, styleOpts)
	var userMsg string

	switch {
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
		case "gemini":
			res = a.callGemini(system, userMsg)
		case "grok":
			res = a.callGrok(system, userMsg)
		default:
			return AIRewriteResult{Error: "unknown provider: " + a.settings.AIProvider}
		}
	}

	// If the model returned diff format, reconstruct full HTML before handing back.
	if res.Error == "" && useDiffFormat {
		res.Result = applyDiffResponse(html, res.Result)
	}

	// Log the result
	if res.Error != "" {
		logAI("ERROR: %s", res.Error)
	} else {
		logAIContent("OUTPUT_RESULT", res.Result)
	}
	logAI("========== RewriteText END ==========")
	return res
}

// RewriteTextCustom applies a custom user prompt to rewrite text.
func (a *App) RewriteTextCustom(text string, prompt string) AIRewriteResult {
	logAI("========== RewriteTextCustom START ==========")
	logAI("prompt=%s ai_mode=%s provider=%s", prompt, a.settings.AIMode, a.settings.AIProvider)
	logAIContent("INPUT_TEXT", text)

	if !a.settings.AIEnabled {
		logAI("ERROR: AI features disabled")
		return AIRewriteResult{Error: "AI features are disabled — enable them in App Settings"}
	}
	if prompt == "" {
		logAI("ERROR: no prompt provided")
		return AIRewriteResult{Error: "no prompt provided"}
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
		case "gemini":
			res = a.callGemini(system, userMsg)
		case "grok":
			res = a.callGrok(system, userMsg)
		default:
			logAI("ERROR: unknown provider: %s", a.settings.AIProvider)
			return AIRewriteResult{Error: "unknown provider: " + a.settings.AIProvider}
		}
	}

	// Log the result
	if res.Error != "" {
		logAI("ERROR: %s", res.Error)
	} else {
		logAIContent("OUTPUT_RESULT", res.Result)
	}
	logAI("========== RewriteTextCustom END ==========")
	return res
}

// InlineGenerateRequest contains the context for inline AI generation
type InlineGenerateRequest struct {
	Instruction   string   `json:"instruction"`    // User's prompt
	BeforeContext string   `json:"before_context"` // Text before cursor (2-3 paragraphs)
	AfterContext  string   `json:"after_context"`  // Text after cursor (1-2 paragraphs)
	Characters    []string `json:"characters"`     // Character names from story bible
	ChapterTitle  string   `json:"chapter_title"`  // Current chapter title
}

// GenerateInlineContent generates new content based on context and instruction
func (a *App) GenerateInlineContent(req InlineGenerateRequest) AIRewriteResult {
	if !a.settings.AIEnabled {
		return AIRewriteResult{Error: "AI features are disabled — enable them in App Settings"}
	}
	if req.Instruction == "" {
		return AIRewriteResult{Error: "no instruction provided"}
	}

	// Build character context
	charContext := ""
	if len(req.Characters) > 0 {
		charContext = fmt.Sprintf("\n\nCharacters in this story: %s", strings.Join(req.Characters, ", "))
	}

	// Build style guide section if configured
	styleBlock := ""
	if a.settings.ProseGuide != "" {
		styleBlock = "\n\nSTYLE GUIDE — match the rhythm, vocabulary, and voice of these examples:\n---\n" + a.settings.ProseGuide + "\n---"
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
		case "gemini":
			res = a.callGemini(system, userMsg)
		case "grok":
			res = a.callGrok(system, userMsg)
		default:
			return AIRewriteResult{Error: "unknown provider: " + a.settings.AIProvider}
		}
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
		"--allowedTools", "",
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

func buildSystemPrompt(mode, proseGuide string, styleOpts *WritingStyleOptions) string {
	styleBlock := ""
	if proseGuide != "" {
		styleBlock = "\n\nSTYLE GUIDE — match the rhythm, vocabulary, and voice of these examples:\n---\n" + proseGuide + "\n---"
	}
	banned := "BANNED words and phrases: tapestry, testament, navigate, delve, underscore, myriad, realm, crucial, pivotal, journey, beacon, vibrant, game-changer"

	aiTellBans := `ABSOLUTELY FORBIDDEN — AI TELL CONSTRUCTIONS:
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
11. Break grammar rules intentionally where rhythm demands it. Fragments are allowed. Sentences can start with And or But. Comma splices are permitted for pacing. Grammatical correctness is not the goal. The sentence is the goal.`

	// Build style mixer instructions for expand/smooth modes
	styleMixerBlock := buildStyleMixerInstructions(styleOpts)

	switch mode {
	case "expand":
		base := `You are a literary prose writer. Expand and enrich the provided HTML text.

Rules:
- Match the existing POV depth, tense, and voice exactly
- Do not introduce new plot events or characters
- Preserve paragraph breaks — return one <p> element per original paragraph (may be longer)
- ` + banned + `

` + aiTellBans + styleBlock

		if styleMixerBlock != "" {
			base += "\n\nSTYLE PREFERENCES (follow these carefully):\n" + styleMixerBlock
		} else {
			base += "\n- Flesh out thin paragraphs — add physical sensation, setting detail, internal thought\n- Show don't tell: replace summary with scene"
		}

		return base + `

CRITICAL: Return ONLY the rewritten HTML content using <p> tags. Do not include any instructions, explanations, system prompts, or meta-commentary. Output raw HTML only.`

	case "smooth":
		base := `You are a line editor focused on flow and rhythm. Smooth the provided HTML text.

Rules:
- Eliminate word repetition within paragraphs (same word used 2+ times nearby)
- Improve sentence-to-sentence transitions
- Vary sentence openings — avoid starting consecutive sentences the same way
- Minimal changes — improve flow without changing meaning or voice
- Preserve paragraph breaks — return one <p> element per original paragraph
- ` + banned + `

` + aiTellBans + styleBlock

		if styleMixerBlock != "" {
			base += "\n\nSTYLE PREFERENCES (follow these carefully):\n" + styleMixerBlock
		}

		return base + `

CRITICAL: Return ONLY the rewritten HTML content using <p> tags. Do not include any instructions, explanations, system prompts, or meta-commentary. Output raw HTML only.`

	default: // "line_edit"
		base := `You are a skilled literary prose editor. Rewrite the provided HTML text, preserving all narrative content, characters, events, and dialogue meaning exactly.

Rewriting rules:
- Vary sentence rhythm: mix short, punchy sentences with longer, flowing ones
- Use strong, precise, concrete words — avoid vague abstractions
- Write in the same tense and POV as the original
- Preserve paragraph breaks — return one <p> element per original paragraph
- ` + banned + `

` + aiTellBans
		if proseGuide != "" {
			base = `You are a skilled literary prose editor. Rewrite the provided HTML text to match the style shown below, while preserving all narrative content exactly.` + styleBlock + `

Rewriting rules:
- Match the rhythm, cadence, and sentence variety of the style examples above
- Vary sentence length as in the examples
- Preserve all story facts: names, places, events, exact dialogue content
- Preserve paragraph breaks — return one <p> element per original paragraph
- ` + banned + `

` + aiTellBans
		}
		return base + `

CRITICAL: Return ONLY the rewritten HTML content using <p> tags. Do not include any instructions, explanations, system prompts, or meta-commentary. Output raw HTML only.`
	}
}

// buildStyleMixerInstructions converts WritingStyleOptions to prompt instructions
func buildStyleMixerInstructions(opts *WritingStyleOptions) string {
	if opts == nil {
		return ""
	}

	intensityWords := []string{"", "subtle", "moderate", "heavy"}
	var instructions []string

	// Metaphors - CRITICAL: these are major AI tells
	if opts.Metaphors == 0 {
		instructions = append(instructions, "- METAPHORS: ABSOLUTELY FORBIDDEN. Never add any new metaphors. Do not write phrases like 'was a [noun]', 'became a [noun]', or any figurative comparisons. Only preserve metaphors that already exist word-for-word in the source text.")
	} else if opts.Metaphors > 0 && opts.Metaphors <= 3 {
		instructions = append(instructions, fmt.Sprintf("- Add %s use of metaphors (figurative comparisons)", intensityWords[opts.Metaphors]))
	}

	// Similes - CRITICAL: these are major AI tells
	if opts.Similes == 0 {
		instructions = append(instructions, "- SIMILES: ABSOLUTELY FORBIDDEN. Never add any new similes. Do not write 'like a...', 'as if...', 'as though...', or any like/as comparisons. Only preserve similes that already exist word-for-word in the source text.")
	} else if opts.Similes > 0 && opts.Similes <= 3 {
		instructions = append(instructions, fmt.Sprintf("- Add %s use of similes (like/as comparisons)", intensityWords[opts.Similes]))
	}

	// Sensory Detail
	if opts.SensoryDetail == 0 {
		instructions = append(instructions, "- SENSORY DETAILS: Do not add new sensory descriptions. Preserve only what exists in the source.")
	} else if opts.SensoryDetail > 0 && opts.SensoryDetail <= 3 {
		instructions = append(instructions, fmt.Sprintf("- Add %s sensory details (sight, sound, smell, touch, taste)", intensityWords[opts.SensoryDetail]))
	}

	// Internal Thought
	if opts.InternalThought == 0 {
		instructions = append(instructions, "- INTERNAL THOUGHT: Do not add character introspection or internal monologue. Preserve only what exists in the source.")
	} else if opts.InternalThought > 0 && opts.InternalThought <= 3 {
		instructions = append(instructions, fmt.Sprintf("- Add %s internal thought and character introspection", intensityWords[opts.InternalThought]))
	}

	// Dialogue
	if opts.Dialogue == 0 {
		instructions = append(instructions, "- DIALOGUE: Do not expand or add dialogue. Preserve only what exists in the source.")
	} else if opts.Dialogue > 0 && opts.Dialogue <= 3 {
		word := intensityWords[opts.Dialogue]
		instructions = append(instructions, fmt.Sprintf("- %s%s expansion of dialogue and conversation", strings.ToUpper(word[:1]), word[1:]))
	}

	// Action
	if opts.Action == 0 {
		instructions = append(instructions, "- ACTION: Do not add physical action or movement beats. Preserve only what exists in the source.")
	} else if opts.Action > 0 && opts.Action <= 3 {
		instructions = append(instructions, fmt.Sprintf("- Add %s physical action and movement beats", intensityWords[opts.Action]))
	}

	// Description
	if opts.Description == 0 {
		instructions = append(instructions, "- DESCRIPTION: Do not add setting description or atmosphere. Preserve only what exists in the source.")
	} else if opts.Description > 0 && opts.Description <= 3 {
		instructions = append(instructions, fmt.Sprintf("- Add %s setting description and atmosphere", intensityWords[opts.Description]))
	}

	// Pacing
	if opts.Pacing == 0 {
		instructions = append(instructions, "- Keep sentence rhythm uniform")
	} else if opts.Pacing > 0 && opts.Pacing <= 3 {
		switch opts.Pacing {
		case 1:
			instructions = append(instructions, "- Slight variation in sentence rhythm")
		case 2:
			instructions = append(instructions, "- Moderate variation in sentence rhythm — mix short and long sentences")
		case 3:
			instructions = append(instructions, "- Heavy variation in sentence rhythm — dramatic contrasts between punchy and flowing sentences")
		}
	}

	return strings.Join(instructions, "\n")
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

func (a *App) callGemini(system, userMsg string) AIRewriteResult {
	model := a.settings.AIModel
	if model == "" {
		model = "gemini-1.5-pro"
	}

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

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, a.settings.AIAPIKey)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return AIRewriteResult{Error: err.Error()}
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return AIRewriteResult{Error: "Failed to connect to Gemini API: " + err.Error()}
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

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
		return AIRewriteResult{Error: "Failed to parse Gemini response"}
	}
	if result.Error != nil {
		return AIRewriteResult{Error: result.Error.Message}
	}
	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return AIRewriteResult{Error: "Empty response from Gemini"}
	}
	return AIRewriteResult{Result: strings.TrimSpace(result.Candidates[0].Content.Parts[0].Text)}
}

func (a *App) callGrok(system, userMsg string) AIRewriteResult {
	model := a.settings.AIModel
	if model == "" {
		model = "grok-2"
	}

	// Grok uses OpenAI-compatible API format
	reqBody, _ := json.Marshal(map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": userMsg},
		},
	})

	req, err := http.NewRequest("POST", "https://api.x.ai/v1/chat/completions", bytes.NewBuffer(reqBody))
	if err != nil {
		return AIRewriteResult{Error: err.Error()}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.settings.AIAPIKey)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return AIRewriteResult{Error: "Failed to connect to Grok API: " + err.Error()}
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
		return AIRewriteResult{Error: "Failed to parse Grok response"}
	}
	if result.Error != nil {
		return AIRewriteResult{Error: result.Error.Message}
	}
	if len(result.Choices) == 0 {
		return AIRewriteResult{Error: "Empty response from Grok"}
	}
	return AIRewriteResult{Result: strings.TrimSpace(result.Choices[0].Message.Content)}
}

func (a *App) writeBook(book BookData, path string) SaveResult {
	// Create backup of existing file before overwriting
	if err := a.createBackup(path); err != nil {
		// Log but don't fail the save - backup is best-effort
		fmt.Printf("Backup warning: %v\n", err)
	}

	book.Metadata.Modified = time.Now().Format(time.RFC3339)

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	type entry struct {
		Title    string `json:"title"`
		Subtitle string `json:"subtitle,omitempty"`
		Type     string `json:"type"`
		File     string `json:"file"`
	}
	type manifest struct {
		Version      string              `json:"version"`
		AppVersion   string              `json:"app_version"`
		Metadata     Metadata            `json:"metadata"`
		FrontMatter  []entry             `json:"front_matter"`
		Body         []entry             `json:"body"`
		BackMatter   []entry             `json:"back_matter"`
		WritingGoals WritingGoals        `json:"writing_goals,omitempty"`
		StyleOptions WritingStyleOptions `json:"style_options,omitempty"`
		IsIndexed    bool                `json:"is_indexed,omitempty"`
		LastIndexed  string              `json:"last_indexed,omitempty"`
	}

	mf := manifest{
		Version:      "2.0",
		AppVersion:   AppVersion,
		Metadata:     book.Metadata,
		FrontMatter:  []entry{},
		IsIndexed:    book.IsIndexed,
		LastIndexed:  book.LastIndexed,
		Body:         []entry{},
		BackMatter:   []entry{},
		WritingGoals: book.WritingGoals,
		StyleOptions: book.StyleOptions,
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

	// Save beat_sheet.json if there are beats
	if len(book.BeatSheet.Beats) > 0 {
		beatJSON, _ := json.MarshalIndent(book.BeatSheet, "", "  ")
		if err := addEntry("beat_sheet.json", string(beatJSON)); err != nil {
			return SaveResult{Success: false, Error: err.Error()}
		}
	}

	// Save foreshadowing.json if there are items
	if len(book.ForeshadowingLedger.Items) > 0 {
		foreshadowJSON, _ := json.MarshalIndent(book.ForeshadowingLedger, "", "  ")
		if err := addEntry("foreshadowing.json", string(foreshadowJSON)); err != nil {
			return SaveResult{Success: false, Error: err.Error()}
		}
	}

	// Save knowledge_matrix.json if there are secrets or entries
	if len(book.KnowledgeMatrix.Secrets) > 0 || len(book.KnowledgeMatrix.Entries) > 0 {
		matrixJSON, _ := json.MarshalIndent(book.KnowledgeMatrix, "", "  ")
		if err := addEntry("knowledge_matrix.json", string(matrixJSON)); err != nil {
			return SaveResult{Success: false, Error: err.Error()}
		}
	}

	for i, item := range book.FrontMatter {
		file := fmt.Sprintf("front_matter/%03d.html", i)
		if err := addEntry(file, item.Content); err != nil {
			return SaveResult{Success: false, Error: err.Error()}
		}
		mf.FrontMatter = append(mf.FrontMatter, entry{Title: item.Title, Subtitle: item.Subtitle, Type: item.Type, File: file})
	}
	for i, item := range book.Body {
		file := fmt.Sprintf("body/%03d.html", i)
		if err := addEntry(file, item.Content); err != nil {
			return SaveResult{Success: false, Error: err.Error()}
		}
		mf.Body = append(mf.Body, entry{Title: item.Title, Subtitle: item.Subtitle, Type: item.Type, File: file})
	}
	for i, item := range book.BackMatter {
		file := fmt.Sprintf("back_matter/%03d.html", i)
		if err := addEntry(file, item.Content); err != nil {
			return SaveResult{Success: false, Error: err.Error()}
		}
		mf.BackMatter = append(mf.BackMatter, entry{Title: item.Title, Subtitle: item.Subtitle, Type: item.Type, File: file})
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
