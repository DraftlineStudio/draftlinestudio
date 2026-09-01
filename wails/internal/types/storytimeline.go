package types

// StoryTimelineFacet is one filter derived from source-backed timeline events.
type StoryTimelineFacet struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Count int    `json:"count"`
}

// StoryTimelineEvent is one deduplicated, source-located event. Order is
// always manuscript order; TimeKind describes only what the prose explicitly
// establishes and never implies a guessed calendar date.
type StoryTimelineEvent struct {
	ID              string         `json:"id"`
	EvidenceIDs     []string       `json:"evidence_ids"`
	PrimaryType     string         `json:"primary_type"`
	EventTypes      []string       `json:"event_types"`
	Text            string         `json:"text"`
	SourceText      string         `json:"source_text"`
	ChapterID       string         `json:"chapter_id"`
	ChapterIndex    int            `json:"chapter_index"`
	ChapterTitle    string         `json:"chapter_title"`
	Section         string         `json:"section"`
	SectionIndex    int            `json:"section_index"`
	ParagraphIndex  int            `json:"paragraph_index"`
	SentenceIndex   int            `json:"sentence_index"`
	StartOffset     int            `json:"start_offset"`
	CharacterIDs    []string       `json:"character_ids,omitempty"`
	CharacterNames  []string       `json:"character_names,omitempty"`
	ThreadTerms     []EvidenceTerm `json:"thread_terms,omitempty"`
	Locations       []EvidenceTerm `json:"locations,omitempty"`
	TimeExpressions []string       `json:"time_expressions,omitempty"`
	TimeKind        string         `json:"time_kind"` // anchored | relative | manuscript
	TimeLabel       string         `json:"time_label"`
	Confidence      float64        `json:"confidence"`
	Status          string         `json:"status"`
	Pinned          bool           `json:"pinned,omitempty"`
}

// StoryTimelineChapter summarizes the event density of one manuscript unit.
type StoryTimelineChapter struct {
	ChapterIndex      int    `json:"chapter_index"`
	ChapterTitle      string `json:"chapter_title"`
	EventCount        int    `json:"event_count"`
	ExplicitTimeCount int    `json:"explicit_time_count"`
}

// StoryTimelineResult is the deterministic timeline projection of the local
// evidence index.
type StoryTimelineResult struct {
	Success           bool                   `json:"success"`
	Error             string                 `json:"error,omitempty"`
	Engine            string                 `json:"engine"`
	Events            []StoryTimelineEvent   `json:"events"`
	Chapters          []StoryTimelineChapter `json:"chapters"`
	Characters        []StoryTimelineFacet   `json:"characters"`
	Locations         []StoryTimelineFacet   `json:"locations"`
	EventTypes        []StoryTimelineFacet   `json:"event_types"`
	ExplicitTimeCount int                    `json:"explicit_time_count"`
	RelativeTimeCount int                    `json:"relative_time_count"`
}
