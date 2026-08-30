// Package entityresolution provides rule-based entity clustering for character names.
// It clusters name variations (e.g., "Ruiz", "Officer Ruiz", "Carlos Ruiz") into
// unified entities while allowing manual splits for disambiguation.
package entityresolution

// Mention represents a single occurrence of a name in the text.
type Mention struct {
	ID                   string `json:"id"`          // Unique mention ID
	Text                 string `json:"text"`        // Raw text as it appears (e.g., "Officer Ruiz")
	SentenceID           string `json:"sentence_id"` // ID of the containing sentence
	Chapter              int    `json:"chapter"`     // Chapter index
	CharOffset           int    `json:"char_offset"` // Character offset within chapter
	PersonEvidence       bool   `json:"person_evidence,omitempty"`
	NonPersonEvidence    bool   `json:"non_person_evidence,omitempty"`
	StrongPersonEvidence bool   `json:"strong_person_evidence,omitempty"`

	// Computed during resolution (not stored in JSON input)
	NormalizedHead string   `json:"-"` // Name head after stripping titles (e.g., "Ruiz")
	Titles         []string `json:"-"` // Extracted titles (e.g., ["Officer"])
	Tokens         []string `json:"-"` // Tokenized name parts (e.g., ["Carlos", "Ruiz"])
}

// Entity represents a resolved entity (a single "person" in the story).
type Entity struct {
	ID              string   `json:"id"`          // Unique entity ID
	Canonical       string   `json:"canonical"`   // Best display name (usually most complete form)
	Aliases         []string `json:"aliases"`     // All name variations (including titles)
	MentionIDs      []string `json:"mention_ids"` // IDs of all mentions belonging to this entity
	Confidence      float64  `json:"confidence"`  // Average merge confidence (0.0-1.0)
	Titles          []string `json:"titles"`      // Unique honorifics/titles seen (e.g., ["Officer", "Detective"])
	Kind            string   `json:"kind,omitempty"`
	DetectionStatus string   `json:"detection_status,omitempty"`
	DetectionScore  float64  `json:"detection_score,omitempty"`
}

// SeparatedPair records two mentions that should NOT be merged together.
// This is used after SplitEntity to prevent re-clustering.
type SeparatedPair struct {
	MentionID1 string `json:"mention_id_1"`
	MentionID2 string `json:"mention_id_2"`
	Reason     string `json:"reason,omitempty"` // Optional user-provided reason
}

// ResolvedEntities is the complete output of entity resolution.
type ResolvedEntities struct {
	Entities       []Entity        `json:"entities"`
	SeparatedPairs []SeparatedPair `json:"separated_pairs"`
	MentionMap     map[string]int  `json:"-"` // mentionID -> entityIndex (computed, not stored)
}
