package indexing

import (
	"strings"
	"testing"

	"draftline/internal/types"
)

// Real manuscripts quote dialogue with CURLY double quotes (U+201C/U+201D),
// not straight ASCII. findDialogueRanges must recognize them, and the
// apostrophe inside a contraction ("You're") must NOT be treated as a dialogue
// delimiter. This asserts a directed dialogue interaction is produced.
func TestDetectInteractions_CurlyQuotedDialogue(t *testing.T) {
	// U+201C ... U+201D around the spoken line; a straight apostrophe in
	// "You're" that must not fabricate its own quoted span.
	text := "Marcus stepped forward. “You're late,” he said to Clara."

	marcusOff := strings.Index(text, "Marcus")
	claraOff := strings.Index(text, "Clara")

	scenes := []types.SceneRecord{
		{ID: "sc-0", ChapterIndex: 0, StartOffset: 0, EndOffset: len(text), SceneType: "scene"},
	}
	mentions := []types.MentionRecord{
		{ID: "m0", Chapter: 0, CharOffset: marcusOff},
		{ID: "m1", Chapter: 0, CharOffset: claraOff},
	}
	entityMap := map[string]string{"m0": "e-marcus", "m1": "e-clara"}

	interactions := DetectInteractions(0, text, scenes, mentions, entityMap, DefaultCoOccurrenceConfig())

	var dialogue *types.InteractionRecord
	for i := range interactions {
		if interactions[i].InteractionType == "dialogue" {
			dialogue = &interactions[i]
			break
		}
	}
	if dialogue == nil {
		t.Fatalf("no dialogue interaction detected in curly-quoted text; got %d interactions: %+v", len(interactions), interactions)
	}
	if dialogue.DirectedFrom != "e-marcus" || dialogue.DirectedTo != "e-clara" {
		t.Errorf("expected directed dialogue e-marcus -> e-clara, got %q -> %q", dialogue.DirectedFrom, dialogue.DirectedTo)
	}
}

// A straight apostrophe used as a possessive/contraction must not be treated
// as an opening single quote that fabricates a dialogue span (the old table
// listed straight single ' as a delimiter).
func TestFindDialogueRanges_ApostropheNotDialogue(t *testing.T) {
	text := "Kira's hand brushed Marcus' coat as they walked on."
	if ranges := findDialogueRanges(text); len(ranges) != 0 {
		t.Errorf("apostrophes fabricated %d dialogue ranges: %+v", len(ranges), ranges)
	}
}

// End-to-end pipeline test: index a book (with front matter, so chapter
// indexes must be global) and analyze relationships. Scenes, interactions,
// and relationships must all reference the resolved entity IDs.
func TestAnalyzeBook_EndToEnd(t *testing.T) {
	book := types.BookData{
		FrontMatter: []types.ChapterItem{
			{Title: "Foreword", Type: "foreword", Content: `<p>A note from the publisher.</p>`},
		},
		Body: []types.ChapterItem{
			{
				Title: "Chapter 1",
				Type:  "chapter",
				Content: `<p>Marcus Webb entered the bar. Detective Clara was already there.</p>` +
					`<p>"You're late," Clara said. Marcus shrugged. "Traffic," Marcus said to Clara.</p>`,
			},
			{
				Title:   "Chapter 2",
				Type:    "chapter",
				Content: `<p>Webb woke early. Clara had left a note for Marcus.</p>`,
			},
		},
	}

	if result := IndexBook(&book); !result.Success {
		t.Fatalf("IndexBook failed: %s", result.Error)
	}

	// "Marcus Webb", "Marcus", and "Webb" must be one entity; "Clara" another.
	entities := book.Analysis.EntityResolution.Entities
	if len(entities) != 2 {
		for _, e := range entities {
			t.Logf("entity: %q (mentions %d)", e.Canonical, len(e.MentionIDs))
		}
		t.Fatalf("expected 2 entities, got %d", len(entities))
	}
	entityIDs := map[string]bool{}
	for _, e := range entities {
		entityIDs[e.ID] = true
	}

	analyzer := NewRelationshipAnalyzer()
	relData, err := analyzer.AnalyzeBook(book)
	if err != nil {
		t.Fatalf("AnalyzeBook failed: %v", err)
	}

	// Scenes must pick up characters (offsets share the stripped-text
	// coordinate space with mentions).
	scenesWithChars := 0
	for _, scene := range relData.Scenes {
		for _, id := range scene.CharacterIDs {
			if !entityIDs[id] {
				t.Errorf("scene %s references unknown character ID %q", scene.ID, id)
			}
		}
		if len(scene.CharacterIDs) > 0 {
			scenesWithChars++
		}
	}
	if scenesWithChars == 0 {
		t.Error("no scene has any characters — offsets are misaligned")
	}

	// Both characters appear together, so a relationship must exist and
	// reference resolved entity IDs only.
	if len(relData.Relationships) != 1 {
		t.Fatalf("expected 1 relationship, got %d", len(relData.Relationships))
	}
	rel := relData.Relationships[0]
	if !entityIDs[rel.Character1ID] || !entityIDs[rel.Character2ID] {
		t.Errorf("relationship references unknown IDs: %q, %q", rel.Character1ID, rel.Character2ID)
	}

	// The graph node source (story bible) must contain the same IDs the
	// relationship edges use.
	charIDs := map[string]bool{}
	for _, c := range book.StoryBible.Characters {
		charIDs[c.ID] = true
	}
	if !charIDs[rel.Character1ID] || !charIDs[rel.Character2ID] {
		t.Error("relationship IDs missing from story bible characters — graph would show phantom edges")
	}

	// Determinism: a second run over the same book yields identical IDs.
	book2 := types.BookData{FrontMatter: book.FrontMatter, Body: book.Body}
	if result := IndexBook(&book2); !result.Success {
		t.Fatalf("second IndexBook failed: %s", result.Error)
	}
	relData2, err := analyzer.AnalyzeBook(book2)
	if err != nil {
		t.Fatalf("second AnalyzeBook failed: %v", err)
	}
	if len(relData2.Relationships) != len(relData.Relationships) {
		t.Fatalf("non-deterministic relationship count: %d vs %d", len(relData.Relationships), len(relData2.Relationships))
	}
	if relData2.Relationships[0].ID != rel.ID ||
		relData2.Relationships[0].Character1ID != rel.Character1ID ||
		relData2.Relationships[0].Character2ID != rel.Character2ID {
		t.Error("non-deterministic relationship identity across runs")
	}
}
