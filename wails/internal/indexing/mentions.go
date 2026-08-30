package indexing

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"draftline/internal/entityresolution"
)

// Phase 1 of the character pipeline: mention extraction.
//
// A single greedy scan over stripped chapter text produces immutable mention
// spans. A run of adjacent capitalized name words is ONE mention, never two.
// Leading honorifics (Dr., Det.) stay inside the raw span — entity resolution
// strips them into metadata. Trailing credentials (CPD) never enter a run
// because all-caps tokens are not name words and punctuation breaks runs.

// wordSpan is a scanned token with its position in the source text.
type wordSpan struct {
	raw        string // token as it appears, e.g. "Kira's"
	base       string // token minus possessive/contraction suffix, e.g. "Kira"
	start      int    // byte offset of token start
	baseEnd    int    // byte offset just past base
	rawEnd     int    // byte offset just past raw token
	sentence   int    // sentence index within the chapter
	possessive bool   // token carried a possessive 's suffix
}

// contractionSuffixes are apostrophe suffixes stripped from tokens before
// name matching. A possessive ("Kira's") yields the bare name; a contraction
// ("I'm") yields a base that fails the name-word test.
var contractionSuffixes = []string{"'s", "'m", "'d", "'t", "'ll", "'ve", "'re",
	"’s", "’m", "’d", "’t", "’ll", "’ve", "’re"}

// mentionCandidate is a mention plus the positional evidence used to filter
// sentence-lead false positives ("Traffic," Marcus said / "Resolute Marcus
// strode in").
type mentionCandidate struct {
	mention      entityresolution.Mention
	nlpPerson    bool
	nlpNonPerson bool
	strongPerson bool
	tokens       []string // lowercase name tokens
	quoteInitial bool     // span starts immediately after an opening quote
	firstInitial bool     // first name token opens its sentence or quote
	honorific    bool     // span carries an honorific prefix
	fpKilled     bool     // place/institution indicator — not a character
	// Precomputed reduced span with the first name token dropped, used when
	// an unattested sentence-lead token must be stripped.
	altText     string
	altOffset   int
	altFiltered bool // the reduced span is itself a common word
}

// ExtractMentions scans stripped chapter text and returns all name mentions
// in document order with deterministic IDs and exact byte offsets.
// Corroboration is chapter-local; IndexBook uses ExtractBookMentions for
// book-wide corroboration.
func ExtractMentions(text string, chapter int) []entityresolution.Mention {
	candidates, attested := scanChapter(text, chapter)
	knowledge := collectCandidateKnowledge([][]mentionCandidate{candidates})
	removeNonPersonAttestation(attested, knowledge)
	return filterCandidates(candidates, attested, knowledge)
}

// ExtractBookMentions extracts mentions for every chapter with BOOK-WIDE
// corroboration: a name attested anywhere validates its sentence-initial
// uses everywhere.
func ExtractBookMentions(chapterTexts []string) []entityresolution.Mention {
	allCandidates := make([][]mentionCandidate, len(chapterTexts))
	attested := map[string]bool{}
	linguisticByChapter := analyzeBookLinguisticEvidence(chapterTexts)

	for ch, text := range chapterTexts {
		candidates, chapterAttested := scanChapterWithEvidence(text, ch, linguisticByChapter[ch])
		allCandidates[ch] = candidates
		for tok := range chapterAttested {
			attested[tok] = true
		}
	}

	knowledge := collectCandidateKnowledge(allCandidates)
	removeNonPersonAttestation(attested, knowledge)
	mentions := []entityresolution.Mention{}
	for _, candidates := range allCandidates {
		mentions = append(mentions, filterCandidates(candidates, attested, knowledge)...)
	}
	return mentions
}

// scanChapter produces raw mention candidates plus the set of attested
// tokens: names seen mid-sentence, used as possessives, or belonging to a
// multi-token or honorific-prefixed name.
func scanChapter(text string, chapter int) ([]mentionCandidate, map[string]bool) {
	return scanChapterWithEvidence(text, chapter, analyzeLinguisticEvidence(text))
}

func scanChapterWithEvidence(text string, chapter int, linguistic linguisticEvidence) ([]mentionCandidate, map[string]bool) {
	words := scanWords(text)
	candidates := []mentionCandidate{}

	attested := map[string]bool{}
	for i, w := range words {
		if !isNameWord(w.base) {
			continue
		}
		if w.possessive || !wordIsInitial(text, words, i) {
			attested[strings.ToLower(w.base)] = true
		}
	}

	i := 0
	for i < len(words) {
		// Runs start on a name word, or a CAPITALIZED honorific
		// ("the general" never starts a name).
		if !isNameWord(words[i].base) && !(isHonorificWord(words[i].base) && startsUpper(words[i].base)) {
			i++
			continue
		}
		// An honorific only starts a run if a name word can follow it.
		if isHonorificWord(words[i].base) && !isNameWord(words[i].base) {
			if i+1 >= len(words) || !isNameWord(words[i+1].base) || !joinable(text, words[i], words[i+1]) {
				i++
				continue
			}
		}

		// Greedily extend the run over joinable name words.
		j := i
		for j+1 < len(words) && isNameWord(words[j+1].base) && joinable(text, words[j], words[j+1]) {
			j++
		}

		if c, ok := buildMention(text, words[i:j+1], chapter, len(candidates), wordIsInitial(text, words, i), linguistic); ok {
			candidates = append(candidates, c)
		}
		i = j + 1
	}

	// Tokens of accepted multi-token and honorific-prefixed names attest
	// their components ("Marcus Webb" attests bare "Webb" openers) — except
	// the LEAD token of a sentence-initial span, which cannot vouch for
	// itself ("Resolute Marcus strode" must not attest "resolute").
	for _, c := range candidates {
		if c.fpKilled {
			continue
		}
		if len(c.tokens) >= 2 || c.honorific {
			for k, tok := range c.tokens {
				if k == 0 && c.firstInitial && !c.honorific {
					continue
				}
				attested[tok] = true
			}
		}
	}

	return candidates, attested
}

// wordIsInitial reports whether words[i] opens a sentence or a quote.
func wordIsInitial(text string, words []wordSpan, i int) bool {
	if i == 0 || words[i-1].sentence != words[i].sentence {
		return true
	}
	return isQuoteInitial(text, words[i].start)
}

// filterCandidates applies corroboration evidence to raw candidates:
//
//   - A multi-token span whose FIRST token opens a sentence/quote and is
//     never attested loses that token ("Resolute Marcus strode" → "Marcus").
//   - A single-token span that opens a sentence or quote survives ONLY if
//     the name is attested somewhere ("Thunder rolled." is weather;
//     "Mirela watched." is a person if Mirela appears mid-sentence, as a
//     possessive, or in a full name anywhere in the book).
//
// Honorific-prefixed spans are exempt — a title is strong name evidence.
//
// Tokens from place/institution runs ("Hubbard Street", "Chicago Police
// Department") are blacklisted: a bare "Hubbard" is the street, not a
// person — even mid-sentence — unless the token also appears in a real
// multi-token or honorific-prefixed name.
func filterCandidates(candidates []mentionCandidate, attested map[string]bool, knowledge candidateKnowledge) []entityresolution.Mention {
	mentions := []entityresolution.Mention{}

	for _, c := range candidates {
		if c.fpKilled {
			continue
		}
		if c.nlpNonPerson && !knowledge.personNames[strings.Join(c.tokens, " ")] {
			continue
		}

		single := len(c.tokens) == 1
		initial := c.firstInitial || c.quoteInitial

		if !c.honorific && len(c.tokens) >= 2 && c.firstInitial && !attested[c.tokens[0]] {
			// Strip the unattested sentence-lead token.
			if c.altFiltered {
				continue
			}
			c.mention.Text = c.altText
			c.mention.CharOffset = c.altOffset
			c.tokens = c.tokens[1:]
			single = len(c.tokens) == 1
			initial = false // the remainder followed another word
		}

		if single {
			tok := c.tokens[len(c.tokens)-1]
			if knowledge.hardBlacklist[tok] && !knowledge.strongPersonTokens[tok] {
				continue
			}
			if knowledge.nlpBlacklist[tok] && !knowledge.personTokens[tok] {
				continue
			}
			if !c.honorific && initial && !attested[tok] {
				continue
			}
		}

		mentions = append(mentions, c.mention)
	}
	return mentions
}

type candidateKnowledge struct {
	hardBlacklist      map[string]bool
	nlpBlacklist       map[string]bool
	personTokens       map[string]bool
	strongPersonTokens map[string]bool
	personNames        map[string]bool
}

// collectCandidateKnowledge is deliberately book-wide. A location identified
// as "Hubbard Street" in one chapter must stop an unqualified "Hubbard" in a
// later chapter from becoming a character. Strong person evidence can still
// override the collision when a story genuinely has both a place and person
// sharing a token.
func collectCandidateKnowledge(groups [][]mentionCandidate) candidateKnowledge {
	knowledge := candidateKnowledge{
		hardBlacklist:      map[string]bool{},
		nlpBlacklist:       map[string]bool{},
		personTokens:       map[string]bool{},
		strongPersonTokens: map[string]bool{},
		personNames:        map[string]bool{},
	}
	for _, candidates := range groups {
		for _, candidate := range candidates {
			if candidate.fpKilled {
				for _, token := range candidate.tokens {
					knowledge.hardBlacklist[token] = true
				}
			}
			if candidate.nlpNonPerson {
				for _, token := range candidate.tokens {
					knowledge.nlpBlacklist[token] = true
				}
			}
			if candidate.nlpPerson || candidate.strongPerson {
				knowledge.personNames[strings.Join(candidate.tokens, " ")] = true
				for _, token := range candidate.tokens {
					knowledge.personTokens[token] = true
				}
			}
			if candidate.strongPerson {
				for _, token := range candidate.tokens {
					knowledge.strongPersonTokens[token] = true
				}
			}
		}
	}
	return knowledge
}

func removeNonPersonAttestation(attested map[string]bool, knowledge candidateKnowledge) {
	for token := range attested {
		if knowledge.hardBlacklist[token] && !knowledge.strongPersonTokens[token] {
			delete(attested, token)
			continue
		}
		if knowledge.nlpBlacklist[token] && !knowledge.personTokens[token] {
			delete(attested, token)
		}
	}
}

// isQuoteInitial reports whether the span at offset directly follows an
// OPENING quote character (ignoring whitespace). Directional quotes are
// unambiguous; a straight " counts as opening only when it is itself
// preceded by whitespace — `," Marcus` is a closing quote.
// Scans backwards in O(1); never materializes the text prefix.
func isQuoteInitial(text string, offset int) bool {
	i := offset
	for i > 0 {
		r, size := utf8.DecodeLastRuneInString(text[:i])
		if r == ' ' || r == '\t' {
			i -= size
			continue
		}
		if r == '“' || r == '«' || r == '‘' {
			return true
		}
		if r == '"' || r == '\'' {
			if i-size == 0 {
				return true
			}
			prev, _ := utf8.DecodeLastRuneInString(text[:i-size])
			return unicode.IsSpace(prev)
		}
		return false
	}
	return false
}

// scanWords tokenizes text into letter runs (allowing internal apostrophes
// and hyphens), tracking byte offsets and sentence indexes.
func scanWords(text string) []wordSpan {
	words := []wordSpan{}
	sentence := 0
	pendingSentence := false
	lastWasHonorific := false

	start := -1
	flush := func(end int) {
		if start < 0 {
			return
		}
		raw := text[start:end]
		base, baseEnd := stripContraction(raw, start)
		if base != "" {
			if pendingSentence {
				sentence++
				pendingSentence = false
			}
			suffix := raw[len(base):]
			words = append(words, wordSpan{
				raw:        raw,
				base:       base,
				start:      start,
				baseEnd:    baseEnd,
				rawEnd:     end,
				sentence:   sentence,
				possessive: strings.EqualFold(suffix, "'s") || strings.EqualFold(suffix, "’s"),
			})
			// Only a capitalized honorific ABBREVIATION suppresses the
			// following period as a sentence boundary ("Dr. Chen"). A
			// full-word honorific ending a sentence ("...the general.")
			// is a real boundary.
			lastWasHonorific = entityresolution.IsHonorificAbbrev(base) && startsUpper(base)
		}
		start = -1
	}

	for idx, r := range text {
		isWordRune := unicode.IsLetter(r) || (start >= 0 && (r == '\'' || r == '’' || r == '-'))
		if isWordRune {
			if start < 0 {
				start = idx
			}
			continue
		}
		flush(idx)
		if r == '!' || r == '?' || (r == '.' && !lastWasHonorific) {
			pendingSentence = true
		}
	}
	flush(len(text))

	return words
}

// startsUpper reports whether a word begins with an uppercase letter.
func startsUpper(word string) bool {
	for _, r := range word {
		return unicode.IsUpper(r)
	}
	return false
}

// stripContraction removes a trailing possessive/contraction suffix and any
// trailing apostrophes or hyphens. Returns the base and its end offset.
func stripContraction(raw string, start int) (string, int) {
	base := raw
	for _, suffix := range contractionSuffixes {
		if len(base) > len(suffix) && strings.EqualFold(base[len(base)-len(suffix):], suffix) {
			base = base[:len(base)-len(suffix)]
			break
		}
	}
	base = strings.TrimRight(base, "'’-")
	return base, start + len(base)
}

// isNameWord reports whether a token base looks like part of a proper name:
// starts uppercase, contains a lowercase letter (rejects all-caps credentials
// like CPD), and consists only of letters, apostrophes, and hyphens.
func isNameWord(base string) bool {
	runes := []rune(base)
	if len(runes) < 2 {
		return false
	}
	if !unicode.IsUpper(runes[0]) {
		return false
	}
	hasLower := false
	for _, r := range runes[1:] {
		if !unicode.IsLetter(r) && r != '\'' && r != '’' && r != '-' {
			return false
		}
		if unicode.IsLower(r) {
			hasLower = true
		}
	}
	return hasLower
}

// isHonorificWord reports whether a token is a known honorific/title.
func isHonorificWord(base string) bool {
	return entityresolution.DefaultHonorifics[strings.ToLower(base)]
}

// joinable reports whether word b continues a name run started by word a.
// The gap must be plain spaces, or a period plus spaces when a is a
// capitalized honorific ABBREVIATION ("Dr. Jane" — but never "general. Dr",
// where the period is a sentence boundary). A possessive ends the run
// ("Kira's Bob" is not a name), as does a paragraph break.
func joinable(text string, a, b wordSpan) bool {
	if a.baseEnd != a.rawEnd && !isHonorificWord(a.base) {
		return false // possessive/contraction suffix ends the run
	}
	gap := text[a.rawEnd:b.start]
	if strings.Count(gap, "\n") > 0 {
		return false
	}
	trimmed := strings.TrimLeft(gap, " \t")
	if trimmed == "" {
		return gap != ""
	}
	// Allow "Dr. Jane" — a single period directly after an honorific abbreviation.
	if trimmed[0] == '.' && gap[0] == '.' && entityresolution.IsHonorificAbbrev(a.base) && startsUpper(a.base) {
		rest := strings.TrimLeft(trimmed[1:], " \t")
		return rest == ""
	}
	return false
}

// buildMention filters and assembles a mention candidate from a run of name
// words. Returns false if the run is not a plausible character reference.
func buildMention(text string, run []wordSpan, chapter, seq int, runInitial bool, linguistic linguisticEvidence) (mentionCandidate, bool) {
	// Leading honorifics stay in the span but don't count as name tokens.
	spanStart := run[0].start
	nameWords := run
	for len(nameWords) > 0 && isHonorificWord(nameWords[0].base) {
		nameWords = nameWords[1:]
	}
	hasHonorific := len(nameWords) < len(run)
	fullRun := nameWords // pre-trim run, for false-positive context checks
	if len(fullRun) == 0 {
		return mentionCandidate{}, false
	}

	// Drop sentence-lead common words ("Suddenly Marcus" → "Marcus") and
	// trailing ones ("Gary Mason He" from a missing period → "Gary Mason").
	droppedCommon := false
	for len(nameWords) > 1 && isFilteredWord(nameWords[0].base) {
		nameWords = nameWords[1:]
		droppedCommon = true
	}
	for len(nameWords) > 1 && isFilteredWord(nameWords[len(nameWords)-1].base) {
		nameWords = nameWords[:len(nameWords)-1]
	}
	if len(nameWords) == 0 {
		return mentionCandidate{}, false
	}
	if droppedCommon {
		spanStart = nameWords[0].start
	}

	// Single-token names must not be common English words.
	if len(nameWords) == 1 && isFilteredWord(nameWords[0].base) {
		return mentionCandidate{}, false
	}

	// A place/food/brand/institution indicator ANYWHERE in the original run
	// (before common-word trimming — "Chicago Police Department" must not
	// degrade to "Chicago") or right after it marks a non-character.
	// Flagged rather than dropped so its tokens can blacklist bare
	// references elsewhere in the chapter ("Hubbard was silent").
	fpKilled := false
	for _, w := range fullRun {
		if FalsePositiveContextWords[strings.ToLower(w.base)] {
			fpKilled = true
		}
	}
	if IsStreetDesignator(fullRun[len(fullRun)-1].base) {
		fpKilled = true
	}
	if IsFalsePositiveContext(text, fullRun[len(fullRun)-1].baseEnd) {
		fpKilled = true
	}

	last := nameWords[len(nameWords)-1]
	nlpPerson, nlpNonPerson := linguistic.classification(spanStart, last.baseEnd)
	if IsAddressIntersectionContext(text, spanStart, last.baseEnd) {
		fpKilled = true
	}
	strongPerson := hasHonorific || last.possessive || (nlpPerson && len(nameWords) >= 2)

	commonCount := 0
	tokens := make([]string, 0, len(nameWords))
	for _, w := range nameWords {
		if isFilteredWord(w.base) {
			commonCount++
		}
		tokens = append(tokens, strings.ToLower(w.base))
	}
	if commonCount == len(nameWords) {
		return mentionCandidate{}, false
	}
	if fpKilled {
		// Blacklist every token of the original run, not just the kept ones.
		tokens = tokens[:0]
		for _, w := range fullRun {
			tokens = append(tokens, strings.ToLower(w.base))
		}
	}

	c := mentionCandidate{
		mention: entityresolution.Mention{
			ID:         fmt.Sprintf("m-%d-%d", chapter, seq),
			Text:       text[spanStart:last.baseEnd],
			SentenceID: fmt.Sprintf("ch%d-s%d", chapter, run[0].sentence),
			Chapter:    chapter,
			CharOffset: spanStart,
		},
		nlpPerson:    nlpPerson,
		nlpNonPerson: nlpNonPerson,
		strongPerson: strongPerson,
		tokens:       tokens,
		quoteInitial: isQuoteInitial(text, spanStart),
		firstInitial: runInitial && !hasHonorific && !droppedCommon,
		honorific:    hasHonorific,
		fpKilled:     fpKilled,
	}
	if len(nameWords) >= 2 {
		c.altText = text[nameWords[1].start:last.baseEnd]
		c.altOffset = nameWords[1].start
		c.altFiltered = len(nameWords) == 2 && isFilteredWord(nameWords[1].base)
	}
	return c, true
}

// isFilteredWord reports whether a word is too common to be a name on its own.
func isFilteredWord(word string) bool {
	return IsCommonWord(word) || LooksLikeCommonWord(word)
}
