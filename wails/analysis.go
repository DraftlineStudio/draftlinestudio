package main

import (
	"fmt"

	"draftline/internal/fingerprint"
	"draftline/internal/indexing"
	"draftline/internal/types"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// AnalyzeBook runs the bundled local analysis pipeline as one coherent job.
// Keeping orchestration here prevents app.go from regaining the analysis
// responsibility removed by its package refactor.
func (a *App) AnalyzeBook(bookData types.BookData) types.FullAnalysisResult {
	done, ok := a.beginAnalysis(bookData)
	if !ok {
		return types.FullAnalysisResult{Success: false, Error: analysisBusyMessage}
	}
	defer done()

	chapterCount := len(indexing.AllChapters(&bookData))
	a.emitAnalysisProgress(types.StoryAnalysisProgress{
		Phase: "characters", Message: fmt.Sprintf("Analyzing characters across %d chapters", chapterCount),
		Current: 0, Total: chapterCount, Percent: 5,
	})

	indexResult := indexing.IndexBook(&bookData)
	if !indexResult.Success {
		return types.FullAnalysisResult{Success: false, Error: indexResult.Error}
	}
	bookData = indexResult.Book

	a.emitAnalysisProgress(types.StoryAnalysisProgress{
		Phase: "relationships", Message: "Mapping confirmed character relationships",
		Current: 1, Total: 1, Percent: 45,
	})
	analyzer := indexing.NewRelationshipAnalyzer()
	relationships, err := analyzer.AnalyzeBook(bookData)
	if err != nil {
		return types.FullAnalysisResult{Success: false, Error: err.Error()}
	}
	bookData.Analysis.Relationships = relationships
	bookData.Analysis.Evidence = indexing.AnalyzeEvidence(&bookData, a.emitAnalysisProgress)
	bookData.Analysis.Fingerprint = fingerprint.Build(&bookData, a.emitAnalysisProgress)
	bookData.Analysis.Story = indexing.AnalyzeStory(&bookData, a.emitAnalysisProgress)
	if bookData.Analysis.Version < 4 {
		bookData.Analysis.Version = 4
	}
	a.emitAnalysisProgress(types.StoryAnalysisProgress{
		Phase: "complete", Message: "Story analysis current", Current: chapterCount, Total: chapterCount, Percent: 100,
	})
	return types.FullAnalysisResult{Success: true, Book: bookData}
}

func (a *App) emitAnalysisProgress(progress types.StoryAnalysisProgress) {
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "analysis:progress", progress)
	}
}
