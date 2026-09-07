package main

import (
	"draftline/internal/fingerprint"
	"draftline/internal/types"
)

// GetFingerprintTextDiagnostics returns the three independent, selectable
// manuscript-memory quality reports. It performs no I/O and never sends
// manuscript content to an external service.
func (a *App) GetFingerprintTextDiagnostics(book types.BookData) types.FingerprintTextDiagnostics {
	return fingerprint.TextDiagnostics(&book)
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
	if book.Analysis.Version < 6 {
		book.Analysis.Version = 6
	}
	return types.FullAnalysisResult{Success: true, Book: book}
}
