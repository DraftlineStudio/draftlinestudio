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
	budget, done, ok := a.beginAnalysis(bookData)
	if !ok {
		return types.FullAnalysisResult{Success: false, Error: analysisBusyMessage}
	}
	defer done()

	chapterCount := len(indexing.AllChapters(&bookData))
	a.emitAnalysisProgress(types.StoryAnalysisProgress{
		Phase: "characters", Message: fmt.Sprintf("Analyzing characters across %d chapters", chapterCount),
		Current: 0, Total: chapterCount, Percent: 5,
	})

	// IndexBook invalidates evidence because entity ids may change. Keep the
	// previous cache available to AnalyzeEvidence so unchanged chapters can be
	// reused after entity resolution instead of passing through ProseV3 again.
	priorEvidence := bookData.Analysis.Evidence
	priorEntities := bookData.Analysis.EntityResolution
	pool := indexing.AnalysisPoolOptions{Workers: budget.Workers, MaxInFlightBytes: budget.MaxInFlightBytes, MaxBatchBytes: budget.MaxBatchBytes}
	indexResult := indexing.IndexBookWithOptions(&bookData, pool)
	if !indexResult.Success {
		return types.FullAnalysisResult{Success: false, Error: indexResult.Error}
	}
	bookData = indexResult.Book
	if entityIDsStable(priorEntities, bookData.Analysis.EntityResolution) {
		bookData.Analysis.Evidence = priorEvidence
	}

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
	bookData.Analysis.Evidence = indexing.AnalyzeEvidenceWithOptions(&bookData, a.emitAnalysisProgress, pool)
	bookData.Analysis.Fingerprint = fingerprint.Build(&bookData, a.emitAnalysisProgress)
	bookData.Analysis.Story = indexing.AnalyzeStory(&bookData, a.emitAnalysisProgress)
	if bookData.Analysis.Version < 5 {
		bookData.Analysis.Version = 5
	}
	a.emitAnalysisProgress(types.StoryAnalysisProgress{
		Phase: "complete", Message: "Story analysis current", Current: chapterCount, Total: chapterCount, Percent: 100,
	})
	return types.FullAnalysisResult{Success: true, Book: bookData}
}

func entityIDsStable(before, after *types.EntityData) bool {
	if before == nil || after == nil || len(before.Entities) != len(after.Entities) {
		return false
	}
	ids := make(map[string]string, len(before.Entities))
	for _, entity := range before.Entities {
		ids[entity.Canonical] = entity.ID
	}
	for _, entity := range after.Entities {
		if ids[entity.Canonical] != entity.ID {
			return false
		}
	}
	return true
}

func (a *App) emitAnalysisProgress(progress types.StoryAnalysisProgress) {
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "analysis:progress", progress)
	}
}
