// Package plaintext turns chapter HTML into the plain text every other stage
// measures against. It is the shared coordinate space: mention offsets, scene
// boundaries, dialogue ranges and the manuscript word count all refer to the
// string StripHTML returns.
//
// It is a leaf on purpose. These helpers used to live in internal/indexing
// beside the character pipeline, which imports a natural-language toolkit and
// its trained models; that made the word count — three regexes and a split —
// drag eight megabytes of NLP into anything that wanted it. The mobile build
// binds internal/book into a native library, so that cost was the difference
// between a 3.6 MB and a 15 MB shared object for a function that counts words.
//
// Nothing in here may import anything but the standard library and
// internal/types. If a helper needs the NLP toolkit, it belongs in
// internal/indexing instead.
package plaintext

import (
	"html"
	"regexp"
	"strings"
)

var (
	// Block-level closers become paragraph breaks so scene detection can
	// split on them. <hr> becomes a scene-break marker.
	hrTagRe    = regexp.MustCompile(`(?i)<hr\s*/?>`)
	brTagRe    = regexp.MustCompile(`(?i)<br\s*/?>`)
	blockEndRe = regexp.MustCompile(`(?i)</(p|div|h[1-6]|li|blockquote|tr)>`)
	anyTagRe   = regexp.MustCompile(`<[^>]*>`)

	spacesRe        = regexp.MustCompile(`[ \t]+`)
	spaceAroundNLRe = regexp.MustCompile(`[ \t]*\n[ \t]*`)
	manyNewlinesRe  = regexp.MustCompile(`\n{3,}`)

	// Structural EPUB/editor elements are useful for display but are not
	// manuscript prose. Letting headings and navigation into NER turns chapter
	// titles such as "What Corwin Saw" into character aliases.
	analysisExcludedBlocks = []*regexp.Regexp{
		regexp.MustCompile(`(?is)<h[1-6]\b[^>]*>.*?</h[1-6]>`),
		regexp.MustCompile(`(?is)<nav\b[^>]*>.*?</nav>`),
		regexp.MustCompile(`(?is)<header\b[^>]*>.*?</header>`),
		regexp.MustCompile(`(?is)<footer\b[^>]*>.*?</footer>`),
		regexp.MustCompile(`(?is)<script\b[^>]*>.*?</script>`),
		regexp.MustCompile(`(?is)<style\b[^>]*>.*?</style>`),
	}
)

// StripHTML converts chapter HTML to plain text, preserving paragraph
// boundaries as blank lines and <hr> as a scene-break marker. All pipeline
// stages (mentions, scenes, dialogue) share this coordinate space.
func StripHTML(content string) string {
	text := hrTagRe.ReplaceAllString(content, "\n\n* * *\n\n")
	text = brTagRe.ReplaceAllString(text, "\n")
	text = blockEndRe.ReplaceAllString(text, "\n\n")
	text = anyTagRe.ReplaceAllString(text, "")
	text = html.UnescapeString(text)
	text = spacesRe.ReplaceAllString(text, " ")
	text = spaceAroundNLRe.ReplaceAllString(text, "\n")
	text = manyNewlinesRe.ReplaceAllString(text, "\n\n")
	return strings.TrimSpace(text)
}

// StripHTMLForAnalysis creates the shared coordinate space used by character
// and relationship analysis while excluding display-only structure.
func StripHTMLForAnalysis(content string) string {
	for _, re := range analysisExcludedBlocks {
		content = re.ReplaceAllString(content, " ")
	}
	return StripHTML(content)
}
