package types

// StoryFingerprint is the durable local representation behind chronology,
// threads, continuity and mechanical story questions. Every derived claim
// points back to exact EvidenceRecord IDs.
type StoryFingerprint struct {
	ContentHash         string                  `json:"content_hash"`
	Engine              string                  `json:"engine"`
	LastAnalyzed        string                  `json:"last_analyzed"`
	Version             int                     `json:"version"`
	Contexts            []StoryContext          `json:"contexts"`
	TemporalConstraints []TemporalConstraint    `json:"temporal_constraints"`
	Assertions          []StoryAssertion        `json:"assertions"`
	Events              []FingerprintEvent      `json:"events"`
	States              []StoryStateInterval    `json:"states"`
	Threads             []StoryThread           `json:"threads"`
	Diagnostics         []FingerprintDiagnostic `json:"diagnostics"`
	Profiles            []StoryProfile          `json:"profiles,omitempty"`
	Voices              []CharacterVoiceProfile `json:"voices,omitempty"`
	Structure           *StoryStructure         `json:"structure,omitempty"`
	AuthorModel         StoryAuthorModel        `json:"author_model"`
}

type StoryProfile struct {
	ID         string  `json:"id"`
	Label      string  `json:"label"`
	Confidence float64 `json:"confidence"`
	Source     string  `json:"source"` // inferred | author
}

// CharacterVoiceProfile describes recurring, source-backed speaking habits.
// DialectSignal labels name observable text features, not demographic or
// regional identities that cannot be established safely from prose alone.
type CharacterVoiceProfile struct {
	CharacterID        string               `json:"character_id"`
	CharacterName      string               `json:"character_name"`
	SampleCount        int                  `json:"sample_count"`
	WordCount          int                  `json:"word_count"`
	AverageWords       float64              `json:"average_words"`
	ContractionPercent float64              `json:"contraction_percent"`
	QuestionPercent    float64              `json:"question_percent"`
	ExclamationPercent float64              `json:"exclamation_percent"`
	Vocabulary         []VoiceTerm          `json:"vocabulary,omitempty"`
	AddressForms       []VoiceTerm          `json:"address_forms,omitempty"`
	DialectSignals     []VoiceSignal        `json:"dialect_signals,omitempty"`
	Samples            []DialogueSample     `json:"samples,omitempty"`
	Confidence         float64              `json:"confidence"`
	AuthorNotes        *CharacterVoiceNotes `json:"author_notes,omitempty"`
}

type DialogueSample struct {
	ID             string  `json:"id"`
	Text           string  `json:"text"`
	ChapterID      string  `json:"chapter_id"`
	ChapterIndex   int     `json:"chapter_index"`
	ChapterTitle   string  `json:"chapter_title"`
	StartOffset    int     `json:"start_offset"`
	AttributionCue string  `json:"attribution_cue"`
	Confidence     float64 `json:"confidence"`
}

type VoiceTerm struct {
	Text  string `json:"text"`
	Count int    `json:"count"`
}

type VoiceSignal struct {
	Kind       string   `json:"kind"`
	Label      string   `json:"label"`
	Count      int      `json:"count"`
	Examples   []string `json:"examples,omitempty"`
	Confidence float64  `json:"confidence"`
}

type CharacterVoiceNotes struct {
	CharacterID    string   `json:"character_id"`
	Dialect        string   `json:"dialect,omitempty"`
	Vernacular     []string `json:"vernacular,omitempty"`
	SpeakingTraits []string `json:"speaking_traits,omitempty"`
	Avoids         []string `json:"avoids,omitempty"`
	Notes          string   `json:"notes,omitempty"`
}

// StoryContext separates incompatible clocks or realities without assuming
// that every flashback, dream or simulation follows one author's conventions.
type StoryContext struct {
	ID          string   `json:"id"`
	Kind        string   `json:"kind"` // primary | past | memory | dream | simulation | unknown
	Label       string   `json:"label"`
	ParentID    string   `json:"parent_id,omitempty"`
	EvidenceIDs []string `json:"evidence_ids,omitempty"`
	Confidence  float64  `json:"confidence"`
	Source      string   `json:"source"` // inferred | author
}

// StoryTime preserves both solved and approximate chronology. DayOffset is
// relative to the manuscript's first usable anchor; nil means not solvable.
type StoryTime struct {
	ContextID   string   `json:"context_id"`
	Label       string   `json:"label,omitempty"`
	DayOffset   *float64 `json:"day_offset,omitempty"`
	EarliestDay *float64 `json:"earliest_day,omitempty"`
	LatestDay   *float64 `json:"latest_day,omitempty"`
	Precision   string   `json:"precision"` // exact | day | range | relative | unknown
	Confidence  float64  `json:"confidence"`
}

type TemporalConstraint struct {
	ID             string   `json:"id"`
	FromEvidenceID string   `json:"from_evidence_id"`
	ToEvidenceID   string   `json:"to_evidence_id,omitempty"`
	Relation       string   `json:"relation"` // before | after | same | offset | contained
	OffsetDays     *float64 `json:"offset_days,omitempty"`
	Label          string   `json:"label"`
	Posture        string   `json:"posture"` // narrated | claimed | remembered | inferred | author
	Confidence     float64  `json:"confidence"`
	Source         string   `json:"source"`
}

type StoryAssertion struct {
	ID          string   `json:"id"`
	EvidenceIDs []string `json:"evidence_ids"`
	SubjectID   string   `json:"subject_id,omitempty"`
	Subject     string   `json:"subject,omitempty"`
	Predicate   string   `json:"predicate"`
	ObjectID    string   `json:"object_id,omitempty"`
	Object      string   `json:"object,omitempty"`
	Posture     string   `json:"posture"`  // fact | claim | memory | belief | suspicion | lie | dream
	Polarity    string   `json:"polarity"` // positive | negative | uncertain
	ContextID   string   `json:"context_id"`
	Confidence  float64  `json:"confidence"`
}

type FingerprintEvent struct {
	ID             string         `json:"id"`
	Summary        string         `json:"summary"`
	AuthorSummary  string         `json:"author_summary,omitempty"`
	EvidenceIDs    []string       `json:"evidence_ids"`
	AssertionIDs   []string       `json:"assertion_ids,omitempty"`
	CharacterIDs   []string       `json:"character_ids,omitempty"`
	CharacterNames []string       `json:"character_names,omitempty"`
	Locations      []EvidenceTerm `json:"locations,omitempty"`
	Objects        []EvidenceTerm `json:"objects,omitempty"`
	Kinds          []string       `json:"kinds"`
	ContextID      string         `json:"context_id"`
	StoryTime      StoryTime      `json:"story_time"`
	ChapterID      string         `json:"chapter_id"`
	ChapterIndex   int            `json:"chapter_index"`
	ChapterTitle   string         `json:"chapter_title"`
	ParagraphIndex int            `json:"paragraph_index"`
	StartOffset    int            `json:"start_offset"`
	NarrativeOrder int            `json:"narrative_order"`
	Importance     float64        `json:"importance"`
	Confidence     float64        `json:"confidence"`
	Status         string         `json:"status"`
}

type StoryStateInterval struct {
	ID           string   `json:"id"`
	EntityID     string   `json:"entity_id"`
	EntityName   string   `json:"entity_name"`
	Kind         string   `json:"kind"` // presence | attribute | possession | knowledge | belief | condition | trust | goal
	Value        string   `json:"value"`
	Qualifier    string   `json:"qualifier,omitempty"` // owns | carries | accesses | knows_location
	ContextID    string   `json:"context_id"`
	StartEventID string   `json:"start_event_id"`
	EndEventID   string   `json:"end_event_id,omitempty"`
	EvidenceIDs  []string `json:"evidence_ids"`
	Persistent   bool     `json:"persistent"`
	Confidence   float64  `json:"confidence"`
}

type StoryThread struct {
	ID                string   `json:"id"`
	Label             string   `json:"label"`
	Kind              string   `json:"kind"`
	State             string   `json:"state"` // seeded | active | escalating | dormant | converging | resolved | abandoned | intentionally_deferred
	OpenedByEventID   string   `json:"opened_by_event_id,omitempty"`
	ResolvedByEventID string   `json:"resolved_by_event_id,omitempty"`
	EventIDs          []string `json:"event_ids,omitempty"`
	EvidenceIDs       []string `json:"evidence_ids,omitempty"`
	EntityIDs         []string `json:"entity_ids,omitempty"`
	ParentIDs         []string `json:"parent_ids,omitempty"`
	ChildIDs          []string `json:"child_ids,omitempty"`
	Resolution        float64  `json:"resolution"`
	DormantChapters   int      `json:"dormant_chapters,omitempty"`
	DormantWords      int      `json:"dormant_words,omitempty"`
	Confidence        float64  `json:"confidence"`
	Source            string   `json:"source"`
}

type StoryCheckpoint struct {
	ID                 string                  `json:"id"`
	Title              string                  `json:"title"`
	Description        string                  `json:"description,omitempty"`
	Kind               string                  `json:"kind"`
	Status             string                  `json:"status"` // planned | partial | proposed | fulfilled | dismissed
	Requirements       []CheckpointRequirement `json:"requirements,omitempty"`
	NegativeConditions []CheckpointRequirement `json:"negative_conditions,omitempty"`
	BeforeChapter      *int                    `json:"before_chapter,omitempty"`
	AfterChapter       *int                    `json:"after_chapter,omitempty"`
	MatchedEventIDs    []string                `json:"matched_event_ids,omitempty"`
	Source             string                  `json:"source"`
}

type CheckpointRequirement struct {
	Text       string `json:"text"`
	EntityID   string `json:"entity_id,omitempty"`
	Predicate  string `json:"predicate,omitempty"`
	Object     string `json:"object,omitempty"`
	Satisfied  bool   `json:"satisfied"`
	EvidenceID string `json:"evidence_id,omitempty"`
}

type CanonRule struct {
	ID          string   `json:"id"`
	Subject     string   `json:"subject"`
	Predicate   string   `json:"predicate"`
	Object      string   `json:"object"`
	Polarity    string   `json:"polarity"`
	ContextID   string   `json:"context_id,omitempty"`
	Note        string   `json:"note,omitempty"`
	EvidenceIDs []string `json:"evidence_ids,omitempty"`
}

type FingerprintCorrection struct {
	ID              string   `json:"id"`
	TargetID        string   `json:"target_id"`
	TargetSignature string   `json:"target_signature,omitempty"`
	Kind            string   `json:"kind"`
	Value           string   `json:"value"`
	EvidenceIDs     []string `json:"evidence_ids,omitempty"`
	Status          string   `json:"status"` // active | orphaned | conflict
}

// StructureDecisionDependency snapshots the source block on which an author
// decision depends. It lets reanalysis distinguish a deleted dependency from a
// rewritten one instead of silently applying stale intent to new prose.
type StructureDecisionDependency struct {
	EvidenceID     string `json:"evidence_id"`
	ChapterID      string `json:"chapter_id"`
	ParagraphIndex int    `json:"paragraph_index"`
	ContentHash    string `json:"content_hash"`
}

// StructureAuthorDecision is durable author intent about a rebuildable
// structural inference. Action is confirm | correct | irrelevant | unresolved |
// intentional_ambiguity. Status is active | orphaned | conflict.
type StructureAuthorDecision struct {
	ID                   string                        `json:"id"`
	TargetType           string                        `json:"target_type"`
	TargetID             string                        `json:"target_id,omitempty"`
	TargetSignature      string                        `json:"target_signature,omitempty"`
	Action               string                        `json:"action"`
	Field                string                        `json:"field,omitempty"`
	Value                string                        `json:"value,omitempty"`
	Note                 string                        `json:"note,omitempty"`
	EvidenceIDs          []string                      `json:"evidence_ids,omitempty"`
	Dependencies         []StructureDecisionDependency `json:"dependencies,omitempty"`
	ChangedDependencyIDs []string                      `json:"changed_dependency_ids,omitempty"`
	Status               string                        `json:"status"`
}

type StoryAuthorModel struct {
	Contexts           []StoryContext            `json:"contexts,omitempty"`
	Checkpoints        []StoryCheckpoint         `json:"checkpoints,omitempty"`
	Canon              []CanonRule               `json:"canon,omitempty"`
	Corrections        []FingerprintCorrection   `json:"corrections,omitempty"`
	StructureDecisions []StructureAuthorDecision `json:"structure_decisions,omitempty"`
	Profiles           []string                  `json:"profiles,omitempty"`
	VoiceNotes         []CharacterVoiceNotes     `json:"voice_notes,omitempty"`
}

type FingerprintDiagnostic struct {
	ID             string   `json:"id"`
	Kind           string   `json:"kind"`
	Severity       string   `json:"severity"`
	Title          string   `json:"title"`
	Detail         string   `json:"detail"`
	EventIDs       []string `json:"event_ids,omitempty"`
	EvidenceIDs    []string `json:"evidence_ids,omitempty"`
	ChapterIndices []int    `json:"chapter_indices,omitempty"`
	Confidence     float64  `json:"confidence"`
	Status         string   `json:"status,omitempty"`
}

type FingerprintQueryRequest struct {
	Query string `json:"query"`
	Limit int    `json:"limit,omitempty"`
}

type FingerprintQueryAnswer struct {
	Success        bool                    `json:"success"`
	Error          string                  `json:"error,omitempty"`
	Interpretation string                  `json:"interpretation,omitempty"`
	Answer         string                  `json:"answer,omitempty"`
	Events         []FingerprintEvent      `json:"events,omitempty"`
	States         []StoryStateInterval    `json:"states,omitempty"`
	Threads        []StoryThread           `json:"threads,omitempty"`
	Diagnostics    []FingerprintDiagnostic `json:"diagnostics,omitempty"`
	Voices         []CharacterVoiceProfile `json:"voices,omitempty"`
	EvidenceIDs    []string                `json:"evidence_ids,omitempty"`
	Confidence     float64                 `json:"confidence"`
}
