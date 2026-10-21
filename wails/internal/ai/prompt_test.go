package ai

import (
	"strings"
	"testing"
)

// copy_edit must be a conservative correctness pass: it carries the
// error-fixing instructions and none of the enrichment language from the
// expand mode, and it must not fall through to the line_edit default.
func TestCopyEditPrompt(t *testing.T) {
	p := BuildSystemPrompt("copy_edit", "", nil, "en_US")

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
	p := BuildSystemPrompt("copy_edit", "Sample voice paragraph.", nil, "en_US")
	if strings.Contains(p, "Sample voice paragraph.") {
		t.Fatal("copy_edit prompt includes the prose guide")
	}
}

func TestLineEditPromptIsSelective(t *testing.T) {
	p := BuildSystemPrompt("line_edit", "Sample voice paragraph.", nil, "en_US")
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

// A non-American manuscript carries its convention in every mode; an American
// one says nothing, because American spelling is already every model's default.
func TestDialectSpellingRule(t *testing.T) {
	dialects := map[string]string{
		"en_GB": "British English",
		"en_CA": "Canadian English",
		"en_AU": "Australian English",
	}
	for lang, name := range dialects {
		for _, mode := range []string{"copy_edit", "line_edit", "expand", "smooth"} {
			p := BuildSystemPrompt(mode, "", nil, lang)
			if !strings.Contains(p, name) {
				t.Fatalf("%s prompt for %s never names %s", mode, lang, name)
			}
			if !strings.Contains(p, "colour") {
				t.Fatalf("%s prompt for %s carries no example spellings", mode, lang)
			}
		}
		// copy_edit alone treats American spellings as errors to correct; the
		// rewriting modes only constrain what they write.
		if p := BuildSystemPrompt("copy_edit", "", nil, lang); !strings.Contains(p, "spelling error to correct") {
			t.Fatalf("copy_edit prompt for %s does not correct American spellings", lang)
		}
	}
	for _, lang := range []string{"en_US", ""} {
		if p := BuildSystemPrompt("copy_edit", "", nil, lang); strings.Contains(p, "English:") {
			t.Fatalf("copy_edit prompt for %q carries a dialect rule", lang)
		}
	}
}

// Unknown modes must still fall through to the line_edit default.
func TestUnknownModeFallsBackToLineEdit(t *testing.T) {
	p := BuildSystemPrompt("nonsense", "", nil, "")
	if !strings.Contains(p, "restrained line editor") {
		t.Fatal("unknown mode did not fall back to line_edit prompt")
	}
}
