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

	for _, want := range []string{"copy editor", "objective errors", "NEVER rephrase", "one <p> element per original paragraph"} {
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

// The prose style guide still rides along so continuity fixes can respect
// the author's conventions.
func TestCopyEditPromptIncludesProseGuide(t *testing.T) {
	p := BuildSystemPrompt("copy_edit", "Sample voice paragraph.", nil)
	if !strings.Contains(p, "Sample voice paragraph.") {
		t.Fatal("copy_edit prompt does not include the prose guide")
	}
}

// Unknown modes must still fall through to the line_edit default.
func TestUnknownModeFallsBackToLineEdit(t *testing.T) {
	p := BuildSystemPrompt("nonsense", "", nil)
	if !strings.Contains(p, "literary prose editor") {
		t.Fatal("unknown mode did not fall back to line_edit prompt")
	}
}
