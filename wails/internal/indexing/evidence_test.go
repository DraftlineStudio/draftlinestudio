package indexing

import (
	"strings"
	"testing"

	"draftline/internal/types"
)

func TestAnalyzeEvidenceIndexesDiscoveryWithExactSourceAndCharacters(t *testing.T) {
	text := "Hanlon and Ruiz found the link to IBM beneath the old terminal."
	book := evidenceTestBook(text, map[string]string{"Hanlon": "hanlon", "Ruiz": "ruiz"})

	result := AnalyzeEvidence(&book, nil)
	record := findEvidenceType(result.Records, "discovery")
	if record == nil {
		t.Fatalf("expected discovery evidence, got %+v", result.Records)
	}
	if record.Text != text || len(record.CharacterIDs) != 2 {
		t.Fatalf("evidence lost its exact source or characters: %+v", record)
	}
	if record.ChapterID != "chapter-one" || record.Section != "body" || record.SectionIndex != 0 {
		t.Fatalf("evidence source location is incomplete: %+v", record)
	}
}

func TestAnalyzeEvidenceStableIDSurvivesUnrelatedEarlierParagraph(t *testing.T) {
	sentence := "Hanlon discovered the underground tunnel beneath the plaza."
	book := evidenceTestBook(sentence, map[string]string{"Hanlon": "hanlon"})
	first := findEvidenceType(AnalyzeEvidence(&book, nil).Records, "discovery")
	if first == nil {
		t.Fatal("expected initial discovery")
	}

	book.Body[0].Content = "<p>Rain struck the windows.</p><p>" + sentence + "</p>"
	book.Analysis.EntityResolution.Mentions[0].CharOffset = strings.Index(StripHTMLForAnalysis(book.Body[0].Content), "Hanlon")
	second := findEvidenceType(AnalyzeEvidence(&book, nil).Records, "discovery")
	if second == nil || second.ID != first.ID {
		t.Fatalf("unrelated insertion changed stable evidence ID: first=%+v second=%+v", first, second)
	}
	if second.ParagraphIndex == first.ParagraphIndex {
		t.Fatalf("test did not move the source paragraph: first=%d second=%d", first.ParagraphIndex, second.ParagraphIndex)
	}
}

func TestAnalyzeEvidencePreservesReviewDecisionAndAuthorRecords(t *testing.T) {
	text := "Ruiz learned that the tunnel opened at midnight."
	book := evidenceTestBook(text, map[string]string{"Ruiz": "ruiz"})
	first := AnalyzeEvidence(&book, nil)
	discovery := findEvidenceType(first.Records, "discovery")
	if discovery == nil {
		t.Fatal("expected discovery")
	}
	discovery.Status = "confirmed"
	discovery.AuthorNote = "Main investigation turn"
	author := types.EvidenceRecord{ID: "author-1", Kind: "fact", EvidenceType: "custom", Text: "The author pinned this.", Source: "author", Status: "confirmed", ChapterIndex: 0}
	first.Records = append(first.Records, author)
	book.Analysis.Evidence = first

	second := AnalyzeEvidence(&book, nil)
	preserved := findEvidenceType(second.Records, "discovery")
	if preserved == nil || preserved.Status != "confirmed" || preserved.AuthorNote != "Main investigation turn" {
		t.Fatalf("review decision was not preserved: %+v", preserved)
	}
	if got := findEvidenceID(second.Records, "author-1"); got == nil || got.Text != author.Text {
		t.Fatalf("author evidence was not preserved: %+v", got)
	}
}

func TestAnalyzeEvidenceCapturesSingletonNamedArrivalForReview(t *testing.T) {
	book := evidenceTestBook("Kyle walked into the interrogation room.", nil)
	result := AnalyzeEvidence(&book, nil)
	record := findEvidenceType(result.Records, "transition")
	if record == nil {
		t.Fatalf("expected named arrival evidence for singleton Kyle, got %+v", result.Records)
	}
	if len(record.NamedEntities) == 0 || !strings.EqualFold(record.NamedEntities[0].Text, "Kyle") {
		t.Fatalf("expected Kyle as supporting named evidence, got %+v", record.NamedEntities)
	}
}

func TestAnalyzeEvidenceDoesNotIndexUnanchoredOrdinaryProse(t *testing.T) {
	book := evidenceTestBook("Rain tapped softly against the dark windows.", nil)
	result := AnalyzeEvidence(&book, nil)
	if len(result.Records) != 0 {
		t.Fatalf("ordinary unanchored prose became evidence: %+v", result.Records)
	}
}

func evidenceTestBook(text string, characters map[string]string) types.BookData {
	book := types.BookData{
		Body:     []types.ChapterItem{{ID: "chapter-one", Title: "Chapter One", Type: "chapter", Content: "<p>" + text + "</p>"}},
		Analysis: types.AnalysisData{EntityResolution: &types.EntityData{Version: 1}},
	}
	for name, id := range characters {
		mentionID := "mention-" + strings.ToLower(name)
		book.Analysis.EntityResolution.Mentions = append(book.Analysis.EntityResolution.Mentions, types.MentionRecord{
			ID: mentionID, Text: name, Chapter: 0, CharOffset: strings.Index(text, name),
		})
		book.Analysis.EntityResolution.Entities = append(book.Analysis.EntityResolution.Entities, types.EntityRecord{
			ID: id, Canonical: name, Aliases: []string{name}, MentionIDs: []string{mentionID}, DetectionStatus: "accepted",
		})
	}
	return book
}

func findEvidenceType(records []types.EvidenceRecord, evidenceType string) *types.EvidenceRecord {
	for index := range records {
		if records[index].EvidenceType == evidenceType {
			return &records[index]
		}
	}
	return nil
}

func findEvidenceID(records []types.EvidenceRecord, id string) *types.EvidenceRecord {
	for index := range records {
		if records[index].ID == id {
			return &records[index]
		}
	}
	return nil
}
