package main

import (
	"draftline/internal/continuity"
	"draftline/internal/types"
)

// BuildContinuityReport returns deterministic, source-backed review prompts
// without persisting conclusions or modifying manuscript text.
func (a *App) BuildContinuityReport(book types.BookData) types.ContinuityReport {
	return continuity.Build(book)
}
