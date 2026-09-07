package types

// StoryFingerprint is the durable manuscript-memory model (v5): typed,
// source-grounded frames; canonical state ledgers derived from them;
// structured event identities; synthesized narrative developments; and
// continuity inspections. Every derived claim points back to exact
// EvidenceRecord IDs and verbatim source spans.
//
// v5 doctrine: frames are manuscript memory, not plot beats. Free-text
// payloads are always verbatim slices of source sentences — the engine never
// composes propositions. When a slot cannot be filled confidently the frame
// abstains instead of guessing.
type StoryFingerprint struct {
	ContentHash         string               `json:"content_hash"`
	Engine              string               `json:"engine"`
	LastAnalyzed        string               `json:"last_analyzed"`
	Version             int                  `json:"version"`
	Contexts            []StoryContext       `json:"contexts"`
	TemporalConstraints []TemporalConstraint `json:"temporal_constraints"`
	Frames              []NarrativeFrame     `json:"frames"`
	Ledgers             []StateLedger        `json:"ledgers,omitempty"`
	EventIdentities     []NarrativeEventIdentity `json:"event_identities,omitempty"`
	Developments        []NarrativeDevelopment   `json:"developments,omitempty"`
	Inspections         []NarrativeInspection    `json:"inspections,omitempty"`
	CorpusStats         FingerprintCorpusStats   `json:"corpus_stats"`
	Profiles            []StoryProfile           `json:"profiles,omitempty"`
	Voices              []CharacterVoiceProfile  `json:"voices,omitempty"`
	AuthorModel         StoryAuthorModel         `json:"author_model"`
	// The three text diagnostics are rebuilt on demand, never persisted.
	FrameDiagnostic       string `json:"-"`
	DevelopmentDiagnostic string `json:"-"`
	InspectionDiagnostic  string `json:"-"`
}

// FrameType enumerates the constrained frame vocabulary. There is no
// freeform subject/predicate/object generation in v5.
const (
	FrameEvent        = "event"
	FrameLocation     = "location"
	FramePossession   = "possession"
	FrameTransfer     = "transfer"
	FrameInjury       = "injury"
	FrameLifeStatus   = "life_status"
	FrameKnowledge    = "knowledge"
	FrameBelief       = "belief"
	FrameClaim        = "claim"
	FrameGoal         = "goal"
	FrameDecision     = "decision"
	FrameObligation   = "obligation"
	FrameRelationship = "relationship"
	FrameTime         = "time"
	FrameIdentity     = "identity"
	FrameAccess       = "access"
	FrameCausal       = "causal"
)

// NarrativeFrame is one typed, attributable, scoped unit of manuscript
// memory. Detail (and every participant name) is either a canonical entity
// name or a verbatim slice of the evidence text; nothing is generated.
type NarrativeFrame struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	// Participants are role-tagged and keyed by canonical entity name.
	// Roles by frame type: subject (always), and where applicable
	// item, place, source, recipient, counterparty, cause.
	Participants []NarrativeParticipant `json:"participants"`
	// Detail is the frame payload — a verbatim slice of the source sentence
	// (a condition, fact clause, goal clause, place phrase, item phrase…).
	Detail string `json:"detail,omitempty"`
	// Value is a normalized closed-vocabulary value where the frame type has
	// one (life_status: dead|alive; polarity of access: granted|revoked…).
	Value       string                `json:"value,omitempty"`
	Polarity    string                `json:"polarity"`  // asserted | negated
	Epistemic   string                `json:"epistemic"` // narration | attributed_claim | belief | speculation
	Attribution NarrativeAttribution  `json:"attribution"`
	Scope       NarrativeRealityScope `json:"scope"`
	Temporal    StoryTime             `json:"temporal"`
	ContextID   string                `json:"context_id"`
	ChapterIndex   int `json:"chapter_index"`
	ParagraphIndex int `json:"paragraph_index"`
	SentenceIndex  int `json:"sentence_index"`
	NarrativeOrder int `json:"narrative_order"`
	// Abstentions record slots deliberately left unfilled (with the reason),
	// so uncertainty is visible instead of papered over with guesses.
	Abstentions   []string                `json:"abstentions,omitempty"`
	EvidenceIDs   []string                `json:"evidence_ids"`
	EvidenceSpans []NarrativeEvidenceSpan `json:"evidence_spans"`
	Confidence    float64                 `json:"confidence"`
}

// StateLedger is the canonical history of one aspect of one entity —
// the basis of continuity checking. Conflicting values are preserved as
// separate entries, never averaged away.
type StateLedger struct {
	ID         string `json:"id"`
	EntityName string `json:"entity_name"`
	// Aspect: location | possession | condition | life_status | knowledge |
	// goal | obligation | relationship | access | time
	Aspect string `json:"aspect"`
	// Qualifier narrows the aspect: the item for possession, the fact key
	// for knowledge, the counterparty for relationship.
	Qualifier string             `json:"qualifier,omitempty"`
	Entries   []StateLedgerEntry `json:"entries"`
}

type StateLedgerEntry struct {
	ID      string `json:"id"`
	FrameID string `json:"frame_id"`
	// Value is verbatim or closed-vocabulary, per the originating frame.
	Value string `json:"value"`
	// Operation: set | clear | transfer_in | transfer_out | open | close
	Operation      string                `json:"operation"`
	Epistemic      string                `json:"epistemic"`
	Attribution    NarrativeAttribution  `json:"attribution"`
	Scope          NarrativeRealityScope `json:"scope"`
	Temporal       StoryTime             `json:"temporal"`
	ChapterIndex   int                   `json:"chapter_index"`
	ParagraphIndex int                   `json:"paragraph_index"`
	NarrativeOrder int                   `json:"narrative_order"`
	EvidenceIDs    []string              `json:"evidence_ids"`
	Confidence     float64               `json:"confidence"`
}

// NarrativeEventIdentity links frames that likely describe the same
// underlying event, resolved from structured properties (class,
// participants, time, location, outcome) — never from shared vocabulary.
// Conflicting accounts are preserved as conflicting property values.
type NarrativeEventIdentity struct {
	ID           string                 `json:"id"`
	EventClass   string                 `json:"event_class"`
	Participants []NarrativeParticipant `json:"participants,omitempty"`
	ScopeIDs     []string               `json:"scope_ids"`
	Temporal     StoryTime              `json:"temporal"`
	FrameIDs     []string               `json:"frame_ids"`
	EvidenceIDs  []string               `json:"evidence_ids"`
	Properties   []EventIdentityProperty `json:"properties,omitempty"`
	Status       string                  `json:"status"` // consistent | conflicted
	Confidence   float64                 `json:"confidence"`
}

type EventIdentityProperty struct {
	Name   string               `json:"name"`
	Values []EventPropertyValue `json:"values"`
}

type EventPropertyValue struct {
	Value       string   `json:"value"`
	FrameIDs    []string `json:"frame_ids"`
	EvidenceIDs []string `json:"evidence_ids"`
	Epistemic   string   `json:"epistemic"`
}

// NarrativeDevelopment is a synthesized story-level change, reconstructed
// from combinations of trusted frames, ledgers, and event identities at
// scene/chapter scope. Developments — not raw frames — are the future input
// to PlotWalker. A development may be supported by several frames and
// several evidence spans; no single sentence has to state it outright.
type NarrativeDevelopment struct {
	ID string `json:"id"`
	// Kind: investigation_progress | major_discovery | corroboration |
	// disconfirmation | mystery_introduced | mystery_narrowed |
	// mystery_reframed | mystery_resolved | obstacle_introduced |
	// obstacle_overcome | major_decision | goal_established | goal_change |
	// relationship_change | threat_escalation | threat_reduction |
	// revelation | setup | payoff
	Kind    string `json:"kind"`
	Summary string `json:"summary"`
	// Basis lists the mechanical reasons this development was synthesized.
	Basis          []string                `json:"basis"`
	Entities       []NarrativeParticipant  `json:"entities,omitempty"`
	FrameIDs       []string                `json:"frame_ids"`
	EvidenceIDs    []string                `json:"evidence_ids"`
	EvidenceSpans  []NarrativeEvidenceSpan `json:"evidence_spans,omitempty"`
	ScopeID        string                  `json:"scope_id"`
	ChapterIndex   int                     `json:"chapter_index"`
	SceneIndex     int                     `json:"scene_index"`
	NarrativeOrder int                     `json:"narrative_order"`
	// Before/After capture the state transition where one applies, using
	// verbatim or closed-vocabulary values from the supporting frames.
	Before string `json:"before,omitempty"`
	After  string `json:"after,omitempty"`
	// Advances names the active goal, question, or mystery this development
	// moves, with the frames that established it.
	Advances         string   `json:"advances,omitempty"`
	AdvancesFrameIDs []string `json:"advances_frame_ids,omitempty"`
	// Causal neighbors, filled when the synthesis rules support the link.
	PredecessorID string  `json:"predecessor_id,omitempty"`
	SuccessorID   string  `json:"successor_id,omitempty"`
	Confidence    float64 `json:"confidence"`
}

// NarrativeInspection is one continuity finding over the frame corpus and
// ledgers. ScopeAssessment distinguishes likely errors from deliberate
// narrative devices (flashbacks, lies, simulations, dreams).
type NarrativeInspection struct {
	ID string `json:"id"`
	// Kind: contradictory_state | impossible_location |
	// impossible_possession | knowledge_before_acquisition |
	// repeated_event_conflict | timeline_conflict | unresolved_obligation |
	// causal_prerequisite_failure | setup_without_payoff
	Kind     string `json:"kind"`
	Severity string `json:"severity"` // error | warning | info
	// ScopeAssessment: same_scope_likely_error | cross_scope_divergence |
	// attributed_account_difference
	ScopeAssessment string                    `json:"scope_assessment"`
	Title           string                    `json:"title"`
	Detail          string                    `json:"detail"`
	Sides           []NarrativeInspectionSide `json:"sides"`
	Confidence      float64                   `json:"confidence"`
}

type NarrativeInspectionSide struct {
	Label         string                  `json:"label"`
	FrameIDs      []string                `json:"frame_ids,omitempty"`
	EvidenceSpans []NarrativeEvidenceSpan `json:"evidence_spans"`
}

type FingerprintCorpusStats struct {
	EvidenceAtoms   int `json:"evidence_atoms"`
	Frames          int `json:"frames"`
	Abstained       int `json:"abstained"`
	Ledgers         int `json:"ledgers"`
	EventIdentities int `json:"event_identities"`
	Developments    int `json:"developments"`
	Inspections     int `json:"inspections"`
}

type FingerprintTextDiagnostics struct {
	Frames       string `json:"frames"`
	Developments string `json:"developments"`
	Inspections  string `json:"inspections"`
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

// NarrativeAttribution records whose proposition a frame represents.
// A character-attributed claim never silently becomes narrator/world truth.
type NarrativeAttribution struct {
	Kind       string  `json:"kind"` // narrator | character | document | unknown
	EntityID   string  `json:"entity_id,omitempty"`
	EntityName string  `json:"entity_name,omitempty"`
	Cue        string  `json:"cue,omitempty"`
	Confidence float64 `json:"confidence"`
}

// NarrativeRealityScope prevents frames from incompatible realities from
// being compared as though they happened in the same world state.
type NarrativeRealityScope struct {
	ID          string   `json:"id"`
	Kind        string   `json:"kind"` // current | flashback | dream | vision | hypothetical | remembered | story_within_story | simulation | alternate | uncertain
	Label       string   `json:"label"`
	ParentID    string   `json:"parent_id,omitempty"`
	EvidenceIDs []string `json:"evidence_ids,omitempty"`
	Confidence  float64  `json:"confidence"`
}

type NarrativeParticipant struct {
	EntityID   string `json:"entity_id,omitempty"`
	EntityName string `json:"entity_name"`
	Role       string `json:"role"` // subject | item | place | source | recipient | counterparty | cause | mentioned
}

type NarrativeEvidenceSpan struct {
	EvidenceID     string  `json:"evidence_id"`
	ChapterID      string  `json:"chapter_id"`
	ChapterIndex   int     `json:"chapter_index"`
	Section        string  `json:"section"`
	SectionIndex   int     `json:"section_index"`
	ParagraphIndex int     `json:"paragraph_index"`
	SentenceIndex  int     `json:"sentence_index"`
	StartOffset    int     `json:"start_offset"`
	EndOffset      int     `json:"end_offset"`
	Quote          string  `json:"quote"`
	Confidence     float64 `json:"confidence"`
}

// FingerprintCorrection is durable author intent about a derived record,
// re-attached across rebuilds by target ID or evidence signature.
type FingerprintCorrection struct {
	ID              string   `json:"id"`
	TargetID        string   `json:"target_id"`
	TargetSignature string   `json:"target_signature,omitempty"`
	Kind            string   `json:"kind"`
	Value           string   `json:"value"`
	EvidenceIDs     []string `json:"evidence_ids,omitempty"`
	Status          string   `json:"status"` // active | orphaned | conflict
}

type StoryAuthorModel struct {
	Contexts    []StoryContext          `json:"contexts,omitempty"`
	Corrections []FingerprintCorrection `json:"corrections,omitempty"`
	Profiles    []string                `json:"profiles,omitempty"`
	VoiceNotes  []CharacterVoiceNotes   `json:"voice_notes,omitempty"`
}
