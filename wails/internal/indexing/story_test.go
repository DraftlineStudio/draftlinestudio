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

// The Prose panel's short and long sentence shares. The thresholds are
// eight words and twenty-five words; the panel's histogram and its
// short/long copy both rest on them.
func TestAnalyzeStoryReportsShortAndLongSentencePercents(t *testing.T) {
	book := types.BookData{Body: []types.ChapterItem{{
		ID: "one", Title: "Mixed", Type: "Chapter",
		Content: `<p>She ran.</p>` +
			`<p>The lantern room had frosted over in the night and the glass held a thin grey bloom that would not wipe away with a rag or a bare hand.</p>`,
	}}}
	analysis := AnalyzeStory(&book, nil)
	chapter := analysis.Chapters[0]
	if chapter.ShortSentencePercent <= 0 {
		t.Fatalf("expected the two-word sentence to count as short: %#v", chapter)
	}
	if chapter.LongSentencePercent <= 0 {
		t.Fatalf("expected the long sentence to count as long: %#v", chapter)
	}
	if chapter.AverageSentenceWords <= 0 {
		t.Fatalf("expected an average sentence length: %#v", chapter)
	}
}

// The Chapters panel draws one row per chapter from these two fields and
// has nothing else to show.
func TestAnalyzeStoryEmitsKeywordsAndExtractiveSummary(t *testing.T) {
	book := types.BookData{Body: []types.ChapterItem{{
		ID: "one", Title: "The Ledger", Type: "Chapter",
		Content: `<p>The ledger lay open on the keeper's table. Rhea read the ledger twice and found the oil entries short.</p>` +
			`<p>Every page of the ledger named the same supplier. The keeper had signed for oil he never carried up the stair.</p>`,
	}}}
	analysis := AnalyzeStory(&book, nil)
	chapter := analysis.Chapters[0]
	if len(chapter.Keywords) == 0 {
		t.Fatalf("expected keywords for the Chapters panel: %#v", chapter)
	}
	if chapter.ExtractiveSummary == "" {
		t.Fatalf("expected an extractive summary for the Chapters panel: %#v", chapter)
	}
}

// The Review panel ("Worth Reviewing") has exactly one producer: the
// observation list on the story analysis. Its rail badge counts these too.
func TestAnalyzeStoryProducesObservationsForTheReviewPanel(t *testing.T) {
	long := `<p>The keeper climbed the stair with the oil can in one hand and the ledger under his arm and he did not stop at the landing where the window looked out over the water because the light was already failing and he had counted on another hour.</p>`
	book := types.BookData{Body: []types.ChapterItem{
		{ID: "one", Title: "Short", Type: "Chapter", Content: `<p>She ran.</p>`},
		{ID: "two", Title: "Dense", Type: "Chapter", Content: long + long + long},
	}}
	analysis := AnalyzeStory(&book, nil)
	if len(analysis.Observations) == 0 {
		t.Fatal("expected at least one observation; the Review panel has no other source")
	}
	for _, observation := range analysis.Observations {
		if observation.Title == "" || observation.Kind == "" {
			t.Fatalf("observation is missing its label: %#v", observation)
		}
	}
}
