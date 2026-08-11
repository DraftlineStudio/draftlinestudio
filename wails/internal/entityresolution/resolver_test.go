package entityresolution

import (
	"strings"
	"testing"
)

// Test case: "Ruiz"/"Officer Ruiz"/"Detective Ruiz"/"Carlos Ruiz" → 1 entity
func TestResolveEntities_RuizVariants(t *testing.T) {
	mentions := []Mention{
		{ID: "m1", Text: "Ruiz", SentenceID: "s1", Chapter: 0, CharOffset: 100},
		{ID: "m2", Text: "Officer Ruiz", SentenceID: "s2", Chapter: 0, CharOffset: 200},
		{ID: "m3", Text: "Detective Ruiz", SentenceID: "s3", Chapter: 1, CharOffset: 50},
		{ID: "m4", Text: "Carlos Ruiz", SentenceID: "s4", Chapter: 2, CharOffset: 75},
	}

	resolver := NewResolver()
	result := resolver.ResolveEntities(mentions)

	if len(result.Entities) != 1 {
		t.Errorf("Expected 1 entity, got %d", len(result.Entities))
		for i, e := range result.Entities {
			t.Logf("Entity %d: %s (aliases: %v)", i, e.Canonical, e.Aliases)
		}
		return
	}

	entity := result.Entities[0]

	// Should have all 4 mentions
	if len(entity.MentionIDs) != 4 {
		t.Errorf("Expected 4 mentions, got %d", len(entity.MentionIDs))
	}

	// Canonical should be the most complete name
	if entity.Canonical != "Carlos Ruiz" {
		t.Errorf("Expected canonical 'Carlos Ruiz', got '%s'", entity.Canonical)
	}

	// Should have Officer and Detective in titles
	hasOfficer := false
	hasDetective := false
	for _, title := range entity.Titles {
		if strings.EqualFold(title, "Officer") {
			hasOfficer = true
		}
		if strings.EqualFold(title, "Detective") {
			hasDetective = true
		}
	}
	if !hasOfficer || !hasDetective {
		t.Errorf("Expected titles [Officer, Detective], got %v", entity.Titles)
	}
}

// Test case: Keller/Sgt Keller → same person
func TestResolveEntities_KellerWithRank(t *testing.T) {
	mentions := []Mention{
		{ID: "m1", Text: "Keller", SentenceID: "s1", Chapter: 0, CharOffset: 100},
		{ID: "m2", Text: "Sgt Keller", SentenceID: "s2", Chapter: 0, CharOffset: 200},
		{ID: "m3", Text: "Sgt. Keller", SentenceID: "s3", Chapter: 1, CharOffset: 50},
		{ID: "m4", Text: "Sergeant Keller", SentenceID: "s4", Chapter: 2, CharOffset: 75},
	}

	resolver := NewResolver()
	result := resolver.ResolveEntities(mentions)

	if len(result.Entities) != 1 {
		t.Errorf("Expected 1 entity, got %d", len(result.Entities))
		return
	}

	if len(result.Entities[0].MentionIDs) != 4 {
		t.Errorf("Expected 4 mentions, got %d", len(result.Entities[0].MentionIDs))
	}
}

// Test case: SplitEntity separates "Ruiz" (the sister)
func TestSplitEntity_RuizSister(t *testing.T) {
	mentions := []Mention{
		{ID: "m1", Text: "Ruiz", SentenceID: "s1", Chapter: 0, CharOffset: 100},
		{ID: "m2", Text: "Officer Ruiz", SentenceID: "s2", Chapter: 0, CharOffset: 200},
		{ID: "m3", Text: "Carlos Ruiz", SentenceID: "s3", Chapter: 1, CharOffset: 50},
		{ID: "m4", Text: "Ruiz", SentenceID: "s4", Chapter: 2, CharOffset: 75}, // This is the sister
	}

	resolver := NewResolver()
	result := resolver.ResolveEntities(mentions)

	// Initially should be 1 entity
	if len(result.Entities) != 1 {
		t.Fatalf("Expected 1 entity initially, got %d", len(result.Entities))
	}

	entityID := result.Entities[0].ID

	// Split out m4 (the sister)
	newEntities, err := resolver.SplitEntity(result.Entities, entityID, []string{"m4"})
	if err != nil {
		t.Fatalf("SplitEntity failed: %v", err)
	}

	if len(newEntities) != 2 {
		t.Errorf("Expected 2 entities after split, got %d", len(newEntities))
	}

	// Verify the split
	original := newEntities[0]
	if len(original.MentionIDs) != 3 {
		t.Errorf("Expected 3 mentions in original, got %d", len(original.MentionIDs))
	}

	split := newEntities[1]
	if len(split.MentionIDs) != 1 || split.MentionIDs[0] != "m4" {
		t.Errorf("Expected split entity to have only m4, got %v", split.MentionIDs)
	}

	// Verify separation pairs were added
	if len(resolver.SeparatedPairs) != 3 {
		t.Errorf("Expected 3 separated pairs, got %d", len(resolver.SeparatedPairs))
	}
}

// Test case: Re-resolution respects separated pairs
func TestResolveEntities_RespectsSeparatedPairs(t *testing.T) {
	mentions := []Mention{
		{ID: "m1", Text: "Ruiz", SentenceID: "s1", Chapter: 0, CharOffset: 100},
		{ID: "m2", Text: "Officer Ruiz", SentenceID: "s2", Chapter: 0, CharOffset: 200},
		{ID: "m3", Text: "Ruiz", SentenceID: "s3", Chapter: 1, CharOffset: 50}, // Sister
	}

	resolver := NewResolver()

	// Pre-add a separated pair (m3 is the sister)
	resolver.SeparatedPairs = []SeparatedPair{
		{MentionID1: "m1", MentionID2: "m3", Reason: "different person"},
		{MentionID1: "m2", MentionID2: "m3", Reason: "different person"},
	}

	result := resolver.ResolveEntities(mentions)

	if len(result.Entities) != 2 {
		t.Errorf("Expected 2 entities (separated), got %d", len(result.Entities))
		for i, e := range result.Entities {
			t.Logf("Entity %d: mentions %v", i, e.MentionIDs)
		}
	}
}

// Test: Honorific stripping
func TestStripHonorifics(t *testing.T) {
	tests := []struct {
		input    string
		wantHead string
		wantTitles []string
	}{
		{"Officer Ruiz", "Ruiz", []string{"Officer"}},
		{"Detective Ruiz", "Ruiz", []string{"Detective"}},
		{"Dr. Carlos Ruiz", "Carlos Ruiz", []string{"Dr."}},
		{"Sgt. Keller", "Keller", []string{"Sgt."}},
		{"Mr. John Smith", "John Smith", []string{"Mr."}},
		{"Ruiz", "Ruiz", []string{}},
		{"Carlos Ruiz", "Carlos Ruiz", []string{}},
		{"Lt. Col. James Bond", "James Bond", []string{"Lt.", "Col."}},
	}

	for _, tc := range tests {
		head, titles := StripHonorifics(tc.input, nil)
		if head != tc.wantHead {
			t.Errorf("StripHonorifics(%q): head = %q, want %q", tc.input, head, tc.wantHead)
		}
		if len(titles) != len(tc.wantTitles) {
			t.Errorf("StripHonorifics(%q): titles = %v, want %v", tc.input, titles, tc.wantTitles)
		}
	}
}

// Test: Levenshtein distance
func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		s1, s2 string
		want   int
	}{
		{"", "", 0},
		{"abc", "abc", 0},
		{"abc", "ab", 1},
		{"abc", "abcd", 1},
		{"abc", "axc", 1},
		{"kitten", "sitting", 3},
		{"Ruiz", "Riuz", 2}, // Transposition counts as 2 edits
		{"Smith", "Smyth", 1},
		{"Anderson", "Andersen", 1},
	}

	for _, tc := range tests {
		got := LevenshteinDistance(tc.s1, tc.s2)
		if got != tc.want {
			t.Errorf("LevenshteinDistance(%q, %q) = %d, want %d", tc.s1, tc.s2, got, tc.want)
		}
	}
}

// Test: Typo detection
func TestIsLikelyTypo(t *testing.T) {
	tests := []struct {
		s1, s2 string
		want   bool
	}{
		{"Smith", "Smyth", true},   // 1 substitution
		{"Anderson", "Andersen", true}, // 1 substitution
		{"Keller", "Kelller", true}, // 1 insertion
		{"Joe", "Jon", false},      // Too short
		{"Tim", "Tom", false},      // Too short
		{"Smith", "Brown", false},  // Different first letter
		{"Ruiz", "Riuz", false},    // Transposition = 2 edits
	}

	for _, tc := range tests {
		got := IsLikelyTypo(tc.s1, tc.s2)
		if got != tc.want {
			t.Errorf("IsLikelyTypo(%q, %q) = %v, want %v", tc.s1, tc.s2, got, tc.want)
		}
	}
}

// Test: Nickname matching
func TestIsNickname(t *testing.T) {
	tests := []struct {
		n1, n2 string
		want   bool
	}{
		{"William", "Bill", true},
		{"Bill", "William", true},
		{"Bob", "Robert", true},
		{"Jim", "James", true},
		{"Kate", "Katherine", true},
		{"Mike", "Michael", true},
		{"John", "Smith", false},
		{"Carlos", "Charlie", false}, // Not in our table
	}

	for _, tc := range tests {
		got := IsNickname(tc.n1, tc.n2)
		if got != tc.want {
			t.Errorf("IsNickname(%q, %q) = %v, want %v", tc.n1, tc.n2, got, tc.want)
		}
	}
}

// Test: Subset matching
func TestResolveEntities_SubsetMatching(t *testing.T) {
	mentions := []Mention{
		{ID: "m1", Text: "Smith", SentenceID: "s1", Chapter: 0, CharOffset: 100},
		{ID: "m2", Text: "John Smith", SentenceID: "s2", Chapter: 0, CharOffset: 200},
		{ID: "m3", Text: "John David Smith", SentenceID: "s3", Chapter: 1, CharOffset: 50},
	}

	resolver := NewResolver()
	result := resolver.ResolveEntities(mentions)

	if len(result.Entities) != 1 {
		t.Errorf("Expected 1 entity, got %d", len(result.Entities))
		return
	}

	entity := result.Entities[0]
	if entity.Canonical != "John David Smith" {
		t.Errorf("Expected canonical 'John David Smith', got '%s'", entity.Canonical)
	}
}

// Test: Two separate people named John
func TestResolveEntities_TwoJohns(t *testing.T) {
	mentions := []Mention{
		{ID: "m1", Text: "John Smith", SentenceID: "s1", Chapter: 0, CharOffset: 100},
		{ID: "m2", Text: "John Brown", SentenceID: "s2", Chapter: 0, CharOffset: 200},
	}

	resolver := NewResolver()
	result := resolver.ResolveEntities(mentions)

	// Should remain as 2 separate entities (different surnames)
	if len(result.Entities) != 2 {
		t.Errorf("Expected 2 entities (different people), got %d", len(result.Entities))
	}
}

// Test: Nickname with same surname should merge
func TestResolveEntities_NicknameWithSurname(t *testing.T) {
	mentions := []Mention{
		{ID: "m1", Text: "William Smith", SentenceID: "s1", Chapter: 0, CharOffset: 100},
		{ID: "m2", Text: "Bill Smith", SentenceID: "s2", Chapter: 0, CharOffset: 200},
		{ID: "m3", Text: "Billy Smith", SentenceID: "s3", Chapter: 1, CharOffset: 50},
	}

	resolver := NewResolver()
	result := resolver.ResolveEntities(mentions)

	if len(result.Entities) != 1 {
		t.Errorf("Expected 1 entity (nicknames), got %d", len(result.Entities))
		for i, e := range result.Entities {
			t.Logf("Entity %d: %s", i, e.Canonical)
		}
	}
}

// Test: Empty input
func TestResolveEntities_Empty(t *testing.T) {
	resolver := NewResolver()
	result := resolver.ResolveEntities([]Mention{})

	if len(result.Entities) != 0 {
		t.Errorf("Expected 0 entities for empty input, got %d", len(result.Entities))
	}
}

// Test: Single mention
func TestResolveEntities_SingleMention(t *testing.T) {
	mentions := []Mention{
		{ID: "m1", Text: "John", SentenceID: "s1", Chapter: 0, CharOffset: 100},
	}

	resolver := NewResolver()
	result := resolver.ResolveEntities(mentions)

	if len(result.Entities) != 1 {
		t.Errorf("Expected 1 entity, got %d", len(result.Entities))
	}
}

// Benchmark
func BenchmarkResolveEntities(b *testing.B) {
	// Create 1000 mentions with various patterns
	mentions := make([]Mention, 1000)
	names := []string{
		"Ruiz", "Officer Ruiz", "Carlos Ruiz", "Detective Ruiz",
		"Smith", "John Smith", "Mr. Smith",
		"Keller", "Sgt. Keller", "Sergeant Keller",
		"Jones", "Mary Jones", "Dr. Jones",
	}
	for i := 0; i < 1000; i++ {
		mentions[i] = Mention{
			ID:         string(rune(i)),
			Text:       names[i%len(names)],
			SentenceID: "s1",
			Chapter:    i / 100,
			CharOffset: i * 10,
		}
	}

	resolver := NewResolver()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resolver.ResolveEntities(mentions)
	}
}
