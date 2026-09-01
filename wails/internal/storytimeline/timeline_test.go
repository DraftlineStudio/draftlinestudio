package storytimeline

import (
	"testing"

	"draftline/internal/types"
)

func TestBuildDeduplicatesSourceRecordsAndPreservesManuscriptOrder(t *testing.T) {
	book := timelineBook([]types.EvidenceRecord{
		record("later", 1, 40, "event", "transition", "Two hours later, Hanlon entered the tunnel.", []string{"Two hours later"}),
		record("first", 0, 20, "event", "introduction", "Hanlon entered the station.", nil),
		record("duplicate", 0, 20, "event", "transition", "Hanlon entered the station.", nil),
	})

	result := Build(book)
	if !result.Success || len(result.Events) != 2 {
		t.Fatalf("expected two events, got success=%v events=%d error=%q", result.Success, len(result.Events), result.Error)
	}
	if result.Events[0].ID != "first" || result.Events[0].PrimaryType != "transition" {
		t.Fatalf("expected merged first event with transition priority, got %#v", result.Events[0])
	}
	if len(result.Events[0].EvidenceIDs) != 2 {
		t.Fatalf("expected two source records to merge, got %v", result.Events[0].EvidenceIDs)
	}
	if result.Events[1].TimeKind != "relative" || result.RelativeTimeCount != 1 {
		t.Fatalf("expected relative time classification, got %#v", result.Events[1])
	}
}

func TestBuildIncludesTimedFactsAndExcludesOrdinaryOrRejectedFacts(t *testing.T) {
	ordinary := record("ordinary", 0, 10, "fact", "state", "Hanlon had a truck.", nil)
	timed := record("timed", 0, 20, "fact", "time_reference", "At midnight, Hanlon called Ruiz.", []string{"midnight"})
	rejected := record("rejected", 0, 30, "event", "discovery", "Hanlon found the door.", nil)
	rejected.Status = "rejected"

	result := Build(timelineBook([]types.EvidenceRecord{ordinary, timed, rejected}))
	if len(result.Events) != 1 || result.Events[0].ID != "timed" {
		t.Fatalf("expected only timed fact, got %#v", result.Events)
	}
	if result.Events[0].TimeKind != "anchored" || result.ExplicitTimeCount != 1 {
		t.Fatalf("expected anchored explicit time, got %#v", result.Events[0])
	}
}

func TestBuildCleansTokenizerWhitespaceFromTimeLabels(t *testing.T) {
	event := record("countdown", 0, 20, "event", "time_reference", "4:59 4:58 4:57", []string{"4:59", "4:57\n\nA"})
	result := Build(timelineBook([]types.EvidenceRecord{event}))
	if result.Events[0].TimeLabel != "4:59 · 4:57 · explicit time reference" {
		t.Fatalf("unexpected cleaned time label %q", result.Events[0].TimeLabel)
	}
}

func TestBuildCreatesCharacterLocationTypeAndChapterFacets(t *testing.T) {
	record := record("discovery", 0, 10, "event", "discovery", "Hanlon found the hatch beneath Chicago.", nil)
	record.CharacterIDs = []string{"hanlon"}
	record.CharacterNames = []string{"Daniel Hanlon"}
	record.NamedEntities = []types.EvidenceTerm{{Text: "Chicago", Label: "GPE"}, {Text: "IBM", Label: "ORG"}}

	result := Build(timelineBook([]types.EvidenceRecord{record}))
	if len(result.Characters) != 1 || result.Characters[0].Label != "Daniel Hanlon" {
		t.Fatalf("unexpected character facets: %#v", result.Characters)
	}
	if len(result.Locations) != 1 || result.Locations[0].Label != "Chicago" {
		t.Fatalf("unexpected location facets: %#v", result.Locations)
	}
	if len(result.EventTypes) != 1 || result.EventTypes[0].ID != "discovery" {
		t.Fatalf("unexpected type facets: %#v", result.EventTypes)
	}
	if len(result.Chapters) != 1 || result.Chapters[0].ChapterTitle != "Chapter 1" {
		t.Fatalf("unexpected chapters: %#v", result.Chapters)
	}
}

func TestBuildKeepsLocationFacetsConservative(t *testing.T) {
	book := timelineBook([]types.EvidenceRecord{
		locationRecord("character", "Ruiz stood beside Hanlon’s truck.", "Hanlon’s", "GPE"),
		locationRecord("language", "The sign was written in English.", "English", "GPE"),
		locationRecord("acronym", "They waited under the AMA building.", "AMA", "GPE"),
		locationRecord("place", "They met at City Hall Wednesday.", "City Hall Wednesday", "FACILITY"),
	})
	book.StoryBible.Characters = []types.Character{{ID: "hanlon", Name: "Daniel Hanlon", Aliases: []string{"Hanlon"}}}

	result := Build(book)
	if len(result.Locations) != 1 || result.Locations[0].Label != "City Hall" {
		t.Fatalf("expected only cleaned City Hall facet, got %#v", result.Locations)
	}
}

func TestBuildReportsMissingEvidenceIndex(t *testing.T) {
	result := Build(types.BookData{})
	if result.Success || result.Error == "" || result.Events == nil {
		t.Fatalf("expected safe missing-index result, got %#v", result)
	}
}

func TestBuildKeepsExactSourceWhenAuthorInterpretationIsDisplayed(t *testing.T) {
	event := record("author", 0, 10, "event", "discovery", "Hanlon found the original source.", nil)
	event.AuthorText = "Hanlon discovers the hidden evidence."
	result := Build(timelineBook([]types.EvidenceRecord{event}))
	if result.Events[0].Text != event.AuthorText || result.Events[0].SourceText != event.Text {
		t.Fatalf("expected separate display and source text, got %#v", result.Events[0])
	}
}

func timelineBook(records []types.EvidenceRecord) types.BookData {
	return types.BookData{
		Body:     []types.ChapterItem{{ID: "chapter-1", Title: "Chapter 1"}, {ID: "chapter-2", Title: "Chapter 2"}},
		Analysis: types.AnalysisData{Evidence: &types.EvidenceData{Engine: "test", Records: records}},
	}
}

func record(id string, chapter, offset int, kind, evidenceType, text string, times []string) types.EvidenceRecord {
	return types.EvidenceRecord{
		ID: id, ChapterID: "chapter-" + intString(chapter+1), ChapterIndex: chapter, Section: "body", SectionIndex: chapter,
		ParagraphIndex: 0, SentenceIndex: offset / 10, StartOffset: offset, Kind: kind, EvidenceType: evidenceType,
		Text: text, TimeExpressions: times, Confidence: .8, Status: "detected", Source: "auto",
	}
}

func locationRecord(id, text, term, label string) types.EvidenceRecord {
	record := record(id, 0, len(id)*10, "event", "transition", text, nil)
	record.NamedEntities = []types.EvidenceTerm{{Text: term, Label: label}}
	return record
}
