package main

import (
	"draftline/internal/storysearch"
	"draftline/internal/types"
)

// SearchStory exposes the local story-search engine without adding another
// responsibility to app.go. The engine itself is Wails-independent and does
// not persist or transmit manuscript text.
func (a *App) SearchStory(book types.BookData, request types.StorySearchRequest) types.StorySearchResult {
	return storysearch.Search(book, request)
}
