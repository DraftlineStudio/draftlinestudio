package indexing

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"draftline/internal/types"

	"github.com/jdkato/prose/v3"
)

const (
	evidenceEngine     = "prose-v3-evidence-v1"
	maxEvidenceRecords = 50_000
)

var (
	discoveryCueRe   = regexp.MustCompile(`(?i)\b(discover(?:ed|s|ing)?|finds?|found|learn(?:ed|t|s|ing)?|reveal(?:ed|s|ing)?|uncover(?:ed|s|ing)?|realiz(?:ed|es|ing)|identif(?:ied|ies|ying)|determin(?:ed|es|ing)|locat(?:ed|es|ing)|notic(?:ed|es|ing))\b`)
	transitionCueRe  = regexp.MustCompile(`(?i)\b(arriv(?:ed|es|ing)|enter(?:ed|s|ing)?|return(?:ed|s|ing)?|depart(?:ed|s|ing)?|leaves?|left|exit(?:ed|s|ing)?|escap(?:ed|es|ing)|fled|walk(?:ed|s|ing)?|ran|run|drov(?:e|en)|driv(?:es|ing)|rush(?:ed|es|ing)?|approach(?:ed|es|ing)?)\b`)
	interactionCueRe = regexp.MustCompile(`(?i)\b(meets?|met|tells?|told|asks?|asked|hands?|handed|calls?|called|faces?|faced|follows?|followed|joins?|joined|attacks?|attacked|helps?|helped|watches?|watched|confronts?|confronted)\b`)
	stateCueRe       = regexp.MustCompile(`(?i)\b(is|are|was|were|has|have|had|knows?|knew|works?|worked|lives?|lived|belongs?|belonged|called|named)\b`)
	timeCueRe        = regexp.MustCompile(`(?i)\b(?:\d{1,2}:\d{2}(?:\s*[ap]\.?(?:m\.)?)?|(?:19|20)\d{2}|monday|tuesday|wednesday|thursday|friday|saturday|sunday|morning|afternoon|evening|night|midnight|noon|dawn|dusk|today|tonight|tomorrow|yesterday|(?:one|two|three|four|five|six|seven|eight|nine|ten|\d+)\s+(?:minutes?|hours?|days?|weeks?|months?|years?)\s+(?:later|earlier|ago))\b`)
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
}

// AnalyzeEvidence builds a conservative fact/event candidate index from exact
// manuscript sentences. It uses the already bundled prose/v3 models and
// deterministic cue rules; no generative model or network call is involved.
func AnalyzeEvidence(book *types.BookData, progress func(types.StoryAnalysisProgress)) *types.EvidenceData {
	chapters := evidenceChapters(book)
	result := &types.EvidenceData{
		Engine:       evidenceEngine,
		LastAnalyzed: time.Now().Format(time.RFC3339),
		Records:      []types.EvidenceRecord{},
		Version:      1,
	}

	hasher := sha256.New()
	for _, item := range chapters {
		fmt.Fprintf(hasher, "%d\x00%s\x00%s\x00%s\x00", item.globalIndex, item.chapter.ID, item.chapter.Title, item.text)
	}
	result.ContentHash = hex.EncodeToString(hasher.Sum(nil))

	prior := priorEvidence(book.Analysis.Evidence)
	mentionToEntity, canonicalByID, mentionsByChapter := evidenceCharacterLookups(book)
	analyzed := analyzeEvidenceChapters(chapters, mentionsByChapter, mentionToEntity, canonicalByID, progress)
	introduced := make(map[string]bool)
	seenIDs := make(map[string]int)

	for position, item := range chapters {
		if len(result.Records) >= maxEvidenceRecords {
			result.Truncated = true
			break
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
				record := makeEvidenceRecord(item, sentence.paragraphIndex, sentence.sentenceIndex, sentence.start, sentence.end, sentence.text, "event", "introduction", newCharacters, newNames, sentence.named, sentence.action, sentence.timeExpressions, .98, "First confirmed appearance in manuscript order")
				appendEvidence(result, record, prior, seenIDs)
			}

			kind, evidenceType, confidence, rationale := classifyEvidence(sentence.text, sentence.characterIDs, sentence.named, sentence.timeExpressions)
			if evidenceType == "" {
				continue
			}
			record := makeEvidenceRecord(item, sentence.paragraphIndex, sentence.sentenceIndex, sentence.start, sentence.end, sentence.text, kind, evidenceType, sentence.characterIDs, sentence.characterNames, sentence.named, sentence.action, sentence.timeExpressions, confidence, rationale)
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

// analyzeEvidenceChapters runs the model-backed per-chapter work with a small
// bounded worker pool. Four workers substantially reduce large-book latency
// without letting background analysis consume every available core.
func analyzeEvidenceChapters(chapters []evidenceChapter, mentionsByChapter map[int][]types.MentionRecord, mentionToEntity, canonicalByID map[string]string, progress func(types.StoryAnalysisProgress)) [][]evidenceSentence {
	result := make([][]evidenceSentence, len(chapters))
	if len(chapters) == 0 {
		return result
	}
	workerCount := runtime.GOMAXPROCS(0)
	if workerCount > 4 {
		workerCount = 4
	}
	if workerCount > len(chapters) {
		workerCount = len(chapters)
	}

	jobs := make(chan int)
	var workers sync.WaitGroup
	var progressMu sync.Mutex
	completed := 0
	for range workerCount {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for position := range jobs {
				item := chapters[position]
				result[position] = analyzeEvidenceChapter(item, mentionsByChapter[item.globalIndex], mentionToEntity, canonicalByID)
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

func classifyEvidence(text string, characterIDs []string, named []types.EvidenceTerm, times []string) (kind, evidenceType string, confidence float64, rationale string) {
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
	default:
		return "", "", 0, ""
	}
}

func makeEvidenceRecord(item evidenceChapter, paragraphIndex, sentenceIndex, start, end int, text, kind, evidenceType string, characterIDs, characterNames []string, named []types.EvidenceTerm, action string, times []string, confidence float64, rationale string) types.EvidenceRecord {
	return types.EvidenceRecord{
		Kind: kind, EvidenceType: evidenceType,
		ChapterID: item.chapter.ID, ChapterIndex: item.globalIndex, Section: item.section, SectionIndex: item.sectionIndex,
		ParagraphIndex: paragraphIndex, SentenceIndex: sentenceIndex, StartOffset: start, EndOffset: end,
		Text: text, CharacterIDs: characterIDs, CharacterNames: characterNames, NamedEntities: named,
		Action: action, TimeExpressions: times, Confidence: confidence, Rationale: rationale,
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
