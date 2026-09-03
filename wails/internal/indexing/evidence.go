package indexing

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"draftline/internal/types"

	"github.com/jdkato/prose/v3"
)

const (
	evidenceEngine     = "prose-v3-evidence-v3"
	maxEvidenceRecords = 50_000
)

var (
	discoveryCueRe         = regexp.MustCompile(`(?i)\b(discover(?:ed|s|ing)?|finds?|found|learn(?:ed|t|s|ing)?|reveal(?:ed|s|ing)?|uncover(?:ed|s|ing)?|realiz(?:ed|es|ing)|identif(?:ied|ies|ying)|determin(?:ed|es|ing)|locat(?:ed|es|ing)|notic(?:ed|es|ing))\b`)
	transitionCueRe        = regexp.MustCompile(`(?i)\b(arriv(?:ed|es|ing)|enter(?:ed|s|ing)?|return(?:ed|s|ing)?|depart(?:ed|s|ing)?|leaves?|left|exit(?:ed|s|ing)?|escap(?:ed|es|ing)|fled|walk(?:ed|s|ing)?|ran|run|drov(?:e|en)|driv(?:es|ing)|rush(?:ed|es|ing)?|approach(?:ed|es|ing)?)\b`)
	interactionCueRe       = regexp.MustCompile(`(?i)\b(meets?|met|tells?|told|asks?|asked|hands?|handed|calls?|called|faces?|faced|follows?|followed|joins?|joined|attacks?|attacked|helps?|helped|watches?|watched|confronts?|confronted)\b`)
	stateCueRe             = regexp.MustCompile(`(?i)\b(is|are|was|were|has|have|had|knows?|knew|works?|worked|lives?|lived|belongs?|belonged|called|named)\b`)
	timeCueRe              = regexp.MustCompile(`(?i)\b(?:\d{1,2}:\d{2}(?:\s*[ap]\.?(?:m\.)?)?|(?:19|20)\d{2}|monday|tuesday|wednesday|thursday|friday|saturday|sunday|morning|afternoon|evening|night|midnight|noon|dawn|dusk|today|tonight|tomorrow|yesterday|(?:one|two|three|four|five|six|seven|eight|nine|ten|\d+)\s+(?:minutes?|hours?|days?|weeks?|months?|years?)\s+(?:later|earlier|ago))\b`)
	knowledgeLearnRe       = regexp.MustCompile(`(?i)\b(?:learn(?:ed|t|s|ing)?|discover(?:ed|s|ing)?|found\s+out|realiz(?:ed|es|ing)|notic(?:ed|es|ing)|recogniz(?:ed|es|ing)|became\s+aware)\b`)
	knowledgeKnowRe        = regexp.MustCompile(`(?i)\b(?:knows?|knew|underst(?:ood|ands?|anding)|remembers?|remembered|believes?|believed|suspects?|suspected|is\s+aware|was\s+aware|were\s+aware)\b`)
	knowledgeShareRe       = regexp.MustCompile(`(?i)\b(?:tell|tells|told|telling|informs?|informed|warns?|warned|explains?|explained|reveals?|revealed|shares?|shared|shows?|showed|admits?|admitted|confesses?|confessed|reported|withholds?|withheld|withholding)\b`)
	knowledgeNegationRe    = regexp.MustCompile(`(?i)\b(?:didn['’]t|did\s+not|doesn['’]t|does\s+not|don['’]t|do\s+not|couldn['’]t|could\s+not|can['’]t|cannot|never|not)\s*$`)
	knowledgeAttemptRe     = regexp.MustCompile(`(?i)\b(?:tried\s+to|tries\s+to|trying\s+to|attempted\s+to|attempts\s+to)\s*$`)
	knowledgeActorBridgeRe = regexp.MustCompile(`(?i)^\s*(?:(?:had|has|have|hadn['’]t|hasn['’]t|haven['’]t|did|didn['’]t|does|doesn['’]t|do|don['’]t|was|were|is|are|would|wouldn['’]t|could|couldn['’]t|can|can['’]t|will|won['’]t|tries|tried|trying|attempts|attempted|attempting|to|not|never|already|also|just|really|finally|clearly|quietly|suddenly)\s+)*$`)
)

type evidenceChapter struct {
	chapter      types.ChapterItem
	globalIndex  int
	section      string
	sectionIndex int
	text         string
}

type evidenceSentence struct {
	text            string
	start           int
	end             int
	paragraphIndex  int
	sentenceIndex   int
	characterIDs    []string
	characterNames  []string
	named           []types.EvidenceTerm
	action          string
	timeExpressions []string
	knowledgeStates []types.EvidenceKnowledgeState
}

// AnalyzeEvidence builds a conservative fact/event candidate index from exact
// manuscript sentences. It uses the already bundled prose/v3 models and
// deterministic cue rules; no generative model or network call is involved.
func AnalyzeEvidence(book *types.BookData, progress func(types.StoryAnalysisProgress)) *types.EvidenceData {
	return AnalyzeEvidenceWithOptions(book, progress, defaultAnalysisPoolOptions())
}

// AnalyzeEvidenceWithOptions bounds only the ProseV3 evidence workers. The
// unbuffered job channel prevents queued chapters from multiplying memory.
func AnalyzeEvidenceWithOptions(book *types.BookData, progress func(types.StoryAnalysisProgress), pool AnalysisPoolOptions) *types.EvidenceData {
	chapters := evidenceChapters(book)
	result := &types.EvidenceData{
		Engine:        evidenceEngine,
		LastAnalyzed:  time.Now().Format(time.RFC3339),
		Records:       []types.EvidenceRecord{},
		ChapterHashes: map[string]string{},
		Version:       3,
	}

	hasher := sha256.New()
	for _, item := range chapters {
		fmt.Fprintf(hasher, "%d\x00%s\x00%s\x00%s\x00", item.globalIndex, item.chapter.ID, item.chapter.Title, item.text)
	}
	result.ContentHash = hex.EncodeToString(hasher.Sum(nil))

	prior := priorEvidence(book.Analysis.Evidence)
	reusable := reusableEvidenceChapters(book.Analysis.Evidence, chapters, result.ChapterHashes)
	mentionToEntity, canonicalByID, mentionsByChapter := evidenceCharacterLookups(book)
	toAnalyze := append([]evidenceChapter(nil), chapters...)
	for index, item := range toAnalyze {
		if reusable[item.chapter.ID] != nil {
			toAnalyze[index].text = ""
		}
	}
	analyzed := analyzeEvidenceChapters(toAnalyze, mentionsByChapter, mentionToEntity, canonicalByID, progress, pool)
	introduced := make(map[string]bool)
	seenIDs := make(map[string]int)

	for position, item := range chapters {
		if len(result.Records) >= maxEvidenceRecords {
			result.Truncated = true
			break
		}
		if cached := reusable[item.chapter.ID]; cached != nil {
			for _, record := range cached {
				record.ChapterIndex, record.Section, record.SectionIndex = item.globalIndex, item.section, item.sectionIndex
				result.Records = append(result.Records, record)
				for _, id := range record.CharacterIDs {
					introduced[id] = true
				}
			}
			continue
		}
		for _, sentence := range analyzed[position] {
			newCharacters := make([]string, 0, len(sentence.characterIDs))
			newNames := make([]string, 0, len(sentence.characterIDs))
			for i, id := range sentence.characterIDs {
				if !introduced[id] {
					introduced[id] = true
					newCharacters = append(newCharacters, id)
					if i < len(sentence.characterNames) {
						newNames = append(newNames, sentence.characterNames[i])
					}
				}
			}
			if len(newCharacters) > 0 {
				record := makeEvidenceRecord(item, sentence.paragraphIndex, sentence.sentenceIndex, sentence.start, sentence.end, sentence.text, "event", "introduction", newCharacters, newNames, sentence.named, sentence.action, sentence.timeExpressions, nil, .98, "First confirmed appearance in manuscript order")
				appendEvidence(result, record, prior, seenIDs)
			}

			kind, evidenceType, confidence, rationale := classifyEvidence(sentence.text, sentence.characterIDs, sentence.named, sentence.timeExpressions, sentence.knowledgeStates)
			if evidenceType == "" {
				continue
			}
			record := makeEvidenceRecord(item, sentence.paragraphIndex, sentence.sentenceIndex, sentence.start, sentence.end, sentence.text, kind, evidenceType, sentence.characterIDs, sentence.characterNames, sentence.named, sentence.action, sentence.timeExpressions, sentence.knowledgeStates, confidence, rationale)
			appendEvidence(result, record, prior, seenIDs)
			if len(result.Records) >= maxEvidenceRecords {
				result.Truncated = true
				break
			}
		}
	}

	// Author-created records are not rebuildable and must survive analysis.
	if old := book.Analysis.Evidence; old != nil {
		for _, record := range old.Records {
			if record.Source == "author" {
				result.Records = append(result.Records, record)
			}
		}
	}
	sort.SliceStable(result.Records, func(i, j int) bool {
		a, b := result.Records[i], result.Records[j]
		if a.ChapterIndex != b.ChapterIndex {
			return a.ChapterIndex < b.ChapterIndex
		}
		if a.StartOffset != b.StartOffset {
			return a.StartOffset < b.StartOffset
		}
		return a.ID < b.ID
	})
	return result
}

func reusableEvidenceChapters(old *types.EvidenceData, chapters []evidenceChapter, hashes map[string]string) map[string][]types.EvidenceRecord {
	result := map[string][]types.EvidenceRecord{}
	if old == nil || len(old.ChapterHashes) == 0 { // v2 caches safely fall back to one full rebuild.
		for _, item := range chapters {
			hashes[item.chapter.ID] = evidenceChapterHash(item.text)
		}
		return result
	}
	current := map[string]bool{}
	for _, item := range chapters {
		hash := evidenceChapterHash(item.text)
		hashes[item.chapter.ID] = hash
		current[item.chapter.ID] = true
		if old.ChapterHashes[item.chapter.ID] != hash {
			continue
		}
		result[item.chapter.ID] = []types.EvidenceRecord{}
	}
	for _, record := range old.Records {
		if record.Source == "author" || !current[record.ChapterID] {
			continue
		}
		if _, ok := result[record.ChapterID]; ok {
			result[record.ChapterID] = append(result[record.ChapterID], record)
		}
	}
	return result
}

func evidenceChapterHash(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:16])
}

func evidenceChapters(book *types.BookData) []evidenceChapter {
	items := make([]evidenceChapter, 0, len(book.FrontMatter)+len(book.Body)+len(book.BackMatter))
	global := 0
	add := func(section string, chapters []types.ChapterItem) {
		for index, chapter := range chapters {
			text := ""
			if ShouldAnalyzeChapter(chapter) {
				text = StripHTMLForAnalysis(chapter.Content)
			}
			if strings.TrimSpace(text) != "" {
				items = append(items, evidenceChapter{chapter: chapter, globalIndex: global, section: section, sectionIndex: index, text: text})
			}
			global++
		}
	}
	add("front_matter", book.FrontMatter)
	add("body", book.Body)
	add("back_matter", book.BackMatter)
	return items
}

// analyzeEvidenceChapters runs model-backed per-chapter work through the
// stage-local worker and memory bounds supplied by the Wails analysis budget.
func analyzeEvidenceChapters(chapters []evidenceChapter, mentionsByChapter map[int][]types.MentionRecord, mentionToEntity, canonicalByID map[string]string, progress func(types.StoryAnalysisProgress), pool AnalysisPoolOptions) [][]evidenceSentence {
	result := make([][]evidenceSentence, len(chapters))
	if len(chapters) == 0 {
		return result
	}
	pool = normalizeAnalysisPoolOptions(pool, len(chapters))
	jobs := make(chan int)
	memory := newAnalysisMemoryGate(pool.MaxInFlightBytes)
	var workers sync.WaitGroup
	var progressMu sync.Mutex
	completed := 0
	for range pool.Workers {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for position := range jobs {
				item := chapters[position]
				weight := memory.acquire(len(item.text))
				result[position] = analyzeEvidenceChapter(item, mentionsByChapter[item.globalIndex], mentionToEntity, canonicalByID)
				memory.release(weight)
				if progress != nil {
					progressMu.Lock()
					completed++
					progress(types.StoryAnalysisProgress{
						Phase: "evidence", Message: fmt.Sprintf("Indexing facts and events in %s", displayChapterTitle(item.chapter, item.globalIndex)),
						ChapterIndex: item.globalIndex, ChapterTitle: item.chapter.Title,
						Current: completed, Total: len(chapters),
						Percent: 48 + int(math.Round(float64(completed-1)*16/float64(maxInt(1, len(chapters))))),
					})
					progressMu.Unlock()
				}
			}
		}()
	}
	for position := range chapters {
		jobs <- position
	}
	close(jobs)
	workers.Wait()
	return result
}

func analyzeEvidenceChapter(item evidenceChapter, mentions []types.MentionRecord, mentionToEntity, canonicalByID map[string]string) []evidenceSentence {
	doc, err := prose.NewDocument(item.text)
	if err != nil {
		return nil
	}
	paragraphs := splitIntoParagraphs(item.text)
	result := make([]evidenceSentence, 0, len(doc.Sentences()))
	for sentenceIndex, sentence := range doc.Sentences() {
		text := strings.TrimSpace(sentence.Text)
		if text == "" {
			continue
		}
		end := sentence.End()
		characterIDs := charactersInSpan(mentions, mentionToEntity, sentence.Start, end)
		characterNames := make([]string, 0, len(characterIDs))
		for _, id := range characterIDs {
			if name := canonicalByID[id]; name != "" {
				characterNames = append(characterNames, name)
			}
		}
		result = append(result, evidenceSentence{
			text: text, start: sentence.Start, end: end,
			paragraphIndex: paragraphForOffset(paragraphs, sentence.Start), sentenceIndex: sentenceIndex,
			characterIDs: characterIDs, characterNames: characterNames,
			named:           namedEntitiesInSpan(doc.Entities(), sentence.Start, end),
			action:          firstVerbInSpan(doc.Tokens(), sentence.Start, end),
			timeExpressions: uniqueStrings(timeCueRe.FindAllString(text, -1)),
			knowledgeStates: inferKnowledgeStates(text, sentence.Start, mentions, mentionToEntity, canonicalByID),
		})
	}
	return result
}

func evidenceCharacterLookups(book *types.BookData) (map[string]string, map[string]string, map[int][]types.MentionRecord) {
	mentionToEntity := map[string]string{}
	canonicalByID := map[string]string{}
	mentionsByChapter := map[int][]types.MentionRecord{}
	if book.Analysis.EntityResolution == nil {
		return mentionToEntity, canonicalByID, mentionsByChapter
	}
	mentionToEntity = BuildMentionToEntityMap(book.Analysis.EntityResolution.Entities)
	for _, entity := range book.Analysis.EntityResolution.Entities {
		canonicalByID[entity.ID] = entity.Canonical
	}
	for _, mention := range book.Analysis.EntityResolution.Mentions {
		mentionsByChapter[mention.Chapter] = append(mentionsByChapter[mention.Chapter], mention)
	}
	return mentionToEntity, canonicalByID, mentionsByChapter
}

func charactersInSpan(mentions []types.MentionRecord, entityMap map[string]string, start, end int) []string {
	seen := map[string]bool{}
	ids := []string{}
	for _, mention := range mentions {
		if mention.CharOffset < start || mention.CharOffset >= end {
			continue
		}
		if id := entityMap[mention.ID]; id != "" && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	return ids
}

func namedEntitiesInSpan(entities []prose.Entity, start, end int) []types.EvidenceTerm {
	seen := map[string]bool{}
	terms := []types.EvidenceTerm{}
	for _, entity := range entities {
		if entity.Start < start || entity.End() > end {
			continue
		}
		text := strings.TrimSpace(entity.Text)
		key := strings.ToLower(text) + "\x00" + strings.ToUpper(entity.Label)
		if text == "" || seen[key] {
			continue
		}
		seen[key] = true
		terms = append(terms, types.EvidenceTerm{Text: text, Label: strings.ToUpper(entity.Label)})
	}
	return terms
}

func firstVerbInSpan(tokens []prose.Token, start, end int) string {
	for _, token := range tokens {
		if token.Start >= start && token.End() <= end && strings.HasPrefix(token.Tag, "VB") {
			return strings.ToLower(token.Text)
		}
	}
	return ""
}

func paragraphForOffset(paragraphs []ParagraphInfo, offset int) int {
	for index, paragraph := range paragraphs {
		if offset >= paragraph.StartOffset && offset < paragraph.EndOffset {
			return index
		}
	}
	return 0
}

type knowledgeParticipant struct {
	id    string
	name  string
	start int
	end   int
}

type knowledgeCue struct {
	state string
	start int
	end   int
	text  string
}

func inferKnowledgeStates(text string, sentenceStart int, mentions []types.MentionRecord, entityMap, canonicalByID map[string]string) []types.EvidenceKnowledgeState {
	participants := knowledgeParticipants(sentenceStart, sentenceStart+len(text), mentions, entityMap, canonicalByID)
	if len(participants) == 0 {
		return nil
	}
	cues := knowledgeCues(text)
	result := make([]types.EvidenceKnowledgeState, 0, len(cues))
	seen := map[string]bool{}
	for _, cue := range cues {
		before, after := participantsAroundCue(participants, sentenceStart+cue.start, sentenceStart+cue.end)
		actor := explicitKnowledgeActor(text, sentenceStart, cue.start, before)
		if len(actor) == 0 {
			continue
		}
		claim := types.EvidenceKnowledgeState{State: cue.state, Cue: cue.text, Confidence: .82}
		switch cue.state {
		case "shared":
			// The nearest confirmed character before a reporting verb is the
			// conservative active-voice speaker. Earlier names may belong to a
			// preceding clause and are not silently promoted to communicators.
			claim.CharacterIDs, claim.CharacterNames = participantIdentity(actor)
			if knowledgeCueIsNegated(text, cue) || strings.HasPrefix(strings.ToLower(cue.text), "withh") {
				claim.State = "withheld"
			}
			recipients := explicitCounterparty(text, sentenceStart, cue.end, actor[0].id, after)
			claim.CounterpartyIDs, claim.CounterpartyNames = participantIdentity(recipients)
			if len(recipients) == 0 {
				claim.Confidence = .68
			}
		case "learned", "knows":
			claim.CharacterIDs, claim.CharacterNames = participantIdentity(actor)
			if cue.state == "knows" {
				claim.State = knowledgeStateForCue(text, cue)
			}
			if strings.Contains(strings.ToLower(text[cue.end:]), " from ") {
				sources := explicitCounterparty(text, sentenceStart, cue.end, actor[0].id, after)
				claim.CounterpartyIDs, claim.CounterpartyNames = participantIdentity(sources)
			}
		}
		key := claim.State + "\x00" + strings.Join(claim.CharacterIDs, "\x00") + "\x00" + strings.Join(claim.CounterpartyIDs, "\x00")
		if len(claim.CharacterIDs) > 0 && !seen[key] {
			seen[key] = true
			result = append(result, claim)
		}
	}
	return result
}

func explicitKnowledgeActor(text string, sentenceStart, cueStart int, before []knowledgeParticipant) []knowledgeParticipant {
	if len(before) == 0 {
		return nil
	}
	actor := before[len(before)-1]
	gapStart := actor.end - sentenceStart
	if gapStart < 0 || cueStart < gapStart || cueStart-gapStart > 40 {
		return nil
	}
	gap := text[gapStart:cueStart]
	if strings.ContainsAny(gap, ",.;:!?") || !knowledgeActorBridgeRe.MatchString(gap) {
		return nil
	}
	return []knowledgeParticipant{actor}
}

func explicitCounterparty(text string, sentenceStart, cueEnd int, actorID string, after []knowledgeParticipant) []knowledgeParticipant {
	if len(after) == 0 || after[0].id == actorID {
		return nil
	}
	gapEnd := after[0].start - sentenceStart
	if gapEnd < cueEnd || gapEnd-cueEnd > 40 {
		return nil
	}
	gap := text[cueEnd:gapEnd]
	if strings.ContainsAny(gap, ",.!?;") {
		return nil
	}
	return after[:1]
}

func knowledgeStateForCue(text string, cue knowledgeCue) string {
	prefixStart := maxInt(0, cue.start-18)
	prefix := text[prefixStart:cue.start]
	if knowledgeNegationRe.MatchString(prefix) {
		lower := strings.ToLower(cue.text)
		if strings.Contains(lower, "believ") {
			return "does_not_believe"
		}
		if strings.Contains(lower, "suspect") {
			return "does_not_suspect"
		}
		return "does_not_know"
	}
	if knowledgeAttemptRe.MatchString(prefix) {
		return "attempts_to_recall"
	}
	lower := strings.ToLower(cue.text)
	if strings.Contains(lower, "believ") {
		return "believes"
	}
	if strings.Contains(lower, "suspect") {
		return "suspects"
	}
	return "knows"
}

func knowledgeCueIsNegated(text string, cue knowledgeCue) bool {
	prefixStart := maxInt(0, cue.start-18)
	return knowledgeNegationRe.MatchString(text[prefixStart:cue.start])
}

func knowledgeParticipants(start, end int, mentions []types.MentionRecord, entityMap, canonicalByID map[string]string) []knowledgeParticipant {
	result := make([]knowledgeParticipant, 0, 3)
	for _, mention := range mentions {
		if mention.CharOffset < start || mention.CharOffset >= end {
			continue
		}
		id := entityMap[mention.ID]
		if id == "" {
			continue
		}
		name := canonicalByID[id]
		if name == "" {
			name = mention.Text
		}
		result = append(result, knowledgeParticipant{id: id, name: name, start: mention.CharOffset, end: mention.CharOffset + len(mention.Text)})
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].start < result[j].start })
	return result
}

func knowledgeCues(text string) []knowledgeCue {
	result := []knowledgeCue{}
	add := func(state string, re *regexp.Regexp) {
		for _, match := range re.FindAllStringIndex(text, -1) {
			result = append(result, knowledgeCue{state: state, start: match[0], end: match[1], text: text[match[0]:match[1]]})
		}
	}
	add("shared", knowledgeShareRe)
	add("learned", knowledgeLearnRe)
	add("knows", knowledgeKnowRe)
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].start < result[j].start
	})
	compact := result[:0]
	for _, cue := range result {
		if len(compact) > 0 && compact[len(compact)-1].start == cue.start && compact[len(compact)-1].end == cue.end {
			continue
		}
		compact = append(compact, cue)
	}
	return compact
}

func participantsAroundCue(participants []knowledgeParticipant, start, end int) (before, after []knowledgeParticipant) {
	for _, participant := range participants {
		if participant.end <= start {
			before = append(before, participant)
		} else if participant.start >= end {
			after = append(after, participant)
		}
	}
	return uniqueParticipants(before), uniqueParticipants(after)
}

func uniqueParticipants(values []knowledgeParticipant) []knowledgeParticipant {
	seen := map[string]bool{}
	result := make([]knowledgeParticipant, 0, len(values))
	for _, value := range values {
		if !seen[value.id] {
			seen[value.id] = true
			result = append(result, value)
		}
	}
	return result
}

func participantIdentity(values []knowledgeParticipant) (ids, names []string) {
	for _, value := range values {
		ids = append(ids, value.id)
		names = append(names, value.name)
	}
	return ids, names
}

func hasKnowledgeState(states []types.EvidenceKnowledgeState, expected string) bool {
	for _, state := range states {
		if state.State == expected {
			return true
		}
	}
	return false
}

func hasEpistemicKnowledge(states []types.EvidenceKnowledgeState) bool {
	for _, state := range states {
		switch state.State {
		case "knows", "does_not_know", "attempts_to_recall", "believes", "does_not_believe", "suspects", "does_not_suspect":
			return true
		}
	}
	return false
}

func classifyEvidence(text string, characterIDs []string, named []types.EvidenceTerm, times []string, knowledge []types.EvidenceKnowledgeState) (kind, evidenceType string, confidence float64, rationale string) {
	hasSubject := len(characterIDs) > 0 || len(named) > 0
	switch {
	case discoveryCueRe.MatchString(text) && hasSubject:
		return "event", "discovery", .86, "Contains discovery or revelation language with a named story subject"
	case transitionCueRe.MatchString(text) && hasSubject:
		return "event", "transition", .78, "Contains arrival, departure, movement, or return language with a named story subject"
	case len(characterIDs) >= 2 && interactionCueRe.MatchString(text):
		return "event", "interaction", .84, "Contains two confirmed characters and direct interaction language"
	case stateCueRe.MatchString(text) && hasSubject:
		return "fact", "state", .7, "States an identity, possession, knowledge, work, residence, or named condition"
	case len(times) > 0 && hasSubject:
		return "fact", "time_reference", .72, "Contains an explicit time expression tied to a named story subject"
	case hasKnowledgeState(knowledge, "shared") || hasKnowledgeState(knowledge, "withheld"):
		return "event", "knowledge_transfer", .84, "Names a character communicating information to another confirmed character"
	case hasKnowledgeState(knowledge, "learned"):
		return "event", "knowledge_change", .86, "Names a character acquiring or realizing information"
	case hasEpistemicKnowledge(knowledge):
		return "fact", "knowledge_state", .8, "States that a named character knows, believes, remembers, or suspects information"
	default:
		return "", "", 0, ""
	}
}

func makeEvidenceRecord(item evidenceChapter, paragraphIndex, sentenceIndex, start, end int, text, kind, evidenceType string, characterIDs, characterNames []string, named []types.EvidenceTerm, action string, times []string, knowledge []types.EvidenceKnowledgeState, confidence float64, rationale string) types.EvidenceRecord {
	return types.EvidenceRecord{
		Kind: kind, EvidenceType: evidenceType,
		ChapterID: item.chapter.ID, ChapterIndex: item.globalIndex, Section: item.section, SectionIndex: item.sectionIndex,
		ParagraphIndex: paragraphIndex, SentenceIndex: sentenceIndex, StartOffset: start, EndOffset: end,
		Text: text, CharacterIDs: characterIDs, CharacterNames: characterNames, NamedEntities: named,
		Action: action, TimeExpressions: times, KnowledgeStates: knowledge, Confidence: confidence, Rationale: rationale,
		Status: "detected", Source: "auto",
	}
}

func appendEvidence(result *types.EvidenceData, record types.EvidenceRecord, prior map[string]types.EvidenceRecord, seen map[string]int) {
	chapterKey := record.ChapterID
	if chapterKey == "" {
		chapterKey = fmt.Sprintf("chapter-%d", record.ChapterIndex)
	}
	base := strings.Join([]string{chapterKey, record.Kind, record.EvidenceType, normalizeEvidenceText(record.Text)}, "\x00")
	occurrence := seen[base]
	seen[base] = occurrence + 1
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%d", base, occurrence)))
	record.ID = "evidence-" + hex.EncodeToString(sum[:10])
	if old, ok := prior[record.ID]; ok && hasAuthorEvidenceDecision(old) {
		record.Status = old.Status
		record.AuthorText = old.AuthorText
		record.AuthorNote = old.AuthorNote
		record.Pinned = old.Pinned
		record.ReviewedAt = old.ReviewedAt
	}
	result.Records = append(result.Records, record)
}

func hasAuthorEvidenceDecision(record types.EvidenceRecord) bool {
	return record.Status == "confirmed" || record.Status == "rejected" ||
		record.AuthorText != "" || record.AuthorNote != "" || record.Pinned || record.ReviewedAt != ""
}

func priorEvidence(data *types.EvidenceData) map[string]types.EvidenceRecord {
	result := map[string]types.EvidenceRecord{}
	if data == nil {
		return result
	}
	for _, record := range data.Records {
		result[record.ID] = record
	}
	return result
}

func normalizeEvidenceText(text string) string {
	return strings.ToLower(strings.Join(strings.Fields(text), " "))
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		key := strings.ToLower(value)
		if !seen[key] {
			seen[key] = true
			result = append(result, value)
		}
	}
	return result
}
