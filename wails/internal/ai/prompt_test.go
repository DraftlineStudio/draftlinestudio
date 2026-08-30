package ai

import (
	"strings"
	"testing"
)

// copy_edit must be a conservative correctness pass: it carries the
// error-fixing instructions and none of the enrichment language from the
// expand mode, and it must not fall through to the line_edit default.
func TestCopyEditPrompt(t *testing.T) {
	p := BuildSystemPrompt("copy_edit", "", nil)

	for _, want := range []string{"copy editor", "objective mechanical errors", "NEVER rephrase", "capitalization", "one <p> element per original paragraph"} {
		if !strings.Contains(p, want) {
			t.Fatalf("copy_edit prompt missing %q", want)
		}
	}
	for _, forbid := range []string{"Expand and enrich", "sensory", "Vary sentence rhythm", "line editor"} {
		if strings.Contains(p, forbid) {
			t.Fatalf("copy_edit prompt contains enrichment/restyle language %q", forbid)
		}
	}
}

// Copy Edit is intentionally mechanical: including style examples costs
// tokens and nudges models toward subjective rewriting.
func TestCopyEditPromptExcludesProseGuide(t *testing.T) {
	p := BuildSystemPrompt("copy_edit", "Sample voice paragraph.", nil)
	if strings.Contains(p, "Sample voice paragraph.") {
		t.Fatal("copy_edit prompt includes the prose guide")
	}
}

func TestLineEditPromptIsSelective(t *testing.T) {
	p := BuildSystemPrompt("line_edit", "Sample voice paragraph.", nil)
	for _, want := range []string{"restrained line editor", "smallest edit", "preserve it exactly", "never as a reason to rewrite", "constrain wording YOU INTRODUCE"} {
		if !strings.Contains(p, want) {
			t.Fatalf("line_edit prompt missing restraint %q", want)
		}
	}
	for _, forbid := range []string{"Rewrite the provided", "Vary sentence rhythm", "Use strong, precise"} {
		if strings.Contains(p, forbid) {
			t.Fatalf("line_edit prompt still encourages broad rewriting with %q", forbid)
		}
	}
}

// Unknown modes must still fall through to the line_edit default.
func TestUnknownModeFallsBackToLineEdit(t *testing.T) {
	p := BuildSystemPrompt("nonsense", "", nil)
	if !strings.Contains(p, "restrained line editor") {
		t.Fatal("unknown mode did not fall back to line_edit prompt")
	}
}
