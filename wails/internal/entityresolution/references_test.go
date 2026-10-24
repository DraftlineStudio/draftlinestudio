package entityresolution

import (
	"reflect"
	"testing"
)

func TestSharedNameIsNotAssignedByRecency(t *testing.T) {
	mentions := []Mention{
		{ID: "a", Text: "Nelwin Brindle", Chapter: 0, SentenceID: "s1", CharOffset: 0},
		{ID: "b", Text: "Nelwin Mosswick", Chapter: 0, SentenceID: "s2", CharOffset: 30},
		{ID: "c", Text: "Nelwin", Chapter: 0, SentenceID: "s3", CharOffset: 60},
		{ID: "d", Text: "Nelwin", Chapter: 0, SentenceID: "s4", CharOffset: 90},
	}
	result := NewResolver().ResolveEntities(mentions)
	if len(result.Entities) != 2 || !reflect.DeepEqual(result.UnresolvedMentionIDs, []string{"c", "d"}) {
		t.Fatalf("unexpected resolution: %+v", result)
	}
	for _, entity := range result.Entities {
		if len(entity.MentionIDs) != 1 {
			t.Fatalf("ambiguous reference assigned: %+v", entity)
		}
	}
}

func TestSharedNameUsesUniqueChapterOrSentenceContext(t *testing.T) {
	for _, forward := range []bool{false, true} {
		t.Run(map[bool]string{false: "after", true: "before"}[forward], func(t *testing.T) {
			offset := 40
			if forward {
				offset = 0
			}
			mentions := []Mention{
				{ID: "a", Text: "Nelwin Brindle", Chapter: 0, SentenceID: "s1", CharOffset: 20},
				{ID: "b", Text: "Nelwin Mosswick", Chapter: 1, SentenceID: "s2", CharOffset: 20},
				{ID: "c", Text: "Nelwin", Chapter: 0, SentenceID: "s3", CharOffset: offset},
			}
			result := NewResolver().ResolveEntities(mentions)
			if len(result.Entities) != 2 || len(result.UnresolvedMentionIDs) != 0 {
				t.Fatalf("unexpected resolution: %+v", result)
			}
			for _, entity := range result.Entities {
				if entity.Canonical == "Nelwin Brindle" && !reflect.DeepEqual(entity.MentionIDs, []string{"a", "c"}) {
					t.Fatalf("reference lost: %+v", entity)
				}
			}
		})
	}
	mentions := []Mention{
		{ID: "a", Text: "Nelwin Brindle", Chapter: 0, SentenceID: "s1", CharOffset: 0},
		{ID: "b", Text: "Nelwin Mosswick", Chapter: 0, SentenceID: "s2", CharOffset: 20},
		{ID: "c", Text: "Nelwin", Chapter: 0, SentenceID: "s1", CharOffset: 10},
	}
	result := NewResolver().ResolveEntities(mentions)
	if len(result.UnresolvedMentionIDs) != 0 || len(result.Entities) != 2 {
		t.Fatalf("sentence context ignored: %+v", result)
	}
}

func TestFutureSharedNameDoesNotStealEarlierReferences(t *testing.T) {
	mentions := []Mention{
		{ID: "a", Text: "Nelwin Brindle", Chapter: 0, SentenceID: "s1", CharOffset: 0},
		{ID: "b", Text: "Nelwin", Chapter: 0, SentenceID: "s2", CharOffset: 20},
		{ID: "c", Text: "Nelwin Mosswick", Chapter: 0, SentenceID: "s3", CharOffset: 40},
	}
	result := NewResolver().ResolveEntities(mentions)
	if !reflect.DeepEqual(result.UnresolvedMentionIDs, []string{"b"}) {
		t.Fatalf("forward ambiguity ignored: %+v", result)
	}
}
