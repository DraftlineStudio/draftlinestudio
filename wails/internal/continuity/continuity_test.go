package continuity

import (
	"strings"
	"testing"

	"draftline/internal/types"
)

func TestBuildFindsMissingGivenNameAndLateOneOffCharacter(t *testing.T) {
	book := testBook()
	book.StoryBible.Characters = []types.Character{
		{ID: "ruiz", Name: "Ruiz", EntityKind: "person", IsAutoDetected: true, DetectionStatus: "accepted", DetectionScore: .99, Aliases: []string{"Detective Ruiz"}, MentionCount: 12, ChapterMentions: map[int]int{0: 4, 1: 8}},
		{ID: "kyle", Name: "Kyle", EntityKind: "person", IsAutoDetected: true, DetectionStatus: "accepted", DetectionScore: .95, MentionCount: 1, FirstChapter: 4, ChapterMentions: map[int]int{4: 1}},
	}
	book.Analysis.Evidence.Records = []types.EvidenceRecord{
		evidence("ruiz-source", 0, 10, "Ruiz entered the station.", "fact", "state", []string{"ruiz"}, []string{"Ruiz"}),
		evidence("kyle-source", 4, 10, "Kyle walked in and was never seen again.", "event", "introduction", []string{"kyle"}, []string{"Kyle"}),
	}

	report := Build(book)
	assertSignal(t, report, "missing-given-name")
	oneOff := assertSignal(t, report, "one-chapter-character")
	if oneOff.Severity != "review" || !strings.Contains(oneOff.Title, "Kyle") {
		t.Fatalf("expected late Kyle review, got %#v", oneOff)
	}
}

func TestCharacterChecksExcludeNonPeopleLowConfidenceAndResolvedSurnames(t *testing.T) {
	book := testBook()
	book.StoryBible.Characters = []types.Character{
		{ID: "org", Name: "Emergency Response", EntityKind: "organization", IsAutoDetected: true, DetectionStatus: "accepted", DetectionScore: 1, MentionCount: 1, ChapterMentions: map[int]int{4: 1}},
		{ID: "noise", Name: "Christ", EntityKind: "person", IsAutoDetected: true, DetectionStatus: "accepted", DetectionScore: .65, MentionCount: 6, ChapterMentions: map[int]int{1: 6}},
		{ID: "keller-short", Name: "Keller", EntityKind: "person", IsAutoDetected: true, DetectionStatus: "accepted", DetectionScore: 1, MentionCount: 20, ChapterMentions: map[int]int{0: 20}},
		{ID: "keller-full", Name: "Gerald Keller", EntityKind: "person", IsAutoDetected: true, DetectionStatus: "accepted", DetectionScore: 1, MentionCount: 10, ChapterMentions: map[int]int{2: 10}},
	}

	report := Build(book)
	for _, item := range report.Signals {
		if item.Category == "characters" {
			t.Fatalf("expected noisy and surname-resolved characters to be excluded, got %#v", item)
		}
	}
}

func TestBuildFindsKnowledgeReversalAndKnowledgeBeforeLearning(t *testing.T) {
	book := testBook()
	book.Analysis.Evidence.Records = []types.EvidenceRecord{
		knowledgeEvidence("knows", 0, 10, "Hanlon knew the tunnel was sealed.", "knows"),
		knowledgeEvidence("negative", 1, 10, "Hanlon did not know the tunnel was sealed.", "does_not_know"),
		knowledgeEvidence("learned", 2, 10, "Hanlon learned the tunnel was sealed.", "learned"),
	}

	report := Build(book)
	reversal := assertSignal(t, report, "knowledge-reversal")
	if len(reversal.Sources) != 2 || reversal.Sources[0].Text == reversal.Sources[1].Text {
		t.Fatalf("expected two distinct sources, got %#v", reversal.Sources)
	}
	assertSignal(t, report, "knowledge-order")
}

func TestBuildFindsConflictingPhysicalFacts(t *testing.T) {
	book := testBook()
	book.Analysis.Evidence.Records = []types.EvidenceRecord{
		evidence("blue", 0, 10, "Hanlon's eyes were blue.", "fact", "state", []string{"hanlon"}, []string{"Daniel Hanlon"}),
		evidence("green", 2, 10, "Hanlon's eyes were green.", "fact", "state", []string{"hanlon"}, []string{"Daniel Hanlon"}),
	}

	signal := assertSignal(t, Build(book), "attribute-conflict")
	if !strings.Contains(signal.Detail, "blue") || !strings.Contains(signal.Detail, "green") {
		t.Fatalf("expected both values in explanation, got %q", signal.Detail)
	}
}

func TestBuildFindsNearbyClockRegressionButSkipsCountdowns(t *testing.T) {
	book := testBook()
	first := evidence("first", 0, 10, "Hanlon arrived at 9:30.", "event", "transition", []string{"hanlon"}, []string{"Daniel Hanlon"})
	first.TimeExpressions = []string{"9:30"}
	first.ParagraphIndex = 1
	second := evidence("second", 0, 20, "Ruiz entered at 8:00.", "event", "transition", []string{"ruiz"}, []string{"Ruiz"})
	second.TimeExpressions = []string{"8:00"}
	second.ParagraphIndex = 2
	countdown := evidence("countdown", 1, 10, "The countdown dropped from 5:00 to 4:59.", "event", "time_reference", nil, nil)
	countdown.TimeExpressions = []string{"5:00", "4:59"}
	book.Analysis.Evidence.Records = []types.EvidenceRecord{first, second, countdown}

	report := Build(book)
	assertSignal(t, report, "clock-regression")
	for _, item := range report.Signals {
		if item.Kind == "clock-regression" && strings.Contains(item.Detail, "5:00") {
			t.Fatal("countdown should not become a clock regression")
		}
	}
}

func TestBuildReportsMissingEvidenceIndex(t *testing.T) {
	report := Build(types.BookData{})
	if report.Success || report.Error == "" || report.Signals == nil {
		t.Fatalf("expected safe missing-index report, got %#v", report)
	}
}

func testBook() types.BookData {
	body := make([]types.ChapterItem, 6)
	for index := range body {
		body[index] = types.ChapterItem{ID: "chapter-" + string(rune('1'+index)), Title: "Chapter " + string(rune('1'+index)), Content: "<p>Short chapter.</p>"}
	}
	return types.BookData{Body: body, Analysis: types.AnalysisData{Evidence: &types.EvidenceData{Engine: "test", Records: []types.EvidenceRecord{}}}}
}

func evidence(id string, chapter, offset int, text, kind, evidenceType string, characterIDs, characterNames []string) types.EvidenceRecord {
	return types.EvidenceRecord{ID: id, Kind: kind, EvidenceType: evidenceType, ChapterID: "chapter-" + string(rune('1'+chapter)), ChapterIndex: chapter, Section: "body", SectionIndex: chapter, ParagraphIndex: 0, SentenceIndex: offset / 10, StartOffset: offset, Text: text, CharacterIDs: characterIDs, CharacterNames: characterNames, Confidence: .9, Status: "detected", Source: "auto"}
}

func knowledgeEvidence(id string, chapter, offset int, text, state string) types.EvidenceRecord {
	record := evidence(id, chapter, offset, text, "fact", "knowledge_state", []string{"hanlon"}, []string{"Daniel Hanlon"})
	record.KnowledgeStates = []types.EvidenceKnowledgeState{{State: state, CharacterIDs: []string{"hanlon"}, CharacterNames: []string{"Daniel Hanlon"}, Cue: state, Confidence: .9}}
	return record
}

func assertSignal(t *testing.T, report types.ContinuityReport, kind string) types.ContinuitySignal {
	t.Helper()
	for _, signal := range report.Signals {
		if signal.Kind == kind {
			return signal
		}
	}
	t.Fatalf("expected %s signal, got %#v", kind, report.Signals)
	return types.ContinuitySignal{}
}
