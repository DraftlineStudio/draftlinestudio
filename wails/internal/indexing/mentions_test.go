package indexing

import (
	"strings"
	"testing"

	"draftline/internal/types"
)

// Acceptance test for Phase 1 mention extraction.
// A multi-token name is ONE mention; trailing credentials are discarded;
// later single-token references are separate mentions.
func TestExtractMentions_Acceptance(t *testing.T) {
	text := `"Nice to meet you, I'm Daniel Hanlon, CPD." Daniel sat down. Later, Hanlon stood.`

	mentions := ExtractMentions(text, 0)

	if len(mentions) != 3 {
		for _, m := range mentions {
			t.Logf("mention: %q at %d", m.Text, m.CharOffset)
		}
		t.Fatalf("expected 3 mentions, got %d", len(mentions))
	}

	want := []string{"Daniel Hanlon", "Daniel", "Hanlon"}
	for i, w := range want {
		if mentions[i].Text != w {
			t.Errorf("mention %d: got %q, want %q", i, mentions[i].Text, w)
		}
	}

	// Offsets must point at the actual spans in the source text.
	for _, m := range mentions {
		end := m.CharOffset + len(m.Text)
		if m.CharOffset < 0 || end > len(text) || text[m.CharOffset:end] != m.Text {
			t.Errorf("mention %q offset %d does not match source text", m.Text, m.CharOffset)
		}
	}
}

func TestReindexPreservesCuratedAutoCharacterFields(t *testing.T) {
	book := types.BookData{Body: []types.ChapterItem{{
		Title: "Chapter 1", Type: "chapter",
		Content: `<p>Detective Mara Ionescu entered. Mara studied the room.</p>`,
	}}}
	if result := IndexBook(&book); !result.Success {
		t.Fatalf("first IndexBook failed: %s", result.Error)
	}
	if len(book.StoryBible.Characters) != 1 {
		t.Fatalf("expected one character, got %d", len(book.StoryBible.Characters))
	}
	book.StoryBible.Characters[0].Role = "protagonist"
	book.StoryBible.Characters[0].Description = "Captain of the Echo"
	book.StoryBible.Characters[0].Notes = "User-authored note"

	if result := IndexBook(&book); !result.Success {
		t.Fatalf("second IndexBook failed: %s", result.Error)
	}
	got := book.StoryBible.Characters[0]
	if got.Role != "protagonist" || got.Description != "Captain of the Echo" || got.Notes != "User-authored note" {
		t.Fatalf("curated fields were lost on re-index: %+v", got)
	}
}

// Honorifics are part of the raw span (metadata is extracted during
// resolution), but never split a multi-token name into two mentions.
func TestExtractMentions_Honorifics(t *testing.T) {
	text := `Dr. Jane Smith opened the door. Det. Hanlon watched.`

	mentions := ExtractMentions(text, 0)

	if len(mentions) != 2 {
		for _, m := range mentions {
			t.Logf("mention: %q at %d", m.Text, m.CharOffset)
		}
		t.Fatalf("expected 2 mentions, got %d", len(mentions))
	}
	if mentions[0].Text != "Dr. Jane Smith" {
		t.Errorf("mention 0: got %q, want %q", mentions[0].Text, "Dr. Jane Smith")
	}
	if mentions[1].Text != "Det. Hanlon" {
		t.Errorf("mention 1: got %q, want %q", mentions[1].Text, "Det. Hanlon")
	}
}

// Sentence-leading common words must not become mentions or prefix a name.
func TestExtractMentions_CommonWordFilter(t *testing.T) {
	text := `Suddenly Marcus ran. The street was empty. Perhaps tomorrow. Fine, he thought.`

	mentions := ExtractMentions(text, 0)

	if len(mentions) != 1 {
		for _, m := range mentions {
			t.Logf("mention: %q at %d", m.Text, m.CharOffset)
		}
		t.Fatalf("expected 1 mention, got %d", len(mentions))
	}
	if mentions[0].Text != "Marcus" {
		t.Errorf("got %q, want %q", mentions[0].Text, "Marcus")
	}
}

// Place/food/brand patterns are not characters.
func TestExtractMentions_FalsePositives(t *testing.T) {
	text := `They met on Hubbard Street. He ordered General Tso's Chicken while Kira laughed.`

	mentions := ExtractMentions(text, 0)

	for _, m := range mentions {
		if strings.Contains(m.Text, "Hubbard") || strings.Contains(m.Text, "Tso") {
			t.Errorf("false positive extracted: %q", m.Text)
		}
	}

	found := false
	for _, m := range mentions {
		if m.Text == "Kira" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected to find mention %q, got %v", "Kira", mentions)
	}
}

// Possessives resolve to the bare name; contractions are not names.
func TestExtractMentions_Possessives(t *testing.T) {
	text := `Kira's eyes narrowed. I'm sure O'Brien noticed.`

	mentions := ExtractMentions(text, 0)

	got := map[string]bool{}
	for _, m := range mentions {
		got[m.Text] = true
	}
	if !got["Kira"] {
		t.Errorf("expected possessive to yield %q, got %v", "Kira", mentions)
	}
	if !got["O'Brien"] {
		t.Errorf("expected apostrophe name %q, got %v", "O'Brien", mentions)
	}
	for text := range got {
		if strings.HasPrefix(text, "I'") {
			t.Errorf("contraction extracted as mention: %q", text)
		}
	}
}

// Mention IDs must be unique and deterministic.
func TestExtractMentions_DeterministicIDs(t *testing.T) {
	text := `Alice met Bob. Alice waved. Bob nodded. Carol joined Alice and Bob.`

	a := ExtractMentions(text, 2)
	b := ExtractMentions(text, 2)

	if len(a) != len(b) {
		t.Fatalf("non-deterministic mention count: %d vs %d", len(a), len(b))
	}
	seen := map[string]bool{}
	for i := range a {
		if a[i].ID != b[i].ID {
			t.Errorf("non-deterministic ID at %d: %q vs %q", i, a[i].ID, b[i].ID)
		}
		if seen[a[i].ID] {
			t.Errorf("duplicate mention ID %q", a[i].ID)
		}
		seen[a[i].ID] = true
	}
}

// End-to-end acceptance: indexing a book with the acceptance text must
// produce exactly ONE character with canonical name "Daniel Hanlon" and
// three mentions resolving to it.
func TestIndexBook_Acceptance(t *testing.T) {
	book := types.BookData{
		Body: []types.ChapterItem{
			{
				Title:   "Chapter 1",
				Type:    "chapter",
				Content: `<p>&quot;Nice to meet you, I'm Daniel Hanlon, CPD.&quot; Daniel sat down. Later, Hanlon stood.</p>`,
			},
		},
	}

	result := IndexBook(&book)
	if !result.Success {
		t.Fatalf("IndexBook failed: %s", result.Error)
	}

	autoChars := []types.Character{}
	for _, c := range book.StoryBible.Characters {
		if c.IsAutoDetected {
			autoChars = append(autoChars, c)
		}
	}
	if len(autoChars) != 1 {
		for _, c := range autoChars {
			t.Logf("character: %q (aliases %v)", c.Name, c.Aliases)
		}
		t.Fatalf("expected exactly 1 auto-detected character, got %d", len(autoChars))
	}
	if autoChars[0].Name != "Daniel Hanlon" {
		t.Errorf("canonical name: got %q, want %q", autoChars[0].Name, "Daniel Hanlon")
	}
	if autoChars[0].MentionCount != 3 {
		t.Errorf("mention count: got %d, want 3", autoChars[0].MentionCount)
	}

	// Entity data must be stored with all mentions resolving to the entity.
	ed := book.Analysis.EntityResolution
	if ed == nil {
		t.Fatal("entity resolution data not stored on book")
	}
	if len(ed.Entities) != 1 {
		t.Fatalf("expected 1 entity, got %d", len(ed.Entities))
	}
	if len(ed.Mentions) != 3 {
		t.Fatalf("expected 3 mentions, got %d", len(ed.Mentions))
	}
	if len(ed.Entities[0].MentionIDs) != 3 {
		t.Errorf("expected entity to own 3 mentions, got %d", len(ed.Entities[0].MentionIDs))
	}
}

// Mentions in front matter must not leak into body chapter indexes:
// chapter indexes are global across front matter, body, and back matter,
// and relationship analysis must use the same enumeration.
func TestIndexBook_GlobalChapterIndexes(t *testing.T) {
	book := types.BookData{
		FrontMatter: []types.ChapterItem{
			{Title: "Foreword", Type: "foreword", Content: `<p>Alice wrote this foreword.</p>`},
		},
		Body: []types.ChapterItem{
			{Title: "Chapter 1", Type: "chapter", Content: `<p>Marcus met Kira. "Hello," Marcus said. Kira smiled at Marcus.</p>`},
		},
	}

	result := IndexBook(&book)
	if !result.Success {
		t.Fatalf("IndexBook failed: %s", result.Error)
	}

	// Front matter is chapter 0; body starts at 1.
	for _, m := range book.Analysis.EntityResolution.Mentions {
		if m.Text == "Alice" && m.Chapter != 0 {
			t.Errorf("front matter mention chapter: got %d, want 0", m.Chapter)
		}
		if (m.Text == "Marcus" || m.Text == "Kira") && m.Chapter != 1 {
			t.Errorf("body mention %q chapter: got %d, want 1", m.Text, m.Chapter)
		}
	}
}

// Compound military ranks are honorific prefixes, and full-word honorifics
// at the END of a previous sentence must never glue onto the next name
// ("...saluted the general. Dr. Chen" is NOT "general. Dr. Chen").
func TestExtractMentions_RanksAndSentenceBoundaries(t *testing.T) {
	text := `Staff Sgt. Chen saluted the general. Dr. Chen frowned. She told her mother. Kira left. Nobody stopped Kira.`

	mentions := ExtractMentions(text, 0)

	want := []string{"Staff Sgt. Chen", "Dr. Chen", "Kira", "Kira"}
	if len(mentions) != len(want) {
		for _, m := range mentions {
			t.Logf("mention: %q at %d", m.Text, m.CharOffset)
		}
		t.Fatalf("expected %d mentions, got %d", len(want), len(mentions))
	}
	for i, w := range want {
		if mentions[i].Text != w {
			t.Errorf("mention %d: got %q, want %q", i, mentions[i].Text, w)
		}
	}
}

// A capitalized sentence-lead word that never appears mid-sentence must not
// become part of a name ("Resolute Marcus strode" is Marcus, not Resolute
// Marcus). A real given name used mid-sentence elsewhere survives.
func TestExtractMentions_SentenceInitialAdjectives(t *testing.T) {
	text := `Resolute Marcus strode in. Everyone watched Marcus go. Carlos Ruiz stood. They saluted Carlos Ruiz.`

	mentions := ExtractMentions(text, 0)

	for _, m := range mentions {
		if strings.Contains(m.Text, "Resolute") {
			t.Errorf("sentence-lead adjective leaked into mention: %q", m.Text)
		}
	}

	got := map[string]int{}
	for _, m := range mentions {
		got[m.Text]++
	}
	if got["Marcus"] != 2 {
		t.Errorf("expected 2 bare Marcus mentions, got %v", got)
	}
	// "Carlos Ruiz stood" leads a sentence, but Carlos is corroborated by
	// the mid-sentence "saluted Carlos Ruiz" - the full name must survive.
	if got["Carlos Ruiz"] != 2 {
		t.Errorf("expected 2 Carlos Ruiz mentions, got %v", got)
	}
}

// Manual merge: two entities the resolver kept apart are declared the same
// person. The merge persists as a name rule that survives re-indexing.
func TestMergeCharacterEntities(t *testing.T) {
	book := types.BookData{
		Body: []types.ChapterItem{
			{Title: "Ch1", Type: "chapter", Content: `<p>Rook watched the door. Marcus Webb waited outside. Everyone called Marcus by his nickname, and Rook answered to it.</p>`},
		},
	}
	if r := IndexBook(&book); !r.Success {
		t.Fatalf("IndexBook failed: %s", r.Error)
	}

	// "Rook" (a nickname) and "Marcus Webb" resolve separately.
	var rookID, webbID string
	for _, e := range book.Analysis.EntityResolution.Entities {
		switch e.Canonical {
		case "Rook":
			rookID = e.ID
		case "Marcus Webb":
			webbID = e.ID
		}
	}
	if rookID == "" || webbID == "" {
		t.Fatalf("expected Rook and Marcus Webb entities, got %+v", book.Analysis.EntityResolution.Entities)
	}

	if err := MergeCharacterEntities(&book, []string{webbID, rookID}, ""); err != nil {
		t.Fatalf("MergeCharacterEntities failed: %v", err)
	}

	if len(book.Analysis.EntityResolution.Entities) != 1 {
		t.Fatalf("expected 1 entity after merge, got %d", len(book.Analysis.EntityResolution.Entities))
	}
	merged := book.Analysis.EntityResolution.Entities[0]
	if merged.Canonical != "Marcus Webb" {
		t.Errorf("canonical after merge: got %q, want %q", merged.Canonical, "Marcus Webb")
	}
	hasRookAlias := false
	for _, a := range merged.Aliases {
		if a == "Rook" {
			hasRookAlias = true
		}
	}
	if !hasRookAlias {
		t.Errorf("expected Rook as alias, got %v", merged.Aliases)
	}

	// Re-index: the merge rule must be applied again automatically.
	if r := IndexBook(&book); !r.Success {
		t.Fatalf("re-IndexBook failed: %s", r.Error)
	}
	if len(book.Analysis.EntityResolution.Entities) != 1 {
		for _, e := range book.Analysis.EntityResolution.Entities {
			t.Logf("entity: %q", e.Canonical)
		}
		t.Errorf("merge rule not applied on re-index: got %d entities", len(book.Analysis.EntityResolution.Entities))
	}
}

// Regression for real-manuscript junk: organizations, locations, floor
// designations, and trailing pronouns must not become characters.
func TestExtractMentions_OrganizationsAndPlaces(t *testing.T) {
	text := `He reported to the Chicago Police Department that evening. At Sub-Level Six they found the lab. ` +
		`The offices on Hubbard Street were dark, and Hubbard was silent at that hour. ` +
		`"Get down," said Gary Mason He pulled her behind the crates toward Kira. Kira nodded.`

	mentions := ExtractMentions(text, 0)

	banned := []string{"Chicago", "Police", "Department", "At", "Sub-Level", "Six", "Hubbard", "Level", "He"}
	for _, m := range mentions {
		for _, b := range banned {
			for _, tok := range strings.Fields(m.Text) {
				if strings.TrimSuffix(tok, ",") == b {
					t.Errorf("junk token %q leaked into mention %q", b, m.Text)
				}
			}
		}
	}

	got := map[string]int{}
	for _, m := range mentions {
		got[m.Text]++
	}
	if got["Gary Mason"] == 0 {
		t.Errorf("expected Gary Mason (trailing pronoun dropped), got %v", got)
	}
	if got["Kira"] == 0 {
		t.Errorf("expected Kira, got %v", got)
	}
}

// Abstract nouns opening sentences must not become characters: a
// single-token name at a sentence start needs corroboration elsewhere in
// the book (mid-sentence use, possessive, or a full name containing it).
func TestExtractMentions_SentenceOpeners(t *testing.T) {
	text := `Thunder rolled across the bay. Silence followed. Dust settled on the sill. ` +
		`Mirela watched it all. Later that night, Mirela slept.`

	mentions := ExtractMentions(text, 0)

	for _, m := range mentions {
		if m.Text == "Thunder" || m.Text == "Silence" || m.Text == "Dust" {
			t.Errorf("sentence-opener noun extracted as character: %q", m.Text)
		}
	}
	count := 0
	for _, m := range mentions {
		if m.Text == "Mirela" {
			count++
		}
	}
	// "Mirela watched" opens a sentence, but "Later that night, Mirela slept"
	// corroborates her mid-sentence — both mentions must survive.
	if count != 2 {
		t.Errorf("expected 2 Mirela mentions, got %d (%v)", count, mentions)
	}
}

// Corroboration is book-wide: a name attested in one chapter validates its
// sentence-initial uses in other chapters.
func TestIndexBook_CrossChapterAttestation(t *testing.T) {
	book := types.BookData{
		Body: []types.ChapterItem{
			{Title: "Ch1", Type: "chapter", Content: `<p>Vasquez stood alone in the rain.</p>`},
			{Title: "Ch2", Type: "chapter", Content: `<p>They all feared Vasquez by then.</p>`},
		},
	}
	if r := IndexBook(&book); !r.Success {
		t.Fatalf("IndexBook failed: %s", r.Error)
	}
	for _, e := range book.Analysis.EntityResolution.Entities {
		if e.Canonical == "Vasquez" && len(e.MentionIDs) == 2 {
			return
		}
	}
	t.Errorf("expected Vasquez with 2 mentions (ch2 attests ch1), got %+v", book.Analysis.EntityResolution.Entities)
}
