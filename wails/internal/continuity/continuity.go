package continuity

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"draftline/internal/indexing"
	"draftline/internal/storytimeline"
	"draftline/internal/types"
)

const engine = "source-continuity-v1"

var (
	wordRe       = regexp.MustCompile(`[\pL\pN]+(?:['’][\pL\pN]+)?`)
	clockRe      = regexp.MustCompile(`(?i)^(\d{1,2}):(\d{2})`)
	ageRe        = regexp.MustCompile(`(?i)\b(\d{1,3})[- ]year[- ]old\b`)
	colorRe      = regexp.MustCompile(`(?i)\b(amber|auburn|black|blond|blonde|blue|brown|chestnut|gray|grey|green|hazel|red|silver|white)\b`)
	handednessRe = regexp.MustCompile(`(?i)\b(left|right)[- ]handed\b`)
)

var conceptStop = map[string]bool{
	"about": true, "after": true, "again": true, "already": true, "also": true, "and": true, "are": true, "because": true,
	"before": true, "believe": true, "believed": true, "believes": true, "but": true, "could": true, "did": true, "didn't": true,
	"does": true, "doesn't": true, "don't": true, "from": true, "had": true, "has": true, "have": true, "into": true, "is": true,
	"know": true, "knows": true, "knew": true, "learned": true, "learns": true, "not": true, "realized": true, "remembered": true,
	"said": true, "she": true, "suspect": true, "suspected": true, "that": true, "the": true, "their": true, "them": true, "then": true,
	"they": true, "this": true, "told": true, "was": true, "were": true, "what": true, "when": true, "with": true, "would": true,
	"you": true, "your": true,
}

// Build derives review prompts from the current evidence, character, and
// timeline fingerprints. It does not persist conclusions or modify the book.
func Build(book types.BookData) types.ContinuityReport {
	report := types.ContinuityReport{Success: true, Engine: engine, Signals: []types.ContinuitySignal{}, Categories: []types.ContinuityFacet{}, Characters: []types.ContinuityFacet{}}
	if book.Analysis.Evidence == nil {
		report.Success = false
		report.Error = "Story analysis has not produced an evidence index yet."
		return report
	}
	report.Signals = append(report.Signals, characterSignals(book)...)
	report.Signals = append(report.Signals, knowledgeSignals(book)...)
	report.Signals = append(report.Signals, attributeSignals(book)...)
	report.Signals = append(report.Signals, fingerprintSignals(book)...)
	timeline := storytimeline.Build(book)
	if timeline.Success {
		report.Signals = append(report.Signals, clockSignals(timeline)...)
		report.Signals = append(report.Signals, thinChapterSignals(book, timeline)...)
	}
	sortSignals(report.Signals)
	finishReport(&report, book)
	return report
}

// fingerprintSignals projects manuscript-memory inspections (v5) into
// continuity signals. Cross-scope divergences are reported as informational
// so deliberate flashbacks, dreams, and simulations are not flagged as
// errors.
func fingerprintSignals(book types.BookData) []types.ContinuitySignal {
	if book.Analysis.Fingerprint == nil {
		return nil
	}
	result := []types.ContinuitySignal{}
	records := map[string]types.EvidenceRecord{}
	for _, record := range book.Analysis.Evidence.Records {
		records[record.ID] = record
	}
	for _, item := range book.Analysis.Fingerprint.Inspections {
		sources := []types.ContinuitySource{}
		for _, side := range item.Sides {
			for _, span := range side.EvidenceSpans {
				if record, ok := records[span.EvidenceID]; ok {
					sources = append(sources, sourceFromRecord(book, record))
				}
			}
		}
		severity := item.Severity
		if item.ScopeAssessment == "cross_scope_divergence" {
			severity = "info"
		}
		result = append(result, signal("memory-"+item.Kind, "story", severity, item.Title, item.Detail, nil, nil, sources, item.Confidence))
	}
	return result
}

func characterSignals(book types.BookData) []types.ContinuitySignal {
	result := []types.ContinuitySignal{}
	for _, character := range book.StoryBible.Characters {
		if !continuityCharacter(character) {
			continue
		}
		source := firstCharacterSource(book, character.ID)
		if character.MentionCount >= 5 && !hasGivenNameInBook(book, character) {
			result = append(result, signal("missing-given-name", "characters", "review", character.Name+" has no established given name", fmt.Sprintf("%s appears %d times, but every accepted name form is still a single name or title plus surname. This may be intentional; review where a fuller identity would naturally appear.", character.Name, character.MentionCount), []string{character.ID}, []string{character.Name}, sourceList(source), .96))
		}
		chapters := nonzeroChapterMentions(character.ChapterMentions)
		if len(chapters) == 1 && character.MentionCount <= 3 {
			severity := "info"
			detail := fmt.Sprintf("%s appears %d %s in only one chapter. That can be deliberate, but it is worth checking whether the character is introduced, resolved, or accidentally abandoned.", character.Name, character.MentionCount, plural(character.MentionCount, "time", "times"))
			if character.MentionCount == 1 && character.FirstChapter >= max(2, len(book.Body)/3) {
				severity = "review"
				detail = fmt.Sprintf("%s appears once, late in the manuscript, and is not seen again. Confirm that this is an intentional one-off rather than an orphaned introduction.", character.Name)
			}
			result = append(result, signal("one-chapter-character", "characters", severity, character.Name+" appears in only one chapter", detail, []string{character.ID}, []string{character.Name}, sourceList(source), .9))
		}
	}
	return result
}

type knowledgePoint struct {
	state  string
	record types.EvidenceRecord
	tokens map[string]bool
}

func knowledgeSignals(book types.BookData) []types.ContinuitySignal {
	byCharacter := map[string][]knowledgePoint{}
	for _, record := range book.Analysis.Evidence.Records {
		if record.Status == "rejected" {
			continue
		}
		for _, state := range record.KnowledgeStates {
			for index, id := range state.CharacterIDs {
				names := append([]string(nil), state.CharacterNames...)
				if index < len(names) {
					names = append(names, names[index])
				}
				tokens := conceptTokens(record.Text, names)
				if len(tokens) >= 2 {
					byCharacter[id] = append(byCharacter[id], knowledgePoint{state: state.State, record: record, tokens: tokens})
				}
			}
		}
	}
	result := []types.ContinuitySignal{}
	seen := map[string]bool{}
	for characterID, points := range byCharacter {
		sort.SliceStable(points, func(i, j int) bool { return recordLess(points[i].record, points[j].record) })
		for i := 0; i < len(points); i++ {
			for j := i + 1; j < len(points); j++ {
				if similarity(points[i].tokens, points[j].tokens) < .6 {
					continue
				}
				kind, title, detail, confidence := knowledgeConcern(points[i].state, points[j].state)
				if kind == "" {
					continue
				}
				key := characterID + "\x00" + kind + "\x00" + points[i].record.ID + "\x00" + points[j].record.ID
				if seen[key] {
					continue
				}
				seen[key] = true
				name := characterName(points[i].record, characterID)
				result = append(result, signal(kind, "knowledge", "review", name+" "+title, detail, []string{characterID}, []string{name}, []types.ContinuitySource{sourceFromRecord(book, points[i].record), sourceFromRecord(book, points[j].record)}, confidence))
			}
		}
	}
	return result
}

func knowledgeConcern(earlier, later string) (kind, title, detail string, confidence float64) {
	if positiveKnowledge(earlier) && negativeCounterpart(earlier) == later {
		return "knowledge-reversal", "may reverse an established knowledge state", "An earlier passage establishes this knowledge or belief, while a later passage explicitly negates a closely matching concept. This may represent forgetting, deception, or intentional uncertainty; review the transition.", .84
	}
	if negativeKnowledge(earlier) && positiveCounterpart(earlier) == later {
		return "knowledge-reversal", "may gain knowledge without a visible bridge", "An earlier passage explicitly denies this knowledge or belief, while a later passage states a closely matching positive state. Check whether the intervening acquisition is established clearly enough.", .82
	}
	if earlier == "knows" && later == "learned" {
		return "knowledge-order", "appears to know something before learning it", "A closely matching concept is stated as known before a later passage explicitly presents it as learned or realized. Flashbacks can make this correct; review the two sources in narrative context.", .78
	}
	return "", "", "", 0
}

type attributePoint struct {
	attribute string
	value     string
	record    types.EvidenceRecord
}

func attributeSignals(book types.BookData) []types.ContinuitySignal {
	byKey := map[string][]attributePoint{}
	for _, record := range book.Analysis.Evidence.Records {
		if record.Status == "rejected" || len(record.CharacterIDs) != 1 {
			continue
		}
		attribute, value := physicalAttribute(record.Text)
		if attribute == "" {
			continue
		}
		for _, id := range record.CharacterIDs {
			key := id + "\x00" + attribute
			byKey[key] = append(byKey[key], attributePoint{attribute: attribute, value: value, record: record})
		}
	}
	result := []types.ContinuitySignal{}
	for key, points := range byKey {
		if len(points) < 2 {
			continue
		}
		sort.SliceStable(points, func(i, j int) bool { return recordLess(points[i].record, points[j].record) })
		for i := 0; i < len(points); i++ {
			for j := i + 1; j < len(points); j++ {
				if points[i].value == points[j].value {
					continue
				}
				characterID := strings.SplitN(key, "\x00", 2)[0]
				name := characterName(points[i].record, characterID)
				title := fmt.Sprintf("%s has conflicting %s details", name, points[i].attribute)
				detail := fmt.Sprintf("One source says %s; another says %s. If this is not a deliberate change, disguise, or unreliable observation, the physical detail may need reconciliation.", points[i].value, points[j].value)
				result = append(result, signal("attribute-conflict", "facts", "review", title, detail, []string{characterID}, []string{name}, []types.ContinuitySource{sourceFromRecord(book, points[i].record), sourceFromRecord(book, points[j].record)}, .94))
				break
			}
		}
	}
	return result
}

func clockSignals(timeline types.StoryTimelineResult) []types.ContinuitySignal {
	result := []types.ContinuitySignal{}
	var previous *types.StoryTimelineEvent
	for index := range timeline.Events {
		event := &timeline.Events[index]
		minutes, ok := firstClock(event.TimeExpressions)
		if !ok || skipClockComparison(event.Text) {
			continue
		}
		if previous != nil && previous.ChapterIndex == event.ChapterIndex && event.ParagraphIndex-previous.ParagraphIndex <= 3 {
			previousMinutes, previousOK := firstClock(previous.TimeExpressions)
			if previousOK && minutes+10 < previousMinutes && !(previousMinutes >= 11*60 && minutes <= 1*60) && !skipClockComparison(previous.Text) {
				detail := fmt.Sprintf("A nearby clock reference moves from %s to %s without an indexed relative-time bridge or chapter break. This may be a flashback, comparison, or countdown; review the source order.", firstClockLabel(previous.TimeExpressions), firstClockLabel(event.TimeExpressions))
				result = append(result, signal("clock-regression", "chronology", "review", event.ChapterTitle+" may move backward in clock time", detail, event.CharacterIDs, event.CharacterNames, []types.ContinuitySource{sourceFromTimeline(*previous), sourceFromTimeline(*event)}, .72))
			}
		}
		previous = event
	}
	return result
}

func thinChapterSignals(book types.BookData, timeline types.StoryTimelineResult) []types.ContinuitySignal {
	counts := map[int]int{}
	for _, chapter := range timeline.Chapters {
		counts[chapter.ChapterIndex] = chapter.EventCount
	}
	eligibleCounts := []int{}
	global := len(book.FrontMatter)
	for _, chapter := range book.Body {
		if len(strings.Fields(indexing.StripHTMLForAnalysis(chapter.Content))) >= 100 {
			eligibleCounts = append(eligibleCounts, counts[global])
		}
		global++
	}
	if len(eligibleCounts) < 5 {
		return nil
	}
	sorted := append([]int(nil), eligibleCounts...)
	sort.Ints(sorted)
	median := sorted[len(sorted)/2]
	threshold := max(1, median/4)
	result := []types.ContinuitySignal{}
	global = len(book.FrontMatter)
	for index, chapter := range book.Body {
		words := len(strings.Fields(indexing.StripHTMLForAnalysis(chapter.Content)))
		count := counts[global]
		if words >= 100 && count <= threshold {
			detail := fmt.Sprintf("This chapter contains %d indexed %s, compared with a manuscript median of %d. Low event density is not inherently a problem, but it can reveal a connective chapter whose plot movement deserves a second look.", count, plural(count, "event", "events"), median)
			source := types.ContinuitySource{ChapterID: chapter.ID, ChapterIndex: global, ChapterTitle: chapter.Title, Section: "body", SectionIndex: index}
			result = append(result, signal("thin-chapter", "structure", "info", chapter.Title+" has unusually thin event coverage", detail, nil, nil, []types.ContinuitySource{source}, .76))
		}
		global++
	}
	return result
}

func physicalAttribute(text string) (attribute, value string) {
	lower := strings.ToLower(text)
	if color := colorRe.FindString(lower); color != "" {
		if strings.Contains(lower, "eye") {
			return "eye color", color
		}
		if strings.Contains(lower, "hair") {
			return "hair color", color
		}
	}
	if match := ageRe.FindStringSubmatch(lower); len(match) == 2 {
		return "age", match[1] + " years old"
	}
	if match := handednessRe.FindStringSubmatch(lower); len(match) == 2 {
		return "handedness", strings.ToLower(match[1]) + "-handed"
	}
	return "", ""
}

func conceptTokens(text string, names []string) map[string]bool {
	excluded := map[string]bool{}
	for _, name := range names {
		for _, token := range wordRe.FindAllString(strings.ToLower(name), -1) {
			excluded[token] = true
		}
	}
	result := map[string]bool{}
	for _, token := range wordRe.FindAllString(strings.ToLower(text), -1) {
		if len([]rune(token)) < 3 || conceptStop[token] || excluded[token] {
			continue
		}
		result[token] = true
	}
	return result
}

func similarity(a, b map[string]bool) float64 {
	intersection, union := 0, len(a)
	for token := range b {
		if a[token] {
			intersection++
		} else {
			union++
		}
	}
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

func firstClock(expressions []string) (int, bool) {
	for _, expression := range expressions {
		match := clockRe.FindStringSubmatch(strings.TrimSpace(expression))
		if len(match) != 3 {
			continue
		}
		hour, hourErr := strconv.Atoi(match[1])
		minute, minuteErr := strconv.Atoi(match[2])
		if hourErr == nil && minuteErr == nil && hour <= 23 && minute <= 59 {
			return hour*60 + minute, true
		}
	}
	return 0, false
}

func firstClockLabel(expressions []string) string {
	for _, expression := range expressions {
		if clockRe.MatchString(strings.TrimSpace(expression)) {
			return strings.TrimSpace(expression)
		}
	}
	return "the earlier time"
}

func skipClockComparison(text string) bool {
	lower := strings.ToLower(text)
	return strings.Contains(lower, "countdown") || strings.Contains(lower, "detonation") || strings.Contains(lower, "timer") || strings.Contains(lower, "time remaining")
}

func confirmedCharacter(character types.Character) bool {
	return !character.IsAutoDetected || character.DetectionStatus == "accepted"
}

func continuityCharacter(character types.Character) bool {
	if !confirmedCharacter(character) || (character.EntityKind != "" && character.EntityKind != "person") {
		return false
	}
	return !character.IsAutoDetected || character.DetectionScore >= .8
}

func hasGivenNameInBook(book types.BookData, character types.Character) bool {
	if hasGivenName(character) {
		return true
	}
	parts := personNameParts(character.Name)
	if len(parts) != 1 {
		return false
	}
	surname := parts[0]
	for _, candidate := range book.StoryBible.Characters {
		if candidate.ID == character.ID || !confirmedCharacter(candidate) {
			continue
		}
		values := append([]string{candidate.Name}, candidate.Aliases...)
		for _, value := range values {
			candidateParts := personNameParts(value)
			if len(candidateParts) >= 2 && candidateParts[len(candidateParts)-1] == surname {
				return true
			}
		}
	}
	return false
}

func hasGivenName(character types.Character) bool {
	values := append([]string{character.Name}, character.Aliases...)
	for _, value := range values {
		parts := personNameParts(value)
		if len(parts) >= 2 {
			return true
		}
	}
	return false
}

func personNameParts(value string) []string {
	titles := map[string]bool{"captain": true, "chief": true, "colonel": true, "detective": true, "doctor": true, "dr": true, "lieutenant": true, "officer": true, "sergeant": true, "sgt": true, "staff": true}
	result := []string{}
	for _, part := range wordRe.FindAllString(strings.ToLower(value), -1) {
		if !titles[part] {
			result = append(result, part)
		}
	}
	return result
}

func firstCharacterSource(book types.BookData, characterID string) *types.ContinuitySource {
	for _, record := range book.Analysis.Evidence.Records {
		if record.Status != "rejected" && contains(record.CharacterIDs, characterID) {
			source := sourceFromRecord(book, record)
			return &source
		}
	}
	return nil
}

func sourceFromRecord(book types.BookData, record types.EvidenceRecord) types.ContinuitySource {
	return types.ContinuitySource{EvidenceID: record.ID, Text: record.Text, ChapterID: record.ChapterID, ChapterIndex: record.ChapterIndex, ChapterTitle: chapterTitle(book, record), Section: record.Section, SectionIndex: record.SectionIndex, ParagraphIndex: record.ParagraphIndex, StartOffset: record.StartOffset}
}

func sourceFromTimeline(event types.StoryTimelineEvent) types.ContinuitySource {
	evidenceID := ""
	if len(event.EvidenceIDs) > 0 {
		evidenceID = event.EvidenceIDs[0]
	}
	return types.ContinuitySource{EvidenceID: evidenceID, Text: event.SourceText, ChapterID: event.ChapterID, ChapterIndex: event.ChapterIndex, ChapterTitle: event.ChapterTitle, Section: event.Section, SectionIndex: event.SectionIndex, ParagraphIndex: event.ParagraphIndex, StartOffset: event.StartOffset}
}

func sourceList(source *types.ContinuitySource) []types.ContinuitySource {
	if source == nil {
		return nil
	}
	return []types.ContinuitySource{*source}
}

func signal(kind, category, severity, title, detail string, characterIDs, characterNames []string, sources []types.ContinuitySource, confidence float64) types.ContinuitySignal {
	key := kind + "\x00" + title
	for _, source := range sources {
		key += "\x00" + source.EvidenceID + "\x00" + source.ChapterID
	}
	sum := sha256.Sum256([]byte(key))
	return types.ContinuitySignal{ID: "continuity-" + hex.EncodeToString(sum[:8]), Kind: kind, Category: category, Severity: severity, Title: title, Detail: detail, CharacterIDs: characterIDs, CharacterNames: characterNames, Sources: sources, Confidence: confidence}
}

func finishReport(report *types.ContinuityReport, book types.BookData) {
	applyDecisions(report, book)
	categories, characters := map[string]*types.ContinuityFacet{}, map[string]*types.ContinuityFacet{}
	for _, item := range report.Signals {
		// Counts describe outstanding work, so a question the author has already
		// reviewed or dismissed is still listed but no longer counted.
		if item.Status == "" {
			if item.Severity == "review" {
				report.ReviewCount++
			} else {
				report.InfoCount++
			}
		}
		incrementFacet(categories, item.Category, categoryLabel(item.Category))
		for index, id := range item.CharacterIDs {
			label := id
			if index < len(item.CharacterNames) && item.CharacterNames[index] != "" {
				label = item.CharacterNames[index]
			}
			incrementFacet(characters, id, label)
		}
	}
	report.Categories = sortedFacets(categories)
	report.Characters = sortedFacets(characters)
	for _, chapter := range book.Body {
		if indexing.ShouldAnalyzeChapter(chapter) {
			report.ChaptersChecked++
		}
	}
}

// applyDecisions stamps stored author decisions onto freshly built signals. A
// decision whose signal no longer exists is simply ignored: the question it
// answered is gone, so the record has nothing to say about this manuscript.
func applyDecisions(report *types.ContinuityReport, book types.BookData) {
	if book.Analysis.Continuity == nil {
		return
	}
	status := make(map[string]string, len(book.Analysis.Continuity.Decisions))
	for _, decision := range book.Analysis.Continuity.Decisions {
		if decision.Status == "reviewed" || decision.Status == "dismissed" {
			status[decision.SignalID] = decision.Status
		}
	}
	for index := range report.Signals {
		report.Signals[index].Status = status[report.Signals[index].ID]
	}
}

func sortSignals(signals []types.ContinuitySignal) {
	sort.SliceStable(signals, func(i, j int) bool {
		if signals[i].Severity != signals[j].Severity {
			return signals[i].Severity == "review"
		}
		ai, aj := firstSourceIndex(signals[i]), firstSourceIndex(signals[j])
		if ai != aj {
			return ai < aj
		}
		return signals[i].Title < signals[j].Title
	})
}

func firstSourceIndex(item types.ContinuitySignal) int {
	if len(item.Sources) == 0 {
		return 1 << 30
	}
	return item.Sources[0].ChapterIndex
}

func incrementFacet(values map[string]*types.ContinuityFacet, id, label string) {
	if id == "" {
		return
	}
	if values[id] == nil {
		values[id] = &types.ContinuityFacet{ID: id, Label: label}
	}
	values[id].Count++
}

func sortedFacets(values map[string]*types.ContinuityFacet) []types.ContinuityFacet {
	result := make([]types.ContinuityFacet, 0, len(values))
	for _, value := range values {
		result = append(result, *value)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Count != result[j].Count {
			return result[i].Count > result[j].Count
		}
		return result[i].Label < result[j].Label
	})
	return result
}

func positiveKnowledge(state string) bool {
	return state == "knows" || state == "believes" || state == "suspects"
}
func negativeKnowledge(state string) bool {
	return state == "does_not_know" || state == "does_not_believe" || state == "does_not_suspect"
}

func negativeCounterpart(state string) string {
	switch state {
	case "knows":
		return "does_not_know"
	case "believes":
		return "does_not_believe"
	case "suspects":
		return "does_not_suspect"
	default:
		return ""
	}
}

func positiveCounterpart(state string) string {
	switch state {
	case "does_not_know":
		return "knows"
	case "does_not_believe":
		return "believes"
	case "does_not_suspect":
		return "suspects"
	default:
		return ""
	}
}

func characterName(record types.EvidenceRecord, characterID string) string {
	for index, id := range record.CharacterIDs {
		if id == characterID && index < len(record.CharacterNames) {
			return record.CharacterNames[index]
		}
	}
	for _, state := range record.KnowledgeStates {
		for index, id := range state.CharacterIDs {
			if id == characterID && index < len(state.CharacterNames) {
				return state.CharacterNames[index]
			}
		}
	}
	return "A character"
}

func recordLess(a, b types.EvidenceRecord) bool {
	if a.ChapterIndex != b.ChapterIndex {
		return a.ChapterIndex < b.ChapterIndex
	}
	if a.StartOffset != b.StartOffset {
		return a.StartOffset < b.StartOffset
	}
	return a.ID < b.ID
}

func chapterTitle(book types.BookData, record types.EvidenceRecord) string {
	var chapters []types.ChapterItem
	switch record.Section {
	case "front_matter":
		chapters = book.FrontMatter
	case "back_matter":
		chapters = book.BackMatter
	default:
		chapters = book.Body
	}
	if record.SectionIndex >= 0 && record.SectionIndex < len(chapters) && chapters[record.SectionIndex].Title != "" {
		return chapters[record.SectionIndex].Title
	}
	return "Chapter " + strconv.Itoa(record.ChapterIndex+1)
}

func nonzeroChapterMentions(values map[int]int) []int {
	result := []int{}
	for chapter, count := range values {
		if count > 0 {
			result = append(result, chapter)
		}
	}
	sort.Ints(result)
	return result
}

func contains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func categoryLabel(value string) string {
	switch value {
	case "characters":
		return "Characters"
	case "knowledge":
		return "Who knows what"
	case "facts":
		return "Story facts"
	case "chronology":
		return "Chronology"
	case "structure":
		return "Story structure"
	default:
		return value
	}
}

func plural(count int, singular, plural string) string {
	if count == 1 {
		return singular
	}
	return plural
}
