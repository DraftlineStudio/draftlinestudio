package fingerprint

// Typed frame extraction (v5). The doctrine, enforced here and by tests:
//
//   - Only the constrained frame vocabulary in types (Frame*) is emitted;
//     there is no freeform subject/predicate/object generation.
//   - Every Detail payload is a VERBATIM slice of the evidence sentence,
//     obtained exclusively through index-based slicing — the extractor can
//     select source text but can never compose it.
//   - The subject slot fills only from a canonical entity alias anchored at
//     the start of the sentence (or its possessive); secondary slots fill
//     only from canonical aliases or named-entity terms found in the text.
//   - When a slot cannot be filled confidently the extractor abstains: the
//     slot stays empty and the abstention is recorded, or no frame is
//     emitted at all. Unknown is acceptable; guessing is not.

import (
	"regexp"
	"sort"
	"strings"

	"draftline/internal/types"
)

// maxDetailWords caps how much of a sentence a single Detail slice may carry
// so a frame never swallows a whole compound sentence.
const maxDetailWords = 25

// ── canonical roster ─────────────────────────────────────────────────────────

type aliasEntry struct {
	alias     string
	canonical string
}

type rosterMatcher struct {
	entries []aliasEntry
}

// buildRoster collects canonical character names and aliases. Recall-first:
// review-status detections stay in (a name the extractor cannot see corrupts
// other frames); explicit rejections stay out. Keys are canonical names —
// entity IDs churn across re-indexing.
func buildRoster(book *types.BookData) *rosterMatcher {
	seen := map[string]bool{}
	entries := []aliasEntry{}
	add := func(alias, canonical string) {
		alias = strings.TrimSpace(alias)
		if len(alias) < 2 || seen[alias+"\x00"+canonical] {
			return
		}
		seen[alias+"\x00"+canonical] = true
		entries = append(entries, aliasEntry{alias: alias, canonical: canonical})
	}
	if book.Analysis.EntityResolution != nil {
		for _, entity := range book.Analysis.EntityResolution.Entities {
			if entity.DetectionStatus == "rejected" || entity.Canonical == "" {
				continue
			}
			add(entity.Canonical, entity.Canonical)
			for _, alias := range entity.Aliases {
				add(alias, entity.Canonical)
			}
		}
	}
	for _, character := range book.StoryBible.Characters {
		if character.Name == "" {
			continue
		}
		add(character.Name, character.Name)
		for _, alias := range character.Aliases {
			add(alias, character.Name)
		}
	}
	sort.SliceStable(entries, func(i, j int) bool { return len(entries[i].alias) > len(entries[j].alias) })
	return &rosterMatcher{entries: entries}
}

func wordBoundary(text string, start, end int) bool {
	if start > 0 {
		prev := text[start-1]
		if isWordByte(prev) {
			return false
		}
	}
	if end < len(text) && isWordByte(text[end]) {
		return false
	}
	return true
}

func isWordByte(b byte) bool {
	return b == '_' || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}

// subjectAt anchors the subject at the start of the sentence: an alias at
// position zero (optionally after an opening quote/dash) or its possessive.
func (r *rosterMatcher) subjectAt(text string) (canonical string, rest string, ok bool) {
	offset := 0
	for offset < len(text) && strings.ContainsRune("“\"'‘—- ", rune(text[offset])) {
		offset++
	}
	body := text[offset:]
	for _, entry := range r.entries {
		if !strings.HasPrefix(body, entry.alias) {
			continue
		}
		end := len(entry.alias)
		if end < len(body) && isWordByte(body[end]) {
			continue
		}
		rest := body[end:]
		// Possessive subject ("Ruiz's hands were shaking") is the same anchor.
		rest = strings.TrimPrefix(rest, "'s")
		rest = strings.TrimPrefix(rest, "’s")
		return entry.canonical, rest, true
	}
	return "", "", false
}

// findIn locates any canonical alias inside the text (word-bounded), for
// secondary slots such as transfer recipients.
func (r *rosterMatcher) findIn(text string) (string, bool) {
	for _, entry := range r.entries {
		index := strings.Index(text, entry.alias)
		for index >= 0 {
			if wordBoundary(text, index, index+len(entry.alias)) {
				return entry.canonical, true
			}
			next := strings.Index(text[index+1:], entry.alias)
			if next < 0 {
				break
			}
			index += 1 + next
		}
	}
	return "", false
}

// ── extraction context ───────────────────────────────────────────────────────

type frameContext struct {
	book    *types.BookData
	record  types.EvidenceRecord
	order   int
	point   types.StoryTime
	scope   types.NarrativeRealityScope
	roster  *rosterMatcher
	negated bool
}

func spanFromRecord(record types.EvidenceRecord) types.NarrativeEvidenceSpan {
	return types.NarrativeEvidenceSpan{
		EvidenceID: record.ID, ChapterID: record.ChapterID, ChapterIndex: record.ChapterIndex,
		Section: record.Section, SectionIndex: record.SectionIndex,
		ParagraphIndex: record.ParagraphIndex, SentenceIndex: record.SentenceIndex,
		StartOffset: record.StartOffset, EndOffset: record.EndOffset,
		Quote: record.Text, Confidence: record.Confidence,
	}
}

func (fc *frameContext) newFrame(frameType, detail string, confidence float64) types.NarrativeFrame {
	polarity := "asserted"
	if fc.negated {
		polarity = "negated"
	}
	return types.NarrativeFrame{
		ID:          stableID("frame", fc.record.ID, frameType, detail),
		Type:        frameType,
		Detail:      detail,
		Polarity:    polarity,
		Epistemic:   "narration",
		Attribution: types.NarrativeAttribution{Kind: "narrator", Confidence: .9},
		Scope:       fc.scope,
		Temporal:    fc.point,
		ContextID:   fc.point.ContextID,
		ChapterIndex: fc.record.ChapterIndex, ParagraphIndex: fc.record.ParagraphIndex,
		SentenceIndex: fc.record.SentenceIndex, NarrativeOrder: fc.order,
		EvidenceIDs:   []string{fc.record.ID},
		EvidenceSpans: []types.NarrativeEvidenceSpan{spanFromRecord(fc.record)},
		Confidence:    confidence,
	}
}

func participant(role, name string) types.NarrativeParticipant {
	return types.NarrativeParticipant{EntityName: name, Role: role}
}

// detailSlice cuts a verbatim payload out of text starting at from: up to the
// first clause-ending punctuation, capped at maxDetailWords. The returned
// string is always a contiguous substring of text.
func detailSlice(text string, from int) (string, bool) {
	if from < 0 || from >= len(text) {
		return "", false
	}
	end := len(text)
	for i := from; i < len(text); i++ {
		if strings.ContainsRune(".;!?”\"", rune(text[i])) {
			end = i
			break
		}
	}
	slice := strings.TrimSpace(text[from:end])
	if slice == "" {
		return "", false
	}
	words := strings.Fields(slice)
	truncated := false
	if len(words) > maxDetailWords {
		// Re-derive the verbatim prefix covering the first maxDetailWords
		// words: find the offset of the word after the cap in the slice.
		count, cut := 0, len(slice)
		inWord := false
		for i := 0; i < len(slice); i++ {
			isSpace := slice[i] == ' ' || slice[i] == '\t'
			if !isSpace && !inWord {
				count++
				if count > maxDetailWords {
					cut = i
					break
				}
			}
			inWord = !isSpace
		}
		slice = strings.TrimSpace(slice[:cut])
		truncated = true
	}
	return slice, truncated
}

var itemStopRe = regexp.MustCompile(`(?i)\s+(from|on|in|at|off|under|behind|beside|into|onto|out of|with|before|after|and then)\s`)

// itemDetail slices an item noun phrase: like detailSlice, but also stops at
// the first preposition so "the brass key from the desk" yields "brass key".
func itemDetail(text string, from int) string {
	detail, _ := detailSlice(text, from)
	if detail == "" {
		return ""
	}
	if stop := itemStopRe.FindStringIndex(detail); stop != nil {
		detail = strings.TrimSpace(detail[:stop[0]])
	}
	return detail
}

var negationRe = regexp.MustCompile(`(?i)\b(never|not|no longer|didn't|did not|wasn't|was not|couldn't|could not|refused to)\b`)
var speculationRe = regexp.MustCompile(`(?i)\b(might|may have|perhaps|possibly|seemed to|appeared to)\b`)

// ── the extractor ────────────────────────────────────────────────────────────

type extractor struct {
	frameType  string
	pattern    *regexp.Regexp
	confidence float64
	// build turns a match into a frame; returning false abstains entirely.
	build func(fc *frameContext, subject, rest string, match []int) (types.NarrativeFrame, bool)
}

var (
	lifeStatusRe  = regexp.MustCompile(`(?i)\b(was killed|had died|died|was dead|is dead|passed away|survived|was alive|still alive)\b`)
	locationSetRe = regexp.MustCompile(`(?i)\b(?:was|were|stood|sat|waited|lay|remained|lived)\s+(?:back\s+)?(in|at|inside|outside|near)\s+`)
	locationMoveRe = regexp.MustCompile(`(?i)\b(entered|arrived at|reached|walked into|stepped into|returned to|went to|drove to|headed to|climbed to|crossed into)\s+`)
	locationLeaveRe = regexp.MustCompile(`(?i)\b(left|departed|exited|fled|abandoned)\s+(the\s+|his\s+|her\s+)?`)
	possessionGetRe = regexp.MustCompile(`(?i)\b(picked up|took|grabbed|pocketed|carried|held|clutched|drew|retrieved|kept)\s+(?:up\s+)?(the|a|an|his|her|their|its)\s+`)
	possessionLoseRe = regexp.MustCompile(`(?i)\b(dropped|lost|surrendered|discarded|tossed away|left behind)\s+(the|a|an|his|her|their)\s+`)
	transferRe    = regexp.MustCompile(`(?i)\b(handed|gave|passed|tossed|slid|returned)\s+(the|a|an|his|her|their)?\s*`)
	injuryRe      = regexp.MustCompile(`(?i)\b(wounded|bleeding|broken (?:arm|leg|rib|nose|wrist|ankle|hand)|bruised|shot|stabbed|burned|limping|concussion|sprained|fractured|injured)\b`)
	knowledgeRe   = regexp.MustCompile(`(?i)\b(knew|learned|realized|discovered|remembered|noticed|recognized|understood)\s+(that\s+)?`)
	beliefRe      = regexp.MustCompile(`(?i)\b(believed|suspected|assumed|thought|feared|hoped|doubted)\s+(that\s+)?`)
	goalRe        = regexp.MustCompile(`(?i)\b(wanted|needed|planned|intended|hoped|aimed)\s+to\s+`)
	decisionRe    = regexp.MustCompile(`(?i)\b(decided|chose|agreed|resolved|refused)\s+(?:to\s+|not\s+to\s+|that\s+)?`)
	obligationRe  = regexp.MustCompile(`(?i)\b(had to|promised to|was supposed to|owed|was ordered to|swore to|must)\s+`)
	relationshipRe = regexp.MustCompile(`(?i)^,?\s*(?:'s|\x{2019}s)?\s*(partner|wife|husband|brother|sister|mother|father|son|daughter|boss|friend|mentor|ex-wife|ex-husband)\b`)
	accessGainRe  = regexp.MustCompile(`(?i)\b(had|knew|got|received|memorized|copied)\s+(?:the|a|an|his|her)?\s*(key(?:card)?|code|password|combination|badge|passcode|access)\b`)
	claimTagRe    = regexp.MustCompile(`(?i)[,”"]\s*([A-Z][\p{L}'\x{2019}.-]*(?:\s+[A-Z][\p{L}'\x{2019}.-]*)?)\s+(said|says|told|insisted|claimed|replied|answered|explained|admitted|warned|whispered|shouted|muttered|announced)\b`)
	quotedRe      = regexp.MustCompile(`[“"]([^”"]{2,400})[”"]`)
)

// extractFrames turns evidence records into typed frames.
func extractFrames(
	book *types.BookData,
	records []types.EvidenceRecord,
	contextByEvidence map[string]string,
	points map[string]types.StoryTime,
	scopes map[string]types.NarrativeRealityScope,
) []types.NarrativeFrame {
	roster := buildRoster(book)
	frames := []types.NarrativeFrame{}
	for order, record := range records {
		point := points[record.ID]
		scope := scopes[contextByEvidence[record.ID]]
		if scope.ID == "" {
			scope = types.NarrativeRealityScope{ID: contextByEvidence[record.ID], Kind: "uncertain", Label: "Unclassified narrative reality", Confidence: .3}
		}
		fc := &frameContext{book: book, record: record, order: order, point: point, scope: scope, roster: roster}
		frames = append(frames, extractFromRecord(fc)...)
	}
	sort.SliceStable(frames, func(i, j int) bool { return frames[i].NarrativeOrder < frames[j].NarrativeOrder })
	return frames
}

func extractFromRecord(fc *frameContext) []types.NarrativeFrame {
	text := fc.record.Text
	frames := []types.NarrativeFrame{}

	// Attributed speech first: a quoted span with a canonical speaker tag
	// becomes a claim frame; the quote is never treated as narrator truth.
	if frame, ok := claimFrame(fc, text); ok {
		frames = append(frames, frame)
	}

	subject, rest, hasSubject := fc.roster.subjectAt(text)
	if !hasSubject {
		// Knowledge states can still anchor on their recorded knower even
		// when the sentence doesn't open with the name.
		frames = append(frames, knowledgeStateFrames(fc)...)
		return capFrames(frames)
	}
	fc.negated = negationRe.MatchString(firstWords(rest, 6))

	// Typed extractors in specificity order; each may add at most one frame.
	frames = append(frames, subjectFrames(fc, text, subject, rest)...)
	frames = append(frames, knowledgeStateFrames(fc)...)
	return capFrames(frames)
}

// capFrames bounds frames-per-sentence and deduplicates by ID so one
// sentence can never flood the corpus.
func capFrames(frames []types.NarrativeFrame) []types.NarrativeFrame {
	seen := map[string]bool{}
	result := []types.NarrativeFrame{}
	for _, frame := range frames {
		if seen[frame.ID] {
			continue
		}
		seen[frame.ID] = true
		result = append(result, frame)
		if len(result) == 3 {
			break
		}
	}
	return result
}

func firstWords(text string, count int) string {
	words := strings.Fields(text)
	if len(words) > count {
		words = words[:count]
	}
	return strings.Join(words, " ")
}

func subjectFrames(fc *frameContext, text, subject, rest string) []types.NarrativeFrame {
	frames := []types.NarrativeFrame{}
	restOffset := len(text) - len(rest)

	if match := lifeStatusRe.FindStringIndex(rest); match != nil {
		value := "dead"
		lowered := strings.ToLower(rest[match[0]:match[1]])
		if strings.Contains(lowered, "alive") || strings.Contains(lowered, "survived") {
			value = "alive"
		}
		frame := fc.newFrame(types.FrameLifeStatus, strings.TrimSpace(rest[match[0]:match[1]]), .92)
		frame.Value = value
		frame.Participants = []types.NarrativeParticipant{participant("subject", subject)}
		frames = append(frames, frame)
	}

	if match := locationSetRe.FindStringSubmatchIndex(rest); match != nil {
		if frame, ok := placeFrame(fc, text, subject, rest, restOffset, match[1], "set", .88); ok {
			frames = append(frames, frame)
		}
	} else if match := locationMoveRe.FindStringSubmatchIndex(rest); match != nil {
		if frame, ok := placeFrame(fc, text, subject, rest, restOffset, match[1], "set", .85); ok {
			frames = append(frames, frame)
		}
	} else if match := locationLeaveRe.FindStringSubmatchIndex(rest); match != nil {
		if frame, ok := placeFrame(fc, text, subject, rest, restOffset, match[1], "clear", .8); ok {
			frames = append(frames, frame)
		}
	}

	if match := transferRe.FindStringSubmatchIndex(rest); match != nil {
		if frame, ok := transferFrame(fc, subject, rest, match[1]); ok {
			frames = append(frames, frame)
		}
	} else if match := possessionGetRe.FindStringSubmatchIndex(rest); match != nil {
		if detail := itemDetail(rest, match[1]); detail != "" {
			frame := fc.newFrame(types.FramePossession, detail, .85)
			frame.Value = "holds"
			frame.Participants = []types.NarrativeParticipant{participant("subject", subject), participant("item", detail)}
			frames = append(frames, frame)
		}
	} else if match := possessionLoseRe.FindStringSubmatchIndex(rest); match != nil {
		if detail := itemDetail(rest, match[1]); detail != "" {
			frame := fc.newFrame(types.FramePossession, detail, .8)
			frame.Value = "relinquished"
			frame.Participants = []types.NarrativeParticipant{participant("subject", subject), participant("item", detail)}
			frames = append(frames, frame)
		}
	}

	if match := injuryRe.FindStringIndex(rest); match != nil {
		frame := fc.newFrame(types.FrameInjury, strings.TrimSpace(rest[match[0]:match[1]]), .85)
		frame.Participants = []types.NarrativeParticipant{participant("subject", subject)}
		frames = append(frames, frame)
	}

	if match := knowledgeRe.FindStringSubmatchIndex(rest); match != nil {
		if detail, _ := detailSlice(rest, match[1]); detail != "" {
			frame := fc.newFrame(types.FrameKnowledge, detail, .85)
			frame.Value = "knows"
			frame.Participants = []types.NarrativeParticipant{participant("subject", subject)}
			frames = append(frames, frame)
		}
	} else if match := beliefRe.FindStringSubmatchIndex(rest); match != nil {
		if detail, _ := detailSlice(rest, match[1]); detail != "" {
			frame := fc.newFrame(types.FrameBelief, detail, .8)
			frame.Epistemic = "belief"
			frame.Participants = []types.NarrativeParticipant{participant("subject", subject)}
			frames = append(frames, frame)
		}
	}

	if match := goalRe.FindStringSubmatchIndex(rest); match != nil {
		if detail, _ := detailSlice(rest, match[1]); detail != "" {
			frame := fc.newFrame(types.FrameGoal, detail, .82)
			frame.Participants = []types.NarrativeParticipant{participant("subject", subject)}
			frames = append(frames, frame)
		}
	} else if match := decisionRe.FindStringSubmatchIndex(rest); match != nil {
		if detail, _ := detailSlice(rest, match[1]); detail != "" {
			frame := fc.newFrame(types.FrameDecision, detail, .82)
			frame.Participants = []types.NarrativeParticipant{participant("subject", subject)}
			frames = append(frames, frame)
		}
	} else if match := obligationRe.FindStringSubmatchIndex(rest); match != nil {
		if detail, _ := detailSlice(rest, match[1]); detail != "" {
			frame := fc.newFrame(types.FrameObligation, detail, .8)
			frame.Value = "open"
			frame.Participants = []types.NarrativeParticipant{participant("subject", subject)}
			frames = append(frames, frame)
		}
	}

	if match := relationshipRe.FindStringSubmatchIndex(rest); match != nil {
		role := strings.ToLower(strings.TrimSpace(rest[match[2]:match[3]]))
		after := rest[match[1]:]
		if counterparty, ok := fc.roster.findIn(firstWords(after, 5)); ok && counterparty != subject {
			frame := fc.newFrame(types.FrameRelationship, role, .85)
			frame.Value = role
			frame.Participants = []types.NarrativeParticipant{participant("subject", subject), participant("counterparty", counterparty)}
			frames = append(frames, frame)
		} else {
			frame := fc.newFrame(types.FrameRelationship, role, .6)
			frame.Value = role
			frame.Participants = []types.NarrativeParticipant{participant("subject", subject)}
			frame.Abstentions = append(frame.Abstentions, "counterparty: no canonical name near the relationship term")
			frames = append(frames, frame)
		}
	}

	if match := accessGainRe.FindStringSubmatchIndex(rest); match != nil {
		if detail, _ := detailSlice(rest, match[2*2]); detail != "" {
			frame := fc.newFrame(types.FrameAccess, detail, .8)
			frame.Value = "granted"
			frame.Participants = []types.NarrativeParticipant{participant("subject", subject)}
			frames = append(frames, frame)
		}
	}

	if speculationRe.MatchString(firstWords(rest, 8)) {
		for index := range frames {
			if frames[index].Epistemic == "narration" {
				frames[index].Epistemic = "speculation"
				frames[index].Confidence -= .15
			}
		}
	}

	// Generic witnessed event: only as a fallback, only for event-kind
	// records with an anchored subject and a detected action verb.
	if len(frames) == 0 && fc.record.Kind == "event" && fc.record.Action != "" {
		if detail, truncated := detailSlice(rest, 0); detail != "" {
			frame := fc.newFrame(types.FrameEvent, detail, .7)
			frame.Participants = []types.NarrativeParticipant{participant("subject", subject)}
			if truncated {
				frame.Abstentions = append(frame.Abstentions, "detail: truncated to the leading clause")
			}
			frames = append(frames, frame)
		}
	}
	return frames
}

func placeFrame(fc *frameContext, text, subject, rest string, restOffset, from int, value string, confidence float64) (types.NarrativeFrame, bool) {
	detail, _ := detailSlice(rest, from)
	if detail == "" {
		return types.NarrativeFrame{}, false
	}
	// Prefer a named location term when prose/v3 tagged one inside the slice.
	place := ""
	for _, term := range fc.record.NamedEntities {
		label := strings.ToUpper(term.Label)
		if (label == "GPE" || label == "FAC" || label == "LOC" || label == "ORG") && strings.Contains(detail, term.Text) {
			place = term.Text
			break
		}
	}
	frame := fc.newFrame(types.FrameLocation, detail, confidence)
	frame.Value = value
	frame.Participants = []types.NarrativeParticipant{participant("subject", subject)}
	if place != "" {
		frame.Participants = append(frame.Participants, participant("place", place))
	} else {
		frame.Abstentions = append(frame.Abstentions, "place: no named location term; verbatim phrase retained")
	}
	return frame, true
}

func transferFrame(fc *frameContext, subject, rest string, from int) (types.NarrativeFrame, bool) {
	toIndex := strings.Index(strings.ToLower(rest[from:]), " to ")
	if toIndex < 0 {
		return types.NarrativeFrame{}, false
	}
	item := itemDetail(rest[:from+toIndex], from)
	if item == "" {
		return types.NarrativeFrame{}, false
	}
	afterTo := rest[from+toIndex+4:]
	recipient, ok := fc.roster.findIn(firstWords(afterTo, 4))
	if !ok || recipient == subject {
		// A transfer without a resolvable, distinct recipient abstains to a
		// plain possession-relinquish rather than inventing a counterparty.
		frame := fc.newFrame(types.FramePossession, item, .7)
		frame.Value = "relinquished"
		frame.Participants = []types.NarrativeParticipant{participant("subject", subject), participant("item", item)}
		frame.Abstentions = append(frame.Abstentions, "recipient: no distinct canonical name after 'to'")
		return frame, true
	}
	frame := fc.newFrame(types.FrameTransfer, item, .87)
	frame.Participants = []types.NarrativeParticipant{
		participant("source", subject), participant("recipient", recipient), participant("item", item),
	}
	return frame, true
}

// claimFrame captures attributed quoted speech: subject = the speaker, the
// payload = the quoted text, epistemic = attributed_claim. Unattributed
// quotes abstain entirely.
func claimFrame(fc *frameContext, text string) (types.NarrativeFrame, bool) {
	quote := quotedRe.FindStringSubmatchIndex(text)
	if quote == nil {
		return types.NarrativeFrame{}, false
	}
	tag := claimTagRe.FindStringSubmatchIndex(text)
	if tag == nil {
		return types.NarrativeFrame{}, false
	}
	speakerText := text[tag[2]:tag[3]]
	speaker, ok := fc.roster.findIn(speakerText)
	if !ok {
		return types.NarrativeFrame{}, false
	}
	content := strings.TrimSpace(text[quote[2]:quote[3]])
	if content == "" {
		return types.NarrativeFrame{}, false
	}
	frame := fc.newFrame(types.FrameClaim, content, .85)
	frame.Epistemic = "attributed_claim"
	frame.Attribution = types.NarrativeAttribution{Kind: "character", EntityName: speaker, Cue: strings.TrimSpace(text[tag[2]:tag[5]]), Confidence: .85}
	frame.Participants = []types.NarrativeParticipant{participant("subject", speaker)}
	return frame, true
}

// knowledgeStateFrames projects the evidence layer's conservative,
// sentence-local knowledge states into knowledge/belief frames. The knower
// comes from the upstream record, already canonical.
func knowledgeStateFrames(fc *frameContext) []types.NarrativeFrame {
	frames := []types.NarrativeFrame{}
	for _, state := range fc.record.KnowledgeStates {
		if len(state.CharacterNames) == 0 {
			continue
		}
		frameType := types.FrameKnowledge
		value := state.State
		epistemic := "narration"
		switch state.State {
		case "believes", "suspects":
			frameType = types.FrameBelief
			epistemic = "belief"
		case "learned", "knows", "does_not_know", "shared", "withheld", "attempts_to_recall":
			// knowledge frame as typed
		default:
			continue
		}
		detail := strings.TrimSpace(state.Cue)
		if detail == "" || !strings.Contains(fc.record.Text, detail) {
			// The cue must be verbatim; otherwise carry the whole sentence.
			detail = fc.record.Text
		}
		frame := fc.newFrame(frameType, detail, minFloat(.9, state.Confidence+.1))
		frame.Value = value
		frame.Epistemic = epistemic
		if state.State == "does_not_know" {
			frame.Polarity = "negated"
		}
		frame.Participants = []types.NarrativeParticipant{participant("subject", state.CharacterNames[0])}
		for _, counterparty := range state.CounterpartyNames {
			frame.Participants = append(frame.Participants, participant("counterparty", counterparty))
		}
		frames = append(frames, frame)
	}
	return frames
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
