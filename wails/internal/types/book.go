// Package types contains shared data structures for the Draftline application.
package types

import "strings"

// ISBNEntry is one format-specific ISBN. A hardcover, paperback, ebook, and
// audiobook each carry their own ISBN, so a book holds a list; the legacy
// single ISBN field stays populated for older readers of the .draftline
// format and always mirrors the first entry.
type ISBNEntry struct {
	Format string `json:"format"` // hardcover | paperback | ebook | audiobook | large_print | other | ""
	Value  string `json:"value"`
}

// Metadata contains book metadata information.
type Metadata struct {
	Title    string `json:"title"`
	Subtitle string `json:"subtitle,omitempty"`
	Author   string `json:"author"`
	// SeriesName and SeriesNumber place the book in a series. The number is a
	// string because series numbers are not always integers: "2", "2.5", and
	// "Book Two" are all things a storefront prints.
	SeriesName   string      `json:"series_name,omitempty"`
	SeriesNumber string      `json:"series_number,omitempty"`
	ISBN         string      `json:"isbn"`
	ISBNs        []ISBNEntry `json:"isbns,omitempty"`
	Publisher    string      `json:"publisher"`
	// Imprint is the publishing line a book appears under, which may differ
	// from the publisher that owns it.
	Imprint string `json:"imprint,omitempty"`
	// Language is a BCP 47 tag ("en-US"). Empty means the export falls back
	// to "en", which is what every book got before this field existed.
	Language string `json:"language,omitempty"`
	// CopyrightHolder is the name the copyright line is made out to, which is
	// not always the author.
	CopyrightHolder string `json:"copyright_holder,omitempty"`
	// Catalogue: what a storefront needs and an exporter can declare.
	BISAC1           string `json:"bisac_1,omitempty"`
	BISAC2           string `json:"bisac_2,omitempty"`
	Audience         string `json:"audience,omitempty"`
	Keywords         string `json:"keywords,omitempty"`
	ShortDescription string `json:"short_description,omitempty"`
	// Contributors is free text ("Cover: Mara Quist · Copy edit: A. Feld").
	// Roles are not modelled until something reads them.
	Contributors string `json:"contributors,omitempty"`
	Created      string `json:"created"`
	Modified     string `json:"modified"`
	// WordCount is the manuscript word count, computed in Go on open and
	// save so the frontend never re-counts a whole book on its main thread.
	// Additive field; older readers of the format ignore it.
	WordCount int `json:"word_count,omitempty"`
	// BookID names this book rather than its file. A path cannot do the job:
	// dragging a project into a Dropbox folder renames its path and would
	// orphan the working copy holding its unsaved work, and a book opened on
	// a laptop and a phone has two paths and is one book. Additive field;
	// older readers of the format ignore it. See EnsureBookID.
	BookID string `json:"book_id,omitempty"`
}

// NormalizeISBNs reconciles the per-format ISBN list with the legacy single
// field: blank entries are dropped, a legacy-only book seeds the list, and
// the legacy field mirrors the first listed ISBN so older readers keep
// seeing one.
func (m *Metadata) NormalizeISBNs() {
	cleaned := make([]ISBNEntry, 0, len(m.ISBNs))
	for _, entry := range m.ISBNs {
		entry.Value = strings.TrimSpace(entry.Value)
		if entry.Value != "" {
			cleaned = append(cleaned, entry)
		}
	}
	m.ISBNs = cleaned
	if len(m.ISBNs) == 0 {
		if legacy := strings.TrimSpace(m.ISBN); legacy != "" {
			m.ISBNs = []ISBNEntry{{Value: legacy}}
		}
	}
	if len(m.ISBNs) > 0 {
		m.ISBN = m.ISBNs[0].Value
	}
}

// ISBNFor returns the ISBN registered for the given format, or "".
func (m *Metadata) ISBNFor(format string) string {
	for _, entry := range m.ISBNs {
		if entry.Format == format {
			return entry.Value
		}
	}
	return ""
}

// ChapterItem represents a single chapter or section in a book.
type ChapterItem struct {
	ID       string `json:"id,omitempty"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle,omitempty"` // Optional chapter subheading
	Type     string `json:"type"`
	Content  string `json:"content"`
}

// ChapterSnapshotRequest describes a chapter state to add to the version
// history embedded in a .draftline archive.
type ChapterSnapshotRequest struct {
	ChapterID    string `json:"chapter_id"`
	Section      string `json:"section"`
	ChapterTitle string `json:"chapter_title"`
	Content      string `json:"content"`
	Reason       string `json:"reason,omitempty"`
}

// ChapterHistoryEntry is the lightweight metadata shown by the history UI.
// Snapshot content remains in the archive until explicitly requested.
type ChapterHistoryEntry struct {
	ID           string `json:"id"`
	ChapterID    string `json:"chapter_id"`
	Section      string `json:"section"`
	ChapterTitle string `json:"chapter_title"`
	CreatedAt    string `json:"created_at"`
	Reason       string `json:"reason"`
	WordCount    int    `json:"word_count"`
	ContentHash  string `json:"content_hash"`
	File         string `json:"file"`
}

// ChapterHistorySnapshot includes the content for one selected history entry.
type ChapterHistorySnapshot struct {
	Entry   ChapterHistoryEntry `json:"entry"`
	Content string              `json:"content"`
}

// WritingGoals tracks the author's writing progress and targets.
type WritingGoals struct {
	TargetWordCount int    `json:"target_word_count"`
	DailyWordGoal   int    `json:"daily_word_goal"`
	WordsToday      int    `json:"words_today"`
	LastWritingDate string `json:"last_writing_date"`
}

// WritingStyleOptions controls AI behavior for Expand/Smooth modes.
// Each value: 0 = off, 1 = subtle, 2 = moderate, 3 = heavy
type WritingStyleOptions struct {
	Metaphors       int `json:"metaphors"`
	Similes         int `json:"similes"`
	SensoryDetail   int `json:"sensory_detail"`
	InternalThought int `json:"internal_thought"`
	Dialogue        int `json:"dialogue"`
	Action          int `json:"action"`
	Description     int `json:"description"`
	Pacing          int `json:"pacing"`
}

// ReadAloudCast persists per-book Read Aloud voice casting: whether cast
// mode is on and which TTS voice each character speaks in. Keys of Voices
// are lowercased canonical character names (entity IDs churn across
// re-indexing; names survive it). Stored as an optional read_aloud_cast.json
// archive member — absent for books that never used cast mode.
type ReadAloudCast struct {
	CastMode bool              `json:"cast_mode"`
	Voices   map[string]string `json:"voices,omitempty"`
}

// PlannerLane is a story line on the Planner timeline: the main plot, a typed
// subplot, or a character's arc (CharacterID set). Lanes and cards refer to
// each other by ID.
type PlannerLane struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Kind        string `json:"kind"` // main | subplot | character
	Color       string `json:"color"`
	CharacterID string `json:"character_id,omitempty"`
}

// PlannerLink ties a card to a scene: the chapter's stable ID and the
// 1-based scene index within it (scenes are separated by scene breaks).
type PlannerLink struct {
	ChapterID string `json:"chapter_id"`
	Scene     int    `json:"scene"`
}

// PlannerCard is one plot card. It sits on the first line in Lines at the
// chapter in ChapterID (empty = "Later", not yet pinned to a chapter); other
// lines are drawn as crossings. Who are character IDs; WhoNames are the same
// people's names at the time Who was set, so a card still names its people
// after re-indexing reassigns entity IDs. Changes and Stakes are the promise
// the card makes, kept as separate fields.
type PlannerCard struct {
	ID string `json:"id"`
	// SourceID names the scratch note an imported outline card came from.
	SourceID string `json:"source_id,omitempty"`
	// SourceKey identifies the movement within that note across repeated imports.
	SourceKey string `json:"source_key,omitempty"`
	// Origin is how the card was made: by hand or from an imported outline.
	Origin    string       `json:"origin,omitempty"`
	Title     string       `json:"title"`
	Synopsis  string       `json:"synopsis"`
	Lines     []string     `json:"lines"`
	Who       []string     `json:"who"`
	WhoNames  []string     `json:"who_names,omitempty"`
	Changes   string       `json:"changes,omitempty"`
	Stakes    string       `json:"stakes,omitempty"`
	ChapterID string       `json:"chapter_id"`
	Link      *PlannerLink `json:"link,omitempty"`
	Status    string       `json:"status"` // planned | drafted
	Updated   string       `json:"updated,omitempty"`
}

// PlannerNote is a scratch note. System "dead" marks the automatic Dead Ideas
// note that deleted cards are written into; Excluded notes are ignored by
// Propose Cards.
type PlannerNote struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Body     string `json:"body"`
	Updated  string `json:"updated,omitempty"`
	Excluded bool   `json:"excluded,omitempty"`
	System   string `json:"system,omitempty"`
}

// PlannerData is the Planner's state for one book, stored as the optional
// archive member planner.json. Synopsis holds per-chapter edits keyed by
// chapter ID; chapters without an entry are generated from their cards.
type PlannerData struct {
	Version      int               `json:"version"`
	Lanes        []PlannerLane     `json:"lanes"`
	Cards        []PlannerCard     `json:"cards"`
	Notes        []PlannerNote     `json:"notes"`
	Synopsis     map[string]string `json:"synopsis,omitempty"`
	BeatTemplate string            `json:"beat_template,omitempty"`
	HiddenLanes  []string          `json:"hidden_lanes,omitempty"`
	Compact      bool              `json:"compact,omitempty"`
}

// BookData is the main container for all book content.
type BookData struct {
	Version             string              `json:"version"`
	Metadata            Metadata            `json:"metadata"`
	Copyright           string              `json:"copyright"`
	FrontMatter         []ChapterItem       `json:"front_matter"`
	Body                []ChapterItem       `json:"body"`
	BackMatter          []ChapterItem       `json:"back_matter"`
	FilePath            string              `json:"file_path,omitempty"`
	StoryBible          StoryBible          `json:"story_bible,omitempty"`
	WritingGoals        WritingGoals        `json:"writing_goals,omitempty"`
	StyleOptions        WritingStyleOptions `json:"style_options,omitempty"`
	IsIndexed           bool                `json:"is_indexed,omitempty"`
	LastIndexed         string              `json:"last_indexed,omitempty"`
	BeatSheet           BeatSheet           `json:"beat_sheet,omitempty"`
	ForeshadowingLedger ForeshadowingLedger `json:"foreshadowing,omitempty"`
	KnowledgeMatrix     KnowledgeMatrix     `json:"knowledge_matrix,omitempty"`
	ReadAloudCast       ReadAloudCast       `json:"read_aloud_cast,omitempty"`
	// Planner is the story-line timeline: lanes, cards, notes. Optional
	// archive member planner.json; nil for books that never opened it.
	Planner *PlannerData `json:"planner,omitempty"`
	// Editions is the book's publishing record: which editions exist, which
	// formats each one was published in, and the ISBN and specification of
	// each format. Optional archive member editions/index.json; nil for books
	// that never registered one. These are small JSON records and ride the
	// bridge with the rest of the book. Cover images and frozen manuscripts
	// are bytes and deliberately do NOT join them here.
	Editions *EditionIndex `json:"editions,omitempty"`
	// Analysis contains entity resolution and other analysis results.
	// This is a future-proof container that can be extended without schema changes.
	Analysis AnalysisData `json:"analysis,omitempty"`
}
