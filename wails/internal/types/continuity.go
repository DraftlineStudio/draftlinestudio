package types

// ContinuitySource is an exact manuscript location supporting a continuity
// signal. A signal may have two sources when Draftline is comparing passages.
type ContinuitySource struct {
	EvidenceID     string `json:"evidence_id,omitempty"`
	Text           string `json:"text,omitempty"`
	ChapterID      string `json:"chapter_id,omitempty"`
	ChapterIndex   int    `json:"chapter_index"`
	ChapterTitle   string `json:"chapter_title"`
	Section        string `json:"section"`
	SectionIndex   int    `json:"section_index"`
	ParagraphIndex int    `json:"paragraph_index,omitempty"`
	StartOffset    int    `json:"start_offset,omitempty"`
}

// ContinuitySignal is a conservative, explainable review prompt. Severity is
// review or info; deterministic heuristics never declare prose to be wrong.
type ContinuitySignal struct {
	ID             string             `json:"id"`
	Kind           string             `json:"kind"`
	Category       string             `json:"category"`
	Severity       string             `json:"severity"`
	Title          string             `json:"title"`
	Detail         string             `json:"detail"`
	CharacterIDs   []string           `json:"character_ids,omitempty"`
	CharacterNames []string           `json:"character_names,omitempty"`
	Sources        []ContinuitySource `json:"sources,omitempty"`
	Confidence     float64            `json:"confidence"`
	// Status carries the author's stored decision (reviewed | dismissed) and
	// is empty for questions not yet acted on. It is applied when the report
	// is built; it is not part of the signal's identity.
	Status         string             `json:"status,omitempty"`
}

// ContinuityFacet describes one report filter.
type ContinuityFacet struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Count int    `json:"count"`
}

// ContinuityReport is rebuilt on demand from the current story fingerprint.
type ContinuityReport struct {
	Success         bool               `json:"success"`
	Error           string             `json:"error,omitempty"`
	Engine          string             `json:"engine"`
	Signals         []ContinuitySignal `json:"signals"`
	Categories      []ContinuityFacet  `json:"categories"`
	Characters      []ContinuityFacet  `json:"characters"`
	ReviewCount     int                `json:"review_count"`
	InfoCount       int                `json:"info_count"`
	ChaptersChecked int                `json:"chapters_checked"`
}

// ContinuityDecision records an author's judgement on one continuity question.
// Signal IDs are derived from the question's kind, title, and source evidence,
// so a decision survives rebuilds of the report but is deliberately NOT carried
// over when the underlying prose changes — an edited passage produces a new
// signal ID and the question resurfaces for a fresh look.
type ContinuityDecision struct {
	SignalID  string `json:"signal_id"`
	Status    string `json:"status"` // reviewed | dismissed
	DecidedAt string `json:"decided_at,omitempty"`
}

// ContinuityData persists author decisions across sessions. The report itself
// is always rebuilt from the current fingerprint and never stored.
type ContinuityData struct {
	Decisions []ContinuityDecision `json:"decisions,omitempty"`
	Version   int                  `json:"version,omitempty"`
}
