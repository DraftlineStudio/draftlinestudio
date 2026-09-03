package main

import (
	"draftline/internal/fingerprint"
	"draftline/internal/types"
)

// QueryStoryFingerprint answers local, deterministic story questions without
// AI, filesystem access, or manuscript mutation.
func (a *App) QueryStoryFingerprint(book types.BookData, request types.FingerprintQueryRequest) types.FingerprintQueryAnswer {
	return fingerprint.Query(book, request)
}

// UpdateStoryAuthorModel applies explicit author intent and rebuilds only the
// semantic fingerprint. It does not rerun prose/v3 or mutate manuscript text.
func (a *App) UpdateStoryAuthorModel(book types.BookData, model types.StoryAuthorModel) types.FullAnalysisResult {
	if book.Analysis.Evidence == nil {
		return types.FullAnalysisResult{Success: false, Error: "Run story analysis before editing its author model."}
	}
	if book.Analysis.Fingerprint == nil {
		book.Analysis.Fingerprint = &types.StoryFingerprint{}
	}
	book.Analysis.Fingerprint.AuthorModel = model
	book.Analysis.Fingerprint = fingerprint.Build(&book, nil)
	if book.Analysis.Version < 5 {
		book.Analysis.Version = 5
	}
	return types.FullAnalysisResult{Success: true, Book: book}
}
