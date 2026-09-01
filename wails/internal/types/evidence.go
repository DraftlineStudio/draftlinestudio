package types

// EvidenceData is the rebuildable local fact/event index persisted in
// analysis.json. Records always retain their exact source sentence and never
// claim more certainty than the deterministic rule that admitted them.
type EvidenceData struct {
	ContentHash  string           `json:"content_hash"`
	Engine       string           `json:"engine"`
	LastAnalyzed string           `json:"last_analyzed"`
	Records      []EvidenceRecord `json:"records"`
	Truncated    bool             `json:"truncated,omitempty"`
	Version      int              `json:"version"`
}

// EvidenceRecord is one source-located candidate fact or event. IDs are
// derived from stable chapter IDs and normalized evidence text, so an
// unrelated edit earlier in a chapter does not renumber every later record.
type EvidenceRecord struct {
	ID              string         `json:"id"`
	Kind            string         `json:"kind"`          // event | fact
	EvidenceType    string         `json:"evidence_type"` // introduction | discovery | transition | interaction | state | time_reference
	ChapterID       string         `json:"chapter_id"`
	ChapterIndex    int            `json:"chapter_index"`
	Section         string         `json:"section"`
	SectionIndex    int            `json:"section_index"`
	ParagraphIndex  int            `json:"paragraph_index"`
	SentenceIndex   int            `json:"sentence_index"`
	StartOffset     int            `json:"start_offset"`
	EndOffset       int            `json:"end_offset"`
	Text            string         `json:"text"`
	CharacterIDs    []string       `json:"character_ids,omitempty"`
	CharacterNames  []string       `json:"character_names,omitempty"`
	NamedEntities   []EvidenceTerm `json:"named_entities,omitempty"`
	Action          string         `json:"action,omitempty"`
	TimeExpressions []string       `json:"time_expressions,omitempty"`
	Confidence      float64        `json:"confidence"`
	Rationale       string         `json:"rationale"`
	Status          string         `json:"status"` // detected | confirmed | rejected
	Source          string         `json:"source"` // auto | author
	AuthorNote      string         `json:"author_note,omitempty"`
}

// EvidenceTerm preserves a named term and prose/v3's local entity label.
type EvidenceTerm struct {
	Text  string `json:"text"`
	Label string `json:"label"`
}
