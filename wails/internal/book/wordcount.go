package book

import (
	"strings"

	"draftline/internal/indexing"
	"draftline/internal/types"
)

// RefreshWordCount recomputes the manuscript word count into the book's
// metadata. It runs on open and save (Go-side, off the UI thread) so the
// frontend can read metadata.word_count instead of re-counting the whole
// manuscript in JavaScript.
func RefreshWordCount(b *types.BookData) {
	total := 0
	for _, sections := range [][]types.ChapterItem{b.FrontMatter, b.Body, b.BackMatter} {
		for _, chapter := range sections {
			total += len(strings.Fields(indexing.StripHTMLForAnalysis(chapter.Content)))
		}
	}
	b.Metadata.WordCount = total
}
