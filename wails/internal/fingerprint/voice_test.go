package fingerprint

import (
	"strings"
	"testing"

	"draftline/internal/types"
)

func TestVoiceProfileAttributesDialogueAndRecordsVernacular(t *testing.T) {
	text := `"Y'all ain't going nowhere," Mara said to Hanlon. Mara whispered, "Captain, keep walkin'."`
	first := strings.Index(text, "Mara")
	second := strings.LastIndex(text, "Mara")
	book := testBook(nil)
	book.Body = []types.ChapterItem{{ID: "chapter-1", Title: "Chapter 1", Type: "chapter", Content: text}}
	book.Analysis.EntityResolution = &types.EntityData{
		Mentions: []types.MentionRecord{{ID: "m1", Text: "Mara", Chapter: 0, CharOffset: first}, {ID: "m2", Text: "Mara", Chapter: 0, CharOffset: second}},
		Entities: []types.EntityRecord{{ID: "mara", Canonical: "Mara", Kind: "person", DetectionStatus: "accepted", MentionIDs: []string{"m1", "m2"}}},
	}
	book.Analysis.Fingerprint = &types.StoryFingerprint{AuthorModel: types.StoryAuthorModel{VoiceNotes: []types.CharacterVoiceNotes{{CharacterID: "mara", Dialect: "author-declared test dialect", SpeakingTraits: []string{"addresses officers by rank"}}}}}
	model := Build(&book, nil)
	if len(model.Voices) != 1 || model.Voices[0].SampleCount != 2 {
		t.Fatalf("expected two attributed samples, got %#v", model.Voices)
	}
	profile := model.Voices[0]
	if profile.AuthorNotes == nil || profile.AuthorNotes.Dialect == "" {
		t.Fatal("author voice notes did not survive analysis")
	}
	if !hasVoiceSignal(profile.DialectSignals, "vernacular-contraction") || !hasVoiceSignal(profile.DialectSignals, "dropped-final-g") || !hasVoiceSignal(profile.DialectSignals, "negative-concord") {
		t.Fatalf("missing dialect signals: %#v", profile.DialectSignals)
	}
	if len(profile.AddressForms) == 0 || profile.AddressForms[0].Text != "captain" {
		t.Fatalf("expected captain address form, got %#v", profile.AddressForms)
	}
	book.Analysis.Fingerprint = model
	answer := Query(book, types.FingerprintQueryRequest{Query: "How does Mara speak?"})
	if len(answer.Voices) != 1 || answer.Voices[0].CharacterID != "mara" {
		t.Fatalf("voice query did not return Mara: %#v", answer)
	}
}

func TestVoiceProfileDoesNotGuessSpeakerWithoutAttribution(t *testing.T) {
	text := `"Nobody knows who said this."`
	book := testBook(nil)
	book.Body = []types.ChapterItem{{ID: "chapter-1", Title: "Chapter 1", Type: "chapter", Content: text}}
	book.Analysis.EntityResolution = &types.EntityData{Entities: []types.EntityRecord{{ID: "mara", Canonical: "Mara", Kind: "person", DetectionStatus: "accepted"}}}
	model := Build(&book, nil)
	if len(model.Voices) != 0 {
		t.Fatalf("unattributed dialogue was assigned to a character: %#v", model.Voices)
	}
}

func hasVoiceSignal(values []types.VoiceSignal, kind string) bool {
	for _, value := range values {
		if value.Kind == kind {
			return true
		}
	}
	return false
}
