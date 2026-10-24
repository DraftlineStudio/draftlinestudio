package indexing

import (
	"encoding/json"
	"reflect"
	"testing"

	"draftline/internal/entityresolution"
	"draftline/internal/types"
)

func TestSpellingAliasesExcludeLikelyTypos(t *testing.T) {
	aliases := []string{"Nelwin Brindle", "Nelwin Brindel", "Nelwin Brindls", "Brindls", "Dr. Nelwin Brindls", "Dr. Nelwin Brindle", "Nell Brindle"}
	characters := ConvertEntitiesToCharacters([]entityresolution.Entity{{
		ID: "person", Canonical: "Nelwin Brindle", Aliases: aliases, DetectionStatus: "accepted",
	}}, nil)
	data, err := json.Marshal(characters[0])
	if err != nil {
		t.Fatal(err)
	}
	var got types.Character
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	want := []string{"Nelwin Brindle", "Nelwin Brindel", "Dr. Nelwin Brindle", "Nell Brindle"}
	if !reflect.DeepEqual(got.SpellingAliases, want) {
		t.Fatalf("spelling aliases = %v, want %v", got.SpellingAliases, want)
	}
	if !reflect.DeepEqual(got.Aliases, aliases) {
		t.Fatal("mention aliases were changed")
	}
}

func TestSpellingAliasesPreservesEmptyList(t *testing.T) {
	got := spellingAliases("Nelwin Brindle", []string{"Nelwin Brindls"})
	if got == nil || len(got) != 0 {
		t.Fatalf("expected an explicit empty alias list, got %v", got)
	}
}
