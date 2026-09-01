package main

import (
	"draftline/internal/storytimeline"
	"draftline/internal/types"
)

// BuildStoryTimeline returns a deterministic, source-backed manuscript
// timeline without resolving ambiguous prose into invented calendar dates.
func (a *App) BuildStoryTimeline(book types.BookData) types.StoryTimelineResult {
	return storytimeline.Build(book)
}
