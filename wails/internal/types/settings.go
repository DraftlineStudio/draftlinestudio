package types

import (
	"maps"
	"slices"
)

// AppSettings contains user preferences and configuration.
type AppSettings struct {
	// Application
	DefaultAuthor    string `json:"default_author"`
	DefaultPublisher string `json:"default_publisher"`
	// DefaultCopyright is retired. It was a template advertising [YEAR] and
	// [AUTHOR] placeholders that nothing substituted, and it was never applied
	// to a new book either; the copyright page is now written from the book's
	// own record on the Book & Editions screen. The field stays so that a
	// settings file written before the change still parses and a round-trip
	// through the settings screen does not silently delete what it holds.
	DefaultCopyright        string   `json:"default_copyright"`
	DefaultSaveDir          string   `json:"default_save_dir"`
	DarkMode                bool     `json:"dark_mode"`
	ThemeMode               string   `json:"theme_mode"` // "light" | "dark" | "auto"
	AutoThemeUseManual      bool     `json:"auto_theme_use_manual"`
	AutoThemeDawn           string   `json:"auto_theme_dawn"` // "HH:MM" format
	AutoThemeDusk           string   `json:"auto_theme_dusk"` // "HH:MM" format
	ActivityAutoSaveEnabled bool     `json:"activity_autosave_enabled"`
	CustomDictionary        []string `json:"custom_dictionary,omitempty"`
	SpellCheckEnabled       bool     `json:"spell_check_enabled"`
	GrammarCheckEnabled     bool     `json:"grammar_check_enabled"`
	CastEnabled             bool     `json:"cast_enabled"`
	StoryBibleEnabled       bool     `json:"story_bible_enabled"`
	AnalysisEnabled         bool     `json:"analysis_enabled"`
	// Read Aloud is opt-in: enabling it offers a one-time local voice-model
	// download; synthesis then runs entirely offline in the webview.
	ReadAloudEnabled bool    `json:"read_aloud_enabled"`
	ReadAloudVoice   string  `json:"read_aloud_voice"`
	ReadAloudSpeed   float64 `json:"read_aloud_speed"`
	// ReadAloudDevice is retained for settings-file compatibility. Native is
	// the only supported Read Aloud protocol.
	// (opt-in fp32 fast path). ReadAloudThreads: "auto" (default; uses all
	// but one core when cross-origin isolation is active) | "single".
	ReadAloudDevice  string `json:"read_aloud_device"`
	ReadAloudThreads string `json:"read_aloud_threads"`
	// ReadAloudVolume is the player's master gain, 0–1. Mute is session-only.
	ReadAloudVolume float64 `json:"read_aloud_volume"`
	// ReadAloudGlow toggles the player's glow accents (progress playhead dot
	// and play-button ring).
	ReadAloudGlow bool `json:"read_aloud_glow"`
	// AnalysisCPUProfile controls each analysis stage's local worker-pool size:
	// "adaptive" | "gentle" | "balanced" | "fast". It never changes the
	// process-wide Go scheduler.
	AnalysisCPUProfile string `json:"analysis_cpu_profile"`
	// CharactersLaneView remembers the codex lane style: "grid" | "heat" | "weave".
	CharactersLaneView string `json:"characters_lane_view"`
	// AI
	AIEnabled bool   `json:"ai_enabled"`
	AIMode    string `json:"ai_mode"` // "claudecode" | "codex" | "api" | an AIProviders id
	// AITaskRoutes optionally overrides AIMode for individual editing tasks.
	// Missing keys inherit AIMode. Supported keys are line_edit, copy_edit,
	// expand, smooth, and custom.
	AITaskRoutes map[string]string `json:"ai_task_routes,omitempty"`
	AIProvider   string            `json:"ai_provider"` // "claude" | "openai" | ""
	// AIProviders are endpoints the writer configured themselves. Anything
	// beyond Claude and OpenAI is reached this way rather than being built in.
	AIProviders []AIProvider `json:"ai_providers,omitempty"`
	// AIAPIKey is legacy: keys now live in the OS keyring. The tag is kept
	// (with omitempty) so old settings.json files can still be read and
	// migrated; it is never returned to the frontend or written back once the
	// keyring holds the key.
	AIAPIKey string `json:"ai_api_key,omitempty"`
	// HasAPIKey tells the frontend whether a key is stored, without exposing it.
	HasAPIKey bool `json:"has_api_key"`
	// AIDebugLogging opts in to writing prompts/manuscript text to local logs.
	AIDebugLogging bool   `json:"ai_debug_logging"`
	AIModel        string `json:"ai_model"`
	// AILocalEndpoint and AILocalModel are legacy: the local endpoint is now
	// one AIProviders entry. Kept so older settings files migrate.
	AILocalEndpoint string `json:"ai_local_endpoint,omitempty"`
	AILocalModel    string `json:"ai_local_model,omitempty"`
	ProseGuide      string `json:"prose_guide"`
	// Book defaults
	BookFont          string `json:"book_font"`
	EditorFontSize    string `json:"editor_font_size"`  // "small"|"normal"|"large" (12/14/16px)
	BookFontSize      int    `json:"book_font_size"`    // Export font size in points
	BookLineSpacing   string `json:"book_line_spacing"` // "1.0"|"1.25"|"1.5"|"2.0"
	BookDropCaps      bool   `json:"book_drop_caps"`
	BookTrimSize      string `json:"book_trim_size"` // "6x9"|"5.5x8.5"|"5x8"|"7x10"|"A5"
	SidebarPanelWidth int    `json:"sidebar_panel_width"`
	// SidebarActiveSection remembers which tools-sidebar pane is open
	// ("dashboard", "characters", "ai", …); "" means the sidebar is closed.
	SidebarActiveSection string `json:"sidebar_active_section"`
	// UpdateCheckEnabled lets the app query GitHub releases on launch and
	// hourly to show the subtle title-bar update indicator.
	UpdateCheckEnabled bool `json:"update_check_enabled"`
	// Plugin platform. Plugins are installed machine-wide (one copy per
	// machine); these two maps are the entire per-user footprint: which
	// installed plugins this user has switched on, and each plugin's own
	// settings bag (opaque to the host, keyed by plugin id).
	PluginsEnabled map[string]bool           `json:"plugins_enabled,omitempty"`
	PluginSettings map[string]map[string]any `json:"plugin_settings,omitempty"`
}

// ClaudeCodeStatus contains the status of the Claude Code CLI installation.
type ClaudeCodeStatus struct {
	Installed     bool   `json:"installed"`
	Authenticated bool   `json:"authenticated"`
	NpmAvailable  bool   `json:"npm_available"`
	Version       string `json:"version"`
	Error         string `json:"error,omitempty"`
}

// AIProvider is an OpenAI-compatible endpoint the writer configured. Kind
// "cloud" sends the stored key as a bearer token; "local" sends no credentials.
type AIProvider struct {
	ID       string `json:"id"`
	Nickname string `json:"nickname"`
	Kind     string `json:"kind"` // "cloud" | "local"
	BaseURL  string `json:"base_url"`
	Model    string `json:"model"`
	// APIKey is the fallback for machines with no keyring, mirroring AIAPIKey.
	APIKey string `json:"api_key,omitempty"`
}

// Clone returns a copy of the settings sharing no slice or map with the
// original.
//
// AppSettings travels by value, and a value copy duplicates only the headers
// of its slice and map fields: two holders of a "copy" write into the same
// memory. The backend hands settings to callers that read them without a lock
// while other goroutines edit them, so both directions clone.
func (s AppSettings) Clone() AppSettings {
	s.CustomDictionary = slices.Clone(s.CustomDictionary)
	s.AIProviders = slices.Clone(s.AIProviders)
	s.AITaskRoutes = maps.Clone(s.AITaskRoutes)
	s.PluginsEnabled = maps.Clone(s.PluginsEnabled)
	if s.PluginSettings != nil {
		bags := make(map[string]map[string]any, len(s.PluginSettings))
		for id, bag := range s.PluginSettings {
			bags[id] = maps.Clone(bag)
		}
		s.PluginSettings = bags
	}
	return s
}
