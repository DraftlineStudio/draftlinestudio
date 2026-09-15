package indexing

import (
	"errors"
	"fmt"

	"draftline/internal/types"
)

// Chapter-space anchors: Planner evidence whose offsets index a chapter's
// stripped analysis text (StripHTMLForAnalysis), the coordinate space the
// evidence index, the manuscript memory's evidence spans, and scene numbers
// (SceneAt) already share. An anchor is only ever emitted when its quote is
// the text at its offsets, and it is only ever trusted after Locate says so
// against the text as it is now.

// Locator resolves chapter-space anchors against one reading of the book:
// the current content hash and each analyzable chapter's stripped text,
// computed once so many anchors can be checked without re-stripping.
type Locator struct {
	revision string
	texts    map[string]string
}

// NewLocator reads the book once. Chapters without an ID cannot be
// addressed by an anchor and are skipped; a chapter the analysis excludes
// (front-matter apparatus) has no text here either.
func NewLocator(book *types.BookData) *Locator {
	locator := &Locator{revision: ContentHash(book), texts: map[string]string{}}
	for _, chapter := range AllChapters(book) {
		if chapter.ID == "" || !ShouldAnalyzeChapter(chapter) {
			continue
		}
		if _, seen := locator.texts[chapter.ID]; seen {
			continue
		}
		locator.texts[chapter.ID] = StripHTMLForAnalysis(chapter.Content)
	}
	return locator
}

// Revision is the content hash of the text the locator read.
func (l *Locator) Revision() string { return l.revision }

// ChapterText returns a chapter's stripped analysis text.
func (l *Locator) ChapterText(chapterID string) (string, bool) {
	text, ok := l.texts[chapterID]
	return text, ok
}

// Locate reports whether an anchor still points at its passage in the
// current text, with a reason when it does not:
//
//	legacy_block_anchor  the anchor is in the isolated narrative engine's
//	                     block space (Space empty), which this locator
//	                     cannot read;
//	chapter_missing      no analyzable chapter has the anchor's ID;
//	stale_revision       the book has changed since the anchor was made and
//	                     the quote is no longer at its offsets;
//	quote_mismatch       the book has not changed and the quote still is not
//	                     at its offsets — the anchor was never valid;
//	scene_mismatch       the quote is at its offsets but in a different
//	                     scene than the anchor records.
//
// When the anchor locates, the reason is "located", or "relocated" when the
// book has changed elsewhere but this passage has not: the caller may then
// refresh the anchor's revision without doubting the passage.
func (l *Locator) Locate(e types.PlannerEvidence) (bool, string) {
	if e.Space != types.PlannerEvidenceChapterSpace {
		return false, "legacy_block_anchor"
	}
	text, ok := l.texts[e.ChapterID]
	if !ok {
		return false, "chapter_missing"
	}
	if !quoteAt(text, e.Start, e.End, e.Quote) {
		if e.Revision != l.revision {
			return false, "stale_revision"
		}
		return false, "quote_mismatch"
	}
	if SceneAt(text, e.Start) != e.Scene {
		return false, "scene_mismatch"
	}
	if e.Revision != l.revision {
		return true, "relocated"
	}
	return true, "located"
}

// AnchorFromSpan converts a manuscript-memory evidence span into a
// chapter-space anchor under the locator's revision. It refuses, with an
// error, any span whose quote is not the text at its offsets, so a stale or
// malformed span never becomes an anchor.
func (l *Locator) AnchorFromSpan(span types.NarrativeEvidenceSpan) (types.PlannerEvidence, error) {
	text, ok := l.texts[span.ChapterID]
	if !ok {
		return types.PlannerEvidence{}, fmt.Errorf("evidence span names unknown chapter %q", span.ChapterID)
	}
	if span.Quote == "" {
		return types.PlannerEvidence{}, errors.New("evidence span has no quote")
	}
	if !quoteAt(text, span.StartOffset, span.EndOffset, span.Quote) {
		return types.PlannerEvidence{}, fmt.Errorf("evidence span does not locate in chapter %q at %d-%d", span.ChapterID, span.StartOffset, span.EndOffset)
	}
	return types.PlannerEvidence{
		Revision:  l.revision,
		ChapterID: span.ChapterID,
		Scene:     SceneAt(text, span.StartOffset),
		Start:     span.StartOffset,
		End:       span.EndOffset,
		Quote:     span.Quote,
		Space:     types.PlannerEvidenceChapterSpace,
	}, nil
}

// LocateAnchor is Locate over a fresh reading of the book.
func LocateAnchor(book *types.BookData, e types.PlannerEvidence) (bool, string) {
	return NewLocator(book).Locate(e)
}

// AnchorFromSpan is the locator's AnchorFromSpan over a fresh reading of the
// book, stamped with the given revision. The quote is still checked against
// the current text: a revision is a label, never a reason to trust offsets.
func AnchorFromSpan(book *types.BookData, span types.NarrativeEvidenceSpan, revision string) (types.PlannerEvidence, error) {
	anchor, err := NewLocator(book).AnchorFromSpan(span)
	if err != nil {
		return types.PlannerEvidence{}, err
	}
	anchor.Revision = revision
	return anchor, nil
}

func quoteAt(text string, start, end int, quote string) bool {
	if quote == "" || start < 0 || end > len(text) || start >= end {
		return false
	}
	return text[start:end] == quote
}
