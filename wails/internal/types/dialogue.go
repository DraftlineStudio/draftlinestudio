package types

// DialogueSpan is one stretch of quoted speech located in a chapter's
// analysis plaintext. Offsets are byte offsets into the stripped chapter
// text (internal/plaintext), the same coordinate space as EvidenceRecord.
// The speaker is carried by canonical name as well as entity ID because
// entity IDs are positional and change between index runs, while names
// survive them (the ReadAloudCast precedent).
type DialogueSpan struct {
	// ID is a stable hash of the chapter content hash and the span's
	// offsets, so author decisions keyed to a span lapse when the prose
	// around it changes.
	ID           string `json:"id"`
	ChapterIndex int    `json:"chapter_index"`
	StartOffset  int    `json:"start_offset"`
	EndOffset    int    `json:"end_offset"`

	// SpeakerEntityID is empty when the span could not be attributed.
	SpeakerEntityID string `json:"speaker_entity_id,omitempty"`
	SpeakerName     string `json:"speaker_name,omitempty"`

	// Kind is "dialogue" for a pure quotation or "mixed" when narration
	// shares the sentence.
	Kind string `json:"kind"`

	// Attribution names the evidence that chose the speaker: "tag",
	// "adjacent-tag", "pretag", "continuation", "nearby", "alternation",
	// or "none".
	Attribution string  `json:"attribution"`
	Confidence  float64 `json:"confidence"`
}

// DialogueData is the persisted dialogue layer of a book's analysis. It is
// rebuildable, cached per chapter by content hash like EvidenceData, and
// consumed by style inspections (dialogue-aware policy), character voice
// features, and Read Aloud casting.
type DialogueData struct {
	Spans []DialogueSpan `json:"spans,omitempty"`

	// ChapterHashes records the content hash each chapter's spans were
	// computed from, so unchanged chapters are not re-analyzed.
	ChapterHashes map[int]string `json:"chapter_hashes,omitempty"`

	// Engine identifies the producer and its version; a change invalidates
	// every cached chapter.
	Engine  string `json:"engine,omitempty"`
	Version int    `json:"version,omitempty"`
}
