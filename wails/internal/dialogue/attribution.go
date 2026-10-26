package dialogue

import (
	"regexp"
	"sort"
	"strings"
)

// Speaker attribution, a port of the Read Aloud attributor
// (frontend/src/services/readaloud/attribution.ts). The frontend copy drives
// live cast-mode playback; this one produces the persisted dialogue layer.
// Keep the two in step: every rule here has a numbered twin over there.

// Verbs that mark a dialogue tag ("...," she said). Lowercase; matched
// case-insensitively against the token next to a name or pronoun.
var tagVerbs = map[string]bool{
	"said": true, "asked": true, "replied": true, "whispered": true,
	"shouted": true, "muttered": true, "murmured": true, "cried": true,
	"called": true, "answered": true, "added": true, "continued": true,
	"snapped": true, "hissed": true, "growled": true, "breathed": true,
	"began": true, "insisted": true, "admitted": true, "agreed": true,
	"demanded": true, "offered": true, "warned": true, "told": true,
	"repeated": true, "echoed": true, "managed": true, "pressed": true,
	"countered": true, "blurted": true, "stammered": true, "sighed": true,
	"laughed": true,
}

// A textblock containing only separator glyphs is a scene break.
var sceneSeparatorRe = regexp.MustCompile(`^[\s*\-–—~•#_.]+$`)

var (
	tagAfterRe       = regexp.MustCompile(`^\s*,?\s*(?:had\s+|would\s+)?([A-Za-z']+)`)
	tagBeforeRe      = regexp.MustCompile(`([A-Za-z']+)\s*$`)
	pronounTagRe     = regexp.MustCompile(`(?i)(?:^|\W)(he|she|they)\s+(?:had\s+|would\s+)?([A-Za-z']+)`)
	leadingPronounRe = regexp.MustCompile(`^(?i:(he|she|they))\b`)
)

const (
	speakerNarrator = "narrator"
	speakerUnknown  = "unknown"
)

// sentence is one segmented sentence of a chapter's plaintext.
type sentence struct {
	text  string
	start int // byte offset of the sentence within the chapter plaintext
	block int // paragraph ordinal within the chapter
}

// ── Phase B: attribution ─────────────────────────────────────────────────────

type pronounIdx int

const (
	pronounHe pronounIdx = iota
	pronounShe
	pronounThey
	pronounNone pronounIdx = -1
)

func pronounOf(word string) pronounIdx {
	switch strings.ToLower(word) {
	case "he":
		return pronounHe
	case "she":
		return pronounShe
	case "they":
		return pronounThey
	}
	return pronounNone
}

type genderEvidence [3]int

// Positive support: the pronoun has been observed for this character and is
// their majority pronoun. (Empty evidence is NOT support — guessing at
// whoever was last named is how a character ends up with the wrong voice.)
func (e *genderEvidence) supports(p pronounIdx) bool {
	if e == nil {
		return false
	}
	count := e[p]
	return count > 0 && count >= e[0] && count >= e[1] && count >= e[2]
}

func (e *genderEvidence) empty() bool {
	return e == nil || e[0]+e[1]+e[2] == 0
}

// Finds "<Name> said" / "said <Name>" in unquoted text; returns the speaker.
func findNamedTag(unquoted string, matchers []matcher) string {
	for _, hit := range findRosterHits(unquoted, matchers) {
		if after := tagAfterRe.FindStringSubmatch(unquoted[hit.index+hit.length:]); after != nil &&
			tagVerbs[strings.ToLower(after[1])] {
			return hit.key
		}
		if before := tagBeforeRe.FindStringSubmatch(unquoted[:hit.index]); before != nil &&
			tagVerbs[strings.ToLower(before[1])] {
			return hit.key
		}
	}
	return ""
}

func findPronounTag(unquoted string) pronounIdx {
	if match := pronounTagRe.FindStringSubmatch(unquoted); match != nil &&
		tagVerbs[strings.ToLower(match[2])] {
		return pronounOf(match[1])
	}
	return pronounNone
}

func findLeadingPronoun(text string) pronounIdx {
	if match := leadingPronounRe.FindStringSubmatch(strings.TrimSpace(text)); match != nil {
		return pronounOf(match[1])
	}
	return pronounNone
}

// RosterEntry is one attributable character. Key is the canonical speaker
// key (the lowercased name); aliases are matched case-sensitively as typed.
type RosterEntry struct {
	Key     string
	Name    string
	Aliases []string
}

type attributedSentence struct {
	speaker    string // narrator | unknown | roster key
	kind       string // narration | dialogue | mixed
	confidence string // tag | adjacent-tag | pretag | continuation | nearby | alternation | none
}

func attributeSpeakers(sentences []sentence, roster []RosterEntry) ([]attributedSentence, []quoteScan) {
	scans := scanQuotes(sentences)
	matchers := buildMatchers(roster)

	out := make([]attributedSentence, 0, len(sentences))
	evidence := map[string]*genderEvidence{}

	// Scene-scoped context; reset at separator blocks.
	lastSpeaker := ""
	prevParaSpeaker := ""       // last CHARACTER dialogue speaker of the previous paragraph
	paraCharSpeaker := ""       // last character dialogue speaker within THIS paragraph
	var recentSpeakers []string // last two DISTINCT dialogue speakers, most recent first
	lastNamedKey := ""
	lastNamedBlock := -1
	sceneSpeakers := map[string]bool{}

	resetScene := func() {
		lastSpeaker = ""
		prevParaSpeaker = ""
		paraCharSpeaker = ""
		recentSpeakers = nil
		lastNamedKey = ""
		lastNamedBlock = -1
		sceneSpeakers = map[string]bool{}
	}

	bumpEvidence := func(key string, p pronounIdx) {
		e := evidence[key]
		if e == nil {
			e = &genderEvidence{}
			evidence[key] = e
		}
		e[p]++
	}

	// Pre-pass: harvest gender evidence from narration co-reference — a solo
	// character mention followed (within a paragraph) by a He/She/They-initial
	// narration sentence. Gives pronoun tags something solid to resolve
	// against from the first page.
	{
		soloKey := ""
		soloBlock := -1
		for i, sent := range sentences {
			if sceneSeparatorRe.MatchString(sent.text) {
				soloKey = ""
				continue
			}
			if scans[i].kind != "narration" {
				continue
			}
			trimmed := strings.TrimSpace(sent.text)
			if lead := findLeadingPronoun(trimmed); lead != pronounNone && soloKey != "" &&
				sent.block-soloBlock <= 1 && startsUpper(trimmed) {
				bumpEvidence(soloKey, lead)
			}
			hits := findRosterHits(sent.text, matchers)
			if len(hits) == 1 {
				soloKey, soloBlock = hits[0].key, sent.block
			} else if len(hits) > 1 {
				soloKey = ""
			}
		}
	}

	// A dialogue tag is often its own segmented sentence between quotations:
	// treat that narration sentence as an attribution anchor for adjacent
	// spans in the same paragraph.
	adjacentSpanSpeakers := map[int]string{}
	conflictedSpans := map[int]bool{}
	anchorSpan := func(spanID int, speaker string) {
		if spanID < 0 || conflictedSpans[spanID] {
			return
		}
		if existing, ok := adjacentSpanSpeakers[spanID]; ok && existing != speaker {
			delete(adjacentSpanSpeakers, spanID)
			conflictedSpans[spanID] = true
			return
		}
		adjacentSpanSpeakers[spanID] = speaker
	}
	resolveStandalonePronoun := func(index int, p pronounIdx) string {
		nearestWithoutEvidence := ""
		block := sentences[index].block
		for j := index - 1; j >= 0; j-- {
			if block-sentences[j].block > 3 {
				break
			}
			if scans[j].kind != "narration" {
				continue
			}
			hits := findRosterHits(sentences[j].text, matchers)
			for h := len(hits) - 1; h >= 0; h-- {
				key := hits[h].key
				if evidence[key].supports(p) {
					return key
				}
				if nearestWithoutEvidence == "" && evidence[key].empty() {
					nearestWithoutEvidence = key
				}
			}
		}
		type supported struct {
			key   string
			count int
		}
		var all []supported
		for key, e := range evidence {
			if e.supports(p) {
				all = append(all, supported{key, e[p]})
			}
		}
		sort.Slice(all, func(a, b int) bool {
			if all[a].count != all[b].count {
				return all[a].count > all[b].count
			}
			return all[a].key < all[b].key
		})
		if len(all) == 1 || (len(all) > 1 && all[0].count > all[1].count) {
			return all[0].key
		}
		return nearestWithoutEvidence
	}
	for i := range sentences {
		if scans[i].kind != "narration" {
			continue
		}
		sameBlock := func(index int) bool {
			return index >= 0 && index < len(sentences) && sentences[index].block == sentences[i].block
		}
		previous, next := -1, -1
		if sameBlock(i-1) && scans[i-1].kind != "narration" {
			previous = scans[i-1].spanID
		}
		if sameBlock(i+1) && scans[i+1].kind != "narration" {
			next = scans[i+1].spanID
		}
		if previous < 0 && next < 0 {
			continue
		}

		speaker := findNamedTag(sentences[i].text, matchers)
		p := pronounNone
		if speaker == "" {
			if p = findPronounTag(sentences[i].text); p != pronounNone {
				speaker = resolveStandalonePronoun(i, p)
			}
		}
		if speaker == "" {
			continue
		}
		anchorSpan(previous, speaker)
		anchorSpan(next, speaker)
		if p != pronounNone {
			bumpEvidence(speaker, p)
		}
	}

	prevBlock := -1

	for i, sent := range sentences {
		scan := scans[i]
		newParagraph := i == 0 || sent.block != prevBlock

		if newParagraph {
			if paraCharSpeaker != "" {
				prevParaSpeaker = paraCharSpeaker
			}
			paraCharSpeaker = ""
		}
		prevBlock = sent.block

		if sceneSeparatorRe.MatchString(sent.text) {
			resetScene()
			out = append(out, attributedSentence{speaker: speakerNarrator, kind: "narration", confidence: "none"})
			continue
		}

		if scan.kind == "narration" {
			if hits := findRosterHits(sent.text, matchers); len(hits) > 0 {
				lastNamedKey, lastNamedBlock = hits[len(hits)-1].key, sent.block
			}
			out = append(out, attributedSentence{speaker: speakerNarrator, kind: "narration", confidence: "none"})
			continue
		}

		// Dialogue or mixed — rules in order, first hit wins.
		speaker := ""
		confidence := "none"
		p := findPronounTag(scan.unquoted)

		// 0. A standalone narration tag immediately beside this quote is more
		// explicit than all continuity and alternation guesses below.
		if scan.spanID >= 0 {
			if adjacent, ok := adjacentSpanSpeakers[scan.spanID]; ok {
				speaker = adjacent
				confidence = "adjacent-tag"
			}
		}

		// 1a/1b. Same-sentence named tag.
		if speaker == "" {
			if named := findNamedTag(scan.unquoted, matchers); named != "" {
				speaker = named
				confidence = "tag"
			}
		}

		// 1c. Same-sentence pronoun tag ("he said"): resolve against POSITIVE
		//     gender evidence first — the fresh narration name, then recent
		//     speakers. A no-evidence fallback to the last-named character
		//     runs only when nobody in the scene matches the pronoun.
		if speaker == "" && p != pronounNone {
			// A narration name only anchors a pronoun while it's fresh — this
			// paragraph or the one before.
			fresh := lastNamedKey != "" && sent.block-lastNamedBlock <= 1
			if fresh && evidence[lastNamedKey].supports(p) {
				speaker = lastNamedKey
				confidence = "tag"
			} else {
				for _, key := range recentSpeakers {
					if key != speakerUnknown && evidence[key].supports(p) {
						speaker = key
						confidence = "tag"
						break
					}
				}
				if speaker == "" {
					// Anyone in the ledger with supporting evidence beats a
					// no-evidence guess.
					best := ""
					bestCount := 0
					keys := make([]string, 0, len(evidence))
					for key := range evidence {
						keys = append(keys, key)
					}
					sort.Strings(keys)
					for _, key := range keys {
						if e := evidence[key]; e.supports(p) && e[p] > bestCount {
							best = key
							bestCount = e[p]
						}
					}
					if best != "" {
						speaker = best
						confidence = "tag"
					} else if fresh && evidence[lastNamedKey].empty() {
						// Cold start: truly no one has evidence yet — the
						// fresh name is the best available guess.
						speaker = lastNamedKey
						confidence = "tag"
					}
				}
			}
		}

		// 2. Pre-quote tag / action beat: name in the unquoted text before
		//    the quote opens, or in the immediately preceding narration
		//    sentence of the same paragraph.
		if speaker == "" && scan.quoteOpensHere && !scan.startsInsideQuote {
			beforeHits := findRosterHits(scan.unquotedBeforeQuote, matchers)
			if len(beforeHits) > 0 {
				speaker = beforeHits[len(beforeHits)-1].key
				confidence = "pretag"
			} else if i > 0 && scans[i-1].kind == "narration" && sentences[i-1].block == sent.block {
				prevHits := findRosterHits(sentences[i-1].text, matchers)
				if len(prevHits) == 1 {
					speaker = prevHits[0].key
					confidence = "pretag"
				} else if len(prevHits) == 0 {
					if lead := findLeadingPronoun(sentences[i-1].text); lead != pronounNone {
						if resolved := resolveStandalonePronoun(i-1, lead); resolved != "" {
							speaker = resolved
							confidence = "pretag"
						}
					}
				}
			}
		}

		// 3. Quote continuation: the segmenter split inside the quote, or the
		//    speech continued across a paragraph boundary.
		if speaker == "" && scan.startsInsideQuote && lastSpeaker != "" {
			speaker = lastSpeaker
			confidence = "continuation"
		}

		// 3b. One paragraph, one speaker: further dialogue in a paragraph
		//     whose earlier dialogue already resolved continues that
		//     character.
		if speaker == "" && paraCharSpeaker != "" {
			speaker = paraCharSpeaker
			confidence = "continuation"
		}

		// 4. Nearby mention within this paragraph. A previous paragraph is
		//    held until after turn-taking.
		if speaker == "" {
			for j := i - 1; j >= 0 && sentences[j].block == sent.block; j-- {
				if scans[j].kind != "narration" {
					continue
				}
				hits := findRosterHits(sentences[j].text, matchers)
				if len(hits) == 1 {
					speaker = hits[0].key
					confidence = "nearby"
					break
				}
				if len(hits) > 1 {
					break
				}
			}
		}

		// 5. Alternation: an untagged fresh dialogue paragraph answers the
		//    previous paragraph's speaker with the other of the two most
		//    recent speakers.
		if speaker == "" && newParagraph && len(recentSpeakers) == 2 &&
			prevParaSpeaker != "" && prevParaSpeaker == recentSpeakers[0] {
			speaker = recentSpeakers[1]
			confidence = "alternation"
		}

		// 5b. Monologue continuation: with exactly one speaker active in the
		//     scene, an untagged dialogue paragraph continues that speaker.
		if speaker == "" && newParagraph && len(sceneSpeakers) == 1 &&
			prevParaSpeaker != "" && sceneSpeakers[prevParaSpeaker] {
			speaker = prevParaSpeaker
			confidence = "alternation"
		}

		// 5c. Previous-paragraph mention is a last-resort local hint after
		//     stronger conversational continuity has had its chance.
		if speaker == "" {
			scope := sent.block - 1
			for j := i - 1; j >= 0; j-- {
				if sentences[j].block != scope {
					continue
				}
				if scans[j].kind != "narration" {
					continue
				}
				hits := findRosterHits(sentences[j].text, matchers)
				if len(hits) == 1 {
					speaker = hits[0].key
					confidence = "nearby"
					break
				}
				if len(hits) > 1 {
					break
				}
			}
		}

		// 6. Unknown.
		if speaker == "" {
			speaker = speakerUnknown
			confidence = "none"
		}

		// A pronoun tag observed alongside a confident resolution feeds the
		// gender ledger (guessed continuations/alternations don't).
		if p != pronounNone && speaker != speakerUnknown &&
			(confidence == "tag" || confidence == "adjacent-tag" || confidence == "pretag" || confidence == "nearby") {
			bumpEvidence(speaker, p)
		}

		// Tagged sentences name people in their unquoted parts too —
		// remember them for later pronoun tags.
		if scan.unquoted != "" {
			if hits := findRosterHits(scan.unquoted, matchers); len(hits) > 0 {
				lastNamedKey, lastNamedBlock = hits[len(hits)-1].key, sent.block
			}
		}

		out = append(out, attributedSentence{speaker: speaker, kind: scan.kind, confidence: confidence})

		lastSpeaker = speaker
		if speaker != speakerUnknown && speaker != speakerNarrator {
			paraCharSpeaker = speaker
			sceneSpeakers[speaker] = true
			if len(recentSpeakers) == 0 || recentSpeakers[0] != speaker {
				next := []string{speaker}
				for _, key := range recentSpeakers {
					if key != speaker {
						next = append(next, key)
					}
				}
				if len(next) > 2 {
					next = next[:2]
				}
				recentSpeakers = next
			}
		}
	}

	// Backfill: a tag anywhere in a quote span names the whole span.
	spanSpeaker := map[int]string{}
	for i := range out {
		spanID := scans[i].spanID
		if spanID >= 0 && out[i].speaker != speakerUnknown && out[i].speaker != speakerNarrator {
			if _, ok := spanSpeaker[spanID]; !ok {
				spanSpeaker[spanID] = out[i].speaker
			}
		}
	}
	for i := range out {
		spanID := scans[i].spanID
		if spanID < 0 || out[i].speaker != speakerUnknown {
			continue
		}
		if speaker, ok := spanSpeaker[spanID]; ok {
			out[i].speaker = speaker
			out[i].confidence = "continuation"
		}
	}

	// One paragraph, one speaker (post-pass): when a paragraph's dialogue
	// resolved to exactly one character, its remaining unknown dialogue joins
	// that character.
	blockStart := 0
	for blockStart < len(out) {
		block := sentences[blockStart].block
		blockEnd := blockStart
		for blockEnd < len(out) && sentences[blockEnd].block == block {
			blockEnd++
		}
		blockSpeakers := map[string]bool{}
		for j := blockStart; j < blockEnd; j++ {
			s := out[j].speaker
			if out[j].kind != "narration" && s != speakerUnknown && s != speakerNarrator {
				blockSpeakers[s] = true
			}
		}
		if len(blockSpeakers) == 1 {
			only := ""
			for key := range blockSpeakers {
				only = key
			}
			for j := blockStart; j < blockEnd; j++ {
				if out[j].kind != "narration" && out[j].speaker == speakerUnknown {
					out[j].speaker = only
					out[j].confidence = "continuation"
				}
			}
		}
		blockStart = blockEnd
	}

	return out, scans
}

func startsUpper(text string) bool {
	return text != "" && text[0] >= 'A' && text[0] <= 'Z'
}
