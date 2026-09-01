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

// StorySearchChapterSummary describes how a searched detail travels through
// one chapter. Counts cover every matching scene, even when the source list is
// capped for display.
type StorySearchChapterSummary struct {
	ChapterIndex  int    `json:"chapter_index"`
	ChapterTitle  string `json:"chapter_title"`
	Occurrences   int    `json:"occurrences"`
	EvidenceCount int    `json:"evidence_count"`
	EventCount    int    `json:"event_count"`
	FactCount     int    `json:"fact_count"`
}

// StorySearchRelatedTerm is a named detail that repeatedly occurs in indexed
// evidence supporting the search trail.
type StorySearchRelatedTerm struct {
	Text  string `json:"text"`
	Label string `json:"label"`
	Count int    `json:"count"`
}

// StorySearchSignal is a deterministic, source-backed observation about a
// detail trail. Kind is answer, attention, or context.
type StorySearchSignal struct {
	Kind   string `json:"kind"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

// StorySearchKnowledgeState is one source-backed point in the searched
// detail's knowledge trail, retained in manuscript order.
type StorySearchKnowledgeState struct {
	EvidenceID        string   `json:"evidence_id"`
	State             string   `json:"state"`
	CharacterIDs      []string `json:"character_ids,omitempty"`
	CharacterNames    []string `json:"character_names,omitempty"`
	CounterpartyIDs   []string `json:"counterparty_ids,omitempty"`
	CounterpartyNames []string `json:"counterparty_names,omitempty"`
	ChapterIndex      int      `json:"chapter_index"`
	ChapterTitle      string   `json:"chapter_title"`
	Section           string   `json:"section"`
	SectionIndex      int      `json:"section_index"`
	Text              string   `json:"text"`
	Cue               string   `json:"cue"`
	Confidence        float64  `json:"confidence"`
}

// StorySearchInsight turns raw scene hits into a compact manuscript trail.
// It summarizes only measured occurrences and indexed evidence; it never
// invents an answer when the source does not establish one.
type StorySearchInsight struct {
	Intent           string                      `json:"intent"`
	InterpretedQuery string                      `json:"interpreted_query"`
	ChapterCount     int                         `json:"chapter_count"`
	EvidenceCount    int                         `json:"evidence_count"`
	EventCount       int                         `json:"event_count"`
	FactCount        int                         `json:"fact_count"`
	DiscoveryCount   int                         `json:"discovery_count"`
	KnowledgeCount   int                         `json:"knowledge_count"`
	Chapters         []StorySearchChapterSummary `json:"chapters"`
	KnowledgeStates  []StorySearchKnowledgeState `json:"knowledge_states,omitempty"`
	RelatedTerms     []StorySearchRelatedTerm    `json:"related_terms,omitempty"`
	Signals          []StorySearchSignal         `json:"signals,omitempty"`
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
	Insight          *StorySearchInsight `json:"insight,omitempty"`
	Total            int                 `json:"total"`
	Error            string              `json:"error,omitempty"`
}
