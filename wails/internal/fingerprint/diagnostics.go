package fingerprint

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"draftline/internal/types"
)

var (
	leaveCueRe  = regexp.MustCompile(`(?i)\b(?:left|exited|departed|walked out|went out)\b`)
	aloneCueRe  = regexp.MustCompile(`(?i)\b(?:alone|by (?:him|her|them)self|only [A-Z][\pL'-]+ remained)\b`)
	directionRe = regexp.MustCompile(`(?i)\b([A-Z][\pL'-]+(?:\s+[A-Z][\pL'-]+)?)\s+(?:was|is|lay|stood)?\s*(north|south|east|west)\s+of\s+([A-Z0-9][\pL\pN .'-]+)`)
	nameFactRe  = regexp.MustCompile(`(?i)\b([A-Z][\pL'-]+)'s first name (?:was|is) ([A-Z][\pL'-]+)\b`)
	htmlTagRe   = regexp.MustCompile(`<[^>]+>`)
)

func buildDiagnostics(book *types.BookData, model *types.StoryFingerprint, records []types.EvidenceRecord, supportEvents []types.FingerprintEvent) []types.FingerprintDiagnostic {
	result := []types.FingerprintDiagnostic{}
	result = append(result, attributeDiagnostics(model.States, supportEvents)...)
	result = append(result, canonDiagnostics(model.AuthorModel.Canon, model.States, supportEvents)...)
	result = append(result, presenceDiagnostics(supportEvents, records)...)
	result = append(result, orphanCharacterDiagnostics(records, supportEvents)...)
	result = append(result, directionDiagnostics(records)...)
	result = append(result, identityDiagnostics(records)...)
	result = append(result, duplicateChapterDiagnostics(book)...)
	for _, thread := range model.Threads {
		if thread.State == "dormant" && thread.Kind != "temporal" {
			result = append(result, types.FingerprintDiagnostic{ID: stableID("diagnostic", thread.ID, "dormant"), Kind: "dormant_thread", Severity: "review", Title: "A story obligation may have gone quiet", Detail: fmt.Sprintf("%s has no matching evidence for %d chapters. It may be intentionally deferred.", thread.Label, thread.DormantChapters), EventIDs: clone(thread.EventIDs), EvidenceIDs: clone(thread.EvidenceIDs), Confidence: thread.Confidence})
		}
	}
	return dedupeDiagnostics(result)
}

func attributeDiagnostics(states []types.StoryStateInterval, events []types.FingerprintEvent) []types.FingerprintDiagnostic {
	byKey := map[string][]types.StoryStateInterval{}
	for _, state := range states {
		if state.Kind == "attribute" {
			byKey[state.EntityID+"\x00"+state.Kind] = append(byKey[state.EntityID+"\x00"+state.Kind], state)
		}
	}
	result := []types.FingerprintDiagnostic{}
	for _, values := range byKey {
		for i := 0; i < len(values); i++ {
			for j := i + 1; j < len(values); j++ {
				if strings.EqualFold(values[i].Value, values[j].Value) {
					continue
				}
				result = append(result, diagnosticFromStates("attribute_conflict", "A persistent character detail conflicts", fmt.Sprintf("%s is established as %s and later as %s.", values[i].EntityName, values[i].Value, values[j].Value), values[i], values[j], events, .94))
				break
			}
		}
	}
	return result
}

func canonDiagnostics(rules []types.CanonRule, states []types.StoryStateInterval, events []types.FingerprintEvent) []types.FingerprintDiagnostic {
	result := []types.FingerprintDiagnostic{}
	for _, rule := range rules {
		for _, state := range states {
			if !strings.EqualFold(rule.Subject, state.EntityName) && rule.Subject != state.EntityID {
				continue
			}
			if rule.Predicate != "" && !strings.Contains(strings.ToLower(state.Kind+" "+state.Value), strings.ToLower(rule.Predicate)) {
				continue
			}
			conflicts := rule.Polarity != "negative" && rule.Object != "" && !strings.Contains(strings.ToLower(state.Value), strings.ToLower(rule.Object))
			if rule.Polarity == "negative" {
				conflicts = strings.Contains(strings.ToLower(state.Value), strings.ToLower(rule.Object))
			}
			if conflicts {
				event := findEvent(state.StartEventID, events)
				result = append(result, types.FingerprintDiagnostic{ID: stableID("diagnostic", rule.ID, state.ID), Kind: "canon_conflict", Severity: "review", Title: "The manuscript may break declared canon", Detail: fmt.Sprintf("Canon says %s %s %s; this passage establishes %s.", rule.Subject, rule.Predicate, rule.Object, state.Value), EventIDs: []string{state.StartEventID}, EvidenceIDs: clone(state.EvidenceIDs), ChapterIndices: eventChapters(event), Confidence: .95})
			}
		}
	}
	return result
}

func presenceDiagnostics(events []types.FingerprintEvent, records []types.EvidenceRecord) []types.FingerprintDiagnostic {
	recordByID := map[string]types.EvidenceRecord{}
	for _, record := range records {
		recordByID[record.ID] = record
	}
	active := map[string]map[string]string{}
	result := []types.FingerprintDiagnostic{}
	for _, event := range events {
		key := fmt.Sprintf("%d\x00%s", event.ChapterIndex, event.ContextID)
		if active[key] == nil {
			active[key] = map[string]string{}
		}
		text := eventSourceText(event, recordByID)
		if leaveCueRe.MatchString(text) {
			for index, id := range event.CharacterIDs {
				name := id
				if index < len(event.CharacterNames) {
					name = event.CharacterNames[index]
				}
				delete(active[key], id)
				_ = name
			}
		} else if presenceRe.MatchString(text) || containsKind(event.Kinds, "introduction") || containsKind(event.Kinds, "transition") {
			for index, id := range event.CharacterIDs {
				name := id
				if index < len(event.CharacterNames) {
					name = event.CharacterNames[index]
				}
				active[key][id] = name
			}
		}
		if aloneCueRe.MatchString(text) {
			named := map[string]bool{}
			for _, id := range event.CharacterIDs {
				named[id] = true
			}
			stranded := []string{}
			for id, name := range active[key] {
				if !named[id] {
					stranded = append(stranded, name)
				}
			}
			if len(stranded) > 0 {
				sort.Strings(stranded)
				result = append(result, types.FingerprintDiagnostic{ID: stableID("diagnostic", event.ID, "unclosed-presence", strings.Join(stranded, "|")), Kind: "unclosed_presence", Severity: "review", Title: "Someone may still be present", Detail: "The passage says a character is alone, but no indexed exit accounts for " + strings.Join(stranded, ", ") + ". This may be an omitted exit, an accidental introduction, or an extraction error.", EventIDs: []string{event.ID}, EvidenceIDs: clone(event.EvidenceIDs), ChapterIndices: []int{event.ChapterIndex}, Confidence: .82})
			}
		}
	}
	return result
}

func orphanCharacterDiagnostics(records []types.EvidenceRecord, events []types.FingerprintEvent) []types.FingerprintDiagnostic {
	counts := map[string]int{}
	names := map[string]string{}
	first := map[string]types.EvidenceRecord{}
	for _, record := range records {
		for index, id := range record.CharacterIDs {
			counts[id]++
			if index < len(record.CharacterNames) {
				names[id] = record.CharacterNames[index]
			}
			if _, exists := first[id]; !exists {
				first[id] = record
			}
		}
	}
	result := []types.FingerprintDiagnostic{}
	for id, count := range counts {
		record := first[id]
		if count == 1 && record.EvidenceType == "introduction" {
			result = append(result, types.FingerprintDiagnostic{ID: stableID("diagnostic", id, record.ID, "orphan-character"), Kind: "orphan_character", Severity: "review", Title: names[id] + " is introduced but never referenced again", Detail: "A named character enters the story once without later presence, reference, or resolution. Confirm that this is an intentional one-off.", EvidenceIDs: []string{record.ID}, ChapterIndices: []int{record.ChapterIndex}, Confidence: .88})
		}
	}
	return result
}

type directionFact struct {
	subject, direction, object string
	record                     types.EvidenceRecord
}

func directionDiagnostics(records []types.EvidenceRecord) []types.FingerprintDiagnostic {
	byPair := map[string][]directionFact{}
	for _, record := range records {
		match := directionRe.FindStringSubmatch(record.Text)
		if len(match) != 4 {
			continue
		}
		fact := directionFact{strings.ToLower(match[1]), strings.ToLower(match[2]), strings.ToLower(strings.TrimSpace(match[3])), record}
		byPair[fact.subject+"\x00"+fact.object] = append(byPair[fact.subject+"\x00"+fact.object], fact)
	}
	result := []types.FingerprintDiagnostic{}
	for _, facts := range byPair {
		for i := 0; i < len(facts); i++ {
			for j := i + 1; j < len(facts); j++ {
				if oppositeDirection(facts[i].direction) == facts[j].direction {
					result = append(result, pairDiagnostic("direction_conflict", "A location direction may conflict", facts[i].record, facts[j].record, .91))
				}
			}
		}
	}
	return result
}

func identityDiagnostics(records []types.EvidenceRecord) []types.FingerprintDiagnostic {
	bySubject := map[string][]struct {
		name   string
		record types.EvidenceRecord
	}{}
	for _, record := range records {
		match := nameFactRe.FindStringSubmatch(record.Text)
		if len(match) == 3 {
			key := strings.ToLower(match[1])
			bySubject[key] = append(bySubject[key], struct {
				name   string
				record types.EvidenceRecord
			}{strings.ToLower(match[2]), record})
		}
	}
	result := []types.FingerprintDiagnostic{}
	for _, facts := range bySubject {
		for i := 0; i < len(facts); i++ {
			for j := i + 1; j < len(facts); j++ {
				if facts[i].name != facts[j].name {
					result = append(result, pairDiagnostic("identity_conflict", "A character's established name conflicts", facts[i].record, facts[j].record, .98))
				}
			}
		}
	}
	return result
}

func duplicateChapterDiagnostics(book *types.BookData) []types.FingerprintDiagnostic {
	chapters := append(append([]types.ChapterItem{}, book.FrontMatter...), book.Body...)
	chapters = append(chapters, book.BackMatter...)
	result := []types.FingerprintDiagnostic{}
	for i := 0; i < len(chapters); i++ {
		leftText := stripMarkup(chapters[i].Content)
		left := shingles(leftText)
		if len(strings.Fields(leftText)) < 30 {
			continue
		}
		for j := i + 1; j < len(chapters); j++ {
			rightText := stripMarkup(chapters[j].Content)
			right := shingles(rightText)
			score := setSimilarity(left, right)
			if strings.Contains(leftText, rightText) || strings.Contains(rightText, leftText) {
				shorter, longer := len(leftText), len(rightText)
				if shorter > longer {
					shorter, longer = longer, shorter
				}
				if longer > 0 && float64(shorter)/float64(longer) > score {
					score = float64(shorter) / float64(longer)
				}
			}
			if score >= .9 {
				result = append(result, types.FingerprintDiagnostic{ID: stableID("diagnostic", chapters[i].ID, chapters[j].ID, "near-duplicate"), Kind: "near_duplicate_chapter", Severity: "review", Title: "Two chapters are nearly identical", Detail: fmt.Sprintf("%s and %s share %.0f%% of their normalized phrasing. This may be an intentional loop or reset and needs a temporal/reality decision.", chapters[i].Title, chapters[j].Title, score*100), ChapterIndices: []int{i, j}, Confidence: score})
			}
		}
	}
	return result
}

func correctionDiagnostics(corrections []types.FingerprintCorrection) []types.FingerprintDiagnostic {
	result := []types.FingerprintDiagnostic{}
	for _, correction := range corrections {
		if correction.Status == "orphaned" {
			result = append(result, types.FingerprintDiagnostic{ID: stableID("diagnostic", correction.ID, "orphaned"), Kind: "orphaned_correction", Severity: "review", Title: "A manual story correction lost its source", Detail: "The corrected event or context no longer exists after reanalysis. Reattach or remove the correction.", EvidenceIDs: clone(correction.EvidenceIDs), Confidence: 1})
		}
	}
	return result
}

func diagnosticFromStates(kind, title, detail string, left, right types.StoryStateInterval, events []types.FingerprintEvent, confidence float64) types.FingerprintDiagnostic {
	a, b := findEvent(left.StartEventID, events), findEvent(right.StartEventID, events)
	return types.FingerprintDiagnostic{ID: stableID("diagnostic", left.ID, right.ID, kind), Kind: kind, Severity: "review", Title: title, Detail: detail, EventIDs: []string{left.StartEventID, right.StartEventID}, EvidenceIDs: appendUnique(clone(left.EvidenceIDs), right.EvidenceIDs...), ChapterIndices: appendUniqueInt(eventChapters(a), eventChapters(b)...), Confidence: confidence}
}
func pairDiagnostic(kind, title string, left, right types.EvidenceRecord, confidence float64) types.FingerprintDiagnostic {
	return types.FingerprintDiagnostic{ID: stableID("diagnostic", left.ID, right.ID, kind), Kind: kind, Severity: "review", Title: title, Detail: "Two source passages establish incompatible values. Review their story-time and reality contexts before changing prose.", EvidenceIDs: []string{left.ID, right.ID}, ChapterIndices: []int{left.ChapterIndex, right.ChapterIndex}, Confidence: confidence}
}
func findEvent(id string, events []types.FingerprintEvent) *types.FingerprintEvent {
	for index := range events {
		if events[index].ID == id {
			return &events[index]
		}
	}
	return nil
}
func eventChapters(event *types.FingerprintEvent) []int {
	if event == nil {
		return nil
	}
	return []int{event.ChapterIndex}
}
func containsKind(values []string, kind string) bool {
	for _, value := range values {
		if value == kind {
			return true
		}
	}
	return false
}
func oppositeDirection(value string) string {
	return map[string]string{"north": "south", "south": "north", "east": "west", "west": "east"}[value]
}
func stripMarkup(text string) string {
	return strings.ToLower(strings.Join(strings.Fields(htmlTagRe.ReplaceAllString(text, " ")), " "))
}
func shingles(text string) map[string]bool {
	words := strings.Fields(text)
	result := map[string]bool{}
	for i := 0; i+4 < len(words); i++ {
		result[strings.Join(words[i:i+5], " ")] = true
	}
	return result
}
func setSimilarity(left, right map[string]bool) float64 {
	common, union := 0, len(left)
	for value := range right {
		if left[value] {
			common++
		} else {
			union++
		}
	}
	if union == 0 {
		return 0
	}
	return float64(common) / float64(union)
}
func appendUniqueInt(values []int, additions ...int) []int {
	seen := map[int]bool{}
	for _, v := range values {
		seen[v] = true
	}
	for _, v := range additions {
		if !seen[v] {
			values = append(values, v)
			seen[v] = true
		}
	}
	return values
}
func dedupeDiagnostics(values []types.FingerprintDiagnostic) []types.FingerprintDiagnostic {
	result := []types.FingerprintDiagnostic{}
	seen := map[string]bool{}
	for _, value := range values {
		if !seen[value.ID] {
			result = append(result, value)
			seen[value.ID] = true
		}
	}
	return result
}
