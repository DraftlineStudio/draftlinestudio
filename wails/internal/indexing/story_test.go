package indexing

import (
	"testing"

	"draftline/internal/types"
)

func TestAnalyzeStoryProducesChapterMetricsAndProgress(t *testing.T) {
	book := types.BookData{Body: []types.ChapterItem{
		{ID: "one", Title: "The Door", Type: "Chapter", Content: `<p>“Run now,” Mara said. Hanlon followed her through the door.</p><p>They stopped outside.</p>`},
		{ID: "two", Title: "Afterward", Type: "Chapter", Content: `<p>Three days later, Mara returned to the quiet station. She waited beside the window and listened.</p>`},
	}}
	progressCount := 0
	analysis := AnalyzeStory(&book, func(progress types.StoryAnalysisProgress) {
		progressCount++
		if progress.ChapterTitle == "" || progress.Percent < 55 || progress.Percent > 100 {
			t.Fatalf("invalid progress: %#v", progress)
		}
	})

	if len(analysis.Chapters) != 2 || progressCount != 2 {
		t.Fatalf("expected two analyzed chapters and progress calls, got chapters=%d progress=%d", len(analysis.Chapters), progressCount)
	}
	if analysis.ContentHash == "" || analysis.Engine != storyAnalysisEngine {
		t.Fatalf("missing analysis identity: %#v", analysis)
	}
	if analysis.Overview.WordCount == 0 || analysis.Chapters[0].SentenceCount == 0 {
		t.Fatalf("expected non-zero prose metrics: %#v", analysis.Overview)
	}
	if analysis.Chapters[0].DialoguePercent <= 0 {
		t.Fatalf("expected dialogue to be measured: %#v", analysis.Chapters[0])
	}
}

func TestAnalyzeStorySkipsNonNarrativeSections(t *testing.T) {
	book := types.BookData{
		FrontMatter: []types.ChapterItem{
			{Title: "Acknowledgments", Type: "Acknowledgments", Content: "<p>Thanks to Alice and Bob.</p>"},
			{Title: "Author's Note", Type: "Author's Note", Content: "<p>A note about writing this book.</p>"},
		},
		Body: []types.ChapterItem{{Title: "Chapter One", Type: "Chapter", Content: "<p>Mara entered the room.</p>"}},
	}
	analysis := AnalyzeStory(&book, nil)
	if len(analysis.Chapters) != 1 || analysis.Chapters[0].Title != "Chapter One" {
		t.Fatalf("unexpected analyzed chapters: %#v", analysis.Chapters)
	}
}
