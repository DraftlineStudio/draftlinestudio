package types

// StyleFinding is one style suggestion located in a chapter's analysis
// plaintext. Offsets are byte offsets into the stripped chapter text, the
// same coordinate space as EvidenceRecord and DialogueSpan. Because the
// editor cannot reliably map plaintext offsets into its own document, each
// finding also carries its exact text with a little context; the editor
// places the underline by finding Prefix+Snippet+Suffix and skips findings
// it can no longer place.
type StyleFinding struct {
	// ID is a stable hash of the chapter content hash, the rule, and the
	// offsets, so a dismissal lapses when the prose around it changes.
	ID           string `json:"id"`
	Rule         string `json:"rule"`
	Severity     string `json:"severity"` // "suggestion" | "hint"
	ChapterIndex int    `json:"chapter_index"`
	StartOffset  int    `json:"start_offset"`
	EndOffset    int    `json:"end_offset"`
	Message      string `json:"message"`

	// The flagged text and up to a few dozen bytes around it.
	Snippet string `json:"snippet"`
	Prefix  string `json:"prefix,omitempty"`
	Suffix  string `json:"suffix,omitempty"`

	Replacements []string `json:"replacements,omitempty"`
}

// StyleDecision records the author silencing one finding. Finding IDs hash
// the surrounding prose, so a decision lapses naturally when the text
// changes, the continuity-decisions precedent.
type StyleDecision struct {
	FindingID string `json:"finding_id"`
	Status    string `json:"status"` // "dismissed"
	DecidedAt string `json:"decided_at,omitempty"`
}

// StyleData is the persisted style layer of a book's analysis. Findings are
// rebuildable and cached per chapter by content hash; decisions are the
// author's and are preserved across rebuilds while their findings exist.
type StyleData struct {
	Findings  []StyleFinding  `json:"findings,omitempty"`
	Decisions []StyleDecision `json:"decisions,omitempty"`

	ChapterHashes map[int]string `json:"chapter_hashes,omitempty"`
	Engine        string         `json:"engine,omitempty"`
	Version       int            `json:"version,omitempty"`
	// Truncated reports that the findings list hit its cap.
	Truncated bool `json:"truncated,omitempty"`
}
