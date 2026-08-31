package indexing

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
	"unicode"

	"draftline/internal/types"

	"github.com/jdkato/prose/v3"
	"github.com/jdkato/prose/v3/summarize"
)

const storyAnalysisEngine = "prose-v3"

// AnalyzeStory measures narrative chapters with the bundled Prose models.
// Progress is optional and is called after each chapter so UI clients can
// display truthful, chapter-specific work without polling.
func AnalyzeStory(book *types.BookData, progress func(types.StoryAnalysisProgress)) *types.StoryAnalysisData {
	chapters := AllChapters(book)
	analyzable := make([]struct {
		index   int
		chapter types.ChapterItem
		text    string
	}, 0, len(chapters))
	hasher := sha256.New()

	for index, chapter := range chapters {
		if !ShouldAnalyzeChapter(chapter) {
			continue
		}
		text := StripHTMLForAnalysis(chapter.Content)
		if strings.TrimSpace(text) == "" {
			continue
		}
		fmt.Fprintf(hasher, "%d\x00%s\x00%s\x00%s\x00", index, chapter.ID, chapter.Title, text)
		analyzable = append(analyzable, struct {
			index   int
			chapter types.ChapterItem
			text    string
		}{index: index, chapter: chapter, text: text})
	}

	result := &types.StoryAnalysisData{
		ContentHash:  hex.EncodeToString(hasher.Sum(nil)),
		Engine:       storyAnalysisEngine,
		LastAnalyzed: time.Now().Format(time.RFC3339),
		Chapters:     make([]types.ChapterAnalysis, 0, len(analyzable)),
		Version:      1,
	}

	for position, item := range analyzable {
		if progress != nil {
			current := position + 1
			progress(types.StoryAnalysisProgress{
				Phase:        "story",
				Message:      fmt.Sprintf("Analyzing story and pacing in %s", displayChapterTitle(item.chapter, item.index)),
				ChapterIndex: item.index,
				ChapterTitle: item.chapter.Title,
				Current:      current,
				Total:        len(analyzable),
				Percent:      55 + int(math.Round(float64(position)*44/float64(maxInt(1, len(analyzable))))),
			})
		}
		chapter := analyzeStoryChapter(item.chapter, item.index, item.text)
		result.Chapters = append(result.Chapters, chapter)
	}

	result.Overview = aggregateStoryOverview(result.Chapters)
	result.Observations = buildStoryObservations(result.Chapters, result.Overview)
	return result
}

func analyzeStoryChapter(chapter types.ChapterItem, index int, text string) types.ChapterAnalysis {
	doc := summarize.NewDocument(text)
	assessment := doc.Assess()
	wordCount := int(doc.NumWords)
	sentenceCount := int(doc.NumSentences)
	paragraphCount := int(doc.NumParagraphs)

	shortSentences := 0
	longSentences := 0
	for _, sentence := range doc.Sentences {
		if sentence.Length <= 8 {
			shortSentences++
		}
		if sentence.Length >= 25 {
			longSentences++
		}
	}

	verbCount, adverbCount, adjectiveCount, taggedWords := countPOSTags(text)
	dialogueWords := countDialogueWords(text)
	dialoguePercent := percent(dialogueWords, wordCount)
	avgSentence := ratio(wordCount, sentenceCount)
	tempo := tempoScore(avgSentence, dialoguePercent, percent(shortSentences, sentenceCount), percent(longSentences, sentenceCount))

	return types.ChapterAnalysis{
		ChapterID:             chapter.ID,
		ChapterIndex:          index,
		Title:                 chapter.Title,
		WordCount:             wordCount,
		SentenceCount:         sentenceCount,
		ParagraphCount:        paragraphCount,
		SceneBreakCount:       strings.Count(text, "* * *"),
		DialoguePercent:       round1(dialoguePercent),
		AverageSentenceWords:  round1(avgSentence),
		AverageParagraphWords: round1(ratio(wordCount, paragraphCount)),
		ReadingEase:           round1(finite(assessment.ReadingEase)),
		MeanGradeLevel:        round1(finite(assessment.MeanGradeLevel)),
		MeanWordLength:        round1(finite(doc.MeanWordLength())),
		ShortSentencePercent:  round1(percent(shortSentences, sentenceCount)),
		LongSentencePercent:   round1(percent(longSentences, sentenceCount)),
		VerbPercent:           round1(percent(verbCount, taggedWords)),
		AdverbPercent:         round1(percent(adverbCount, taggedWords)),
		AdjectivePercent:      round1(percent(adjectiveCount, taggedWords)),
		TempoScore:            round1(tempo),
		TempoLabel:            tempoLabel(tempo),
		Keywords:              topKeywords(doc.Keywords(), 6),
		ExtractiveSummary:     extractSummary(doc),
	}
}

func countPOSTags(text string) (verbs, adverbs, adjectives, words int) {
	doc, err := prose.NewDocument(text, prose.WithExtraction(false), prose.WithSegmentation(false))
	if err != nil {
		return 0, 0, 0, 0
	}
	for _, token := range doc.Tokens() {
		if !containsLetter(token.Text) {
			continue
		}
		words++
		switch {
		case strings.HasPrefix(token.Tag, "VB"):
			verbs++
		case strings.HasPrefix(token.Tag, "RB"):
			adverbs++
		case strings.HasPrefix(token.Tag, "JJ"):
			adjectives++
		}
	}
	return
}

func containsLetter(text string) bool {
	for _, r := range text {
		if unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

func countDialogueWords(text string) int {
	inDialogue := false
	var segment strings.Builder
	total := 0
	flush := func() {
		if segment.Len() > 0 {
			total += len(strings.Fields(segment.String()))
			segment.Reset()
		}
	}
	for _, r := range text {
		switch r {
		case '“':
			flush()
			inDialogue = true
		case '”':
			flush()
			inDialogue = false
		case '"':
			flush()
			inDialogue = !inDialogue
		default:
			if inDialogue {
				segment.WriteRune(r)
			}
		}
	}
	flush()
	return total
}

func extractSummary(doc *summarize.Document) string {
	paragraphs := doc.Summary(1)
	if len(paragraphs) == 0 {
		return ""
	}
	parts := make([]string, 0, len(paragraphs[0].Sentences))
	for _, sentence := range paragraphs[0].Sentences {
		parts = append(parts, strings.TrimSpace(sentence.Text))
	}
	return strings.Join(parts, " ")
}

func topKeywords(keywords map[string]int, limit int) []types.TermCount {
	items := make([]types.TermCount, 0, len(keywords))
	for term, count := range keywords {
		items = append(items, types.TermCount{Term: term, Count: count})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count == items[j].Count {
			return items[i].Term < items[j].Term
		}
		return items[i].Count > items[j].Count
	})
	if len(items) > limit {
		items = items[:limit]
	}
	return items
}

func aggregateStoryOverview(chapters []types.ChapterAnalysis) types.StoryAnalysisOverview {
	var result types.StoryAnalysisOverview
	result.ChapterCount = len(chapters)
	for _, chapter := range chapters {
		result.WordCount += chapter.WordCount
		result.SentenceCount += chapter.SentenceCount
		result.ParagraphCount += chapter.ParagraphCount
		result.DialoguePercent += chapter.DialoguePercent * float64(chapter.WordCount)
		result.ReadingEase += chapter.ReadingEase * float64(chapter.WordCount)
		result.MeanGradeLevel += chapter.MeanGradeLevel * float64(chapter.WordCount)
		result.TempoScore += chapter.TempoScore * float64(chapter.WordCount)
	}
	result.AverageChapterWords = round1(ratio(result.WordCount, result.ChapterCount))
	result.AverageSentenceWords = round1(ratio(result.WordCount, result.SentenceCount))
	if result.WordCount > 0 {
		weight := float64(result.WordCount)
		result.DialoguePercent = round1(result.DialoguePercent / weight)
		result.ReadingEase = round1(result.ReadingEase / weight)
		result.MeanGradeLevel = round1(result.MeanGradeLevel / weight)
		result.TempoScore = round1(result.TempoScore / weight)
	}
	return result
}

func buildStoryObservations(chapters []types.ChapterAnalysis, overview types.StoryAnalysisOverview) []types.StoryAnalysisObservation {
	observations := []types.StoryAnalysisObservation{}
	if len(chapters) < 2 || overview.AverageChapterWords <= 0 {
		return observations
	}
	for _, chapter := range chapters {
		title := displayChapterTitle(types.ChapterItem{Title: chapter.Title}, chapter.ChapterIndex)
		if float64(chapter.WordCount) < overview.AverageChapterWords*0.45 {
			observations = append(observations, types.StoryAnalysisObservation{
				Kind: "structure", Level: "notice", Title: "Shorter chapter",
				Detail:       fmt.Sprintf("%s is %d words, well below the manuscript average of %.0f.", title, chapter.WordCount, overview.AverageChapterWords),
				ChapterIndex: chapter.ChapterIndex,
			})
		} else if float64(chapter.WordCount) > overview.AverageChapterWords*1.8 {
			observations = append(observations, types.StoryAnalysisObservation{
				Kind: "structure", Level: "notice", Title: "Longer chapter",
				Detail:       fmt.Sprintf("%s is %d words, well above the manuscript average of %.0f.", title, chapter.WordCount, overview.AverageChapterWords),
				ChapterIndex: chapter.ChapterIndex,
			})
		}
		if chapter.LongSentencePercent >= 30 {
			observations = append(observations, types.StoryAnalysisObservation{
				Kind: "pacing", Level: "notice", Title: "Dense sentence pattern",
				Detail:       fmt.Sprintf("%.0f%% of sentences in %s contain at least 25 words.", chapter.LongSentencePercent, title),
				ChapterIndex: chapter.ChapterIndex,
			})
		}
	}
	if len(observations) > 12 {
		observations = observations[:12]
	}
	return observations
}

func tempoScore(avgSentence, dialogue, shortSentences, longSentences float64) float64 {
	// Tempo is a descriptive signal, not a quality score. Short sentences and
	// dialogue raise it; long-sentence concentration lowers it.
	score := 55.0 + (16-avgSentence)*1.8 + dialogue*0.22 + shortSentences*0.18 - longSentences*0.22
	return math.Max(0, math.Min(100, score))
}

func tempoLabel(score float64) string {
	switch {
	case score >= 68:
		return "brisk"
	case score <= 42:
		return "measured"
	default:
		return "balanced"
	}
}

func displayChapterTitle(chapter types.ChapterItem, index int) string {
	if title := strings.TrimSpace(chapter.Title); title != "" {
		return title
	}
	return fmt.Sprintf("Chapter %d", index+1)
}

func ratio(numerator, denominator int) float64 {
	if denominator == 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}

func percent(numerator, denominator int) float64 {
	return ratio(numerator, denominator) * 100
}

func finite(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	return value
}

func round1(value float64) float64 {
	return math.Round(finite(value)*10) / 10
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
