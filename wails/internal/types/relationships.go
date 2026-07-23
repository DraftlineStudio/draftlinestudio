package types

// SceneRecord represents a detected scene or paragraph boundary in the text.
type SceneRecord struct {
	ID           string   `json:"id"`
	ChapterIndex int      `json:"chapter_index"`
	StartOffset  int      `json:"start_offset"`
	EndOffset    int      `json:"end_offset"`
	SceneType    string   `json:"scene_type"` // "paragraph", "scene_break", "chapter"
	CharacterIDs []string `json:"character_ids"`
}

// InteractionRecord represents a detected interaction between characters.
type InteractionRecord struct {
	ID              string   `json:"id"`
	Participants    []string `json:"participants"`
	ChapterIndex    int      `json:"chapter_index"`
	SceneID         string   `json:"scene_id,omitempty"`
	SentenceID      string   `json:"sentence_id,omitempty"`
	InteractionType string   `json:"interaction_type"` // "dialogue", "co_occurrence", "reference"
	DirectedFrom    string   `json:"directed_from,omitempty"`
	DirectedTo      string   `json:"directed_to,omitempty"`
	Confidence      float64  `json:"confidence"`
	TextSnippet     string   `json:"text_snippet,omitempty"`
}

// RelationshipRecord represents an aggregated relationship between two characters.
type RelationshipRecord struct {
	ID               string         `json:"id"`
	Character1ID     string         `json:"character1_id"`
	Character2ID     string         `json:"character2_id"`
	FirstChapter     int            `json:"first_chapter"`
	LastChapter      int            `json:"last_chapter"`
	InteractionCount int            `json:"interaction_count"`
	Strength         float64        `json:"strength"`
	ChapterHistory   []int          `json:"chapter_history"`
	InteractionIDs   []string       `json:"interaction_ids,omitempty"`
	TypeBreakdown    map[string]int `json:"type_breakdown"`
}

// CharacterEvent represents a significant plot point tied to characters.
type CharacterEvent struct {
	ID             string   `json:"id"`
	CharacterIDs   []string `json:"character_ids"`
	ChapterIndex   int      `json:"chapter_index"`
	EventType      string   `json:"event_type"` // "introduction", "meeting", "conflict", "resolution", "death", "custom"
	Description    string   `json:"description"`
	IsAutoDetected bool     `json:"is_auto_detected"`
}

// RelationshipData stores all relationship and interaction analysis.
type RelationshipData struct {
	Scenes        []SceneRecord        `json:"scenes,omitempty"`
	Interactions  []InteractionRecord  `json:"interactions,omitempty"`
	Relationships []RelationshipRecord `json:"relationships,omitempty"`
	Events        []CharacterEvent     `json:"events,omitempty"`
	LastAnalyzed  string               `json:"last_analyzed,omitempty"`
	Version       int                  `json:"version,omitempty"`
}

// RelationshipAnalysisResult contains the result of relationship analysis.
type RelationshipAnalysisResult struct {
	Success            bool     `json:"success"`
	Error              string   `json:"error,omitempty"`
	Book               BookData `json:"book,omitempty"`
	ScenesDetected     int      `json:"scenes_detected"`
	InteractionsFound  int      `json:"interactions_found"`
	RelationshipsBuilt int      `json:"relationships_built"`
}

// CharacterTimelineResult contains timeline data for a character.
type CharacterTimelineResult struct {
	Success     bool                     `json:"success"`
	Error       string                   `json:"error,omitempty"`
	CharacterID string                   `json:"character_id"`
	Events      []CharacterTimelineEvent `json:"events"`
}

// CharacterTimelineEvent is a single event in a character's timeline.
type CharacterTimelineEvent struct {
	Chapter      int      `json:"chapter"`
	EventType    string   `json:"event_type"` // "mention", "dialogue", "interaction", "event"
	Description  string   `json:"description"`
	RelatedChars []string `json:"related_chars,omitempty"`
}
