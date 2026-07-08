package types

// Character represents a character in the story bible.
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
	ChapterMentions map[int]int       `json:"chapter_mentions,omitempty"` // Chapter index -> mention count
	Attributes      map[string]string `json:"attributes,omitempty"`       // Extracted attributes (eye_color, hair_color, etc.)
}

// StoryBible contains characters, plot notes, and timeline for the book.
type StoryBible struct {
	Characters []Character `json:"characters"`
	PlotNotes  string      `json:"plot_notes"`
	Timeline   string      `json:"timeline"`
}
