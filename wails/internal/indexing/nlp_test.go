package indexing

import (
	"strings"
	"testing"

	"draftline/internal/types"
)

func TestIndexBook_NLPRejectsChicagoAddressEntities(t *testing.T) {
	book := types.BookData{Body: []types.ChapterItem{{
		Title: "Chapter 1",
		Type:  "chapter",
		Content: `<p>The detective crossed Chicago at Hubbard Street. He turned onto Ontario at Wells Ave and followed the river north toward Lower Wacker Drive.</p>` +
			`<p>Detective Daniel Hanlon waited beneath the awning. Hanlon's coat was wet. Mara Ionescu waved to Hanlon. The detective thanked Mara.</p>`,
	}}}

	result := IndexBook(&book)
	if !result.Success {
		t.Fatalf("IndexBook failed: %s", result.Error)
	}

	got := make([]string, 0, len(book.Analysis.EntityResolution.Entities))
	for _, entity := range book.Analysis.EntityResolution.Entities {
		got = append(got, entity.Canonical)
	}
	if len(got) != 2 || !containsFold(got, "Daniel Hanlon") || !containsFold(got, "Mara Ionescu") {
		t.Fatalf("expected only Daniel Hanlon and Mara Ionescu, got %v", got)
	}

	banned := []string{
		"The", "He", "Chicago", "Hubbard", "Street", "Ontario", "Wells",
		"Ave", "River", "North", "Lower", "Upper", "Wacker", "Drive", "Detective",
	}
	for _, entity := range got {
		for _, word := range banned {
			if strings.EqualFold(entity, word) {
				t.Errorf("non-character %q was promoted into the cast", entity)
			}
		}
	}
}

func TestIndexBook_NonPersonEvidenceAndPersonOverrideAreBookWide(t *testing.T) {
	book := types.BookData{Body: []types.ChapterItem{
		{Title: "Chapter 1", Type: "chapter", Content: `<p>They waited beside Hubbard Street until dawn.</p>`},
		{Title: "Chapter 2", Type: "chapter", Content: `<p>Detective Hubbard arrived. Everyone trusted Hubbard immediately.</p>`},
	}}

	if result := IndexBook(&book); !result.Success {
		t.Fatalf("IndexBook failed: %s", result.Error)
	}

	entities := book.Analysis.EntityResolution.Entities
	if len(entities) != 1 || !strings.EqualFold(entities[0].Canonical, "Hubbard") {
		t.Fatalf("expected the titled person Hubbard, not the street, got %+v", entities)
	}
	if len(entities[0].MentionIDs) != 2 {
		t.Errorf("expected two person mentions, got %d", len(entities[0].MentionIDs))
	}
}

func TestIndexBook_PreservesInventedNames(t *testing.T) {
	book := types.BookData{Body: []types.ChapterItem{{
		Title: "Chapter 1",
		Type:  "chapter",
		Content: `<p>Mara Ionescu entered the chamber. Captain Glorpashlrop followed Mara.</p>` +
			`<p>Veyr studied Qalis beneath the red sun. Later, Qalis answered Veyr.</p>`,
	}}}

	if result := IndexBook(&book); !result.Success {
		t.Fatalf("IndexBook failed: %s", result.Error)
	}

	names := make([]string, 0, len(book.Analysis.EntityResolution.Entities))
	for _, entity := range book.Analysis.EntityResolution.Entities {
		names = append(names, entity.Canonical)
	}
	for _, expected := range []string{"Mara Ionescu", "Glorpashlrop", "Veyr", "Qalis"} {
		if !containsFold(names, expected) {
			t.Errorf("invented character %q was lost; got %v", expected, names)
		}
	}
}

func TestCommonSentenceGrammarDoesNotBecomePartOfNames(t *testing.T) {
	book := types.BookData{Body: []types.ChapterItem{{
		Title: "Chapter 1", Type: "chapter",
		Content: `<p>Maeve waited. Could Maeve have known? Yrene answered. Would Yrene agree? Thank Anneith for that.</p>`,
	}}}
	if result := IndexBook(&book); !result.Success {
		t.Fatalf("IndexBook failed: %s", result.Error)
	}
	names := make([]string, 0, len(book.StoryBible.Characters))
	for _, char := range book.StoryBible.Characters {
		names = append(names, char.Name)
	}
	for _, expected := range []string{"Maeve", "Yrene", "Anneith"} {
		if !containsFold(names, expected) {
			t.Errorf("expected %q in %v", expected, names)
		}
	}
	for _, name := range names {
		lower := strings.ToLower(name)
		if strings.Contains(lower, "could") || strings.Contains(lower, "would") || strings.Contains(lower, "thank") {
			t.Errorf("sentence grammar leaked into character name %q", name)
		}
	}
}

func containsFold(values []string, expected string) bool {
	for _, value := range values {
		if strings.EqualFold(value, expected) {
			return true
		}
	}
	return false
}
