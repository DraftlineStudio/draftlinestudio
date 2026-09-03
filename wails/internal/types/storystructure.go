package types

// StoryStructure is the rebuildable aggregation layer between the lossless
// Story Fingerprint and author-facing structural views. Every aggregate keeps
// the complete evidence ids that justify it; no manuscript facts are removed.
type StoryStructure struct {
	ContentHash       string                  `json:"content_hash"`
	Engine            string                  `json:"engine"`
	LastAnalyzed      string                  `json:"last_analyzed"`
	Version           int                     `json:"version"`
	SignificantEvents []SignificantStoryEvent `json:"significant_events"`
	Scenes            []SemanticScene         `json:"scenes"`
	Sequences         []StorySequence         `json:"sequences"`
	NarrativeThreads  []NarrativeThread       `json:"narrative_threads"`
	Arcs              []StoryArc              `json:"arcs"`
	Cache             StoryStructureCache     `json:"cache"`
}

type SalienceBreakdown struct {
	Base            float64 `json:"base"`
	StateChange     float64 `json:"state_change,omitempty"`
	Movement        float64 `json:"movement,omitempty"`
	Decision        float64 `json:"decision,omitempty"`
	Discovery       float64 `json:"discovery,omitempty"`
	TemporalChange  float64 `json:"temporal_change,omitempty"`
	ThreadChange    float64 `json:"thread_change,omitempty"`
	LaterReferences float64 `json:"later_references,omitempty"`
	Contradiction   float64 `json:"contradiction,omitempty"`
	DescriptionOnly float64 `json:"description_only,omitempty"`
	LowConfidence   float64 `json:"low_confidence,omitempty"`
}

type SignificantStoryEvent struct {
	ID                  string            `json:"id"`
	Summary             string            `json:"summary"`
	AuthorSummary       string            `json:"author_summary,omitempty"`
	EvidenceIDs         []string          `json:"evidence_ids"`
	FingerprintEventIDs []string          `json:"fingerprint_event_ids"`
	AssertionIDs        []string          `json:"assertion_ids,omitempty"`
	CharacterIDs        []string          `json:"character_ids,omitempty"`
	CharacterNames      []string          `json:"character_names,omitempty"`
	Kinds               []string          `json:"kinds,omitempty"`
	Locations           []EvidenceTerm    `json:"locations,omitempty"`
	Objects             []EvidenceTerm    `json:"objects,omitempty"`
	StateIDs            []string          `json:"state_ids,omitempty"`
	ObligationThreadIDs []string          `json:"obligation_thread_ids,omitempty"`
	ContextID           string            `json:"context_id"`
	StoryTime           StoryTime         `json:"story_time"`
	ChapterID           string            `json:"chapter_id"`
	ChapterIndex        int               `json:"chapter_index"`
	ChapterTitle        string            `json:"chapter_title"`
	Section             string            `json:"section"`
	SectionIndex        int               `json:"section_index"`
	ParagraphStart      int               `json:"paragraph_start"`
	ParagraphEnd        int               `json:"paragraph_end"`
	StartOffset         int               `json:"start_offset"`
	EndOffset           int               `json:"end_offset"`
	NarrativeOrder      int               `json:"narrative_order"`
	Salience            float64           `json:"salience"`
	SalienceReasons     SalienceBreakdown `json:"salience_reasons"`
	Confidence          float64           `json:"confidence"`
}

type SemanticScene struct {
	ID                 string         `json:"id"`
	Summary            string         `json:"summary"`
	EventIDs           []string       `json:"event_ids"`
	EvidenceIDs        []string       `json:"evidence_ids"`
	CharacterIDs       []string       `json:"character_ids,omitempty"`
	CharacterNames     []string       `json:"character_names,omitempty"`
	Locations          []EvidenceTerm `json:"locations,omitempty"`
	ObjectiveTerms     []EvidenceTerm `json:"objective_terms,omitempty"`
	ContextID          string         `json:"context_id"`
	StoryTime          StoryTime      `json:"story_time"`
	ChapterIDs         []string       `json:"chapter_ids"`
	ChapterStart       int            `json:"chapter_start"`
	ChapterEnd         int            `json:"chapter_end"`
	NarrativeStart     int            `json:"narrative_start"`
	NarrativeEnd       int            `json:"narrative_end"`
	StartOffset        int            `json:"start_offset"`
	EndOffset          int            `json:"end_offset"`
	ParagraphEnd       int            `json:"paragraph_end"`
	Salience           float64        `json:"salience"`
	Confidence         float64        `json:"confidence"`
	BoundaryConfidence float64        `json:"boundary_confidence"`
	BoundarySource     string         `json:"boundary_source"`
}

type StorySequence struct {
	ID             string         `json:"id"`
	Summary        string         `json:"summary"`
	SceneIDs       []string       `json:"scene_ids"`
	EventIDs       []string       `json:"event_ids"`
	EvidenceIDs    []string       `json:"evidence_ids"`
	CharacterIDs   []string       `json:"character_ids,omitempty"`
	CharacterNames []string       `json:"character_names,omitempty"`
	ContextIDs     []string       `json:"context_ids,omitempty"`
	Locations      []EvidenceTerm `json:"locations,omitempty"`
	ObjectiveTerms []EvidenceTerm `json:"objective_terms,omitempty"`
	NarrativeStart int            `json:"narrative_start"`
	NarrativeEnd   int            `json:"narrative_end"`
	Salience       float64        `json:"salience"`
	Confidence     float64        `json:"confidence"`
}

// NarrativeThread models a temporarily independent storyline. Existing
// StoryThread records remain plot obligations; these are deliberately separate.
type NarrativeThread struct {
	ID                  string   `json:"id"`
	Label               string   `json:"label"`
	State               string   `json:"state"`
	SequenceIDs         []string `json:"sequence_ids"`
	SceneIDs            []string `json:"scene_ids"`
	EventIDs            []string `json:"event_ids"`
	EvidenceIDs         []string `json:"evidence_ids"`
	CharacterIDs        []string `json:"character_ids,omitempty"`
	CharacterNames      []string `json:"character_names,omitempty"`
	ObjectiveTerms      []string `json:"objective_terms,omitempty"`
	LocationTerms       []string `json:"location_terms,omitempty"`
	ContextIDs          []string `json:"context_ids,omitempty"`
	ObligationThreadIDs []string `json:"obligation_thread_ids,omitempty"`
	ParentIDs           []string `json:"parent_ids,omitempty"`
	ChildIDs            []string `json:"child_ids,omitempty"`
	ConvergenceEventIDs []string `json:"convergence_event_ids,omitempty"`
	SeparationEventIDs  []string `json:"separation_event_ids,omitempty"`
	NarrativeStart      int      `json:"narrative_start"`
	NarrativeEnd        int      `json:"narrative_end"`
	Salience            float64  `json:"salience"`
	Confidence          float64  `json:"confidence"`
}

type StoryArc struct {
	ID             string   `json:"id"`
	Label          string   `json:"label"`
	ThreadIDs      []string `json:"thread_ids"`
	SequenceIDs    []string `json:"sequence_ids"`
	SceneIDs       []string `json:"scene_ids"`
	EventIDs       []string `json:"event_ids"`
	EvidenceIDs    []string `json:"evidence_ids"`
	NarrativeStart int      `json:"narrative_start"`
	NarrativeEnd   int      `json:"narrative_end"`
	Salience       float64  `json:"salience"`
	Confidence     float64  `json:"confidence"`
}

type StoryStructureBlockCache struct {
	ID             string   `json:"id"`
	ChapterID      string   `json:"chapter_id"`
	ContentHash    string   `json:"content_hash"`
	ChapterIndex   int      `json:"chapter_index"`
	ParagraphIndex int      `json:"paragraph_index"`
	EvidenceIDs    []string `json:"evidence_ids,omitempty"`
	AggregateIDs   []string `json:"aggregate_ids,omitempty"`
}

type StoryStructureCache struct {
	SourceBlocks       []StoryStructureBlockCache `json:"source_blocks"`
	EvidenceDependents map[string][]string        `json:"evidence_dependents"`
	AggregateParents   map[string][]string        `json:"aggregate_parents"`
}
