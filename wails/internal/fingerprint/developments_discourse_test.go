package fingerprint

import (
	"strings"
	"testing"

	"draftline/internal/types"
)

// A recollection in the current scene neither answers nor narrows an open
// question; a later narrated acquisition does, and the development names
// the question it answers.
func TestRecollectionDoesNotResolveOrNarrowAQuestion(t *testing.T) {
	book := frameBook([]types.EvidenceRecord{
		evidence("e1", 0, "Avery suspected that the courier had hidden the vault key in the loft.", "fact", "knowledge_state", "suspected"),
		evidence("e2", 1, "Avery remembered that the courier had hidden the vault key beneath the loft stair.", "fact", "knowledge_state", "remembered"),
		evidence("e3", 2, "Avery discovered that the courier had hidden the vault key behind the loft stair.", "event", "discovery", "discovered"),
	})
	model := Build(&book, nil)
	for _, frame := range model.Frames {
		if frame.Type == types.FrameKnowledge && frame.ChapterIndex == 1 && frame.Value != "recalled" {
			t.Fatalf("a remembered fact must carry value recalled: %+v", frame)
		}
	}
	for _, development := range model.Developments {
		if development.ChapterIndex == 1 {
			t.Fatalf("a recollection completed a beat: %s %q", development.Kind, development.Summary)
		}
	}
	resolved := developmentOfKind(model.Developments, "mystery_resolved")
	if resolved == nil || resolved.ChapterIndex != 2 || resolved.Discourse != "narration" {
		t.Fatalf("the narrated discovery must resolve the question with narration discourse: %+v", resolved)
	}
	if !strings.Contains(resolved.Advances, "vault key in the loft") {
		t.Fatalf("the resolution must name the question it answers: %q", resolved.Advances)
	}
}

// A fact learned inside a flashback is not a current discovery and does not
// advance a current goal; the flashback's own goal opens no current pursuit.
func TestFlashbackFactsDoNotAdvanceCurrentStakes(t *testing.T) {
	book := frameBook([]types.EvidenceRecord{
		evidence("e1", 0, "Avery wanted to find the vault ledger before the river froze.", "fact", "state", "wanted"),
		evidence("e2", 1, "Avery had worked the vault years earlier, when the courier still kept the ledger.", "fact", "state", "worked"),
		evidence("e3", 1, "Avery learned that the vault ledger sat under the counting bench.", "event", "discovery", "learned"),
		evidence("e4", 1, "Avery wanted to reach the counting bench before the courier returned.", "fact", "state", "wanted"),
	})
	model := Build(&book, nil)
	for _, development := range model.Developments {
		if development.ChapterIndex == 1 {
			t.Fatalf("a flashback produced a development: %s %q (discourse %s)", development.Kind, development.Summary, development.Discourse)
		}
	}
	if goal := developmentOfKind(model.Developments, "goal_established"); goal == nil || goal.ChapterIndex != 0 {
		t.Fatalf("the current goal must still be established: %+v", goal)
	}
}

// A knowledge sentence inside quoted speech is the speaker's claim: it is
// attributed, and it never answers the open question.
func TestQuotedKnowledgeIsAClaimNotNarration(t *testing.T) {
	book := frameBook([]types.EvidenceRecord{
		evidence("e1", 0, "Avery suspected that the courier had hidden the vault key in the loft.", "fact", "knowledge_state", "suspected"),
		evidence("e2", 1, "“Avery learned that the courier had hidden the vault key in the loft,” Mira said.", "event", "discovery", "learned"),
	})
	model := Build(&book, nil)
	knowledge := 0
	for _, frame := range model.Frames {
		if frame.Type != types.FrameKnowledge {
			continue
		}
		knowledge++
		if frame.Epistemic != "attributed_claim" || frame.Attribution.EntityName != "Mira Brannick" {
			t.Fatalf("quoted knowledge must be attributed to the speaker: %+v", frame)
		}
	}
	if knowledge == 0 {
		t.Fatal("expected a knowledge frame for the quoted sentence")
	}
	if resolved := developmentOfKind(model.Developments, "mystery_resolved"); resolved != nil {
		t.Fatalf("a quoted claim resolved the question: %+v", resolved)
	}
	if narrowed := developmentOfKind(model.Developments, "mystery_narrowed"); narrowed != nil {
		t.Fatalf("a quoted claim narrowed the question: %+v", narrowed)
	}
}

// An injury term inside a learned clause belongs to the clause, not to the
// sentence's subject.
func TestInjuryInsideKnownClauseAbstains(t *testing.T) {
	book := frameBook([]types.EvidenceRecord{
		evidence("e1", 0, "Avery learned that the counting house had burned to the ground.", "event", "discovery", "learned"),
		evidence("e2", 0, "Avery was bleeding from a cut along her wrist.", "fact", "state", "was"),
	})
	frames := extract(t, book)
	injuries := 0
	for _, frame := range frames {
		if frame.Type == types.FrameInjury {
			injuries++
			if frame.Detail != "bleeding" {
				t.Fatalf("the burned building became an injury: %+v", frame)
			}
		}
	}
	if injuries != 1 {
		t.Fatalf("expected exactly the wrist injury, got %d injury frames", injuries)
	}
}

// Every development carries the engine's discourse label; a claim tied to
// an open question is labelled reported on the record itself.
func TestDevelopmentsCarryDiscourse(t *testing.T) {
	book := frameBook([]types.EvidenceRecord{
		evidence("e1", 0, "Avery suspected that the courier had hidden the vault key in the loft.", "fact", "knowledge_state", "suspected"),
		evidence("e2", 1, "“The courier drowned with the vault key in his coat,” Mira said.", "event", "interaction", "said"),
	})
	model := Build(&book, nil)
	if len(model.Developments) == 0 {
		t.Fatal("expected developments")
	}
	reported := false
	for _, development := range model.Developments {
		if development.Discourse == "" {
			t.Fatalf("development without discourse: %+v", development)
		}
		if development.Kind == "investigation_progress" && development.Discourse == "reported" {
			reported = true
		}
	}
	if !reported {
		t.Fatalf("the claim tied to the question must be labelled reported: %+v", model.Developments)
	}
}

// A goal's object named late in an arrival still ties the arrival to the
// goal: the stake keeps its clause head, the fact contributes its whole
// phrase.
func TestArrivalTiesToGoalNamedLateInThePhrase(t *testing.T) {
	book := frameBook([]types.EvidenceRecord{
		evidence("e1", 0, "Avery wanted to bring the second ledger to the village before the ice closed the road.", "fact", "state", "wanted"),
		evidence("e2", 1, "Avery returned to the village with the vault key and the second ledger.", "event", "transition", "returned"),
	})
	model := Build(&book, nil)
	progress := developmentOfKind(model.Developments, "investigation_progress")
	if progress == nil || !strings.Contains(progress.Summary, "makes progress toward") || !strings.Contains(progress.Advances, "second ledger") {
		t.Fatalf("the arrival must count as progress toward the goal: %+v", progress)
	}
}
