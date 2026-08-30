package indexing

import (
	"testing"

	"draftline/internal/entityresolution"
	"draftline/internal/types"
)

func TestClassifyResolvedEntitiesTriagesCandidates(t *testing.T) {
	entities := []entityresolution.Entity{
		{ID: "person", Canonical: "Mara Ionescu", MentionIDs: []string{"m1", "m2"}, Titles: []string{"Captain"}},
		{ID: "object", Canonical: "Honda Accord", MentionIDs: []string{"m3"}},
		{ID: "unknown", Canonical: "Veyr", MentionIDs: []string{"m4", "m5"}},
		{ID: "org", Canonical: "FleetCom", MentionIDs: []string{"m6", "m7", "m8"}},
	}
	mentions := []entityresolution.Mention{
		{ID: "m1", PersonEvidence: true, StrongPersonEvidence: true},
		{ID: "m2", PersonEvidence: true},
		{ID: "m3"},
		{ID: "m4"}, {ID: "m5"},
		{ID: "m6"}, {ID: "m7"}, {ID: "m8"},
	}

	ClassifyResolvedEntities(entities, mentions)
	if entities[0].Kind != "person" || entities[0].DetectionStatus != "accepted" {
		t.Fatalf("strong person was not accepted: %+v", entities[0])
	}
	if entities[1].Kind != "object" || entities[1].DetectionStatus != "rejected" {
		t.Fatalf("obvious object was not rejected: %+v", entities[1])
	}
	if entities[2].DetectionStatus != "review" {
		t.Fatalf("invented ambiguous name should remain reviewable: %+v", entities[2])
	}
	if entities[3].Kind != "organization" || entities[3].DetectionStatus != "review" {
		t.Fatalf("story organization should remain reviewable: %+v", entities[3])
	}
}

func TestConvertEntitiesToCharactersOmitsRejectedCandidates(t *testing.T) {
	entities := []entityresolution.Entity{
		{ID: "keep", Canonical: "Mara", MentionIDs: []string{"m1"}, DetectionStatus: "review"},
		{ID: "drop", Canonical: "Boom", MentionIDs: []string{"m2"}, DetectionStatus: "rejected"},
	}
	mentions := []entityresolution.Mention{{ID: "m1"}, {ID: "m2"}}
	characters := ConvertEntitiesToCharacters(entities, mentions)
	if len(characters) != 1 || characters[0].Name != "Mara" {
		t.Fatalf("rejected candidate leaked into Codex: %+v", characters)
	}
}

func TestRejectedCandidatesDoNotEnterRelationshipMap(t *testing.T) {
	entities := []types.EntityRecord{
		{ID: "keep", MentionIDs: []string{"m1"}, DetectionStatus: "accepted"},
		{ID: "drop", MentionIDs: []string{"m2"}, DetectionStatus: "rejected"},
	}
	lookup := BuildMentionToEntityMap(entities)
	if lookup["m1"] != "keep" {
		t.Fatal("accepted character missing from relationship lookup")
	}
	if _, exists := lookup["m2"]; exists {
		t.Fatal("rejected candidate leaked into relationship lookup")
	}
}

func TestApplyEntityDecisionsSurvivesChangedEntityIDs(t *testing.T) {
	entities := []types.EntityRecord{
		{ID: "new-id", Canonical: "Mara Ionescu", Aliases: []string{"Captain Ionescu"}, DetectionStatus: "review"},
		{ID: "noise-id", Canonical: "Wacker Drive", DetectionStatus: "review"},
	}
	decisions := []types.EntityDecision{
		{Names: []string{"Captain Ionescu"}, Status: "accepted"},
		{Names: []string{"Wacker Drive"}, Status: "rejected"},
	}

	ApplyEntityDecisions(entities, decisions)
	if entities[0].DetectionStatus != "accepted" {
		t.Fatalf("accepted author decision was not restored: %+v", entities[0])
	}
	if entities[1].DetectionStatus != "rejected" {
		t.Fatalf("rejected author decision was not restored: %+v", entities[1])
	}
}

func TestApplyEntityDecisionsSkipsAmbiguousNames(t *testing.T) {
	entities := []types.EntityRecord{
		{ID: "one", Canonical: "Alex North", Aliases: []string{"Alex"}, DetectionStatus: "review"},
		{ID: "two", Canonical: "Alex Reed", Aliases: []string{"Alex"}, DetectionStatus: "review"},
	}
	ApplyEntityDecisions(entities, []types.EntityDecision{{Names: []string{"Alex"}, Status: "rejected"}})
	for _, entity := range entities {
		if entity.DetectionStatus != "review" {
			t.Fatalf("ambiguous rule changed entity %s", entity.ID)
		}
	}
}
