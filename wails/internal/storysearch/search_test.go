package storysearch

import (
	"testing"

	"draftline/internal/types"
)

func TestSearchExpandsConfirmedCharacterAliasesAcrossScene(t *testing.T) {
	book := testBook(`<p>Detective Hanlon reviewed the old files.</p><p>Danny found the IBM receipt under the desk.</p>`)
	book.Analysis.EntityResolution = &types.EntityData{Entities: []types.EntityRecord{{
		ID: "hanlon", Canonical: "Daniel Hanlon", Aliases: []string{"Daniel Hanlon", "Hanlon", "Danny"}, DetectionStatus: "accepted",
	}}}

	result := Search(book, types.StorySearchRequest{Query: "Hanlon IBM"})
	if result.Error != "" {
		t.Fatalf("unexpected error: %s", result.Error)
	}
	if result.Total != 1 || len(result.Matches) != 1 {
		t.Fatalf("expected one scene match, got total=%d matches=%d", result.Total, len(result.Matches))
	}
	if len(result.ResolvedEntities) != 1 || result.ResolvedEntities[0].Canonical != "Daniel Hanlon" {
		t.Fatalf("expected Hanlon alias expansion, got %+v", result.ResolvedEntities)
	}
	if result.Matches[0].MatchedTerms[0] != "Hanlon" || result.Matches[0].MatchedTerms[1] != "IBM" {
		t.Fatalf("unexpected matched evidence: %+v", result.Matches[0].MatchedTerms)
	}
}

func TestSearchDoesNotCrossSceneBreak(t *testing.T) {
	book := testBook(`<p>Hanlon found the access card.</p><hr><p>Ruiz found the tunnel.</p>`)

	result := Search(book, types.StorySearchRequest{Query: "Hanlon tunnel"})
	if result.Total != 0 {
		t.Fatalf("expected no cross-scene match, got %+v", result.Matches)
	}
}

func TestSearchDoesNotExpandUnconfirmedEntity(t *testing.T) {
	book := testBook(`<p>Danny entered the lobby.</p>`)
	book.Analysis.EntityResolution = &types.EntityData{Entities: []types.EntityRecord{{
		ID: "hanlon", Canonical: "Daniel Hanlon", Aliases: []string{"Hanlon", "Danny"}, DetectionStatus: "review",
	}}}

	result := Search(book, types.StorySearchRequest{Query: "Hanlon"})
	if len(result.ResolvedEntities) != 0 {
		t.Fatalf("review entity must not expand aliases: %+v", result.ResolvedEntities)
	}
	if result.Total != 0 {
		t.Fatalf("literal Hanlon is absent; expected no match, got %+v", result.Matches)
	}
}

func TestSearchReportsTotalBeyondLimitInStoryOrder(t *testing.T) {
	book := testBook(`<p>The key appeared.</p><hr><p>The second key appeared.</p><hr><p>The final key appeared.</p>`)

	result := Search(book, types.StorySearchRequest{Query: "key", Limit: 2})
	if result.Total != 3 || len(result.Matches) != 2 {
		t.Fatalf("expected total 3 limited to 2, got total=%d matches=%d", result.Total, len(result.Matches))
	}
	if result.Matches[0].SceneIndex != 0 || result.Matches[1].SceneIndex != 1 {
		t.Fatalf("results not in story order: %+v", result.Matches)
	}
}

func TestSearchHonorsQuotedPhrase(t *testing.T) {
	book := testBook(`<p>They searched One IBM Plaza before dawn.</p><hr><p>IBM occupied one plaza downtown.</p>`)

	result := Search(book, types.StorySearchRequest{Query: `"One IBM Plaza"`})
	if result.Total != 1 {
		t.Fatalf("expected exact phrase to match once, got %d", result.Total)
	}
}

func TestSearchAttachesPersistedEvidenceAndExcludesRejectedRecords(t *testing.T) {
	book := testBook(`<p>Hanlon discovered the underground tunnel beneath the plaza.</p>`)
	book.Analysis.Evidence = &types.EvidenceData{Records: []types.EvidenceRecord{
		{ID: "kept", Kind: "event", EvidenceType: "discovery", ChapterIndex: 0, Text: "Hanlon discovered the underground tunnel beneath the plaza.", Status: "detected"},
		{ID: "rejected", Kind: "fact", EvidenceType: "state", ChapterIndex: 0, Text: "Hanlon discovered the underground tunnel beneath the plaza.", Status: "rejected"},
	}}

	result := Search(book, types.StorySearchRequest{Query: "underground tunnel"})
	if len(result.Matches) != 1 || len(result.Matches[0].Evidence) != 1 || result.Matches[0].Evidence[0].ID != "kept" {
		t.Fatalf("search evidence attachment is wrong: %+v", result.Matches)
	}
}

func testBook(content string) types.BookData {
	return types.BookData{
		Body: []types.ChapterItem{{ID: "chapter-1", Title: "First", Type: "chapter", Content: content}},
	}
}
