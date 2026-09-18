package plaintext

import (
	"strings"
	"testing"
)

// These cases are the contract the manuscript word count is built on, and the
// mobile app's TypeScript port of this file mirrors them case for case
// (capacitor/src/book/wordcount.ts). A change here is a change there.

func TestStripHTMLKeepsParagraphBoundaries(t *testing.T) {
	got := StripHTML("<p>One.</p><p>Two.</p>")
	if want := "One.\n\nTwo."; got != want {
		t.Fatalf("StripHTML = %q, want %q", got, want)
	}
}

func TestStripHTMLTurnsSceneBreakIntoThreeFields(t *testing.T) {
	// <hr> becomes "* * *", which strings.Fields counts as three words. The
	// count is a published number, so this is deliberate, not incidental.
	got := StripHTML("<p>One.</p><hr><p>Two.</p>")
	if n := len(strings.Fields(got)); n != 5 {
		t.Fatalf("fields = %d, want 5 (one + three + one); text was %q", n, got)
	}
}

func TestStripHTMLUnescapesEntities(t *testing.T) {
	got := StripHTML("<p>Rock&nbsp;and&nbsp;roll &amp; blues</p>")
	if n := len(strings.Fields(got)); n != 5 {
		t.Fatalf("fields = %d, want 5; text was %q", n, got)
	}
}

func TestStripHTMLTurnsLineBreaksIntoNewlines(t *testing.T) {
	got := StripHTML("<p>First line<br>second line</p>")
	if want := "First line\nsecond line"; got != want {
		t.Fatalf("StripHTML = %q, want %q", got, want)
	}
}

func TestStripHTMLForAnalysisDropsHeadings(t *testing.T) {
	got := StripHTMLForAnalysis("<h1>A Chapter Title Here</h1><p>One two three.</p>")
	if n := len(strings.Fields(got)); n != 3 {
		t.Fatalf("fields = %d, want 3; a heading is not manuscript prose. text was %q", n, got)
	}
}

func TestStripHTMLForAnalysisDropsStructuralBlocks(t *testing.T) {
	for _, tag := range []string{"nav", "header", "footer", "script", "style"} {
		in := "<" + tag + ">discarded words here</" + tag + "><p>kept.</p>"
		if got := StripHTMLForAnalysis(in); got != "kept." {
			t.Errorf("<%s>: StripHTMLForAnalysis = %q, want %q", tag, got, "kept.")
		}
	}
}

func TestStripHTMLOnEmptyContent(t *testing.T) {
	if got := StripHTML("<p></p>"); got != "" {
		t.Fatalf("StripHTML = %q, want empty", got)
	}
}
