package fingerprint

import (
	"regexp"
	"sort"
	"strings"

	"draftline/internal/types"
)

var (
	commitmentCueRe = regexp.MustCompile(`(?i)\b(?:will|would|must|need(?:s|ed)? to|promised|swore|keep an eye|look into|find out|figure out|investigate)\b`)
	threatCueRe     = regexp.MustCompile(`(?i)\b(?:threatened|vowed|kill|destroy|stop them|coming for)\b`)
	resolutionCueRe = regexp.MustCompile(`(?i)\b(?:found|discovered|revealed|answered|learned|confirmed|handed|delivered|returned|rescued|escaped|defeated|killed|stopped|closed|solved|explained)\b`)
	escalationCueRe = regexp.MustCompile(`(?i)\b(?:but|however|failed|blocked|worse|another|deeper|threat|attack|urgent)\b`)
)

const (
	maxAutoThreads      = 5000
	maxThreadCandidates = 4096
)

func buildThreads(events []types.FingerprintEvent, records []types.EvidenceRecord, contexts []types.StoryContext) []types.StoryThread {
	recordByID := map[string]types.EvidenceRecord{}
	for _, record := range records {
		recordByID[record.ID] = record
	}
	eventTokens, postings := indexThreadEvents(events, recordByID)
	threads := contextThreads(events, contexts)
	for eventIndex, event := range events {
		if len(threads) >= maxAutoThreads {
			break
		}
		text := eventSourceText(event, recordByID)
		kind, opens := obligationKind(text, event)
		if !opens {
			continue
		}
		label := obligationLabel(text)
		thread := types.StoryThread{ID: stableID("thread", event.ID, kind, strings.ToLower(label)), Label: label, Kind: kind, State: "seeded", OpenedByEventID: event.ID, EventIDs: []string{event.ID}, EvidenceIDs: clone(event.EvidenceIDs), EntityIDs: clone(event.CharacterIDs), Confidence: event.Confidence, Source: "auto"}
		advanceThread(&thread, eventIndex, events, recordByID, eventTokens, postings)
		threads = append(threads, thread)
	}
	markConvergence(threads)
	sort.SliceStable(threads, func(i, j int) bool { return threadStart(threads[i], events) < threadStart(threads[j], events) })
	return threads
}

func contextThreads(events []types.FingerprintEvent, contexts []types.StoryContext) []types.StoryThread {
	result := []types.StoryThread{}
	for _, context := range contexts {
		if context.Kind == "primary" || context.Source == "author" && context.Kind == "unknown" {
			continue
		}
		eventIDs, evidenceIDs := []string{}, []string{}
		first := ""
		for _, event := range events {
			if event.ContextID == context.ID {
				if first == "" {
					first = event.ID
				}
				eventIDs = append(eventIDs, event.ID)
				evidenceIDs = appendUnique(evidenceIDs, event.EvidenceIDs...)
			}
		}
		if len(eventIDs) > 0 {
			result = append(result, types.StoryThread{ID: stableID("thread", context.ID), Label: context.Label, Kind: "temporal", State: "active", OpenedByEventID: first, EventIDs: eventIDs, EvidenceIDs: evidenceIDs, Confidence: context.Confidence, Source: context.Source})
		}
	}
	return result
}

func obligationKind(text string, event types.FingerprintEvent) (string, bool) {
	trimmed := strings.TrimSpace(text)
	if strings.Contains(trimmed, "?") {
		return "question", true
	}
	if threatCueRe.MatchString(trimmed) {
		return "threat", true
	}
	if commitmentCueRe.MatchString(trimmed) {
		return "commitment", true
	}
	for _, kind := range event.Kinds {
		if kind == "discovery" && (strings.Contains(strings.ToLower(text), "unknown") || strings.Contains(strings.ToLower(text), "mystery")) {
			return "mystery", true
		}
	}
	return "", false
}

func advanceThread(thread *types.StoryThread, opened int, events []types.FingerprintEvent, records map[string]types.EvidenceRecord, eventTokens []map[string]bool, postings map[string][]int) {
	seedTokens := meaningfulTokens(thread.Label)
	for _, name := range events[opened].CharacterNames {
		for token := range meaningfulTokens(name) {
			delete(seedTokens, token)
		}
	}
	lastChapter := events[opened].ChapterIndex
	// The rarest meaningful seed term is the best deterministic discriminator
	// and prevents common characters/verbs from turning this into O(n²).
	var candidates []int
	for token := range seedTokens {
		future := []int{}
		for _, index := range postings[token] {
			if index > opened {
				future = append(future, index)
			}
		}
		if len(future) > 0 && (candidates == nil || len(future) < len(candidates)) {
			candidates = future
		}
	}
	filtered := make([]int, 0, min(len(candidates), maxThreadCandidates))
	for _, index := range candidates {
		filtered = append(filtered, index)
	}
	if len(filtered) > maxThreadCandidates {
		half := maxThreadCandidates / 2
		filtered = append(append([]int(nil), filtered[:half]...), filtered[len(filtered)-half:]...)
	}
	candidates = filtered
	for _, index := range candidates {
		event := events[index]
		score := tokenOverlap(seedTokens, eventTokens[index])
		if overlap(thread.EntityIDs, event.CharacterIDs) {
			score += .18
		}
		if score < .22 {
			continue
		}
		thread.EventIDs = appendUnique(thread.EventIDs, event.ID)
		thread.EvidenceIDs = appendUnique(thread.EvidenceIDs, event.EvidenceIDs...)
		thread.EntityIDs = appendUnique(thread.EntityIDs, event.CharacterIDs...)
		lastChapter = event.ChapterIndex
		if resolutionCueRe.MatchString(eventSourceText(event, records)) && score >= .35 {
			thread.Resolution = 1
			thread.State = "resolved"
			thread.ResolvedByEventID = event.ID
			return
		}
		if escalationCueRe.MatchString(eventSourceText(event, records)) {
			thread.State = "escalating"
		} else {
			thread.State = "active"
			if thread.Resolution < .5 {
				thread.Resolution = .5
			}
		}
	}
	gap := events[len(events)-1].ChapterIndex - lastChapter
	if gap >= 3 {
		thread.State = "dormant"
		thread.DormantChapters = gap
	}
}

func indexThreadEvents(events []types.FingerprintEvent, records map[string]types.EvidenceRecord) ([]map[string]bool, map[string][]int) {
	tokens := make([]map[string]bool, len(events))
	postings := map[string][]int{}
	for index, event := range events {
		tokens[index] = meaningfulTokens(eventSourceText(event, records))
		for token := range tokens[index] {
			postings[token] = append(postings[token], index)
		}
	}
	return tokens, postings
}

func markConvergence(threads []types.StoryThread) {
	byEvent := map[string][]int{}
	for index, thread := range threads {
		for _, eventID := range thread.EventIDs {
			byEvent[eventID] = append(byEvent[eventID], index)
		}
	}
	for _, indices := range byEvent {
		if len(indices) < 2 {
			continue
		}
		for _, index := range indices {
			for _, other := range indices {
				if other != index {
					threads[index].ParentIDs = appendUnique(threads[index].ParentIDs, threads[other].ID)
				}
			}
			if threads[index].State != "resolved" {
				threads[index].State = "converging"
			}
		}
	}
}

func eventSourceText(event types.FingerprintEvent, records map[string]types.EvidenceRecord) string {
	parts := []string{}
	for _, id := range event.EvidenceIDs {
		parts = append(parts, records[id].Text)
	}
	return strings.Join(parts, " ")
}

func obligationLabel(text string) string {
	text = strings.TrimSpace(strings.Join(strings.Fields(text), " "))
	if len([]rune(text)) > 140 {
		text = string([]rune(text)[:137]) + "…"
	}
	return text
}

func meaningfulTokens(text string) map[string]bool {
	result := map[string]bool{}
	stop := map[string]bool{"the": true, "a": true, "an": true, "and": true, "or": true, "but": true, "he": true, "she": true, "they": true, "it": true, "to": true, "of": true, "in": true, "on": true, "for": true, "with": true, "was": true, "is": true, "had": true, "have": true, "would": true, "will": true, "said": true}
	for _, raw := range strings.Fields(strings.ToLower(text)) {
		token := strings.Trim(raw, "\"'“”‘’.,!?;:()[]{}")
		if len(token) > 2 && !stop[token] {
			result[token] = true
		}
	}
	return result
}

func tokenOverlap(left, right map[string]bool) float64 {
	if len(left) == 0 || len(right) == 0 {
		return 0
	}
	common := 0
	for token := range left {
		if right[token] {
			common++
		}
	}
	denominator := len(left)
	if len(right) < denominator {
		denominator = len(right)
	}
	return float64(common) / float64(denominator)
}

func threadStart(thread types.StoryThread, events []types.FingerprintEvent) int {
	for _, event := range events {
		if event.ID == thread.OpenedByEventID {
			return event.NarrativeOrder
		}
	}
	return len(events)
}
