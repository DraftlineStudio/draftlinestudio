package indexing

import (
	"testing"

	"draftline/internal/entityresolution"
)

func TestCharacterAccuracyCorpus(t *testing.T) {
	cases := []struct {
		name, text string
		people     []string
	}{
		{"dialogue", "Nelwin nodded. Nelwin replied quietly. Nelwin asked for a cup.", []string{"Nelwin"}},
		{"gesture", "Tavren shrugged. Tavren smiled. Tavren frowned at the empty shelf.", []string{"Tavren"}},
		{"two people", "Nelwin Brindle nodded. Tavren Mosswick replied. Nelwin Brindle asked again. Tavren Mosswick sighed.", []string{"Nelwin Brindle", "Tavren Mosswick"}},
		{"possessives", "The label bore Vellora's mark. The box bore Vellora's mark. The lid bore Vellora's mark.", nil},
		{"place", "They reached Copper Hollow. The map showed Copper Hollow. They left Copper Hollow behind.", nil},
		{"organisation", "They called Amber Network. The invoice named Amber Network. Payment went to Amber Network.", nil},
		{"object", "They checked Relay Nodes. The diagram showed Relay Nodes. The repair required Relay Nodes.", nil},
		{"weather", "Thunder rolled. Thunder shook the windows. Thunder faded.", nil},
		{"quote openings", "“Tomorrow,” she said. “Tomorrow,” he repeated. “Tomorrow,” they agreed.", nil},
	}
	truePositive, falsePositive, expected, recovered := 0, 0, 0, 0
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mentions := ExtractMentions(tc.text, 0)
			result := entityresolution.NewResolver().ResolveEntities(mentions)
			ClassifyResolvedEntities(result.Entities, mentions)
			expected += len(tc.people)
			for _, person := range tc.people {
				found, accepted := false, false
				for _, entity := range result.Entities {
					if entity.Canonical == person {
						found = true
						accepted = entity.DetectionStatus == "accepted"
					}
				}
				if found {
					recovered++
				}
				if !accepted {
					t.Errorf("expected accepted person %q: %+v", person, result.Entities)
				}
			}
			for _, entity := range result.Entities {
				if entity.DetectionStatus != "accepted" {
					continue
				}
				person := false
				for _, name := range tc.people {
					person = person || entity.Canonical == name
				}
				if person {
					truePositive++
				} else {
					falsePositive++
					t.Errorf("false accepted character: %+v", entity)
				}
			}
		})
	}
	t.Logf("invented corpus: accepted true=%d false=%d expected=%d recovered=%d", truePositive, falsePositive, expected, recovered)
}
