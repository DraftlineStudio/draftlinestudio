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
	if result.Engine != "prose-v3-evidence-v3" || result.Version != 3 {
		t.Fatalf("unexpected evidence schema identity: engine=%s version=%d", result.Engine, result.Version)
	}
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

func TestAnalyzeEvidenceReusesUnchangedChapterCache(t *testing.T) {
	book := evidenceTestBook("Hanlon discovered the tunnel beneath IBM.", map[string]string{"Hanlon": "hanlon"})
	first := AnalyzeEvidence(&book, nil)
	if first.ChapterHashes["chapter-one"] == "" {
		t.Fatal("missing chapter cache hash")
	}
	book.Analysis.Evidence = first
	// If the chapter were passed through extraction again this deliberately
	// absent mention span would remove its confirmed character link.
	book.Analysis.EntityResolution.Mentions = nil
	second := AnalyzeEvidence(&book, nil)
	discovery := findEvidenceType(second.Records, "discovery")
	if discovery == nil || len(discovery.CharacterIDs) != 1 || discovery.CharacterIDs[0] != "hanlon" {
		t.Fatalf("unchanged chapter was not reused: %+v", discovery)
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
	discovery.AuthorText = "Ruiz confirms the tunnel opens at midnight."
	discovery.AuthorNote = "Main investigation turn"
	discovery.Pinned = true
	discovery.ReviewedAt = "2026-09-01T07:00:00Z"
	author := types.EvidenceRecord{ID: "author-1", Kind: "fact", EvidenceType: "custom", Text: "The author pinned this.", Source: "author", Status: "confirmed", ChapterIndex: 0}
	first.Records = append(first.Records, author)
	book.Analysis.Evidence = first

	second := AnalyzeEvidence(&book, nil)
	preserved := findEvidenceType(second.Records, "discovery")
	if preserved == nil || preserved.Status != "confirmed" || preserved.AuthorText != discovery.AuthorText ||
		preserved.AuthorNote != "Main investigation turn" || !preserved.Pinned || preserved.ReviewedAt != discovery.ReviewedAt {
		t.Fatalf("review decision was not preserved: %+v", preserved)
	}
	if got := findEvidenceID(second.Records, "author-1"); got == nil || got.Text != author.Text {
		t.Fatalf("author evidence was not preserved: %+v", got)
	}
}

func TestAnalyzeEvidencePreservesPinnedDetectedRecord(t *testing.T) {
	book := evidenceTestBook("Ruiz learned that the tunnel opened at midnight.", map[string]string{"Ruiz": "ruiz"})
	first := AnalyzeEvidence(&book, nil)
	discovery := findEvidenceType(first.Records, "discovery")
	if discovery == nil {
		t.Fatal("expected discovery")
	}
	discovery.Pinned = true
	discovery.AuthorNote = "Review this when the timeline is built"
	book.Analysis.Evidence = first

	preserved := findEvidenceType(AnalyzeEvidence(&book, nil).Records, "discovery")
	if preserved == nil || !preserved.Pinned || preserved.AuthorNote != discovery.AuthorNote || preserved.Status != "detected" {
		t.Fatalf("pinned unconfirmed record was not preserved: %+v", preserved)
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

func TestAnalyzeEvidenceLinksKnowledgeAcquisitionToSourceCharacter(t *testing.T) {
	book := evidenceTestBook("Ruiz learned that IBM controlled the tunnel.", map[string]string{"Ruiz": "ruiz"})
	record := findEvidenceType(AnalyzeEvidence(&book, nil).Records, "discovery")
	if record == nil || len(record.KnowledgeStates) != 1 {
		t.Fatalf("expected source-backed knowledge acquisition, got %+v", record)
	}
	claim := record.KnowledgeStates[0]
	if claim.State != "learned" || len(claim.CharacterIDs) != 1 || claim.CharacterIDs[0] != "ruiz" || claim.Cue != "learned" {
		t.Fatalf("knowledge acquisition linked to the wrong character: %+v", claim)
	}
}

func TestAnalyzeEvidenceLinksKnowledgeTransferParticipants(t *testing.T) {
	book := evidenceTestBook("Hanlon told Ruiz that IBM controlled the tunnel.", map[string]string{"Hanlon": "hanlon", "Ruiz": "ruiz"})
	record := findEvidenceType(AnalyzeEvidence(&book, nil).Records, "interaction")
	if record == nil || len(record.KnowledgeStates) != 1 {
		t.Fatalf("expected source-backed knowledge transfer, got %+v", record)
	}
	claim := record.KnowledgeStates[0]
	if claim.State != "shared" || len(claim.CharacterIDs) != 1 || claim.CharacterIDs[0] != "hanlon" ||
		len(claim.CounterpartyIDs) != 1 || claim.CounterpartyIDs[0] != "ruiz" {
		t.Fatalf("knowledge transfer participants are wrong: %+v", claim)
	}
}

func TestAnalyzeEvidenceDoesNotInventSpeakerWithoutCharacterBeforeCue(t *testing.T) {
	book := evidenceTestBook("The report informed Ruiz about the tunnel.", map[string]string{"Ruiz": "ruiz"})
	result := AnalyzeEvidence(&book, nil)
	for _, record := range result.Records {
		if len(record.KnowledgeStates) > 0 {
			t.Fatalf("passive or non-character source was promoted to a speaker: %+v", record.KnowledgeStates)
		}
	}
}

func TestAnalyzeEvidencePreservesExplicitLackOfKnowledge(t *testing.T) {
	book := evidenceTestBook("Hanlon didn't know the tunnel existed.", map[string]string{"Hanlon": "hanlon"})
	record := findEvidenceType(AnalyzeEvidence(&book, nil).Records, "state")
	if record == nil || len(record.KnowledgeStates) != 1 || record.KnowledgeStates[0].State != "does_not_know" {
		t.Fatalf("negative knowledge was flattened into positive knowledge: %+v", record)
	}
}

func TestAnalyzeEvidenceDistinguishesBeliefFromKnowledge(t *testing.T) {
	book := evidenceTestBook("Hanlon believed the tunnel was beneath IBM.", map[string]string{"Hanlon": "hanlon"})
	record := findEvidenceType(AnalyzeEvidence(&book, nil).Records, "state")
	if record == nil || len(record.KnowledgeStates) != 1 || record.KnowledgeStates[0].State != "believes" {
		t.Fatalf("belief was promoted to certain knowledge: %+v", record)
	}
}

func TestAnalyzeEvidencePreservesDisbeliefAndWithholding(t *testing.T) {
	disbeliefBook := evidenceTestBook("Keller would never believe the scream was real.", map[string]string{"Keller": "keller"})
	disbelief := findEvidenceType(AnalyzeEvidence(&disbeliefBook, nil).Records, "state")
	if disbelief == nil || len(disbelief.KnowledgeStates) != 1 || disbelief.KnowledgeStates[0].State != "does_not_believe" {
		t.Fatalf("disbelief was flattened into belief: %+v", disbelief)
	}

	withheldBook := evidenceTestBook("Hanlon didn't tell Ruiz about the tunnel.", map[string]string{"Hanlon": "hanlon", "Ruiz": "ruiz"})
	withheld := findEvidenceType(AnalyzeEvidence(&withheldBook, nil).Records, "interaction")
	if withheld == nil || len(withheld.KnowledgeStates) != 1 || withheld.KnowledgeStates[0].State != "withheld" ||
		len(withheld.KnowledgeStates[0].CounterpartyIDs) != 1 || withheld.KnowledgeStates[0].CounterpartyIDs[0] != "ruiz" {
		t.Fatalf("withholding was flattened into sharing: %+v", withheld)
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
