package indexing

import (
	"fmt"
	"testing"

	"draftline/internal/entityresolution"
)

func TestAcceptanceRequiresIndependentPersonContexts(t *testing.T) {
	cases := []struct {
		name                   string
		actors, persons        int
		sameSentence, conflict bool
		want                   string
	}{
		{name: "repeated weak label", persons: 1, want: "review"},
		{name: "repeated model labels", persons: 12, want: "review"},
		{name: "one speaking context", actors: 1, persons: 1, want: "review"},
		{name: "same sentence", actors: 4, persons: 4, sameSentence: true, want: "review"},
		{name: "independent actor contexts", actors: 2, want: "accepted"},
		{name: "model corroborates actor", actors: 1, persons: 2, want: "accepted"},
		{name: "conflicting labels", actors: 2, persons: 3, conflict: true, want: "review"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mentions := []entityresolution.Mention{}
			entity := entityresolution.Entity{Canonical: "Nelwin Brindle"}
			for i := 0; i < 12; i++ {
				sentence := fmt.Sprintf("s%d", i)
				if tc.sameSentence {
					sentence = "same"
				}
				id := fmt.Sprintf("m%d", i)
				mentions = append(mentions, entityresolution.Mention{ID: id, SentenceID: sentence,
					ActorEvidence: i < tc.actors, PersonEvidence: i < tc.persons, NonPersonEvidence: tc.conflict && i >= tc.persons && i >= tc.actors})
				entity.MentionIDs = append(entity.MentionIDs, id)
			}
			entities := []entityresolution.Entity{entity}
			ClassifyResolvedEntities(entities, mentions)
			if entities[0].DetectionStatus != tc.want {
				t.Fatalf("got %+v, want %s", entities[0], tc.want)
			}
		})
	}
}

func TestActorEvidenceFindsInventedSentenceInitialName(t *testing.T) {
	mentions := ExtractMentions("Nelwin nodded. Nelwin replied softly.", 0)
	if len(mentions) != 2 {
		t.Fatalf("expected two mentions, got %+v", mentions)
	}
	result := entityresolution.NewResolver().ResolveEntities(mentions)
	ClassifyResolvedEntities(result.Entities, mentions)
	if len(result.Entities) != 1 || result.Entities[0].DetectionStatus != "accepted" {
		t.Fatalf("actor not accepted: %+v", result.Entities)
	}
}

func TestPossessionAloneIsNotActorEvidence(t *testing.T) {
	for _, text := range []string{"Nelwin’s nodded", "Nelwin's replied"} {
		for _, mention := range ExtractMentions(text, 0) {
			if mention.ActorEvidence {
				t.Fatalf("possession became actor evidence: %+v", mention)
			}
		}
	}
}
