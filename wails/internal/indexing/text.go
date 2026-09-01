package indexing

import (
	"html"
	"regexp"
	"strings"

	"draftline/internal/types"
)

// The character pipeline runs entirely in "stripped text" coordinates:
// StripHTML converts chapter HTML to plain text once, and mention offsets,
// scene boundaries, and dialogue ranges all refer to that same string.

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
	// titles such as "What Rael Saw" into character aliases.
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

// ShouldAnalyzeChapter reports whether a section is narrative prose. EPUBs
// often place acknowledgments, copyright, contents, and glossaries in the
// spine beside chapters; people named there are real people, but not cast.
func ShouldAnalyzeChapter(chapter types.ChapterItem) bool {
	typeName := strings.ToLower(strings.TrimSpace(chapter.Type))
	title := strings.ToLower(strings.TrimSpace(chapter.Title))
	for _, excluded := range []string{
		"cover", "title page", "copyright", "dedication", "epigraph",
		"contents", "table of contents", "acknowledgments", "acknowledgements",
		"author's note", "author’s note", "authors note", "about the author",
		"also by", "glossary", "index", "colophon",
	} {
		if typeName == excluded || title == excluded {
			return false
		}
	}
	return true
}

// LooksLikeCommonWord checks if a word has suffixes/patterns typical of
// common English words rather than names (adverbs, gerunds, etc.)
func LooksLikeCommonWord(word string) bool {
	lower := strings.ToLower(word)

	for _, suffix := range CommonWordSuffixes {
		if strings.HasSuffix(lower, suffix) && len(lower) > len(suffix)+2 {
			return true
		}
	}

	// Hyphenated designations ("Sub-Level") are common words when their
	// final segment is a common or place word ("level"). Real hyphenated
	// names ("Mary-Jane") end in name-like segments.
	if idx := strings.LastIndexAny(lower, "-"); idx >= 0 && idx < len(lower)-1 {
		seg := lower[idx+1:]
		if commonWordsLower[seg] || FalsePositiveContextWords[seg] {
			return true
		}
	}

	return TemporalWords[lower]
}

// Attribute extraction: ONE pass over the book with precompiled patterns,
// keyed by the possessive name that owns the attribute ("Kira's blue eyes").
// Never re-scan the full text per character — that made indexing take
// minutes on real manuscripts.
var (
	eyeColorRe  = regexp.MustCompile(`([A-Z][A-Za-z'’-]+)['’]s\s+(?i:(blue|green|brown|hazel|gray|grey|black|amber|violet|golden))\s+eyes`)
	hairColorRe = regexp.MustCompile(`([A-Z][A-Za-z'’-]+)['’]s\s+(?i:(blonde|blond|brunette|brown|black|red|auburn|gray|grey|white|silver|golden|dark|light))\s+hair`)
	ageRe       = regexp.MustCompile(`([A-Z][A-Za-z'’-]+)[^.!?\n]{0,30}?\b(\d{1,2})[\s-]year[\s-]old`)
)

// ExtractAllAttributes scans the text once and returns attributes keyed by
// the lowercase name token that owns them.
func ExtractAllAttributes(text string) map[string]map[string]string {
	result := map[string]map[string]string{}

	set := func(name, key, value string) {
		lower := strings.ToLower(name)
		if result[lower] == nil {
			result[lower] = map[string]string{}
		}
		if result[lower][key] == "" {
			result[lower][key] = strings.ToLower(value)
		}
	}

	for _, m := range eyeColorRe.FindAllStringSubmatch(text, -1) {
		set(m[1], "eye_color", m[2])
	}
	for _, m := range hairColorRe.FindAllStringSubmatch(text, -1) {
		set(m[1], "hair_color", m[2])
	}
	for _, m := range ageRe.FindAllStringSubmatch(text, -1) {
		set(m[1], "age", m[2])
	}

	return result
}

// LookupAttributes merges the attributes owned by any of a character's name
// tokens or aliases.
func LookupAttributes(attrsByName map[string]map[string]string, name string, aliases []string) map[string]string {
	attrs := map[string]string{}
	tokens := strings.Fields(strings.ToLower(name))
	for _, alias := range aliases {
		tokens = append(tokens, strings.Fields(strings.ToLower(alias))...)
	}
	for _, tok := range tokens {
		for k, v := range attrsByName[tok] {
			if attrs[k] == "" {
				attrs[k] = v
			}
		}
	}
	return attrs
}
