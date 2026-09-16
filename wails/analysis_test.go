package main

import (
	"testing"

	"draftline/internal/types"
)

// All prose in these fixtures is invented for the tests.

func analyzableBook() types.BookData {
	return types.BookData{
		Version: "2.2",
		Metadata: types.Metadata{Title: "The Lantern Keeper", Author: "R. Vance", Created: "2026-01-02T00:00:00Z"},
		Body: []types.ChapterItem{
			{
				ID: "ch-1", Title: "The Chained Gate", Type: "Chapter",
				Content: "<p>Rhea Calloway climbed the tower stair before dawn. The lantern room was cold and the glass had frosted over.</p>" +
					"<p>“You are late,” Tomas Verhoeven said. He had waited by the rail since midnight, and he did not turn around.</p>" +
					"<p>Rhea set the oil can down. She counted the empty brackets along the wall and found three of them bare.</p>",
			},
			{
				ID: "ch-2", Title: "The Keeper's House", Type: "Chapter",
				Content: "<p>Tomas Verhoeven walked back to the keeper's house through wet grass that soaked his boots.</p>" +
					"<p>Rhea Calloway followed him as far as the door and then stopped. The ledger lay open on the table.</p>" +
					"<p>“Someone has been taking the oil,” she said. Tomas read the page twice and said nothing at all.</p>",
			},
		},
		FrontMatter: []types.ChapterItem{},
		BackMatter:  []types.ChapterItem{},
	}
}

// The four products the analysis pipeline still owes the app after the
// experimental narrative engine was removed: the character index, the
// relationship analysis, the evidence index, and the story statistics. Every
// analysis sidebar, the Characters view, Ask Draftline, the Evidence Index,
// the story timeline, and the Continuity panel read one of these four.
func TestAnalyzeBookProducesCharactersRelationshipsEvidenceAndStory(t *testing.T) {
	app := NewApp()
	result := app.AnalyzeBook(analyzableBook())
	if !result.Success {
		t.Fatalf("AnalyzeBook failed: %s", result.Error)
	}
	analysis := result.Book.Analysis

	if analysis.EntityResolution == nil || len(analysis.EntityResolution.Entities) == 0 {
		t.Fatal("character detection produced no entities")
	}
	if analysis.Relationships == nil {
		t.Fatal("relationship analysis did not run")
	}
	if analysis.Evidence == nil || len(analysis.Evidence.Records) == 0 {
		t.Fatal("the evidence index is empty")
	}
	if analysis.Evidence.ContentHash == "" {
		t.Fatal("the evidence index carries no content hash")
	}
	if analysis.Story == nil {
		t.Fatal("story statistics did not run")
	}
	if analysis.Story.Overview.WordCount == 0 || analysis.Story.Overview.SentenceCount == 0 {
		t.Fatalf("story overview has no counts: %+v", analysis.Story.Overview)
	}
	if len(analysis.Story.Chapters) != 2 {
		t.Fatalf("expected one analysis row per body chapter, got %d", len(analysis.Story.Chapters))
	}
	for _, chapter := range analysis.Story.Chapters {
		if chapter.WordCount == 0 || chapter.SentenceCount == 0 {
			t.Fatalf("chapter %q has no counts: %+v", chapter.ChapterID, chapter)
		}
	}
}
