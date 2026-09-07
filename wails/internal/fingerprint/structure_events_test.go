package fingerprint

import (
	"testing"

	"draftline/internal/types"
)

// These fixtures describe occurrence-level invariants, not desired aggregate
// counts. Later structure stages reuse the same narrative forms for their own
// semantic gates.
func TestSignificantEventFixtureCombinesObservationsIntoAwakening(t *testing.T) {
	records := []types.EvidenceRecord{
		structureRecord("eyes", "fact", "state", "opened", 0, "Daniel opened his eyes."),
		structureRecord("tile", "fact", "state", "recognized", 0, "A familiar ceiling tile came into focus."),
		structureRecord("logo", "fact", "state", "saw", 0, "The Northwestern Memorial Hospital logo hung above him."),
		structureRecord("hospital", "event", "discovery", "realized", 0, "Daniel realized he was at Northwestern Memorial Hospital."),
	}
	for i := range records {
		records[i].CharacterIDs, records[i].CharacterNames = []string{"daniel"}, []string{"Daniel"}
		records[i].NamedEntities = []types.EvidenceTerm{{Text: "Northwestern Memorial Hospital", Label: "FAC"}}
	}
	fp := structureFingerprint(records)
	events := aggregateSignificantEvents(&fp, records)
	aggregate := aggregateContainingEvidence(t, events, "hospital")
	for _, evidenceID := range []string{"eyes", "tile", "logo", "hospital"} {
		if !contains(aggregate.EvidenceIDs, evidenceID) {
			t.Fatalf("awakening occurrence omitted supporting evidence %q: %#v", evidenceID, aggregate.EvidenceIDs)
		}
	}
	if len(aggregate.EvidenceRefs) != len(aggregate.EvidenceIDs) {
		t.Fatalf("every evidence id must retain an inspectable source reference: %#v", aggregate)
	}
	if aggregate.CreationDecision.Rule != "narrative-nucleus" || len(aggregate.MembershipDecisions) < 4 {
		t.Fatalf("aggregate lacks inspectable creation/membership reasons: %#v", aggregate)
	}
}

func TestSignificantEventFixtureKeepsIndependentActionsSeparate(t *testing.T) {
	records := []types.EvidenceRecord{
		structureRecord("lock", "event", "interaction", "locked", 0, "Mara locked the bridge door."),
		structureRecord("fire", "event", "interaction", "fired", 0, "Mara fired the railgun."),
	}
	for i := range records {
		records[i].CharacterIDs, records[i].CharacterNames = []string{"mara"}, []string{"Mara"}
		records[i].NamedEntities = []types.EvidenceTerm{{Text: "bridge", Label: "FAC"}}
	}
	fp := structureFingerprint(records)
	events := aggregateSignificantEvents(&fp, records)
	left := aggregateContainingEvidence(t, events, "lock")
	right := aggregateContainingEvidence(t, events, "fire")
	if left.ID == right.ID || contains(left.EvidenceIDs, "fire") {
		t.Fatalf("different explicit actions were collapsed: left=%#v right=%#v", left, right)
	}
	if len(left.BoundaryDecisions) == 0 || left.BoundaryDecisions[0].Rule != "insufficient-occurrence-evidence" {
		t.Fatalf("separation is not explained: %#v", left.BoundaryDecisions)
	}
}

func TestSignificantEventFixtureLeavesIncidentalFactsAtomic(t *testing.T) {
	records := []types.EvidenceRecord{
		structureRecord("curtains", "fact", "state", "were", 0, "The curtains were blue."),
		structureRecord("ago", "fact", "time_reference", "", 1, "The building had opened two years ago."),
	}
	fp := structureFingerprint(records)
	events := aggregateSignificantEvents(&fp, records)
	if aggregateWithEvidence(events, "curtains") != nil || aggregateWithEvidence(events, "ago") != nil {
		t.Fatalf("incidental state or temporal reference was promoted: %#v", events)
	}
	if len(fp.Events) != 2 {
		t.Fatal("atomic fingerprint events must remain untouched")
	}
}

func TestSignificantEventFixturePromotesPersistentState(t *testing.T) {
	record := structureRecord("injury", "fact", "state", "was shot", 0, "Hanlon had been shot three months earlier.")
	record.CharacterIDs, record.CharacterNames = []string{"hanlon"}, []string{"Hanlon"}
	fp := structureFingerprint([]types.EvidenceRecord{record})
	fp.States = []types.StoryStateInterval{{ID: "state-injury", Kind: "condition", Value: "shot", StartEventID: fp.Events[0].ID, EvidenceIDs: []string{"injury"}, Persistent: true, Confidence: .9}}
	events := aggregateSignificantEvents(&fp, []types.EvidenceRecord{record})
	aggregate := aggregateContainingEvidence(t, events, "injury")
	if !contains(aggregate.StateIDs, "state-injury") || aggregate.CreationDecision.Rule != "narrative-nucleus" {
		t.Fatalf("persistent state was not promoted with a reason: %#v", aggregate)
	}
}

func TestSignificantEventFixtureRecognizesQuietDecisionWithoutActionSpectacle(t *testing.T) {
	record := structureRecord("stay", "fact", "state", "decided", 0, "After the hour changed, Mira decided not to call her sister.")
	record.CharacterIDs, record.CharacterNames = []string{"mira"}, []string{"Mira"}
	fp := structureFingerprint([]types.EvidenceRecord{record})
	event := aggregateContainingEvidence(t, aggregateSignificantEvents(&fp, []types.EvidenceRecord{record}), "stay")
	if event.SalienceReasons.Decision == 0 || event.CreationDecision.Rule != "narrative-nucleus" {
		t.Fatalf("quiet character decision was not recognized as a narrative occurrence: %#v", event)
	}
}

func TestSignificantEventFixtureExplainsLowConfidenceAndMissingContext(t *testing.T) {
	record := structureRecord("uncertain", "event", "discovery", "noticed", 0, "Someone may have noticed a light.")
	record.Confidence = .45
	fp := structureFingerprint([]types.EvidenceRecord{record})
	fp.Events[0].Confidence = .45
	fp.Events[0].ContextID = ""
	event := aggregateContainingEvidence(t, aggregateSignificantEvents(&fp, []types.EvidenceRecord{record}), "uncertain")
	if !hasStructureSignal(event.InterpretationSignals, "low-confidence-extraction") || !hasStructureSignal(event.InterpretationSignals, "missing-context") {
		t.Fatalf("distinct uncertainty causes were not retained: %#v", event.InterpretationSignals)
	}
}

func TestSignificantEventFixtureHonorsAuthorInferenceDecisions(t *testing.T) {
	records := []types.EvidenceRecord{
		structureRecord("noise", "event", "discovery", "noticed", 0, "Hanlon noticed a coffee stain."),
		structureRecord("quiet", "fact", "state", "waited", 1, "Mira waited beside the silent phone."),
	}
	fp := structureFingerprint(records)
	fp.AuthorModel.StructureDecisions = []types.StructureAuthorDecision{
		{ID: "ignore-noise", TargetType: "significant_event", Action: "irrelevant", EvidenceIDs: []string{"noise"}, Status: "active"},
		{ID: "confirm-quiet", TargetType: "significant_event", Action: "confirm", EvidenceIDs: []string{"quiet"}, Status: "active"},
	}
	events := aggregateSignificantEvents(&fp, records)
	if aggregateWithEvidence(events, "noise") != nil {
		t.Fatal("author-marked noise was promoted")
	}
	quiet := aggregateContainingEvidence(t, events, "quiet")
	if quiet.InterpretationStatus != "confirmed" || !contains(quiet.AuthorDecisionIDs, "confirm-quiet") {
		t.Fatalf("confirmed quiet-story occurrence lost author intent: %#v", quiet)
	}
}

func TestStructureDecisionDependencyConflictIsNotSilentlyApplied(t *testing.T) {
	record := structureRecord("new-evidence", "event", "discovery", "found", 0, "Hanlon found a second entrance.")
	oldHash := structureDependencyHash([]string{normalizeFingerprintText("Hanlon found the original entrance.")})
	book := testBook([]types.EvidenceRecord{record})
	book.Analysis.Fingerprint = &types.StoryFingerprint{AuthorModel: types.StoryAuthorModel{StructureDecisions: []types.StructureAuthorDecision{{
		ID: "pinned-placement", TargetType: "significant_event", Action: "confirm", EvidenceIDs: []string{"old-evidence"}, Status: "active",
		Dependencies: []types.StructureDecisionDependency{{EvidenceID: "old-evidence", ChapterID: record.ChapterID, ParagraphIndex: record.ParagraphIndex, ContentHash: oldHash}},
	}}}}
	model := Build(&book, nil)
	decision := model.AuthorModel.StructureDecisions[0]
	if decision.Status != "conflict" || !contains(decision.ChangedDependencyIDs, "old-evidence") {
		t.Fatalf("rewritten dependency should create an inspectable conflict: %#v", decision)
	}
	if model.Structure != nil {
		t.Fatal("legacy structure generation must remain disabled during narrative fingerprint validation")
	}
}

func structureRecord(id, kind, evidenceType, action string, paragraph int, text string) types.EvidenceRecord {
	return types.EvidenceRecord{ID: id, Kind: kind, EvidenceType: evidenceType, ChapterID: "chapter-1", ChapterIndex: 0, Section: "body", ParagraphIndex: paragraph, SentenceIndex: paragraph, StartOffset: paragraph * 100, EndOffset: paragraph*100 + len(text), Text: text, Action: action, Confidence: .9, Status: "detected", Source: "auto"}
}

func structureFingerprint(records []types.EvidenceRecord) types.StoryFingerprint {
	events := make([]types.FingerprintEvent, 0, len(records))
	for i, record := range records {
		locations, objects := eventTerms(record)
		events = append(events, types.FingerprintEvent{ID: "event-" + record.ID, Summary: record.Text, EvidenceIDs: []string{record.ID}, CharacterIDs: clone(record.CharacterIDs), CharacterNames: clone(record.CharacterNames), Locations: locations, Objects: objects, Kinds: []string{record.EvidenceType}, ContextID: "context-primary", StoryTime: types.StoryTime{ContextID: "context-primary", Precision: "unknown", Confidence: .5}, ChapterID: record.ChapterID, ChapterIndex: record.ChapterIndex, ChapterTitle: "Chapter 1", ParagraphIndex: record.ParagraphIndex, StartOffset: record.StartOffset, NarrativeOrder: i, Importance: .6, Confidence: record.Confidence})
	}
	return types.StoryFingerprint{Events: events}
}

func aggregateContainingEvidence(t *testing.T, events []types.SignificantStoryEvent, evidenceID string) types.SignificantStoryEvent {
	t.Helper()
	event := aggregateWithEvidence(events, evidenceID)
	if event == nil {
		t.Fatalf("no significant occurrence contains evidence %q: %#v", evidenceID, events)
	}
	return *event
}

func aggregateWithEvidence(events []types.SignificantStoryEvent, evidenceID string) *types.SignificantStoryEvent {
	for i := range events {
		if contains(events[i].EvidenceIDs, evidenceID) {
			return &events[i]
		}
	}
	return nil
}

func hasStructureSignal(signals []types.StructureSignal, code string) bool {
	for _, signal := range signals {
		if signal.Code == code {
			return true
		}
	}
	return false
}
