package storytimeline

import (
	"regexp"
	"sort"
	"strings"

	"draftline/internal/types"
)

const engine = "source-evidence-timeline-v1"

var relativeTimeRe = regexp.MustCompile(`(?i)\b(?:later|earlier|ago|before|after|tomorrow|yesterday|next|previous|prior)\b`)
var danglingMeridiemRe = regexp.MustCompile(`(?i)^(\d{1,2}:\d{2})\s+[ap]$`)

var typePriority = map[string]int{
	"discovery": 90, "knowledge_change": 85, "knowledge_transfer": 80,
	"interaction": 75, "transition": 70, "introduction": 65,
	"time_reference": 50, "state": 10,
}

// Build projects the persistent evidence index into a deduplicated timeline.
// It intentionally preserves manuscript order. Relative expressions are
// labelled, not resolved against an invented calendar.
func Build(book types.BookData) types.StoryTimelineResult {
	result := types.StoryTimelineResult{Success: true, Engine: engine, Events: []types.StoryTimelineEvent{}, Chapters: []types.StoryTimelineChapter{}, Characters: []types.StoryTimelineFacet{}, Locations: []types.StoryTimelineFacet{}, EventTypes: []types.StoryTimelineFacet{}}
	if book.Analysis.Evidence == nil {
		result.Success = false
		result.Error = "Story analysis has not produced an evidence index yet."
		return result
	}

	bySource := make(map[string]int)
	characterNames := characterNameSet(book)
	for _, record := range book.Analysis.Evidence.Records {
		if !timelineEligible(record) {
			continue
		}
		key := sourceKey(record)
		if index, ok := bySource[key]; ok {
			mergeRecord(&result.Events[index], record, characterNames)
			continue
		}
		event := eventFromRecord(book, record, characterNames)
		bySource[key] = len(result.Events)
		result.Events = append(result.Events, event)
	}
	sort.SliceStable(result.Events, func(i, j int) bool { return eventLess(result.Events[i], result.Events[j]) })
	finish(&result)
	return result
}

func timelineEligible(record types.EvidenceRecord) bool {
	if record.Status == "rejected" {
		return false
	}
	return record.Kind == "event" || len(record.TimeExpressions) > 0 || (record.Source == "author" && (record.Status == "confirmed" || record.Pinned))
}

func sourceKey(record types.EvidenceRecord) string {
	return strings.Join([]string{record.Section, record.ChapterID, intString(record.SectionIndex), intString(record.ParagraphIndex), intString(record.SentenceIndex), normalize(record.Text)}, "\x00")
}

func eventFromRecord(book types.BookData, record types.EvidenceRecord, characterNames map[string]bool) types.StoryTimelineEvent {
	times := cleanTimeExpressions(record.TimeExpressions)
	timeKind, timeLabel := timeMeaning(times)
	text := record.Text
	if strings.TrimSpace(record.AuthorText) != "" {
		text = record.AuthorText
	}
	return types.StoryTimelineEvent{
		ID: record.ID, EvidenceIDs: []string{record.ID}, PrimaryType: record.EvidenceType, EventTypes: []string{record.EvidenceType}, Text: text, SourceText: record.Text,
		ChapterID: record.ChapterID, ChapterIndex: record.ChapterIndex, ChapterTitle: chapterTitle(book, record), Section: record.Section, SectionIndex: record.SectionIndex,
		ParagraphIndex: record.ParagraphIndex, SentenceIndex: record.SentenceIndex, StartOffset: record.StartOffset,
		CharacterIDs: append([]string(nil), record.CharacterIDs...), CharacterNames: append([]string(nil), record.CharacterNames...), Locations: locationTerms(record.NamedEntities, characterNames, record.Text),
		TimeExpressions: times, TimeKind: timeKind, TimeLabel: timeLabel,
		Confidence: record.Confidence, Status: record.Status, Pinned: record.Pinned,
	}
}

func mergeRecord(event *types.StoryTimelineEvent, record types.EvidenceRecord, characterNames map[string]bool) {
	event.EvidenceIDs = appendUnique(event.EvidenceIDs, record.ID)
	event.EventTypes = appendUnique(event.EventTypes, record.EvidenceType)
	if typePriority[record.EvidenceType] > typePriority[event.PrimaryType] {
		event.PrimaryType = record.EvidenceType
	}
	event.CharacterIDs = appendUnique(event.CharacterIDs, record.CharacterIDs...)
	event.CharacterNames = appendUnique(event.CharacterNames, record.CharacterNames...)
	event.TimeExpressions = appendUnique(event.TimeExpressions, cleanTimeExpressions(record.TimeExpressions)...)
	event.Locations = appendUniqueTerms(event.Locations, locationTerms(record.NamedEntities, characterNames, record.Text))
	if record.Confidence > event.Confidence {
		event.Confidence = record.Confidence
	}
	if record.Status == "confirmed" {
		event.Status = "confirmed"
	}
	event.Pinned = event.Pinned || record.Pinned
	event.TimeKind, event.TimeLabel = timeMeaning(event.TimeExpressions)
}

func timeMeaning(expressions []string) (string, string) {
	if len(expressions) == 0 {
		return "manuscript", "Order shown by manuscript placement"
	}
	label := strings.Join(expressions, " · ")
	for _, expression := range expressions {
		if relativeTimeRe.MatchString(expression) {
			return "relative", label + " · relative timing"
		}
	}
	return "anchored", label + " · explicit time reference"
}

func cleanTimeExpressions(expressions []string) []string {
	result := []string{}
	for _, expression := range expressions {
		expression = strings.Join(strings.Fields(expression), " ")
		if match := danglingMeridiemRe.FindStringSubmatch(expression); len(match) == 2 {
			expression = match[1]
		}
		result = appendUnique(result, expression)
	}
	return result
}

func finish(result *types.StoryTimelineResult) {
	chapters := map[int]*types.StoryTimelineChapter{}
	chapterOrder := []int{}
	characters, locations, eventTypes := map[string]*types.StoryTimelineFacet{}, map[string]*types.StoryTimelineFacet{}, map[string]*types.StoryTimelineFacet{}
	for _, event := range result.Events {
		chapter := chapters[event.ChapterIndex]
		if chapter == nil {
			chapter = &types.StoryTimelineChapter{ChapterIndex: event.ChapterIndex, ChapterTitle: event.ChapterTitle}
			chapters[event.ChapterIndex] = chapter
			chapterOrder = append(chapterOrder, event.ChapterIndex)
		}
		chapter.EventCount++
		if event.TimeKind != "manuscript" {
			chapter.ExplicitTimeCount++
			result.ExplicitTimeCount++
		}
		if event.TimeKind == "relative" {
			result.RelativeTimeCount++
		}
		for index, id := range event.CharacterIDs {
			label := id
			if index < len(event.CharacterNames) && event.CharacterNames[index] != "" {
				label = event.CharacterNames[index]
			}
			incrementFacet(characters, id, label)
		}
		for _, location := range event.Locations {
			incrementFacet(locations, strings.ToLower(location.Text), location.Text)
		}
		for _, eventType := range event.EventTypes {
			incrementFacet(eventTypes, eventType, typeLabel(eventType))
		}
	}
	for _, index := range chapterOrder {
		result.Chapters = append(result.Chapters, *chapters[index])
	}
	result.Characters = sortedFacets(characters)
	result.Locations = sortedFacets(locations)
	result.EventTypes = sortedFacets(eventTypes)
}

func incrementFacet(values map[string]*types.StoryTimelineFacet, id, label string) {
	if id == "" {
		return
	}
	if values[id] == nil {
		values[id] = &types.StoryTimelineFacet{ID: id, Label: label}
	}
	values[id].Count++
}

func sortedFacets(values map[string]*types.StoryTimelineFacet) []types.StoryTimelineFacet {
	result := make([]types.StoryTimelineFacet, 0, len(values))
	for _, value := range values {
		result = append(result, *value)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Count != result[j].Count {
			return result[i].Count > result[j].Count
		}
		return strings.ToLower(result[i].Label) < strings.ToLower(result[j].Label)
	})
	return result
}

func locationTerms(terms []types.EvidenceTerm, characterNames map[string]bool, text string) []types.EvidenceTerm {
	result := []types.EvidenceTerm{}
	for _, term := range terms {
		cleaned, ok := cleanLocationTerm(term.Text)
		if !ok {
			continue
		}
		label := strings.ToLower(term.Label)
		if characterNames[normalize(cleaned)] {
			continue
		}
		if (label == "gpe" || label == "loc" || label == "location" || label == "facility" || label == "fac" || label == "place") && hasLocationGrammar(text, cleaned) {
			term.Text = cleaned
			result = appendUniqueTerms(result, []types.EvidenceTerm{term})
		}
	}
	return result
}

func cleanLocationTerm(value string) (string, bool) {
	if strings.ContainsAny(value, "\r\n") || strings.ContainsAny(value, "—–") {
		return "", false
	}
	value = strings.TrimSpace(value)
	if len(value) < 2 || strings.HasSuffix(strings.ToLower(value), "'s") || strings.HasSuffix(strings.ToLower(value), "’s") {
		return "", false
	}
	words := strings.Fields(value)
	if len(words) == 0 || len(words) > 4 {
		return "", false
	}
	timeWords := map[string]bool{"monday": true, "tuesday": true, "wednesday": true, "thursday": true, "friday": true, "saturday": true, "sunday": true, "morning": true, "afternoon": true, "evening": true, "night": true}
	for len(words) > 1 && timeWords[strings.ToLower(strings.Trim(words[len(words)-1], ".,!?\"“”"))] {
		words = words[:len(words)-1]
	}
	value = strings.Join(words, " ")
	upper := true
	for _, letter := range value {
		if letter >= 'a' && letter <= 'z' {
			upper = false
			break
		}
	}
	if upper || strings.EqualFold(value, "Chinese") || strings.EqualFold(value, "English") {
		return "", false
	}
	return value, true
}

func hasLocationGrammar(text, term string) bool {
	pattern := `(?i)\b(?:at|in|into|inside|outside|from|to|toward|towards|through|under|beneath|near|behind|around|across)\s+(?:the\s+)?(?:[\pL\pN'’.-]+\s+){0,2}` + regexp.QuoteMeta(term) + `\b`
	return regexp.MustCompile(pattern).MatchString(text)
}

func characterNameSet(book types.BookData) map[string]bool {
	result := map[string]bool{}
	for _, character := range book.StoryBible.Characters {
		result[normalize(character.Name)] = true
		for _, alias := range character.Aliases {
			result[normalize(alias)] = true
		}
	}
	return result
}

func appendUnique(values []string, additions ...string) []string {
	seen := map[string]bool{}
	for _, value := range values {
		seen[value] = true
	}
	for _, value := range additions {
		if value != "" && !seen[value] {
			seen[value] = true
			values = append(values, value)
		}
	}
	return values
}

func appendUniqueTerms(values, additions []types.EvidenceTerm) []types.EvidenceTerm {
	seen := map[string]bool{}
	for _, value := range values {
		seen[strings.ToLower(value.Label+"\x00"+value.Text)] = true
	}
	for _, value := range additions {
		key := strings.ToLower(value.Label + "\x00" + value.Text)
		if value.Text != "" && !seen[key] {
			seen[key] = true
			values = append(values, value)
		}
	}
	return values
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
	return "Chapter " + intString(record.ChapterIndex+1)
}

func eventLess(a, b types.StoryTimelineEvent) bool {
	if a.ChapterIndex != b.ChapterIndex {
		return a.ChapterIndex < b.ChapterIndex
	}
	if a.StartOffset != b.StartOffset {
		return a.StartOffset < b.StartOffset
	}
	return a.ID < b.ID
}

func typeLabel(value string) string {
	return strings.Title(strings.ReplaceAll(value, "_", " "))
}

func normalize(value string) string { return strings.Join(strings.Fields(strings.ToLower(value)), " ") }

func intString(value int) string {
	if value == 0 {
		return "0"
	}
	digits := [24]byte{}
	position := len(digits)
	negative := value < 0
	if negative {
		value = -value
	}
	for value > 0 {
		position--
		digits[position] = byte('0' + value%10)
		value /= 10
	}
	if negative {
		position--
		digits[position] = '-'
	}
	return string(digits[position:])
}
