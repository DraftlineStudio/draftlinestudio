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
	// MergeRules stores user-confirmed "same person" name pairs, re-applied
	// on every re-index
	MergeRules []MergeRule `json:"merge_rules,omitempty"`
	// Decisions stores author-confirmed admission choices using names rather
	// than volatile entity IDs so they survive a fresh resolution pass.
	Decisions []EntityDecision `json:"decisions,omitempty"`
	// LastResolved is the timestamp of the last entity resolution run
	LastResolved string `json:"last_resolved,omitempty"`
	// Version tracks schema version for future migrations
	Version int `json:"version,omitempty"`
}

// MentionRecord is the stored form of a name mention.
type MentionRecord struct {
	ID                   string `json:"id"`
	Text                 string `json:"text"`        // Raw text as it appears
	SentenceID           string `json:"sentence_id"` // ID of containing sentence
	Chapter              int    `json:"chapter"`     // Chapter index
	CharOffset           int    `json:"char_offset"` // Character offset in chapter
	PersonEvidence       bool   `json:"person_evidence,omitempty"`
	NonPersonEvidence    bool   `json:"non_person_evidence,omitempty"`
	StrongPersonEvidence bool   `json:"strong_person_evidence,omitempty"`
}

// EntityRecord is the stored form of a resolved entity.
type EntityRecord struct {
	ID              string   `json:"id"`
	Canonical       string   `json:"canonical"`   // Best display name
	Aliases         []string `json:"aliases"`     // All name variations
	MentionIDs      []string `json:"mention_ids"` // IDs of all mentions
	Confidence      float64  `json:"confidence"`  // Merge confidence
	Titles          []string `json:"titles"`      // Honorifics/titles seen
	Kind            string   `json:"kind,omitempty"`
	DetectionStatus string   `json:"detection_status,omitempty"`
	DetectionScore  float64  `json:"detection_score,omitempty"`

	// Link to Character record (if user has created one)
	CharacterID string `json:"character_id,omitempty"`
}

// SeparatedPairRecord stores a pair of mentions that should not merge.
type SeparatedPairRecord struct {
	MentionID1 string `json:"mention_id_1"`
	MentionID2 string `json:"mention_id_2"`
	Reason     string `json:"reason,omitempty"`
}

// MergeRule declares that two name forms refer to the same person.
// Name-based (not ID-based) so it survives re-indexing after text edits.
type MergeRule struct {
	Name1 string `json:"name_1"`
	Name2 string `json:"name_2"`
}

// EntityDecision records whether the author considers a resolved candidate a
// character. Names contains the canonical form and aliases known when the
// decision was made; a decision is re-applied only when it matches one entity.
type EntityDecision struct {
	Names  []string `json:"names"`
	Status string   `json:"status"` // accepted | rejected
}

// AnalysisData is a future-proof container for various analysis results.
// This allows adding new analysis types without changing the BookData schema.
type AnalysisData struct {
	// EntityResolution stores character/entity clustering results
	EntityResolution *EntityData `json:"entity_resolution,omitempty"`

	// Relationships stores character interaction and relationship data
	Relationships *RelationshipData `json:"relationships,omitempty"`

	// Future analysis types can be added here:
	// PlotAnalysis     *PlotAnalysisData    `json:"plot_analysis,omitempty"`
	// ThemeAnalysis    *ThemeAnalysisData   `json:"theme_analysis,omitempty"`
	// PacingAnalysis   *PacingAnalysisData  `json:"pacing_analysis,omitempty"`

	// Version tracks schema version for migrations
	Version int `json:"version,omitempty"`
}
