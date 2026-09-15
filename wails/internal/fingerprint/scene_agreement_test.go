package fingerprint

import (
	"strconv"
	"strings"
	"testing"

	"draftline/internal/indexing"
	"draftline/internal/narrative"
	"draftline/internal/types"
)

// All prose in this file is invented for these tests.

// One chapter with every scene-break form the writer might type. The
// shared scene rule (indexing.SceneAt), the isolated narrative engine's
// block scenes, and the manuscript memory's scene locator must place every
// paragraph in the same scene.
func TestSceneAtAgreesWithNarrativeBodyDocumentAndFingerprintLocator(t *testing.T) {
	paragraphs := []string{"Rhea climbed the tower.", "Tomas lit the lantern.", "Eloise rowed out.", "The skiff turned back.", "The harbour was empty.", "A bell rang twice.", "Rhea slept."}
	breaks := []string{"<hr>", "<p>***</p>", "<p>* * *</p>", "<p>⁂</p>", "<p>###</p>", "<p>---</p>"}
	var html strings.Builder
	for i, paragraph := range paragraphs {
		if i > 0 {
			html.WriteString(breaks[i-1])
		}
		html.WriteString("<p>" + paragraph + "</p>")
	}
	book := types.BookData{Body: []types.ChapterItem{{ID: "ch-one", Title: "One", Type: "chapter", Content: html.String()}}}

	doc, err := narrative.BodyDocument(book, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	blockScene := map[string]string{}
	for _, block := range doc.Blocks {
		blockScene[block.Text] = block.Scene
	}
	text := indexing.StripHTMLForAnalysis(book.Body[0].Content)
	locator := newSceneLocator(&book)
	if got := indexing.SceneCount(text); got != len(paragraphs) {
		t.Fatalf("SceneCount = %d, want %d", got, len(paragraphs))
	}
	for i, paragraph := range paragraphs {
		want := i + 1
		offset := strings.Index(text, paragraph)
		if offset < 0 {
			t.Fatalf("paragraph %q not in stripped text", paragraph)
		}
		if got := indexing.SceneAt(text, offset); got != want {
			t.Fatalf("SceneAt(%q) = %d, want %d", paragraph, got, want)
		}
		if got := locator.sceneOf(0, offset) + 1; got != want {
			t.Fatalf("sceneLocator(%q) = %d, want %d", paragraph, got, want)
		}
		if got, ok := blockScene[paragraph]; !ok || got != "ch-one/scene/"+strconv.Itoa(want-1) {
			t.Fatalf("narrative block scene for %q = %q, want scene %d", paragraph, got, want-1)
		}
	}
}

// Participants carry the codex entity ID of accepted people, by exact
// case-folded name or alias; people still under review, rejected ones, and
// unknown names carry none.
func TestParticipantsCarryEntityID(t *testing.T) {
	book := frameBook([]types.EvidenceRecord{
		evidence("e1", 0, "Avery Cole picked up the brass key from the desk.", "event", "interaction", "picked"),
		evidence("e2", 0, "Mira handed the ledger to Dane.", "event", "interaction", "handed"),
	})
	book.StoryBible.Characters = []types.Character{
		{ID: "entity-1", Name: "Avery Cole", Aliases: []string{"Avery", "Cole"}, IsAutoDetected: true, DetectionStatus: "accepted"},
		{ID: "entity-2", Name: "Mira Brannick", Aliases: []string{"Mira", "Brannick"}}, // manual character: accepted
		{ID: "entity-3", Name: "Dane", IsAutoDetected: true, DetectionStatus: "review"},
		{ID: "entity-4", Name: "Ghost", DetectionStatus: "rejected"},
	}
	frames := extract(t, book)
	seen := map[string]string{}
	for _, frame := range frames {
		for _, p := range frame.Participants {
			seen[p.EntityName] = p.EntityID
		}
	}
	if seen["Avery Cole"] != "entity-1" {
		t.Fatalf("accepted subject carries no ID: %v", seen)
	}
	if seen["Mira Brannick"] != "entity-2" {
		t.Fatalf("manual character carries no ID: %v", seen)
	}
	if _, ok := seen["Dane"]; ok && seen["Dane"] != "" {
		t.Fatalf("a review-status person carried an ID: %v", seen)
	}
	model := Build(&book, nil)
	for _, development := range model.Developments {
		for _, p := range development.Entities {
			if p.Role == "subject" && p.EntityName == "Avery Cole" && p.EntityID != "entity-1" {
				t.Fatalf("development participant lost the ID: %+v", p)
			}
		}
	}
	ids := AcceptedPeopleIDs(&book)
	if ids["avery"] != "entity-1" || ids["mira brannick"] != "entity-2" || ids["dane"] != "" || ids["ghost"] != "" {
		t.Fatalf("AcceptedPeopleIDs %v", ids)
	}
}
