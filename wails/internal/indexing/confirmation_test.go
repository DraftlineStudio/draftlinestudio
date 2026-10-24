package indexing

import (
	"draftline/internal/types"
	"testing"
)

func TestRebuildReconsidersAutomaticAcceptance(t *testing.T) {
	for _, confirmed := range []bool{false, true} {
		t.Run(map[bool]string{false: "automatic", true: "writer"}[confirmed], func(t *testing.T) {
			book := types.BookData{
				StoryBible: types.StoryBible{Characters: []types.Character{{ID: "old", Name: "Nelwin", IsAutoDetected: true, DetectionStatus: "accepted", Notes: "Watchful"}}},
				Analysis:   types.AnalysisData{EntityResolution: &types.EntityData{Entities: []types.EntityRecord{{ID: "new", Canonical: "Nelwin", DetectionStatus: "review"}}}},
			}
			if confirmed {
				book.Analysis.EntityResolution.Decisions = []types.EntityDecision{{Names: []string{"Nelwin"}, Status: "accepted"}}
			}
			ApplyEntityDecisions(book.Analysis.EntityResolution.Entities, book.Analysis.EntityResolution.Decisions)
			rebuildCharactersFromEntities(&book)
			got := book.StoryBible.Characters[0]
			want := "review"
			if confirmed {
				want = "accepted"
			}
			if got.DetectionStatus != want || got.Notes != "Watchful" {
				t.Fatalf("unexpected rebuilt character: %+v", got)
			}
		})
	}
}
