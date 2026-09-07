package fingerprint

import (
	"testing"

	"draftline/internal/types"
)

func TestCommitmentAndFulfillmentBecomeRelatedFingerprints(t *testing.T) {
	open := record("open", 0, 0, "Gary promised he would find the IBM building maps for Hanlon.")
	open.CharacterIDs = []string{"gary", "hanlon"}
	open.CharacterNames = []string{"Gary", "Hanlon"}
	open.Action = "promised"
	close := record("close", 4, 0, "Gary handed Hanlon the IBM building maps he found.")
	close.CharacterIDs = []string{"gary", "hanlon"}
	close.CharacterNames = []string{"Gary", "Hanlon"}
	close.Action = "handed"
	book := testBook([]types.EvidenceRecord{open, close})
	model := Build(&book, nil)
	if len(model.Threads) != 0 {
		t.Fatalf("thread inference is intentionally disabled in the semantic reset: %#v", model.Threads)
	}
	if !hasCorpusRelation(model.FingerprintRelations, "fulfills") {
		t.Fatalf("expected source-backed fulfillment relation, got %#v", model.FingerprintRelations)
	}
}

func TestCheckpointRequiresEveryCriterionAndNeverSilentlyFulfills(t *testing.T) {
	discovery := record("tunnel", 2, 0, "Hanlon discovered the underground tunnel.")
	discovery.CharacterIDs = []string{"hanlon"}
	discovery.CharacterNames = []string{"Hanlon"}
	discovery.Action = "discovered"
	book := testBook([]types.EvidenceRecord{discovery})
	book.Analysis.Fingerprint = &types.StoryFingerprint{AuthorModel: types.StoryAuthorModel{Checkpoints: []types.StoryCheckpoint{{
		ID: "checkpoint-1", Title: "Find the tunnel with Ruiz", Status: "planned", Source: "author",
		Requirements: []types.CheckpointRequirement{{Text: "underground tunnel"}, {EntityID: "ruiz"}},
	}}}}
	model := Build(&book, nil)
	checkpoint := model.AuthorModel.Checkpoints[0]
	if checkpoint.Status != "planned" {
		t.Fatalf("checkpoint evaluation must wait for validated higher-level analysis, got %q", checkpoint.Status)
	}
	if checkpoint.Requirements[0].Satisfied || checkpoint.Requirements[1].Satisfied {
		t.Fatalf("semantic reset must not invent checkpoint fulfillment: %#v", checkpoint.Requirements)
	}
}

func TestInferredProfilesCanOverlap(t *testing.T) {
	book := testBook([]types.EvidenceRecord{
		record("one", 0, 0, "The detective began the interrogation with the suspect and examined the evidence."),
		record("two", 1, 0, "The simulation computer hid a quantum mystery inside the underground tunnel."),
		record("three", 2, 0, "They investigate the conspiracy and discovered the secret."),
	})
	model := Build(&book, nil)
	if len(model.Profiles) < 2 {
		t.Fatalf("expected overlapping inferred profiles, got %#v", model.Profiles)
	}
}
