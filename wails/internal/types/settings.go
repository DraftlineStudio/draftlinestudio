package types

// AppSettings contains user preferences and configuration.
type AppSettings struct {
	// Application
	DefaultAuthor           string   `json:"default_author"`
	DefaultPublisher        string   `json:"default_publisher"`
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
	PlotWalkerEnabled       bool     `json:"plot_walker_enabled"`
	AnalysisEnabled         bool     `json:"analysis_enabled"`
	// Read Aloud is opt-in: enabling it offers a one-time local voice-model
	// download; synthesis then runs entirely offline in the webview.
	ReadAloudEnabled bool    `json:"read_aloud_enabled"`
	ReadAloudVoice   string  `json:"read_aloud_voice"`
	ReadAloudSpeed   float64 `json:"read_aloud_speed"`
	// ReadAloudDevice: "wasm" (default, works everywhere) | "webgpu"
	// (opt-in fp32 fast path). ReadAloudThreads: "auto" (default; uses all
	// but one core when cross-origin isolation is active) | "single".
	ReadAloudDevice  string `json:"read_aloud_device"`
	ReadAloudThreads string `json:"read_aloud_threads"`
	// AnalysisCPUProfile controls each analysis stage's local worker-pool size:
	// "adaptive" | "gentle" | "balanced" | "fast". It never changes the
	// process-wide Go scheduler.
	AnalysisCPUProfile string `json:"analysis_cpu_profile"`
	// CharactersLaneView remembers the codex lane style: "grid" | "heat" | "weave".
	CharactersLaneView string `json:"characters_lane_view"`
	// AI
	AIEnabled  bool   `json:"ai_enabled"`
	AIMode     string `json:"ai_mode"`     // "claudecode" | "codex" | "api" | "local"
	AIProvider string `json:"ai_provider"` // "claude" | "openai" | ""
	// AIAPIKey is legacy: keys now live in the OS keyring. The tag is kept
	// (with omitempty) so old settings.json files can still be read and
	// migrated; it is never returned to the frontend or written back once the
	// keyring holds the key.
	AIAPIKey string `json:"ai_api_key,omitempty"`
	// HasAPIKey tells the frontend whether a key is stored, without exposing it.
	HasAPIKey bool `json:"has_api_key"`
	// AIDebugLogging opts in to writing prompts/manuscript text to local logs.
	AIDebugLogging  bool   `json:"ai_debug_logging"`
	AIModel         string `json:"ai_model"`
	AILocalEndpoint string `json:"ai_local_endpoint"` // e.g. http://localhost:11434/v1
	AILocalModel    string `json:"ai_local_model"`
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
}

// ClaudeCodeStatus contains the status of the Claude Code CLI installation.
type ClaudeCodeStatus struct {
	Installed     bool   `json:"installed"`
	Authenticated bool   `json:"authenticated"`
	NpmAvailable  bool   `json:"npm_available"`
	Version       string `json:"version"`
	Error         string `json:"error,omitempty"`
}
