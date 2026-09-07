package fingerprint

import (
	"strings"
	"testing"

	"draftline/internal/types"
)

// frameBook builds a book with a canonical roster and the given evidence.
func frameBook(records []types.EvidenceRecord) types.BookData {
	book := testBook(records)
	book.Analysis.EntityResolution = &types.EntityData{Entities: []types.EntityRecord{
		{ID: "entity-1", Canonical: "Avery Cole", Aliases: []string{"Avery Cole", "Avery", "Cole"}},
		{ID: "entity-2", Canonical: "Mira Voss", Aliases: []string{"Mira Voss", "Mira", "Voss"}},
		{ID: "entity-3", Canonical: "Dane", Aliases: []string{"Dane"}, DetectionStatus: "review"},
		{ID: "entity-4", Canonical: "Ghost", Aliases: []string{"Ghost"}, DetectionStatus: "rejected"},
	}}
	return book
}

func evidence(id string, chapter int, text string, kind, evidenceType, action string) types.EvidenceRecord {
	return types.EvidenceRecord{
		ID: id, Kind: kind, EvidenceType: evidenceType, ChapterID: "chapter-1", ChapterIndex: chapter,
		Text: text, Action: action, Confidence: .9, Status: "detected", Source: "auto",
	}
}

func extract(t *testing.T, book types.BookData) []types.NarrativeFrame {
	t.Helper()
	model := Build(&book, nil)
	return model.Frames
}

func frameOfType(frames []types.NarrativeFrame, frameType string) *types.NarrativeFrame {
	for index := range frames {
		if frames[index].Type == frameType {
			return &frames[index]
		}
	}
	return nil
}

func subjectOf(frame *types.NarrativeFrame) string {
	for _, p := range frame.Participants {
		if p.Role == "subject" {
			return p.EntityName
		}
	}
	return ""
}

// ── the hard invariants ──────────────────────────────────────────────────────

func TestEveryDetailIsVerbatimAndEveryFrameHasEvidence(t *testing.T) {
	book := frameBook([]types.EvidenceRecord{
		evidence("e1", 0, "Avery Cole picked up the brass key from the desk.", "event", "interaction", "picked"),
		evidence("e2", 0, "Mira Voss was killed in the warehouse fire.", "event", "state", "killed"),
		evidence("e3", 0, "Avery wanted to reach the harbor before dawn.", "fact", "state", "wanted"),
		evidence("e4", 1, `"The ledger is a forgery," Mira said.`, "event", "interaction", "said"),
		evidence("e5", 1, "Dane entered the customs office at noon.", "event", "transition", "entered"),
	})
	frames := extract(t, book)
	if len(frames) == 0 {
		t.Fatal("expected frames")
	}
	byID := map[string]types.EvidenceRecord{}
	for _, record := range book.Analysis.Evidence.Records {
		byID[record.ID] = record
	}
	for _, frame := range frames {
		if len(frame.EvidenceIDs) == 0 || len(frame.EvidenceSpans) == 0 {
			t.Fatalf("frame %s has no evidence", frame.ID)
		}
		if frame.Detail == "" {
			continue
		}
		source := byID[frame.EvidenceIDs[0]].Text
		if !strings.Contains(source, frame.Detail) {
			t.Fatalf("frame %s Detail %q is not a verbatim slice of %q", frame.ID, frame.Detail, source)
		}
	}
}

func TestNoSelfReferentialTransfersOrInventedRelationships(t *testing.T) {
	book := frameBook([]types.EvidenceRecord{
		// The v4 failure mode: one character mentioned twice must never
		// become "Avery handed X to Avery".
		evidence("e1", 0, "Avery Cole handed the envelope to a stranger on the platform.", "event", "interaction", "handed"),
		evidence("e2", 0, "Avery handed the ledger to Mira.", "event", "interaction", "handed"),
	})
	frames := extract(t, book)
	for _, frame := range frames {
		if frame.Type != types.FrameTransfer {
			continue
		}
		var source, recipient string
		for _, p := range frame.Participants {
			if p.Role == "source" {
				source = p.EntityName
			}
			if p.Role == "recipient" {
				recipient = p.EntityName
			}
		}
		if source == recipient {
			t.Fatalf("self-referential transfer: %#v", frame)
		}
	}
	// e1 has no canonical recipient: it must abstain into possession, noting why.
	var abstained bool
	for _, frame := range frames {
		if frame.EvidenceIDs[0] == "e1" && frame.Type == types.FramePossession && len(frame.Abstentions) > 0 {
			abstained = true
		}
	}
	if !abstained {
		t.Fatal("transfer without canonical recipient should abstain into possession with a recorded abstention")
	}
	// e2 resolves fully.
	transfer := frameOfType(frames, types.FrameTransfer)
	if transfer == nil {
		t.Fatal("expected a resolved transfer for e2")
	}
}

func TestNoSubjectMeansNoGuessedFrame(t *testing.T) {
	book := frameBook([]types.EvidenceRecord{
		// Clause subject — the v4 engine made this a proposition subject.
		evidence("e1", 0, "The first thing anyone noticed when the door opened was the smell of rain.", "event", "discovery", "noticed"),
		// Rejected entity must not anchor.
		evidence("e2", 0, "Ghost slipped through the fence.", "event", "transition", "slipped"),
	})
	frames := extract(t, book)
	if len(frames) != 0 {
		t.Fatalf("expected total abstention, got %#v", frames)
	}
}

// ── typed extractors ─────────────────────────────────────────────────────────

func TestLifeStatusLocationPossessionInjury(t *testing.T) {
	book := frameBook([]types.EvidenceRecord{
		evidence("e1", 0, "Mira Voss was killed in the warehouse fire.", "event", "state", "killed"),
		evidence("e2", 0, "Avery was at Northgate Station before sunrise.", "fact", "state", "was"),
		evidence("e3", 1, "Avery picked up the brass key from the desk.", "event", "interaction", "picked"),
		evidence("e4", 1, "Mira was bleeding from a cut above her eye.", "fact", "state", "bleeding"),
	})
	book.Analysis.Evidence.Records[1].NamedEntities = []types.EvidenceTerm{{Text: "Northgate Station", Label: "FAC"}}
	frames := extract(t, book)

	life := frameOfType(frames, types.FrameLifeStatus)
	if life == nil || life.Value != "dead" || subjectOf(life) != "Mira Voss" {
		t.Fatalf("life_status: %#v", life)
	}
	location := frameOfType(frames, types.FrameLocation)
	if location == nil || subjectOf(location) != "Avery Cole" {
		t.Fatalf("location: %#v", location)
	}
	var place string
	for _, p := range location.Participants {
		if p.Role == "place" {
			place = p.EntityName
		}
	}
	if place != "Northgate Station" {
		t.Fatalf("expected named place, got %q", place)
	}
	possession := frameOfType(frames, types.FramePossession)
	if possession == nil || possession.Value != "holds" || !strings.Contains(possession.Detail, "brass key") {
		t.Fatalf("possession: %#v", possession)
	}
	injury := frameOfType(frames, types.FrameInjury)
	if injury == nil || subjectOf(injury) != "Mira Voss" {
		t.Fatalf("injury: %#v", injury)
	}
}

func TestKnowledgeGoalObligationDecision(t *testing.T) {
	book := frameBook([]types.EvidenceRecord{
		evidence("e1", 0, "Avery learned that the manifest had been altered.", "fact", "discovery", "learned"),
		evidence("e2", 0, "Mira wanted to reach the harbor before dawn.", "fact", "state", "wanted"),
		evidence("e3", 1, "Avery promised to deliver the report by Friday.", "fact", "state", "promised"),
		evidence("e4", 1, "Mira decided to burn the letters.", "event", "state", "decided"),
	})
	frames := extract(t, book)
	knowledge := frameOfType(frames, types.FrameKnowledge)
	if knowledge == nil || knowledge.Value != "knows" || !strings.Contains(knowledge.Detail, "the manifest had been altered") {
		t.Fatalf("knowledge: %#v", knowledge)
	}
	goal := frameOfType(frames, types.FrameGoal)
	if goal == nil || !strings.Contains(goal.Detail, "reach the harbor") {
		t.Fatalf("goal: %#v", goal)
	}
	obligation := frameOfType(frames, types.FrameObligation)
	if obligation == nil || obligation.Value != "open" || !strings.Contains(obligation.Detail, "deliver the report") {
		t.Fatalf("obligation: %#v", obligation)
	}
	decision := frameOfType(frames, types.FrameDecision)
	if decision == nil || !strings.Contains(decision.Detail, "burn the letters") {
		t.Fatalf("decision: %#v", decision)
	}
}

func TestAttributedClaimNeverBecomesNarratorTruth(t *testing.T) {
	book := frameBook([]types.EvidenceRecord{
		evidence("e1", 0, `"The ledger is a forgery," Mira said.`, "event", "interaction", "said"),
		evidence("e2", 0, `"Nobody will ever find it."`, "event", "interaction", ""),
	})
	frames := extract(t, book)
	claim := frameOfType(frames, types.FrameClaim)
	if claim == nil {
		t.Fatal("expected a claim frame for the attributed quote")
	}
	if claim.Epistemic != "attributed_claim" || claim.Attribution.Kind != "character" || claim.Attribution.EntityName != "Mira Voss" {
		t.Fatalf("claim attribution: %#v", claim)
	}
	if claim.Detail != "The ledger is a forgery," {
		t.Fatalf("claim payload must be the verbatim quote, got %q", claim.Detail)
	}
	// The unattributed quote must abstain, not guess a speaker.
	for _, frame := range frames {
		if frame.EvidenceIDs[0] == "e2" {
			t.Fatalf("unattributed quote produced a frame: %#v", frame)
		}
	}
}

func TestNegationAndSpeculationAreCarried(t *testing.T) {
	book := frameBook([]types.EvidenceRecord{
		evidence("e1", 0, "Avery never learned that the safe had a second combination.", "fact", "state", "learned"),
	})
	frames := extract(t, book)
	knowledge := frameOfType(frames, types.FrameKnowledge)
	if knowledge == nil || knowledge.Polarity != "negated" {
		t.Fatalf("expected negated knowledge frame, got %#v", knowledge)
	}
}

func TestKnowledgeStatesProjectWithCanonicalKnower(t *testing.T) {
	record := evidence("e1", 0, "Somewhere in the files, Mira suspected the audit was staged.", "fact", "knowledge_change", "suspected")
	record.KnowledgeStates = []types.EvidenceKnowledgeState{{State: "suspects", CharacterNames: []string{"Mira Voss"}, Cue: "suspected the audit was staged", Confidence: .8}}
	book := frameBook([]types.EvidenceRecord{record})
	frames := extract(t, book)
	belief := frameOfType(frames, types.FrameBelief)
	if belief == nil || subjectOf(belief) != "Mira Voss" || belief.Epistemic != "belief" {
		t.Fatalf("belief from knowledge state: %#v", belief)
	}
	if !strings.Contains(record.Text, belief.Detail) {
		t.Fatalf("knowledge-state detail is not verbatim: %q", belief.Detail)
	}
}

func TestFramesPerSentenceAreCapped(t *testing.T) {
	book := frameBook([]types.EvidenceRecord{
		evidence("e1", 0, "Avery picked up the brass key, learned that the manifest had been altered, wanted to reach the harbor, and decided to burn the letters.", "event", "interaction", "picked"),
	})
	frames := extract(t, book)
	if len(frames) > 3 {
		t.Fatalf("frames per sentence must be capped, got %d", len(frames))
	}
}

func TestReviewStatusEntitiesAnchorButRejectedDoNot(t *testing.T) {
	book := frameBook([]types.EvidenceRecord{
		evidence("e1", 0, "Dane entered the customs office at noon.", "event", "transition", "entered"),
	})
	frames := extract(t, book)
	location := frameOfType(frames, types.FrameLocation)
	if location == nil || subjectOf(location) != "Dane" {
		t.Fatalf("review-status entity should anchor: %#v", frames)
	}
}
