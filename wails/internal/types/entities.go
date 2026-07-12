package types

// EntityData stores entity resolution results and configuration.
// This is separate from StoryBible to allow independent evolution.
type EntityData struct {
	// Mentions stores all detected name mentions in the text
	Mentions []MentionRecord `json:"mentions,omitempty"`
	// Entities stores resolved entities (clustered mentions)
	Entities []EntityRecord `json:"entities,omitempty"`
	// SeparatedPairs stores mention pairs that should not be merged
	SeparatedPairs []SeparatedPairRecord `json:"separated_pairs,omitempty"`
	// LastResolved is the timestamp of the last entity resolution run
	LastResolved string `json:"last_resolved,omitempty"`
	// Version tracks schema version for future migrations
	Version int `json:"version,omitempty"`
}

// MentionRecord is the stored form of a name mention.
type MentionRecord struct {
	ID         string `json:"id"`
	Text       string `json:"text"`        // Raw text as it appears
	SentenceID string `json:"sentence_id"` // ID of containing sentence
	Chapter    int    `json:"chapter"`     // Chapter index
	CharOffset int    `json:"char_offset"` // Character offset in chapter
}

// EntityRecord is the stored form of a resolved entity.
type EntityRecord struct {
	ID         string   `json:"id"`
	Canonical  string   `json:"canonical"`   // Best display name
	Aliases    []string `json:"aliases"`     // All name variations
	MentionIDs []string `json:"mention_ids"` // IDs of all mentions
	Confidence float64  `json:"confidence"`  // Merge confidence
	Titles     []string `json:"titles"`      // Honorifics/titles seen

	// Link to Character record (if user has created one)
	CharacterID string `json:"character_id,omitempty"`
}

// SeparatedPairRecord stores a pair of mentions that should not merge.
type SeparatedPairRecord struct {
	MentionID1 string `json:"mention_id_1"`
	MentionID2 string `json:"mention_id_2"`
	Reason     string `json:"reason,omitempty"`
}

// AnalysisData is a future-proof container for various analysis results.
// This allows adding new analysis types without changing the BookData schema.
type AnalysisData struct {
	// EntityResolution stores character/entity clustering results
	EntityResolution *EntityData `json:"entity_resolution,omitempty"`

	// Future analysis types can be added here:
	// PlotAnalysis     *PlotAnalysisData    `json:"plot_analysis,omitempty"`
	// ThemeAnalysis    *ThemeAnalysisData   `json:"theme_analysis,omitempty"`
	// DialogueAnalysis *DialogueAnalysisData `json:"dialogue_analysis,omitempty"`
	// PacingAnalysis   *PacingAnalysisData  `json:"pacing_analysis,omitempty"`

	// Version tracks schema version for migrations
	Version int `json:"version,omitempty"`
}
