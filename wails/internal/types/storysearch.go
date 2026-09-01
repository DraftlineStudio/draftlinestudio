package types

// StorySearchRequest describes one explicit search over the open manuscript.
// Searches are submitted rather than run per keystroke so large books are not
// repeatedly serialized across the Wails bridge.
type StorySearchRequest struct {
	Query string `json:"query"`
	Limit int    `json:"limit,omitempty"`
}

// StorySearchEntity reports a confirmed character whose aliases were expanded
// while interpreting the query.
type StorySearchEntity struct {
	ID        string   `json:"id"`
	Canonical string   `json:"canonical"`
	Aliases   []string `json:"aliases"`
}

// StorySearchEvidence is the lightweight evidence metadata attached to a raw
// scene hit. The full source sentence remains in analysis.evidence.
type StorySearchEvidence struct {
	ID           string  `json:"id"`
	Kind         string  `json:"kind"`
	EvidenceType string  `json:"evidence_type"`
	Status       string  `json:"status"`
	Confidence   float64 `json:"confidence"`
}

// StorySearchMatch is one scene-sized evidence hit in manuscript order.
type StorySearchMatch struct {
	Section        string                `json:"section"`
	SectionIndex   int                   `json:"section_index"`
	ChapterIndex   int                   `json:"chapter_index"`
	ChapterID      string                `json:"chapter_id,omitempty"`
	ChapterTitle   string                `json:"chapter_title"`
	SceneIndex     int                   `json:"scene_index"`
	Excerpt        string                `json:"excerpt"`
	MatchedTerms   []string              `json:"matched_terms"`
	AdditionalHits int                   `json:"additional_hits,omitempty"`
	Evidence       []StorySearchEvidence `json:"evidence,omitempty"`
}

// StorySearchResult contains evidence matches and the character aliases used
// to produce them. Total can exceed len(Matches) when a request limit applies.
type StorySearchResult struct {
	Query            string              `json:"query"`
	Matches          []StorySearchMatch  `json:"matches"`
	ResolvedEntities []StorySearchEntity `json:"resolved_entities,omitempty"`
	Total            int                 `json:"total"`
	Error            string              `json:"error,omitempty"`
}
