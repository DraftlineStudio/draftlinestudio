package styleinspect

import (
	"strings"
	"testing"

	"draftline/internal/dialogue"
	"draftline/internal/types"
)

func buildBook(content string, characters ...types.Character) types.BookData {
	book := types.BookData{}
	book.StoryBible.Characters = characters
	book.Body = []types.ChapterItem{{ID: "ch1", Title: "One", Content: content}}
	return book
}

func TestEchoWordFlagsNearbyRepeat(t *testing.T) {
	book := buildBook("<p>The lantern guttered in the cold. A second lantern answered from the shore.</p>")
	data := Analyze(&book, book.Body, nil)
	if len(data.Findings) != 1 {
		t.Fatalf("expected 1 finding, got %+v", data.Findings)
	}
	finding := data.Findings[0]
	if finding.Rule != "echo-word" || finding.Snippet != "lantern" {
		t.Errorf("unexpected finding: %+v", finding)
	}
	text := "The lantern guttered in the cold. A second lantern answered from the shore."
	if text[finding.StartOffset:finding.EndOffset] != "lantern" {
		t.Errorf("offsets do not cover the echo: %q", text[finding.StartOffset:finding.EndOffset])
	}
	if !strings.HasSuffix(finding.Prefix, "second ") || !strings.HasPrefix(finding.Suffix, " answered") {
		t.Errorf("context wrong: prefix %q suffix %q", finding.Prefix, finding.Suffix)
	}
}

func TestEchoWordIgnoresDistantAndStopwords(t *testing.T) {
	filler := strings.Repeat("meanwhile the water moved beneath every board and plank of it, ", 8)
	book := buildBook("<p>The lantern guttered. " + filler + "Another lantern waited.</p>")
	data := Analyze(&book, book.Body, nil)
	for _, finding := range data.Findings {
		if finding.Snippet == "lantern" {
			t.Errorf("distant repeat should not flag: %+v", finding)
		}
		if echoStopwords[strings.ToLower(finding.Snippet)] {
			t.Errorf("stopword flagged: %+v", finding)
		}
	}
}

func TestEchoWordSkipsDialogueAndNames(t *testing.T) {
	content := "<p>“The signal. Give the signal now,” Mara said. Mara waited.</p>"
	book := buildBook(content, types.Character{ID: "c1", Name: "Mara"})
	book.Analysis.Dialogue = dialogue.Analyze(&book, book.Body, nil)
	data := Analyze(&book, book.Body, nil)
	if len(data.Findings) != 0 {
		t.Errorf("dialogue repetition and names must not flag: %+v", data.Findings)
	}
}

func TestAnalyzeCachesAndKeepsLiveDecisions(t *testing.T) {
	book := buildBook("<p>The lantern guttered beside the other lantern.</p>")
	first := Analyze(&book, book.Body, nil)
	if len(first.Findings) != 1 {
		t.Fatalf("expected 1 finding, got %+v", first.Findings)
	}

	first.Decisions = []types.StyleDecision{
		{FindingID: first.Findings[0].ID, Status: "dismissed"},
		{FindingID: "stale-id", Status: "dismissed"},
	}
	book.Analysis.Style = first
	second := Analyze(&book, book.Body, nil)
	if len(second.Findings) != 1 || second.Findings[0].ID != first.Findings[0].ID {
		t.Fatalf("unchanged chapter should reuse findings: %+v", second.Findings)
	}
	if len(second.Decisions) != 1 || second.Decisions[0].FindingID != first.Findings[0].ID {
		t.Errorf("live decision kept, stale dropped — got %+v", second.Decisions)
	}
}
