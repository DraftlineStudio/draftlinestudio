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

func TestSearchBuildsInsightAcrossAllMatchesBeyondDisplayLimit(t *testing.T) {
	book := types.BookData{Body: []types.ChapterItem{
		{ID: "chapter-1", Title: "Arrival", Type: "chapter", Content: `<p>Hanlon found the brass key beside IBM.</p>`},
		{ID: "chapter-2", Title: "Return", Type: "chapter", Content: `<p>Ruiz returned the brass key to Hanlon.</p>`},
	}}
	book.Analysis.Evidence = &types.EvidenceData{Records: []types.EvidenceRecord{
		{ID: "discovery", Kind: "event", EvidenceType: "discovery", ChapterIndex: 0, Text: "Hanlon found the brass key beside IBM.", CharacterNames: []string{"Daniel Hanlon"}, NamedEntities: []types.EvidenceTerm{{Text: "IBM", Label: "ORG"}}, Status: "detected"},
		{ID: "return", Kind: "event", EvidenceType: "transition", ChapterIndex: 1, Text: "Ruiz returned the brass key to Hanlon.", CharacterNames: []string{"Ruiz", "Daniel Hanlon"}, Status: "detected"},
	}}

	result := Search(book, types.StorySearchRequest{Query: "brass key", Limit: 1})
	if result.Total != 2 || len(result.Matches) != 1 {
		t.Fatalf("expected two total matches with one displayed, got total=%d displayed=%d", result.Total, len(result.Matches))
	}
	if result.Insight == nil || result.Insight.ChapterCount != 2 || result.Insight.EventCount != 2 || len(result.Insight.Chapters) != 2 {
		t.Fatalf("insight did not cover the complete trail: %+v", result.Insight)
	}
	if len(result.Insight.RelatedTerms) == 0 || result.Insight.RelatedTerms[0].Text != "Daniel Hanlon" {
		t.Fatalf("expected repeated related character to rank first: %+v", result.Insight.RelatedTerms)
	}
}

func TestSearchInterpretsNaturalResearchQuestion(t *testing.T) {
	book := testBook(`<p>Hanlon and Ruiz researched IBM together.</p>`)
	book.Analysis.EntityResolution = &types.EntityData{Entities: []types.EntityRecord{
		{ID: "hanlon", Canonical: "Daniel Hanlon", Aliases: []string{"Hanlon"}, DetectionStatus: "accepted"},
		{ID: "ruiz", Canonical: "Ruiz", Aliases: []string{"Ruiz"}, DetectionStatus: "accepted"},
	}}

	result := Search(book, types.StorySearchRequest{Query: "What chapter did Hanlon and Ruiz research IBM?"})
	if result.Total != 1 || result.Insight == nil || result.Insight.Intent != "research" {
		t.Fatalf("natural research question was not interpreted: %+v", result)
	}
	if result.Insight.InterpretedQuery != "Hanlon Ruiz IBM" {
		t.Fatalf("unexpected interpreted query: %q", result.Insight.InterpretedQuery)
	}
}

func TestSearchAnswersConfirmedNameQuestionWithoutInventingGivenName(t *testing.T) {
	book := testBook(`<p>Detective Ruiz entered. Hanlon followed him.</p>`)
	book.Analysis.EntityResolution = &types.EntityData{Entities: []types.EntityRecord{
		{ID: "ruiz", Canonical: "Ruiz", Aliases: []string{"Ruiz", "Detective Ruiz"}, DetectionStatus: "accepted"},
		{ID: "hanlon", Canonical: "Daniel Hanlon", Aliases: []string{"Hanlon"}, DetectionStatus: "accepted"},
	}}

	ruiz := Search(book, types.StorySearchRequest{Query: "Ruiz first name"})
	if ruiz.Total != 1 || ruiz.Insight == nil || len(ruiz.Insight.Signals) == 0 || ruiz.Insight.Signals[0].Title != "No confirmed given name found" {
		t.Fatalf("expected an honest missing-name answer, got %+v", ruiz.Insight)
	}

	hanlon := Search(book, types.StorySearchRequest{Query: "What is Hanlon's full name?"})
	if hanlon.Total != 1 || hanlon.Insight == nil || len(hanlon.Insight.Signals) == 0 || hanlon.Insight.Signals[0].Title != "Daniel Hanlon" {
		t.Fatalf("expected confirmed full-name answer, got %+v", hanlon.Insight)
	}
}

func TestSearchFlagsSingletonDetail(t *testing.T) {
	result := Search(testBook(`<p>Kyle entered with the coffee and was never seen again.</p>`), types.StorySearchRequest{Query: "Kyle"})
	if result.Insight == nil || len(result.Insight.Signals) == 0 || result.Insight.Signals[0].Title != "Appears in one scene" {
		t.Fatalf("expected singleton signal, got %+v", result.Insight)
	}
}

func TestSearchRelatedTermsExcludePrimaryPossessivesAndPronounContractions(t *testing.T) {
	book := testBook(`<p>Hanlon reviewed IBM's records.</p>`)
	book.Analysis.EntityResolution = &types.EntityData{Entities: []types.EntityRecord{{
		ID: "hanlon", Canonical: "Daniel Hanlon", Aliases: []string{"Hanlon"}, DetectionStatus: "accepted",
	}}}
	book.Analysis.Evidence = &types.EvidenceData{Records: []types.EvidenceRecord{{
		ID: "record", Kind: "fact", EvidenceType: "state", ChapterIndex: 0,
		Text: "Hanlon reviewed IBM's records.", CharacterNames: []string{"Daniel Hanlon"},
		NamedEntities: []types.EvidenceTerm{{Text: "Hanlon’s", Label: "PERSON"}, {Text: "He’d", Label: "PERSON"}, {Text: "IBM", Label: "ORG"}},
		Status:        "detected",
	}}}

	result := Search(book, types.StorySearchRequest{Query: "Hanlon"})
	if result.Insight == nil || len(result.Insight.RelatedTerms) != 1 || result.Insight.RelatedTerms[0].Text != "IBM" {
		t.Fatalf("unexpected related-term noise: %+v", result.Insight)
	}
}

func testBook(content string) types.BookData {
	return types.BookData{
		Body: []types.ChapterItem{{ID: "chapter-1", Title: "First", Type: "chapter", Content: content}},
	}
}
