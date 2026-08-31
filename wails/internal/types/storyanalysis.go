package types

// StoryAnalysisData contains rebuildable, evidence-based manuscript metrics.
// It intentionally avoids claiming that a structural measurement is a
// narrative fact: semantic plot/event analysis belongs to optional analyzers.
type StoryAnalysisData struct {
	ContentHash  string                     `json:"content_hash"`
	Engine       string                     `json:"engine"`
	LastAnalyzed string                     `json:"last_analyzed"`
	Overview     StoryAnalysisOverview      `json:"overview"`
	Chapters     []ChapterAnalysis          `json:"chapters"`
	Observations []StoryAnalysisObservation `json:"observations,omitempty"`
	Version      int                        `json:"version"`
}

type StoryAnalysisOverview struct {
	ChapterCount         int     `json:"chapter_count"`
	WordCount            int     `json:"word_count"`
	SentenceCount        int     `json:"sentence_count"`
	ParagraphCount       int     `json:"paragraph_count"`
	AverageChapterWords  float64 `json:"average_chapter_words"`
	AverageSentenceWords float64 `json:"average_sentence_words"`
	DialoguePercent      float64 `json:"dialogue_percent"`
	ReadingEase          float64 `json:"reading_ease"`
	MeanGradeLevel       float64 `json:"mean_grade_level"`
	TempoScore           float64 `json:"tempo_score"`
}

type ChapterAnalysis struct {
	ChapterID             string      `json:"chapter_id,omitempty"`
	ChapterIndex          int         `json:"chapter_index"`
	Title                 string      `json:"title"`
	WordCount             int         `json:"word_count"`
	SentenceCount         int         `json:"sentence_count"`
	ParagraphCount        int         `json:"paragraph_count"`
	SceneBreakCount       int         `json:"scene_break_count"`
	DialoguePercent       float64     `json:"dialogue_percent"`
	AverageSentenceWords  float64     `json:"average_sentence_words"`
	AverageParagraphWords float64     `json:"average_paragraph_words"`
	ReadingEase           float64     `json:"reading_ease"`
	MeanGradeLevel        float64     `json:"mean_grade_level"`
	MeanWordLength        float64     `json:"mean_word_length"`
	ShortSentencePercent  float64     `json:"short_sentence_percent"`
	LongSentencePercent   float64     `json:"long_sentence_percent"`
	VerbPercent           float64     `json:"verb_percent"`
	AdverbPercent         float64     `json:"adverb_percent"`
	AdjectivePercent      float64     `json:"adjective_percent"`
	TempoScore            float64     `json:"tempo_score"`
	TempoLabel            string      `json:"tempo_label"`
	Keywords              []TermCount `json:"keywords,omitempty"`
	ExtractiveSummary     string      `json:"extractive_summary,omitempty"`
}

type TermCount struct {
	Term  string `json:"term"`
	Count int    `json:"count"`
}

// StoryAnalysisObservation is a measured signal with a source chapter, not an
// authoritative writing diagnosis. Consumers should present it as something
// worth reviewing rather than an error that must be fixed.
type StoryAnalysisObservation struct {
	Kind         string `json:"kind"`
	Level        string `json:"level"`
	Title        string `json:"title"`
	Detail       string `json:"detail"`
	ChapterIndex int    `json:"chapter_index,omitempty"`
}

type StoryAnalysisProgress struct {
	Phase        string `json:"phase"`
	Message      string `json:"message"`
	ChapterIndex int    `json:"chapter_index,omitempty"`
	ChapterTitle string `json:"chapter_title,omitempty"`
	Current      int    `json:"current"`
	Total        int    `json:"total"`
	Percent      int    `json:"percent"`
}

type FullAnalysisResult struct {
	Success bool     `json:"success"`
	Error   string   `json:"error,omitempty"`
	Book    BookData `json:"book,omitempty"`
}
