// Package types contains shared data structures for the Draftline application.
package types

// Metadata contains book metadata information.
type Metadata struct {
	Title     string `json:"title"`
	Author    string `json:"author"`
	ISBN      string `json:"isbn"`
	Publisher string `json:"publisher"`
	Created   string `json:"created"`
	Modified  string `json:"modified"`
}

// ChapterItem represents a single chapter or section in a book.
type ChapterItem struct {
	Title    string `json:"title"`
	Subtitle string `json:"subtitle,omitempty"` // Optional chapter subheading
	Type     string `json:"type"`
	Content  string `json:"content"`
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
	// Analysis contains entity resolution and other analysis results.
	// This is a future-proof container that can be extended without schema changes.
	Analysis AnalysisData `json:"analysis,omitempty"`
}
