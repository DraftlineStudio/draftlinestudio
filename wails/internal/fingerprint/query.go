package fingerprint

import (
	"fmt"
	"sort"
	"strings"

	"draftline/internal/types"
)

// Query answers mechanical story questions exclusively from persisted,
// source-backed fingerprint records. It never invents connective prose.
func Query(book types.BookData, request types.FingerprintQueryRequest) types.FingerprintQueryAnswer {
	answer := types.FingerprintQueryAnswer{Success: true, Confidence: 0}
	model := book.Analysis.Fingerprint
	query := strings.TrimSpace(request.Query)
	if model == nil {
		answer.Success = false
		answer.Error = "Story Fingerprint analysis has not run yet."
		return answer
	}
	if query == "" {
		answer.Success = false
		answer.Error = "Enter a story question or detail."
		return answer
	}
	limit := request.Limit
	if limit <= 0 || limit > 20 {
		limit = 8
	}
	lower := strings.ToLower(query)
	// Querying uses the complete manuscript-memory corpus. This projection is
	// local to the answer and is never exposed as roadmap or graph events.
	corpusEvents := corpusQueryEvents(book, model.Fingerprints)
	switch {
	case strings.Contains(lower, "voice") || strings.Contains(lower, "dialect") || strings.Contains(lower, "speak") || strings.Contains(lower, "vernacular"):
		answer.Interpretation = "character voice and speaking traits"
		answer.Voices = rankedVoices(model.Voices, query, limit)
		if len(answer.Voices) > 0 {
			profile := answer.Voices[0]
			answer.Answer = fmt.Sprintf("%s has %d attributed dialogue samples averaging %.1f words; %.1f%% contractions.", profile.CharacterName, profile.SampleCount, profile.AverageWords, profile.ContractionPercent)
			answer.Confidence = profile.Confidence
		}
	case strings.Contains(lower, "open thread") || strings.Contains(lower, "unresolved") || strings.Contains(lower, "loose thread"):
		answer.Interpretation = "unresolved story obligations"
		for _, thread := range model.Threads {
			if thread.State != "resolved" && thread.State != "abandoned" {
				answer.Threads = append(answer.Threads, thread)
				answer.EvidenceIDs = appendUnique(answer.EvidenceIDs, thread.EvidenceIDs...)
			}
		}
		if len(answer.Threads) > limit {
			answer.Threads = answer.Threads[:limit]
		}
		answer.Answer = fmt.Sprintf("%d unresolved or deferred story obligations match this view.", len(answer.Threads))
		answer.Confidence = .95
	case strings.Contains(lower, "continuity") || strings.Contains(lower, "contradiction") || strings.Contains(lower, "plot hole"):
		answer.Interpretation = "continuity questions"
		answer.Diagnostics = rankedDiagnostics(model.Diagnostics, query, limit)
		for _, diagnostic := range answer.Diagnostics {
			answer.EvidenceIDs = appendUnique(answer.EvidenceIDs, diagnostic.EvidenceIDs...)
		}
		answer.Answer = fmt.Sprintf("%d source-backed continuity questions match.", len(answer.Diagnostics))
		answer.Confidence = .9
	case strings.Contains(lower, " know") || strings.HasPrefix(lower, "what does") || strings.HasPrefix(lower, "what did") && strings.Contains(lower, "learn"):
		answer.Interpretation = "character knowledge and belief"
		answer.States = rankedStates(model.States, query, []string{"knowledge", "belief"}, limit)
		for _, state := range answer.States {
			answer.EvidenceIDs = appendUnique(answer.EvidenceIDs, state.EvidenceIDs...)
		}
		answer.Answer, answer.Confidence = stateAnswer(answer.States)
	case strings.HasPrefix(lower, "where") || strings.Contains(lower, " location"):
		answer.Interpretation = "physical presence and location"
		answer.States = rankedStates(model.States, query, []string{"presence"}, limit)
		answer.Events = rankedEvents(corpusEvents, book, query, limit)
		answer.Answer, answer.Confidence = eventAnswer(answer.Events)
	case strings.HasPrefix(lower, "when"):
		answer.Interpretation = "in-universe chronology"
		answer.Events = rankedEvents(corpusEvents, book, query, limit)
		sort.SliceStable(answer.Events, func(i, j int) bool { return storyEventLess(answer.Events[i], answer.Events[j]) })
		answer.Answer, answer.Confidence = eventAnswer(answer.Events)
	default:
		answer.Interpretation = "story evidence"
		answer.Events = queryAfterAnchor(corpusEvents, book, query, limit)
		answer.Threads = rankedThreads(model.Threads, query, min(limit, 4))
		answer.Answer, answer.Confidence = trailAnswer(answer.Events)
	}
	for _, event := range answer.Events {
		answer.EvidenceIDs = appendUnique(answer.EvidenceIDs, event.EvidenceIDs...)
	}
	if answer.Answer == "" {
		answer.Answer = "The current fingerprint does not contain enough matching evidence to answer that confidently."
	}
	return answer
}

func corpusQueryEvents(book types.BookData, fingerprints []types.ManuscriptFingerprint) []types.FingerprintEvent {
	records := evidenceRecordMap(nil)
	if book.Analysis.Evidence != nil {
		records = evidenceRecordMap(book.Analysis.Evidence.Records)
	}
	result := make([]types.FingerprintEvent, 0, len(fingerprints))
	for order, fingerprint := range fingerprints {
		if len(fingerprint.EvidenceIDs) == 0 {
			continue
		}
		record, exists := records[fingerprint.EvidenceIDs[0]]
		if !exists {
			continue
		}
		locations, objects := eventTerms(record)
		characterIDs, characterNames := []string{}, []string{}
		for _, participant := range fingerprint.Participants {
			characterIDs = appendUnique(characterIDs, participant.EntityID)
			characterNames = appendUnique(characterNames, participant.EntityName)
		}
		result = append(result, types.FingerprintEvent{
			ID: fingerprint.ID, Summary: fingerprint.Statement, EvidenceIDs: clone(fingerprint.EvidenceIDs), AssertionIDs: clone(fingerprint.AssertionIDs),
			CharacterIDs: characterIDs, CharacterNames: characterNames, Locations: locations, Objects: objects, Kinds: []string{fingerprint.Kind},
			ContextID: fingerprint.Scope.ID, StoryTime: fingerprint.Temporal, ChapterID: record.ChapterID, ChapterIndex: record.ChapterIndex,
			ChapterTitle: chapterTitle(&book, record), ParagraphIndex: record.ParagraphIndex, StartOffset: record.StartOffset,
			NarrativeOrder: order, Confidence: fingerprint.Confidence, Status: fingerprint.Status,
		})
	}
	return result
}

type scoredVoice struct {
	score float64
	voice types.CharacterVoiceProfile
}

func rankedVoices(voices []types.CharacterVoiceProfile, query string, limit int) []types.CharacterVoiceProfile {
	tokens := meaningfulTokens(query)
	items := []scoredVoice{}
	for _, voice := range voices {
		text := voice.CharacterName
		if voice.AuthorNotes != nil {
			text += " " + voice.AuthorNotes.Dialect + " " + strings.Join(voice.AuthorNotes.Vernacular, " ") + " " + strings.Join(voice.AuthorNotes.SpeakingTraits, " ")
		}
		for _, signal := range voice.DialectSignals {
			text += " " + signal.Label
		}
		score := tokenOverlap(tokens, meaningfulTokens(text))
		if score > .05 {
			items = append(items, scoredVoice{score, voice})
		}
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].score > items[j].score })
	result := []types.CharacterVoiceProfile{}
	for _, item := range items {
		result = append(result, item.voice)
		if len(result) >= limit {
			break
		}
	}
	return result
}

type scoredEvent struct {
	score float64
	event types.FingerprintEvent
}

func rankedEvents(events []types.FingerprintEvent, book types.BookData, query string, limit int) []types.FingerprintEvent {
	tokens := meaningfulTokens(query)
	scored := []scoredEvent{}
	records := map[string]string{}
	if book.Analysis.Evidence != nil {
		for _, record := range book.Analysis.Evidence.Records {
			records[record.ID] = record.Text
		}
	}
	for _, event := range events {
		source := ""
		for _, id := range event.EvidenceIDs {
			source += " " + records[id]
		}
		haystack := meaningfulTokens(event.Summary + " " + source + " " + event.ChapterTitle + " " + strings.Join(event.CharacterNames, " "))
		score := tokenOverlap(tokens, haystack)
		for token := range tokens {
			for _, term := range append(event.Locations, event.Objects...) {
				if strings.Contains(strings.ToLower(term.Text), token) {
					score += .15
				}
			}
		}
		if score > .05 {
			scored = append(scored, scoredEvent{score + event.Importance*.08, event})
		}
	}
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score == scored[j].score {
			return scored[i].event.NarrativeOrder < scored[j].event.NarrativeOrder
		}
		return scored[i].score > scored[j].score
	})
	result := []types.FingerprintEvent{}
	for _, item := range scored {
		result = append(result, item.event)
		if len(result) >= limit {
			break
		}
	}
	return result
}

func queryAfterAnchor(events []types.FingerprintEvent, book types.BookData, query string, limit int) []types.FingerprintEvent {
	lower := strings.ToLower(query)
	index := strings.Index(lower, " after ")
	if index < 0 {
		return rankedEvents(events, book, query, limit)
	}
	subject, anchor := strings.TrimSpace(query[:index]), strings.TrimSpace(query[index+7:])
	anchors := rankedEvents(events, book, anchor, 1)
	if len(anchors) == 0 {
		return rankedEvents(events, book, query, limit)
	}
	candidates := []types.FingerprintEvent{}
	for _, event := range events {
		if event.NarrativeOrder > anchors[0].NarrativeOrder {
			candidates = append(candidates, event)
		}
	}
	return rankedEvents(candidates, book, subject, limit)
}

type scoredState struct {
	score float64
	state types.StoryStateInterval
}

func rankedStates(states []types.StoryStateInterval, query string, kinds []string, limit int) []types.StoryStateInterval {
	allowed := map[string]bool{}
	for _, kind := range kinds {
		allowed[kind] = true
	}
	tokens := meaningfulTokens(query)
	scored := []scoredState{}
	for _, state := range states {
		if !allowed[state.Kind] {
			continue
		}
		score := tokenOverlap(tokens, meaningfulTokens(state.EntityName+" "+state.Value+" "+state.Qualifier))
		if score > .05 {
			scored = append(scored, scoredState{score, state})
		}
	}
	sort.SliceStable(scored, func(i, j int) bool { return scored[i].score > scored[j].score })
	result := []types.StoryStateInterval{}
	for _, item := range scored {
		result = append(result, item.state)
		if len(result) >= limit {
			break
		}
	}
	return result
}

type scoredThread struct {
	score  float64
	thread types.StoryThread
}

func rankedThreads(threads []types.StoryThread, query string, limit int) []types.StoryThread {
	tokens := meaningfulTokens(query)
	scored := []scoredThread{}
	for _, thread := range threads {
		score := tokenOverlap(tokens, meaningfulTokens(thread.Label))
		if score > .05 {
			scored = append(scored, scoredThread{score, thread})
		}
	}
	sort.SliceStable(scored, func(i, j int) bool { return scored[i].score > scored[j].score })
	result := []types.StoryThread{}
	for _, item := range scored {
		result = append(result, item.thread)
		if len(result) >= limit {
			break
		}
	}
	return result
}

func rankedDiagnostics(values []types.FingerprintDiagnostic, query string, limit int) []types.FingerprintDiagnostic {
	tokens := meaningfulTokens(query)
	type scored struct {
		score float64
		value types.FingerprintDiagnostic
	}
	items := []scored{}
	for _, value := range values {
		score := tokenOverlap(tokens, meaningfulTokens(value.Title+" "+value.Detail+" "+value.Kind))
		if len(tokens) == 0 || score > .05 {
			items = append(items, scored{score, value})
		}
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].score > items[j].score })
	result := []types.FingerprintDiagnostic{}
	for _, item := range items {
		result = append(result, item.value)
		if len(result) >= limit {
			break
		}
	}
	return result
}

func eventAnswer(events []types.FingerprintEvent) (string, float64) {
	if len(events) == 0 {
		return "", 0
	}
	first := events[0]
	where := first.ChapterTitle
	if first.StoryTime.Label != "" {
		where += ", " + first.StoryTime.Label
	}
	return first.Summary + " — " + where + ".", first.Confidence
}
func trailAnswer(events []types.FingerprintEvent) (string, float64) {
	if len(events) == 0 {
		return "", 0
	}
	count := min(len(events), 3)
	parts := make([]string, 0, count)
	confidence := 0.0
	for _, event := range events[:count] {
		parts = append(parts, event.Summary+" ("+event.ChapterTitle+")")
		confidence += event.Confidence
	}
	return strings.Join(parts, " → ") + ".", confidence / float64(count)
}
func stateAnswer(states []types.StoryStateInterval) (string, float64) {
	if len(states) == 0 {
		return "", 0
	}
	first := states[0]
	return first.EntityName + " — " + first.Kind + ": " + first.Value + ".", first.Confidence
}
func storyEventLess(left, right types.FingerprintEvent) bool {
	if left.ContextID == right.ContextID && left.StoryTime.DayOffset != nil && right.StoryTime.DayOffset != nil && *left.StoryTime.DayOffset != *right.StoryTime.DayOffset {
		return *left.StoryTime.DayOffset < *right.StoryTime.DayOffset
	}
	return left.NarrativeOrder < right.NarrativeOrder
}
func min(left, right int) int {
	if left < right {
		return left
	}
	return right
}
