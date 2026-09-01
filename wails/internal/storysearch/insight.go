package storysearch

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"draftline/internal/types"
)

var (
	identityIntentRe     = regexp.MustCompile(`(?i)\b(?:first|full|given|last|sur)\s*name\b|\bwho\s+is\b`)
	discoveryIntentRe    = regexp.MustCompile(`(?i)\b(?:discover(?:ed|s|ing)?|finds?|found|learn(?:ed|t|s|ing)?|reveal(?:ed|s|ing)?|uncover(?:ed|s|ing)?)\b`)
	researchIntentRe     = regexp.MustCompile(`(?i)\b(?:research(?:ed|es|ing)?|investigat(?:ed|es|ing|ion)|look(?:ed|s|ing)?\s+into)\b`)
	firstIntentRe        = regexp.MustCompile(`(?i)\b(?:first|earliest|initial)\s+(?:appear(?:ance)?|mention|occurrence|establish(?:ment|ed)?|introduc(?:tion|ed)?)\b`)
	knowledgeIntentRe    = regexp.MustCompile(`(?i)\b(?:knows?|knew|learn(?:ed|t|s|ing)?|underst(?:ood|ands?|anding)|remembers?|remembered|believes?|believed|suspects?|suspected|tells?|told|informs?|informed|warns?|warned|explains?|explained|shares?|shared|shows?|showed|withholds?|withheld|withholding|was\s+told)\b`)
	knowledgeNoiseRe     = regexp.MustCompile(`(?i)\b(?:who|whom|what|which|when|how|did|does|do|is|are|was|were|about|with|to|from|the|a|an|this|that|it|chapter|chapters|scene|scenes|story|book)\b`)
	questionNoiseRe      = regexp.MustCompile(`(?i)\b(?:what|which|where|when|how|did|does|do|is|are|was|were|chapter|chapters|scene|scenes|story|book|mentioned|mention|appears?|occurs?|established|introduc(?:e|ed|tion)|and)\b`)
	pronounContractionRe = regexp.MustCompile(`(?i)^(?:he|she|it|i|we|you|they)[\x{2019}'](?:d|s|ve|ll|re|m|t)$`)
)

type insightAccumulator struct {
	intent           string
	interpretedQuery string
	resolved         []types.StorySearchEntity
	chapters         map[int]*types.StorySearchChapterSummary
	chapterOrder     []int
	evidenceSeen     map[string]bool
	related          map[string]*types.StorySearchRelatedTerm
	evidenceCount    int
	eventCount       int
	factCount        int
	discoveryCount   int
	knowledgeSeen    map[string]bool
	knowledgeStates  []types.StorySearchKnowledgeState
}

func interpretDetailQuery(query string) (string, string) {
	intent := "trail"
	switch {
	case knowledgeIntentRe.MatchString(query):
		intent = "knowledge"
	case identityIntentRe.MatchString(query):
		intent = "identity"
	case discoveryIntentRe.MatchString(query):
		intent = "discovery"
	case researchIntentRe.MatchString(query):
		intent = "research"
	case firstIntentRe.MatchString(query):
		intent = "first_occurrence"
	}

	protected, phrases := protectQuotedPhrases(query)
	cleaned := identityIntentRe.ReplaceAllString(protected, " ")
	if intent == "knowledge" {
		cleaned = knowledgeIntentRe.ReplaceAllString(cleaned, " ")
		cleaned = knowledgeNoiseRe.ReplaceAllString(cleaned, " ")
	}
	cleaned = discoveryIntentRe.ReplaceAllString(cleaned, " ")
	cleaned = researchIntentRe.ReplaceAllString(cleaned, " ")
	cleaned = firstIntentRe.ReplaceAllString(cleaned, " ")
	cleaned = questionNoiseRe.ReplaceAllString(cleaned, " ")
	cleaned = restoreQuotedPhrases(cleanSpace(cleaned), phrases)
	cleaned = strings.TrimSpace(strings.Trim(cleaned, "?!.,;:"))
	if cleanSpace(strings.Trim(cleaned, "'\"")) == "" {
		return query, "trail"
	}
	return cleanSpace(cleaned), intent
}

func protectQuotedPhrases(query string) (string, []string) {
	phrases := make([]string, 0, 2)
	protected := quotedRe.ReplaceAllStringFunc(query, func(match string) string {
		parts := quotedRe.FindStringSubmatch(match)
		phrase := parts[1]
		if phrase == "" {
			phrase = parts[2]
		}
		token := fmt.Sprintf("QPHRASE%dTOKEN", len(phrases))
		phrases = append(phrases, phrase)
		return token
	})
	return protected, phrases
}

func restoreQuotedPhrases(query string, phrases []string) string {
	for index, phrase := range phrases {
		query = strings.ReplaceAll(query, fmt.Sprintf("QPHRASE%dTOKEN", index), `"`+phrase+`"`)
	}
	return query
}

func newInsightAccumulator(intent, interpretedQuery string, resolved []types.StorySearchEntity) *insightAccumulator {
	return &insightAccumulator{
		intent: intent, interpretedQuery: interpretedQuery, resolved: resolved,
		chapters: make(map[int]*types.StorySearchChapterSummary), evidenceSeen: make(map[string]bool),
		related: make(map[string]*types.StorySearchRelatedTerm), knowledgeSeen: make(map[string]bool),
	}
}

func (a *insightAccumulator) add(ref chapterRef, evidence []types.EvidenceRecord) {
	chapter := a.chapters[ref.globalIndex]
	if chapter == nil {
		chapter = &types.StorySearchChapterSummary{ChapterIndex: ref.globalIndex, ChapterTitle: chapterTitle(ref)}
		a.chapters[ref.globalIndex] = chapter
		a.chapterOrder = append(a.chapterOrder, ref.globalIndex)
	}
	chapter.Occurrences++

	for _, record := range evidence {
		if a.evidenceSeen[record.ID] {
			continue
		}
		a.evidenceSeen[record.ID] = true
		a.evidenceCount++
		chapter.EvidenceCount++
		switch record.Kind {
		case "event":
			a.eventCount++
			chapter.EventCount++
		case "fact":
			a.factCount++
			chapter.FactCount++
		}
		if record.EvidenceType == "discovery" {
			a.discoveryCount++
		}
		for _, state := range record.KnowledgeStates {
			a.addKnowledge(ref, record, state)
		}
		for _, name := range record.CharacterNames {
			a.addRelated(name, "character")
		}
		for _, entity := range record.NamedEntities {
			a.addRelated(entity.Text, strings.ToLower(entity.Label))
		}
	}
}

func (a *insightAccumulator) addKnowledge(ref chapterRef, record types.EvidenceRecord, state types.EvidenceKnowledgeState) {
	key := record.ID + "\x00" + state.State + "\x00" + strings.Join(state.CharacterIDs, "\x00") + "\x00" + strings.Join(state.CounterpartyIDs, "\x00")
	if a.knowledgeSeen[key] {
		return
	}
	a.knowledgeSeen[key] = true
	a.knowledgeStates = append(a.knowledgeStates, types.StorySearchKnowledgeState{
		EvidenceID: record.ID, State: state.State,
		CharacterIDs: state.CharacterIDs, CharacterNames: state.CharacterNames,
		CounterpartyIDs: state.CounterpartyIDs, CounterpartyNames: state.CounterpartyNames,
		ChapterIndex: ref.globalIndex, ChapterTitle: chapterTitle(ref), Section: ref.section, SectionIndex: ref.sectionIndex,
		Text: record.Text, Cue: state.Cue, Confidence: state.Confidence,
	})
}

func (a *insightAccumulator) addRelated(text, label string) {
	text = cleanSpace(text)
	if text == "" || pronounContractionRe.MatchString(text) || a.isPrimaryTerm(text) {
		return
	}
	key := strings.ToLower(text)
	entry := a.related[key]
	if entry == nil {
		entry = &types.StorySearchRelatedTerm{Text: text, Label: label}
		a.related[key] = entry
	}
	entry.Count++
}

func (a *insightAccumulator) isPrimaryTerm(text string) bool {
	text = comparisonTerm(text)
	for _, entity := range a.resolved {
		for _, alias := range entity.Aliases {
			if strings.EqualFold(comparisonTerm(alias), text) {
				return true
			}
		}
	}
	for _, term := range strings.FieldsFunc(a.interpretedQuery, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsPunct(r)
	}) {
		if strings.EqualFold(term, text) {
			return true
		}
	}
	return false
}

func comparisonTerm(text string) string {
	text = strings.Trim(cleanSpace(text), " \t\r\n.,;:!?\"\u201c\u201d")
	lower := strings.ToLower(text)
	if strings.HasSuffix(lower, "'s") {
		text = text[:len(text)-2]
	} else if strings.HasSuffix(lower, "\u2019s") {
		text = text[:len(text)-len("\u2019s")]
	}
	return text
}

func (a *insightAccumulator) finish(total int) *types.StorySearchInsight {
	if total == 0 {
		return nil
	}
	insight := &types.StorySearchInsight{
		Intent: a.intent, InterpretedQuery: a.interpretedQuery,
		ChapterCount: len(a.chapterOrder), EvidenceCount: a.evidenceCount,
		EventCount: a.eventCount, FactCount: a.factCount, DiscoveryCount: a.discoveryCount,
		KnowledgeCount: len(a.knowledgeStates), KnowledgeStates: a.knowledgeStates,
		Chapters: make([]types.StorySearchChapterSummary, 0, len(a.chapterOrder)),
	}
	for _, chapterIndex := range a.chapterOrder {
		insight.Chapters = append(insight.Chapters, *a.chapters[chapterIndex])
	}
	insight.RelatedTerms = a.sortedRelatedTerms()
	insight.Signals = a.signals(total)
	return insight
}

func (a *insightAccumulator) sortedRelatedTerms() []types.StorySearchRelatedTerm {
	terms := make([]types.StorySearchRelatedTerm, 0, len(a.related))
	for _, term := range a.related {
		terms = append(terms, *term)
	}
	sort.SliceStable(terms, func(i, j int) bool {
		if terms[i].Count != terms[j].Count {
			return terms[i].Count > terms[j].Count
		}
		return strings.ToLower(terms[i].Text) < strings.ToLower(terms[j].Text)
	})
	if len(terms) > 12 {
		terms = terms[:12]
	}
	return terms
}

func (a *insightAccumulator) signals(total int) []types.StorySearchSignal {
	signals := make([]types.StorySearchSignal, 0, 4)
	if a.intent == "identity" {
		signals = append(signals, identitySignals(a.resolved)...)
	}
	if a.intent == "discovery" {
		if a.discoveryCount > 0 {
			signals = append(signals, types.StorySearchSignal{
				Kind: "answer", Title: pluralCount(a.discoveryCount, "indexed discovery", "indexed discoveries"),
				Detail: "The source trail contains explicit discovery or revelation language.",
			})
		} else {
			signals = append(signals, types.StorySearchSignal{
				Kind: "attention", Title: "No indexed discovery event",
				Detail: "The passages mention these details, but the evidence index does not classify any as the moment of discovery.",
			})
		}
	}
	if a.intent == "knowledge" {
		if len(a.knowledgeStates) == 0 {
			signals = append(signals, types.StorySearchSignal{
				Kind: "attention", Title: "No explicit knowledge state found",
				Detail: "The detail appears in these passages, but Draftline cannot prove who learned, knew, or shared it from the indexed grammar.",
			})
		} else {
			signals = append(signals, types.StorySearchSignal{
				Kind: "answer", Title: pluralCount(len(a.knowledgeStates), "source-backed knowledge point", "source-backed knowledge points"),
				Detail: "Each point below names only characters directly tied to knowledge or communication wording in the source sentence.",
			})
		}
	}
	if total == 1 {
		signals = append(signals, types.StorySearchSignal{
			Kind: "attention", Title: "Appears in one scene",
			Detail: "This may be an intentional one-off, a newly introduced thread, or a detail that never returns.",
		})
	} else if len(a.chapterOrder) == 1 {
		signals = append(signals, types.StorySearchSignal{
			Kind: "context", Title: "Confined to one chapter",
			Detail: fmt.Sprintf("All %d matching scenes occur in %s.", total, a.chapters[a.chapterOrder[0]].ChapterTitle),
		})
	}
	if a.evidenceCount == 0 {
		signals = append(signals, types.StorySearchSignal{
			Kind: "context", Title: "Text matches only",
			Detail: "Draftline found the source wording, but no indexed fact or event currently supports these scenes.",
		})
	}
	return signals
}

func identitySignals(entities []types.StorySearchEntity) []types.StorySearchSignal {
	if len(entities) == 0 {
		return []types.StorySearchSignal{{
			Kind: "attention", Title: "No confirmed character match",
			Detail: "The wording appears in the manuscript, but it is not linked to a confirmed character or alias.",
		}}
	}
	result := make([]types.StorySearchSignal, 0, len(entities))
	for _, entity := range entities {
		fullName := probableFullName(entity)
		if fullName == "" {
			result = append(result, types.StorySearchSignal{
				Kind: "attention", Title: "No confirmed given name found",
				Detail: fmt.Sprintf("%s is indexed only by a single name or title. Draftline cannot prove a fuller name from confirmed aliases.", entity.Canonical),
			})
			continue
		}
		detail := "This is the fullest confirmed form linked to the character."
		if len(entity.Aliases) > 1 {
			detail = fmt.Sprintf("Confirmed aliases: %s.", strings.Join(entity.Aliases, ", "))
		}
		result = append(result, types.StorySearchSignal{Kind: "answer", Title: fullName, Detail: detail})
	}
	return result
}

func probableFullName(entity types.StorySearchEntity) string {
	candidates := append([]string{entity.Canonical}, entity.Aliases...)
	for _, candidate := range candidates {
		words := strings.Fields(candidate)
		meaningful := 0
		for _, word := range words {
			switch strings.ToLower(strings.Trim(word, ".,")) {
			case "mr", "mrs", "ms", "miss", "dr", "doctor", "detective", "captain", "officer", "agent", "professor", "sir", "lady", "lord":
			default:
				meaningful++
			}
		}
		if meaningful >= 2 {
			return candidate
		}
	}
	return ""
}

func pluralCount(count int, singular, plural string) string {
	if count == 1 {
		return fmt.Sprintf("1 %s", singular)
	}
	return fmt.Sprintf("%d %s", count, plural)
}
