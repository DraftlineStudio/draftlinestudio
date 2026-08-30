package entityresolution

import (
	"strings"
	"testing"
)

// Acceptance test: "Daniel Hanlon" / "Daniel" / "Hanlon" resolve to exactly
// ONE entity with canonical name "Daniel Hanlon" and all three mentions.
func TestResolveEntities_Acceptance(t *testing.T) {
	mentions := []Mention{
		{ID: "m1", Text: "Daniel Hanlon", SentenceID: "s1", Chapter: 0, CharOffset: 24},
		{ID: "m2", Text: "Daniel", SentenceID: "s2", Chapter: 0, CharOffset: 44},
		{ID: "m3", Text: "Hanlon", SentenceID: "s3", Chapter: 0, CharOffset: 69},
	}

	resolver := NewResolver()
	result := resolver.ResolveEntities(mentions)

	if len(result.Entities) != 1 {
		for i, e := range result.Entities {
			t.Logf("entity %d: %q (mentions %v)", i, e.Canonical, e.MentionIDs)
		}
		t.Fatalf("expected 1 entity, got %d", len(result.Entities))
	}

	e := result.Entities[0]
	if e.Canonical != "Daniel Hanlon" {
		t.Errorf("canonical: got %q, want %q", e.Canonical, "Daniel Hanlon")
	}
	if len(e.MentionIDs) != 3 {
		t.Errorf("expected 3 mentions on entity, got %d: %v", len(e.MentionIDs), e.MentionIDs)
	}
}

// Single-token entities seen BEFORE the multi-token form must merge into it:
// "Daniel" and "Hanlon" already exist when "Daniel Hanlon" arrives.
func TestResolveEntities_MergeIntoLongerForm(t *testing.T) {
	mentions := []Mention{
		{ID: "m1", Text: "Daniel", SentenceID: "s1", Chapter: 0, CharOffset: 10},
		{ID: "m2", Text: "Hanlon", SentenceID: "s2", Chapter: 0, CharOffset: 50},
		{ID: "m3", Text: "Daniel Hanlon", SentenceID: "s3", Chapter: 1, CharOffset: 5},
	}

	resolver := NewResolver()
	result := resolver.ResolveEntities(mentions)

	if len(result.Entities) != 1 {
		for i, e := range result.Entities {
			t.Logf("entity %d: %q (mentions %v)", i, e.Canonical, e.MentionIDs)
		}
		t.Fatalf("expected 1 entity after merge, got %d", len(result.Entities))
	}
	if result.Entities[0].Canonical != "Daniel Hanlon" {
		t.Errorf("canonical: got %q, want %q", result.Entities[0].Canonical, "Daniel Hanlon")
	}
	if len(result.Entities[0].MentionIDs) != 3 {
		t.Errorf("expected 3 mentions, got %v", result.Entities[0].MentionIDs)
	}
}

func TestResolveEntities_DoesNotBridgePeopleThroughSharedCommonToken(t *testing.T) {
	mentions := []Mention{
		{ID: "m1", Text: "Maeve", Chapter: 0, CharOffset: 10},
		{ID: "m2", Text: "Could Maeve", Chapter: 0, CharOffset: 30},
		{ID: "m3", Text: "Yrene", Chapter: 0, CharOffset: 60},
		{ID: "m4", Text: "Could Yrene", Chapter: 0, CharOffset: 90},
		{ID: "m5", Text: "That Could Not Be Broken", Chapter: 1, CharOffset: 10},
	}
	result := NewResolver().ResolveEntities(mentions)
	if len(result.Entities) < 3 {
		t.Fatalf("bridge mentions collapsed distinct people: %+v", result.Entities)
	}
	for _, entity := range result.Entities {
		aliases := strings.ToLower(strings.Join(entity.Aliases, " "))
		if strings.Contains(aliases, "maeve") && strings.Contains(aliases, "yrene") {
			t.Fatalf("Maeve and Yrene were transitively merged: %+v", entity)
		}
	}
}

func TestResolveEntities_DoesNotFuzzyMergeBareWords(t *testing.T) {
	mentions := []Mention{
		{ID: "m1", Text: "Mart", Chapter: 0, CharOffset: 10},
		{ID: "m2", Text: "Mark", Chapter: 0, CharOffset: 30},
		{ID: "m3", Text: "Bank", Chapter: 0, CharOffset: 50},
		{ID: "m4", Text: "Bonk", Chapter: 0, CharOffset: 70},
	}
	result := NewResolver().ResolveEntities(mentions)
	if len(result.Entities) != 4 {
		t.Fatalf("bare words were fuzzy-merged: %+v", result.Entities)
	}
}

func TestResolveEntities_CanonicalPrefersFrequentSingular(t *testing.T) {
	mentions := []Mention{
		{ID: "m1", Text: "Daniel Hanlons", Chapter: 0, CharOffset: 10},
		{ID: "m2", Text: "Daniel Hanlon", Chapter: 0, CharOffset: 30},
		{ID: "m3", Text: "Daniel Hanlon", Chapter: 1, CharOffset: 10},
	}
	result := NewResolver().ResolveEntities(mentions)
	if len(result.Entities) != 1 || result.Entities[0].Canonical != "Daniel Hanlon" {
		t.Fatalf("wrong canonical selection: %+v", result.Entities)
	}
}

// An ambiguous single-token mention resolves to the most recently
// mentioned matching entity.
func TestResolveEntities_RecencyDisambiguation(t *testing.T) {
	mentions := []Mention{
		{ID: "m1", Text: "John Smith", SentenceID: "s1", Chapter: 0, CharOffset: 10},
		{ID: "m2", Text: "John Brown", SentenceID: "s2", Chapter: 0, CharOffset: 100},
		{ID: "m3", Text: "John", SentenceID: "s3", Chapter: 0, CharOffset: 200},
	}

	resolver := NewResolver()
	result := resolver.ResolveEntities(mentions)

	if len(result.Entities) != 2 {
		t.Fatalf("expected 2 entities, got %d", len(result.Entities))
	}

	// m3 ("John") must resolve to John Brown (most recent).
	var brown *Entity
	for i := range result.Entities {
		if result.Entities[i].Canonical == "John Brown" {
			brown = &result.Entities[i]
		}
	}
	if brown == nil {
		t.Fatal("John Brown entity not found")
	}
	found := false
	for _, id := range brown.MentionIDs {
		if id == "m3" {
			found = true
		}
	}
	if !found {
		t.Errorf("ambiguous 'John' should resolve to most recent entity John Brown; got mentions %v", brown.MentionIDs)
	}
}

// Entity IDs must be deterministic across runs and unique within a run.
func TestResolveEntities_DeterministicIDs(t *testing.T) {
	mentions := []Mention{
		{ID: "m1", Text: "Alice", SentenceID: "s1", Chapter: 0, CharOffset: 10},
		{ID: "m2", Text: "Bob", SentenceID: "s2", Chapter: 0, CharOffset: 50},
		{ID: "m3", Text: "Carol", SentenceID: "s3", Chapter: 0, CharOffset: 90},
	}

	a := NewResolver().ResolveEntities(mentions)
	b := NewResolver().ResolveEntities(mentions)

	if len(a.Entities) != 3 || len(b.Entities) != 3 {
		t.Fatalf("expected 3 entities, got %d and %d", len(a.Entities), len(b.Entities))
	}
	seen := map[string]bool{}
	for i := range a.Entities {
		if a.Entities[i].ID != b.Entities[i].ID {
			t.Errorf("non-deterministic entity ID at %d: %q vs %q", i, a.Entities[i].ID, b.Entities[i].ID)
		}
		if seen[a.Entities[i].ID] {
			t.Errorf("duplicate entity ID %q", a.Entities[i].ID)
		}
		seen[a.Entities[i].ID] = true
	}
}

// Conflicting professional titles mean different people: Dr. Chen and
// Staff Sgt. Chen must NOT merge, and neither may inherit the other's title.
func TestResolveEntities_TitleConflict(t *testing.T) {
	mentions := []Mention{
		{ID: "m1", Text: "Dr. Chen", SentenceID: "s1", Chapter: 0, CharOffset: 10},
		{ID: "m2", Text: "Chen", SentenceID: "s2", Chapter: 0, CharOffset: 60},
		{ID: "m3", Text: "Staff Sgt. Chen", SentenceID: "s3", Chapter: 1, CharOffset: 5},
	}

	resolver := NewResolver()
	result := resolver.ResolveEntities(mentions)

	if len(result.Entities) != 2 {
		for _, e := range result.Entities {
			t.Logf("entity: %q titles=%v mentions=%v", e.Canonical, e.Titles, e.MentionIDs)
		}
		t.Fatalf("expected 2 entities (conflicting titles), got %d", len(result.Entities))
	}

	for _, e := range result.Entities {
		hasDr, hasSgt := false, false
		for _, title := range e.Titles {
			lower := strings.ToLower(strings.TrimSuffix(title, "."))
			if lower == "dr" {
				hasDr = true
			}
			if lower == "sgt" || lower == "staff" {
				hasSgt = true
			}
		}
		if hasDr && hasSgt {
			t.Errorf("entity %q inherited titles from both people: %v", e.Canonical, e.Titles)
		}
	}
}
