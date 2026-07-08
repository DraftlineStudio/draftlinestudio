package types

// AppSettings contains user preferences and configuration.
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

// ClaudeCodeStatus contains the status of the Claude Code CLI installation.
type ClaudeCodeStatus struct {
	Installed     bool   `json:"installed"`
	Authenticated bool   `json:"authenticated"`
	NpmAvailable  bool   `json:"npm_available"`
	Version       string `json:"version"`
	Error         string `json:"error,omitempty"`
}
