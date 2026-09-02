package fingerprint

import (
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"draftline/internal/indexing"
	"draftline/internal/types"
)

var (
	voiceWordRe       = regexp.MustCompile(`[\pL\pN]+(?:['’][\pL\pN]+)?`)
	speechVerbRe      = regexp.MustCompile(`(?i)\b(?:said|asked|replied|answered|whispered|shouted|yelled|murmured|muttered|called|added|continued|told|demanded|insisted|explained|warned|promised|admitted|agreed|snapped|growled)\b`)
	contractionRe     = regexp.MustCompile(`(?i)\b[\pL]+(?:n't|'re|'ve|'ll|'d|'m|'s|n’t|’re|’ve|’ll|’d|’m|’s)\b`)
	droppedGRe        = regexp.MustCompile(`(?i)\b[\pL]{3,}in['’](?:\s|[.,!?]|$)`)
	negativeConcordRe = regexp.MustCompile(`(?i)\b(?:ain't|isn't|wasn't|weren't|don't|doesn't|didn't|never)\b[^.!?]{0,70}\b(?:no|nothing|nobody|nowhere)\b`)
	habitualBeRe      = regexp.MustCompile(`(?i)\b(?:i|you|we|they|he|she)\s+be\s+[\pL]+`)
	addressRe         = regexp.MustCompile(`(?i)^\s*(?:well,?\s+)?([\pL][\pL'-]{1,30}),`)
	vernacularRe      = regexp.MustCompile(`(?i)\b(?:ain't|y'all|gonna|wanna|gotta|'em|’em|'cause|’cause)\b`)
)

type attributedDialogue struct {
	characterID, characterName string
	sample                     types.DialogueSample
}

type quoteSpan struct {
	start, end int
	text       string
}

func buildVoiceProfiles(book *types.BookData, notes []types.CharacterVoiceNotes) []types.CharacterVoiceProfile {
	if book.Analysis.EntityResolution == nil {
		return nil
	}
	data := book.Analysis.EntityResolution
	mentionEntity := indexing.BuildMentionToEntityMap(data.Entities)
	entityNames := map[string]string{}
	for _, entity := range data.Entities {
		if entity.Kind == "person" || entity.Kind == "" {
			entityNames[entity.ID] = entity.Canonical
		}
	}
	mentionsByChapter := map[int][]types.MentionRecord{}
	for _, mention := range data.Mentions {
		if mentionEntity[mention.ID] != "" {
			mentionsByChapter[mention.Chapter] = append(mentionsByChapter[mention.Chapter], mention)
		}
	}
	all := []attributedDialogue{}
	for chapterIndex, chapter := range indexing.AllChapters(book) {
		if !indexing.ShouldAnalyzeChapter(chapter) {
			continue
		}
		text := indexing.StripHTMLForAnalysis(chapter.Content)
		for _, span := range dialogueSpans(text) {
			id, cue, confidence := attributeSpeaker(text, span, mentionsByChapter[chapterIndex], mentionEntity)
			if id == "" || entityNames[id] == "" {
				continue
			}
			all = append(all, attributedDialogue{id, entityNames[id], types.DialogueSample{ID: stableID("dialogue", chapter.ID, strconv.Itoa(span.start), span.text), Text: strings.TrimSpace(span.text), ChapterID: chapter.ID, ChapterIndex: chapterIndex, ChapterTitle: chapter.Title, StartOffset: span.start, AttributionCue: cue, Confidence: confidence}})
		}
	}
	grouped := map[string][]attributedDialogue{}
	order := []string{}
	for _, item := range all {
		if _, ok := grouped[item.characterID]; !ok {
			order = append(order, item.characterID)
		}
		grouped[item.characterID] = append(grouped[item.characterID], item)
	}
	notesByID := map[string]types.CharacterVoiceNotes{}
	for _, note := range notes {
		notesByID[note.CharacterID] = note
	}
	result := make([]types.CharacterVoiceProfile, 0, len(order))
	for _, id := range order {
		profile := voiceProfile(grouped[id])
		if note, ok := notesByID[id]; ok {
			copy := note
			profile.AuthorNotes = &copy
		}
		result = append(result, profile)
	}
	return result
}

func dialogueSpans(text string) []quoteSpan {
	pairs := [][2]string{{`"`, `"`}, {"“", "”"}, {"«", "»"}}
	result := []quoteSpan{}
	for _, pair := range pairs {
		position := 0
		for position < len(text) {
			relative := strings.Index(text[position:], pair[0])
			if relative < 0 {
				break
			}
			start := position + relative
			close := strings.Index(text[start+len(pair[0]):], pair[1])
			if close < 0 {
				break
			}
			end := start + len(pair[0]) + close
			spoken := text[start+len(pair[0]) : end]
			if len(strings.Fields(spoken)) > 0 {
				result = append(result, quoteSpan{start, end + len(pair[1]), spoken})
			}
			position = end + len(pair[1])
		}
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].start < result[j].start })
	return result
}

func attributeSpeaker(text string, span quoteSpan, mentions []types.MentionRecord, entityMap map[string]string) (string, string, float64) {
	end := minInt(len(text), span.end+120)
	after := text[span.end:end]
	if speechVerbRe.MatchString(after) {
		if mention := nearestMention(mentions, span.end, end, true); mention != nil {
			return entityMap[mention.ID], strings.TrimSpace(after[:minInt(len(after), 80)]), .96
		}
	}
	start := maxInt(0, span.start-120)
	before := text[start:span.start]
	if speechVerbRe.MatchString(before) {
		if mention := nearestMention(mentions, start, span.start, false); mention != nil {
			return entityMap[mention.ID], strings.TrimSpace(before[maxInt(0, len(before)-80):]), .93
		}
	}
	return "", "", 0
}

func nearestMention(mentions []types.MentionRecord, start, end int, first bool) *types.MentionRecord {
	var found *types.MentionRecord
	best := int(^uint(0) >> 1)
	for index := range mentions {
		mention := &mentions[index]
		if mention.CharOffset < start || mention.CharOffset > end {
			continue
		}
		distance := mention.CharOffset - start
		if !first {
			distance = end - mention.CharOffset
		}
		if distance < best {
			best = distance
			found = mention
		}
	}
	return found
}

func voiceProfile(items []attributedDialogue) types.CharacterVoiceProfile {
	profile := types.CharacterVoiceProfile{CharacterID: items[0].characterID, CharacterName: items[0].characterName, SampleCount: len(items), Samples: []types.DialogueSample{}}
	wordCounts := map[string]int{}
	addresses := map[string]int{}
	questions, exclamations, contractions := 0, 0, 0
	for _, item := range items {
		sample := item.sample
		profile.Samples = append(profile.Samples, sample)
		words := voiceWordRe.FindAllString(sample.Text, -1)
		profile.WordCount += len(words)
		contractions += len(contractionRe.FindAllString(sample.Text, -1))
		if strings.Contains(sample.Text, "?") {
			questions++
		}
		if strings.Contains(sample.Text, "!") {
			exclamations++
		}
		for _, word := range words {
			normalized := strings.ToLower(strings.ReplaceAll(word, "’", "'"))
			if !voiceStopWords[normalized] && len([]rune(normalized)) > 2 {
				wordCounts[normalized]++
			}
		}
		if match := addressRe.FindStringSubmatch(sample.Text); len(match) == 2 {
			addresses[strings.ToLower(match[1])]++
		}
	}
	profile.AverageWords = roundVoice(float64(profile.WordCount) / float64(maxInt(1, profile.SampleCount)))
	profile.ContractionPercent = roundVoice(float64(contractions) * 100 / float64(maxInt(1, profile.WordCount)))
	profile.QuestionPercent = roundVoice(float64(questions) * 100 / float64(maxInt(1, profile.SampleCount)))
	profile.ExclamationPercent = roundVoice(float64(exclamations) * 100 / float64(maxInt(1, profile.SampleCount)))
	profile.Vocabulary = rankVoiceTerms(wordCounts, 16, 2)
	profile.AddressForms = rankVoiceTerms(addresses, 10, 1)
	profile.DialectSignals = voiceSignals(items, profile)
	profile.Confidence = roundVoice(math.Min(.98, .35+float64(profile.SampleCount)*.06+float64(profile.WordCount)*.001))
	if len(profile.Samples) > 24 {
		profile.Samples = profile.Samples[:24]
	}
	return profile
}

func voiceSignals(items []attributedDialogue, profile types.CharacterVoiceProfile) []types.VoiceSignal {
	type counter struct {
		label    string
		count    int
		examples []string
	}
	values := map[string]*counter{"vernacular-contraction": {label: "Uses vernacular contractions"}, "dropped-final-g": {label: "Uses dropped final g spellings"}, "negative-concord": {label: "Uses negative concord"}, "habitual-be": {label: "Uses habitual be constructions"}}
	for _, item := range items {
		text := item.sample.Text
		matches := map[string]bool{"vernacular-contraction": vernacularRe.MatchString(text), "dropped-final-g": droppedGRe.MatchString(text), "negative-concord": negativeConcordRe.MatchString(text), "habitual-be": habitualBeRe.MatchString(text)}
		for kind, matched := range matches {
			if matched {
				value := values[kind]
				value.count++
				if len(value.examples) < 3 {
					value.examples = append(value.examples, text)
				}
			}
		}
	}
	result := []types.VoiceSignal{}
	for kind, value := range values {
		if value.count > 0 {
			result = append(result, types.VoiceSignal{Kind: kind, Label: value.label, Count: value.count, Examples: value.examples, Confidence: roundVoice(math.Min(.96, .55+float64(value.count)*.1))})
		}
	}
	if profile.WordCount >= 80 && profile.ContractionPercent < 1 {
		result = append(result, types.VoiceSignal{Kind: "low-contraction", Label: "Rarely uses contractions", Count: profile.SampleCount, Confidence: .78})
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Count == result[j].Count {
			return result[i].Kind < result[j].Kind
		}
		return result[i].Count > result[j].Count
	})
	return result
}

func rankVoiceTerms(counts map[string]int, limit, minimum int) []types.VoiceTerm {
	result := []types.VoiceTerm{}
	for text, count := range counts {
		if count >= minimum {
			result = append(result, types.VoiceTerm{Text: text, Count: count})
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Count == result[j].Count {
			return result[i].Text < result[j].Text
		}
		return result[i].Count > result[j].Count
	})
	if len(result) > limit {
		result = result[:limit]
	}
	return result
}
func roundVoice(value float64) float64 { return math.Round(value*10) / 10 }
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

var voiceStopWords = map[string]bool{"the": true, "and": true, "that": true, "this": true, "with": true, "from": true, "have": true, "has": true, "had": true, "was": true, "were": true, "are": true, "you": true, "your": true, "his": true, "her": true, "they": true, "them": true, "their": true, "but": true, "not": true, "for": true, "what": true, "when": true, "where": true, "who": true, "why": true, "how": true, "would": true, "could": true, "should": true, "said": true, "just": true, "like": true, "into": true, "out": true, "about": true, "there": true, "here": true, "then": true, "than": true, "all": true, "can": true, "did": true, "does": true, "don't": true, "it's": true, "i'm": true}
