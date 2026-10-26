package dialogue

import (
	"strings"
	"testing"

	"draftline/internal/types"
)

func rosterOf(names ...string) []RosterEntry {
	entries := make([]RosterEntry, len(names))
	for i, name := range names {
		entries[i] = RosterEntry{Key: strings.ToLower(name), Name: name}
	}
	return entries
}

// sentencesOf segments invented paragraphs the way analyzeChapter does.
func sentencesOf(paragraphs ...string) []sentence {
	return chapterSentences(strings.Join(paragraphs, "\n\n"))
}

func TestSegmentTextSplitsProse(t *testing.T) {
	spans := segmentText("The ferry left at dawn. Nobody waved. Was that the end?")
	if len(spans) != 3 {
		t.Fatalf("expected 3 sentences, got %d", len(spans))
	}
	text := "The ferry left at dawn. Nobody waved."
	first := text[spans[0].start:spans[0].end]
	if first != "The ferry left at dawn." {
		t.Errorf("first sentence = %q", first)
	}
}

func TestSegmentTextKeepsAbbreviationsAndQuotes(t *testing.T) {
	cases := map[string]int{
		"Dr. Hale arrived late.":            1,
		"“Go away.” Then he left.":          2,
		"She paused... then knocked twice.": 1,
		"It cost 3.50 at the counter.":      1,
	}
	for text, want := range cases {
		if got := len(segmentText(text)); got != want {
			t.Errorf("%q: expected %d sentences, got %d", text, want, got)
		}
	}
}

func TestNamedTagAttribution(t *testing.T) {
	sentences := sentencesOf("“We should leave before dark,” Mara said.")
	attributed, scans := attributeSpeakers(sentences, rosterOf("Mara"))
	if len(attributed) != 1 {
		t.Fatalf("expected 1 sentence, got %d", len(attributed))
	}
	if attributed[0].speaker != "mara" || attributed[0].confidence != "tag" {
		t.Errorf("got speaker %q confidence %q", attributed[0].speaker, attributed[0].confidence)
	}
	if len(scans[0].quotedRanges) != 1 {
		t.Fatalf("expected one quoted range, got %d", len(scans[0].quotedRanges))
	}
}

func TestPreQuoteActionBeat(t *testing.T) {
	sentences := sentencesOf("Jonas set down the lantern. “It still burns.”")
	attributed, _ := attributeSpeakers(sentences, rosterOf("Jonas", "Mara"))
	last := attributed[len(attributed)-1]
	if last.speaker != "jonas" || last.confidence != "pretag" {
		t.Errorf("got speaker %q confidence %q", last.speaker, last.confidence)
	}
}

func TestAlternationHandsOff(t *testing.T) {
	sentences := sentencesOf(
		"“You counted the boats?” Mara asked.",
		"“Twice,” Jonas said.",
		"“And the oars?”",
		"“Also twice.”",
	)
	attributed, _ := attributeSpeakers(sentences, rosterOf("Mara", "Jonas"))
	// The segmenter splits a tag after ?” into its own narration sentence, so
	// compare only the dialogue lines.
	var speakers []string
	for _, a := range attributed {
		if a.kind != "narration" {
			speakers = append(speakers, a.speaker)
		}
	}
	want := []string{"mara", "jonas", "mara", "jonas"}
	if len(speakers) != len(want) {
		t.Fatalf("expected %d dialogue lines, got %v", len(want), speakers)
	}
	for i, expected := range want {
		if speakers[i] != expected {
			t.Fatalf("line %d: expected %q, got %q (all: %v)", i, expected, speakers[i], speakers)
		}
	}
}

func TestParagraphContinuationAcrossUnclosedQuote(t *testing.T) {
	sentences := sentencesOf(
		"“We walk until the ridge.", // unclosed: speech continues
		"“Then we rest,” Mara said.",
	)
	attributed, _ := attributeSpeakers(sentences, rosterOf("Mara"))
	for i, a := range attributed {
		if a.speaker != "mara" {
			t.Errorf("sentence %d: expected mara, got %q", i, a.speaker)
		}
	}
}

func TestSceneSeparatorResetsContext(t *testing.T) {
	sentences := sentencesOf(
		"“Keep rowing,” Mara said.",
		"* * *",
		"“Anyone there?”",
	)
	attributed, _ := attributeSpeakers(sentences, rosterOf("Mara"))
	last := attributed[len(attributed)-1]
	if last.speaker != speakerUnknown {
		t.Errorf("after a scene break the untagged line should be unknown, got %q", last.speaker)
	}
}

func TestPronounTagUsesGenderEvidence(t *testing.T) {
	sentences := sentencesOf(
		"Mara tied the skiff to the post. She checked the knot twice.",
		"Jonas watched from the dock. He said nothing yet.",
		"“This will hold,” she said.",
	)
	attributed, _ := attributeSpeakers(sentences, rosterOf("Mara", "Jonas"))
	last := attributed[len(attributed)-1]
	if last.speaker != "mara" {
		t.Errorf("pronoun tag should resolve to mara, got %q", last.speaker)
	}
}

func buildBook(content string, characters ...types.Character) types.BookData {
	book := types.BookData{}
	book.StoryBible.Characters = characters
	book.Body = []types.ChapterItem{{ID: "ch1", Title: "One", Content: content}}
	return book
}

func TestAnalyzeProducesSpansWithOffsets(t *testing.T) {
	book := buildBook(
		"<p>Mara pushed the door open.</p><p>“Nobody home,” Mara said.</p>",
		types.Character{ID: "c1", Name: "Mara"},
	)
	data := Analyze(&book, book.Body, nil)
	if len(data.Spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(data.Spans))
	}
	span := data.Spans[0]
	if span.SpeakerName != "Mara" || span.Attribution != "tag" {
		t.Errorf("got speaker %q attribution %q", span.SpeakerName, span.Attribution)
	}
	text := "Mara pushed the door open.\n\n“Nobody home,” Mara said."
	if quoted := text[span.StartOffset:span.EndOffset]; quoted != "“Nobody home,”" {
		t.Errorf("span text = %q", quoted)
	}
	if span.Confidence <= 0.9 {
		t.Errorf("tag confidence should be high, got %f", span.Confidence)
	}
	if span.ID == "" || data.Engine != Engine || data.Version != 1 {
		t.Errorf("span/data identity incomplete: %+v %+v", span, data)
	}
}

func TestAnalyzeSkipsUnconfirmedCharacters(t *testing.T) {
	book := buildBook(
		"<p>“Nobody home,” Mara said.</p>",
		types.Character{ID: "c1", Name: "Mara", IsAutoDetected: true, DetectionStatus: "review"},
	)
	data := Analyze(&book, book.Body, nil)
	if len(data.Spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(data.Spans))
	}
	if data.Spans[0].SpeakerName != "" || data.Spans[0].Attribution != "none" {
		t.Errorf("unconfirmed character must not attribute: %+v", data.Spans[0])
	}
}

func TestAnalyzeReusesUnchangedChapters(t *testing.T) {
	book := buildBook(
		"<p>“Nobody home,” Mara said.</p>",
		types.Character{ID: "c1", Name: "Mara"},
	)
	first := Analyze(&book, book.Body, nil)
	book.Analysis.Dialogue = first
	second := Analyze(&book, book.Body, nil)
	if len(second.Spans) != 1 || second.Spans[0].ID != first.Spans[0].ID {
		t.Errorf("unchanged chapter should reuse its spans: %+v vs %+v", first.Spans, second.Spans)
	}

	// A cast change must invalidate the cache even though the text is the same.
	book.StoryBible.Characters = append(book.StoryBible.Characters, types.Character{ID: "c2", Name: "Jonas"})
	book.Analysis.Dialogue = second
	third := Analyze(&book, book.Body, nil)
	if third.ChapterHashes[0] == second.ChapterHashes[0] {
		t.Error("roster change should change the chapter cache key")
	}
}

func TestAnalyzeMarksMixedSentences(t *testing.T) {
	book := buildBook(
		"<p>“Stay close,” Mara said, and hauled the rope through the cleat.</p>",
		types.Character{ID: "c1", Name: "Mara"},
	)
	data := Analyze(&book, book.Body, nil)
	if len(data.Spans) != 1 || data.Spans[0].Kind != "mixed" {
		t.Fatalf("expected one mixed span, got %+v", data.Spans)
	}
}

func TestSpeakerEntityIDComesFromResolution(t *testing.T) {
	book := buildBook(
		"<p>“Nobody home,” Mara said.</p>",
		types.Character{ID: "c1", Name: "Mara"},
	)
	book.Analysis.EntityResolution = &types.EntityData{
		Entities: []types.EntityRecord{{ID: "entity-3", Canonical: "Mara", CharacterID: "c1"}},
	}
	data := Analyze(&book, book.Body, nil)
	if len(data.Spans) != 1 || data.Spans[0].SpeakerEntityID != "entity-3" {
		t.Fatalf("expected entity-3, got %+v", data.Spans)
	}
}
