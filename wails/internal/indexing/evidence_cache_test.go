package indexing

import (
	"strings"
	"testing"

	"draftline/internal/types"
)

// A cache written by an older evidence engine is never reused by chapter
// hash: the fixed rules must reach books that were analyzed before the fix.
func TestAnalyzeEvidenceRebuildsCacheFromAnOlderEngine(t *testing.T) {
	book := evidenceTestBook("Rhea discovered the tunnel beneath the harbour wall.", map[string]string{"Rhea": "rhea"})
	first := AnalyzeEvidence(&book, nil)
	if first.ChapterHashes["chapter-one"] == "" {
		t.Fatal("missing chapter cache hash")
	}
	// The same text, cached by the previous engine with a record shaped the
	// way that engine produced it.
	stale := *first
	stale.Engine = "prose-v3-evidence-v3"
	stale.Records = append([]types.EvidenceRecord(nil), first.Records...)
	for index := range stale.Records {
		stale.Records[index].Rationale = "written by the previous engine"
	}
	book.Analysis.Evidence = &stale
	second := AnalyzeEvidence(&book, nil)
	for _, record := range second.Records {
		if record.Rationale == "written by the previous engine" {
			t.Fatalf("a record from the older engine was reused: %+v", record)
		}
	}
	if findEvidenceType(second.Records, "discovery") == nil {
		t.Fatal("the chapter was not re-indexed")
	}

	// The same cache under the current engine is reused (the deliberately
	// absent mention would otherwise remove the character link).
	book.Analysis.Evidence = first
	book.Analysis.EntityResolution.Mentions = nil
	third := AnalyzeEvidence(&book, nil)
	discovery := findEvidenceType(third.Records, "discovery")
	if discovery == nil || len(discovery.CharacterIDs) != 1 {
		t.Fatalf("a current-engine cache must still be reused: %+v", discovery)
	}
}

// A sentence that carries only a goal, decision, promise, or obligation cue
// is indexed, so the frame extractors can see it.
func TestEvidenceIndexesIntentSentences(t *testing.T) {
	for _, text := range []string{
		"Rhea decided to search the loft above the wheel before dark.",
		"Rhea wanted to bring the ledger to the village.",
		"Rhea promised to bring the rope to the bridge.",
	} {
		book := evidenceTestBook(text, map[string]string{"Rhea": "rhea"})
		record := findEvidenceType(AnalyzeEvidence(&book, nil).Records, "state")
		if record == nil || record.Text != text {
			t.Fatalf("%q: expected a state record, got %+v", text, record)
		}
		if !strings.Contains(record.Rationale, "goal, decision, promise, or obligation") {
			t.Fatalf("%q: rationale must name the intent cue: %q", text, record.Rationale)
		}
	}
}
