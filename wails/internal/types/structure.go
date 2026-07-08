package types

// Beat represents a story beat in the beat sheet (Save the Cat! structure).
type Beat struct {
	ID           string `json:"id"`
	ChapterIndex int    `json:"chapter_index"`
	BeatType     string `json:"beat_type"` // opening_image, theme_stated, catalyst, midpoint, all_is_lost, finale, custom, etc.
	Description  string `json:"description"`
	Notes        string `json:"notes,omitempty"`
}

// BeatSheet contains all story beats for the book.
type BeatSheet struct {
	Beats []Beat `json:"beats"`
}

// ForeshadowingItem tracks a foreshadowing element through plant -> reinforce -> payoff.
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

// ForeshadowingLedger contains all foreshadowing items for the book.
type ForeshadowingLedger struct {
	Items []ForeshadowingItem `json:"items"`
}

// SecretInfo represents a piece of information/secret in the knowledge matrix.
type SecretInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// KnowledgeEntry tracks when a character learns a secret.
type KnowledgeEntry struct {
	SecretID         string `json:"secret_id"`
	CharacterID      string `json:"character_id"`
	LearnsChapter    *int   `json:"learns_chapter,omitempty"`    // When they definitively learn it
	SuspectedChapter *int   `json:"suspected_chapter,omitempty"` // When they start to suspect
}

// KnowledgeMatrix tracks which characters know which secrets at each chapter.
type KnowledgeMatrix struct {
	Secrets []SecretInfo     `json:"secrets"`
	Entries []KnowledgeEntry `json:"entries"`
}
