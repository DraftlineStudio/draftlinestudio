// Package storysearch provides deterministic, local manuscript search with
// confirmed-character alias expansion. It has no Wails or generative-AI
// dependency and stores no additional copy of the manuscript.
package storysearch

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"draftline/internal/indexing"
	"draftline/internal/types"
)

const (
	defaultLimit     = 100
	maxLimit         = 500
	maxExcerptRunes  = 800
	maxExcerptPieces = 4
)

var (
	paragraphRe = regexp.MustCompile(`\n\s*\n`)
	quotedRe    = regexp.MustCompile(`"([^"]+)"|\x{201C}([^\x{201D}]+)\x{201D}`)
	spaceRe     = regexp.MustCompile(`\s+`)
)

type queryGroup struct {
	alternatives []string
}

type chapterRef struct {
	section      string
	sectionIndex int
	globalIndex  int
	chapter      types.ChapterItem
}

// Search finds scenes containing every query concept. A confirmed character
// concept matches any of that character's names; other words and quoted
// phrases match literally at word boundaries.
func Search(book types.BookData, request types.StorySearchRequest) types.StorySearchResult {
	query := cleanSpace(request.Query)
	result := types.StorySearchResult{Query: query, Matches: []types.StorySearchMatch{}}
	if query == "" {
		result.Error = "Enter a name, place, object, or phrase to search for."
		return result
	}

	interpretedQuery, intent := interpretDetailQuery(query)
	groups, resolved := buildQueryGroups(book, interpretedQuery)
	if len(groups) == 0 {
		result.Error = "The search did not contain any searchable words."
		return result
	}
	result.ResolvedEntities = resolved
	insight := newInsightAccumulator(intent, interpretedQuery, resolved)
	evidenceByChapter := evidenceByChapter(book.Analysis.Evidence)

	limit := request.Limit
	if limit <= 0 {
		limit = defaultLimit
	} else if limit > maxLimit {
		limit = maxLimit
	}

	for _, ref := range allChapters(book) {
		text := indexing.StripHTML(ref.chapter.Content)
		if text == "" {
			continue
		}
		for sceneIndex, paragraphs := range splitScenes(text) {
			matched, pieces := matchScene(paragraphs, groups)
			if len(matched) != len(groups) {
				continue
			}

			result.Total++
			sceneEvidence := evidenceRecordsForScene(evidenceByChapter[ref.globalIndex], pieces)
			insight.add(ref, sceneEvidence)
			if len(result.Matches) >= limit {
				continue
			}
			result.Matches = append(result.Matches, types.StorySearchMatch{
				Section:        ref.section,
				SectionIndex:   ref.sectionIndex,
				ChapterIndex:   ref.globalIndex,
				ChapterID:      ref.chapter.ID,
				ChapterTitle:   chapterTitle(ref),
				SceneIndex:     sceneIndex,
				Excerpt:        buildExcerpt(pieces),
				MatchedTerms:   matched,
				AdditionalHits: max(0, len(pieces)-maxExcerptPieces),
				Evidence:       lightweightEvidence(sceneEvidence),
			})
		}
	}
	result.Insight = insight.finish(result.Total)
	return result
}

func evidenceByChapter(data *types.EvidenceData) map[int][]types.EvidenceRecord {
	result := make(map[int][]types.EvidenceRecord)
	if data == nil {
		return result
	}
	for _, record := range data.Records {
		if record.Status != "rejected" && record.Text != "" {
			result[record.ChapterIndex] = append(result[record.ChapterIndex], record)
		}
	}
	return result
}

func evidenceRecordsForScene(records []types.EvidenceRecord, paragraphs []string) []types.EvidenceRecord {
	scene := strings.ToLower(cleanSpace(strings.Join(paragraphs, " ")))
	result := make([]types.EvidenceRecord, 0, 4)
	for _, record := range records {
		if !strings.Contains(scene, strings.ToLower(cleanSpace(record.Text))) {
			continue
		}
		result = append(result, record)
	}
	return result
}

func lightweightEvidence(records []types.EvidenceRecord) []types.StorySearchEvidence {
	limit := min(len(records), 12)
	result := make([]types.StorySearchEvidence, 0, limit)
	for _, record := range records[:limit] {
		result = append(result, types.StorySearchEvidence{
			ID: record.ID, Kind: record.Kind, EvidenceType: record.EvidenceType,
			Status: record.Status, Confidence: record.Confidence,
		})
	}
	return result
}

func allChapters(book types.BookData) []chapterRef {
	refs := make([]chapterRef, 0, len(book.FrontMatter)+len(book.Body)+len(book.BackMatter))
	global := 0
	appendSection := func(section string, chapters []types.ChapterItem) {
		for i, chapter := range chapters {
			refs = append(refs, chapterRef{section: section, sectionIndex: i, globalIndex: global, chapter: chapter})
			global++
		}
	}
	appendSection("front_matter", book.FrontMatter)
	appendSection("body", book.Body)
	appendSection("back_matter", book.BackMatter)
	return refs
}

func chapterTitle(ref chapterRef) string {
	if title := strings.TrimSpace(ref.chapter.Title); title != "" {
		return title
	}
	return "Chapter " + strconv.Itoa(ref.globalIndex+1)
}

func splitScenes(text string) []([]string) {
	paragraphs := paragraphRe.Split(text, -1)
	scenes := make([][]string, 0, 4)
	current := make([]string, 0, 4)
	flush := func() {
		if len(current) > 0 {
			scenes = append(scenes, current)
			current = nil
		}
	}
	for _, paragraph := range paragraphs {
		paragraph = cleanSpace(paragraph)
		if paragraph == "" {
			continue
		}
		if isSceneBreak(paragraph) {
			flush()
			continue
		}
		current = append(current, paragraph)
	}
	flush()
	return scenes
}

func isSceneBreak(text string) bool {
	compact := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, text)
	if len(compact) < 3 {
		return false
	}
	first := compact[0]
	if first != '*' && first != '#' && first != '-' && first != '~' && first != '.' {
		return false
	}
	for i := 1; i < len(compact); i++ {
		if compact[i] != first {
			return false
		}
	}
	return true
}

func buildQueryGroups(book types.BookData, query string) ([]queryGroup, []types.StorySearchEntity) {
	entities := confirmedEntities(book)
	sort.SliceStable(entities, func(i, j int) bool {
		return longestAlias(entities[i]) > longestAlias(entities[j])
	})

	remaining := query
	groups := make([]queryGroup, 0, 6)
	resolved := make([]types.StorySearchEntity, 0, 2)
	for i := range entities {
		entity := entities[i]
		matchedAlias := ""
		for _, alias := range entity.Aliases {
			if containsTerm(remaining, alias) {
				matchedAlias = alias
				break
			}
		}
		if matchedAlias == "" {
			continue
		}
		groups = append(groups, queryGroup{alternatives: entity.Aliases})
		resolved = append(resolved, entity)
		remaining = removeTerm(remaining, matchedAlias)
	}

	for _, match := range quotedRe.FindAllStringSubmatch(remaining, -1) {
		phrase := match[1]
		if phrase == "" {
			phrase = match[2]
		}
		if phrase = cleanSpace(phrase); phrase != "" {
			groups = append(groups, queryGroup{alternatives: []string{phrase}})
		}
	}
	remaining = quotedRe.ReplaceAllString(remaining, " ")
	for _, term := range strings.FieldsFunc(remaining, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsPunct(r)
	}) {
		if term = strings.TrimSpace(term); len([]rune(term)) > 1 || unicode.IsDigit([]rune(term)[0]) {
			groups = append(groups, queryGroup{alternatives: []string{term}})
		}
	}
	return groups, resolved
}

func confirmedEntities(book types.BookData) []types.StorySearchEntity {
	byName := make(map[string]*types.StorySearchEntity)
	add := func(id, canonical string, aliases []string) {
		canonical = cleanSpace(canonical)
		if canonical == "" {
			return
		}
		key := strings.ToLower(canonical)
		entry := byName[key]
		if entry == nil {
			entry = &types.StorySearchEntity{ID: id, Canonical: canonical}
			byName[key] = entry
		}
		entry.Aliases = uniqueNames(append(append(entry.Aliases, canonical), aliases...))
	}

	for _, character := range book.StoryBible.Characters {
		add(character.ID, character.Name, character.Aliases)
	}
	if data := book.Analysis.EntityResolution; data != nil {
		for _, entity := range data.Entities {
			if entity.DetectionStatus != "accepted" && entity.CharacterID == "" {
				continue
			}
			add(entity.ID, entity.Canonical, entity.Aliases)
		}
	}

	result := make([]types.StorySearchEntity, 0, len(byName))
	for _, entity := range byName {
		result = append(result, *entity)
	}
	return result
}

func uniqueNames(names []string) []string {
	seen := make(map[string]bool, len(names))
	result := make([]string, 0, len(names))
	for _, name := range names {
		name = cleanSpace(name)
		key := strings.ToLower(name)
		if name != "" && !seen[key] {
			seen[key] = true
			result = append(result, name)
		}
	}
	sort.SliceStable(result, func(i, j int) bool { return utf8.RuneCountInString(result[i]) > utf8.RuneCountInString(result[j]) })
	return result
}

func longestAlias(entity types.StorySearchEntity) int {
	longest := 0
	for _, alias := range entity.Aliases {
		longest = max(longest, utf8.RuneCountInString(alias))
	}
	return longest
}

func matchScene(paragraphs []string, groups []queryGroup) ([]string, []string) {
	matched := make([]string, len(groups))
	pieces := make([]string, 0, len(paragraphs))
	for _, paragraph := range paragraphs {
		paragraphMatched := false
		for i, group := range groups {
			if matched[i] != "" {
				continue
			}
			for _, alternative := range group.alternatives {
				if containsTerm(paragraph, alternative) {
					matched[i] = actualTerm(paragraph, alternative)
					paragraphMatched = true
					break
				}
			}
		}
		if paragraphMatched {
			pieces = append(pieces, paragraph)
		}
	}
	compact := matched[:0]
	for _, term := range matched {
		if term != "" {
			compact = append(compact, term)
		}
	}
	return compact, pieces
}

func buildExcerpt(pieces []string) string {
	if len(pieces) == 0 {
		return ""
	}
	if len(pieces) > maxExcerptPieces {
		pieces = pieces[:maxExcerptPieces]
	}
	excerpt := strings.Join(pieces, " … ")
	runes := []rune(excerpt)
	if len(runes) <= maxExcerptRunes {
		return excerpt
	}
	return strings.TrimSpace(string(runes[:maxExcerptRunes])) + "…"
}

func containsTerm(text, term string) bool {
	if term == "" {
		return false
	}
	lowerText, lowerTerm := strings.ToLower(text), strings.ToLower(term)
	start := 0
	for {
		idx := strings.Index(lowerText[start:], lowerTerm)
		if idx < 0 {
			return false
		}
		idx += start
		end := idx + len(lowerTerm)
		if boundaryBefore(lowerText, idx) && boundaryAfter(lowerText, end) {
			return true
		}
		start = idx + 1
	}
}

func removeTerm(text, term string) string {
	lowerText, lowerTerm := strings.ToLower(text), strings.ToLower(term)
	for start := 0; start < len(text); {
		idx := strings.Index(lowerText[start:], lowerTerm)
		if idx < 0 {
			break
		}
		idx += start
		end := idx + len(term)
		if boundaryBefore(lowerText, idx) && boundaryAfter(lowerText, end) {
			return text[:idx] + strings.Repeat(" ", end-idx) + text[end:]
		}
		start = idx + 1
	}
	return text
}

func actualTerm(text, term string) string {
	idx := strings.Index(strings.ToLower(text), strings.ToLower(term))
	if idx < 0 {
		return term
	}
	return text[idx : idx+len(term)]
}

func boundaryBefore(text string, index int) bool {
	if index <= 0 {
		return true
	}
	r, _ := utf8.DecodeLastRuneInString(text[:index])
	return !unicode.IsLetter(r) && !unicode.IsDigit(r)
}

func boundaryAfter(text string, index int) bool {
	if index >= len(text) {
		return true
	}
	r, _ := utf8.DecodeRuneInString(text[index:])
	return !unicode.IsLetter(r) && !unicode.IsDigit(r)
}

func cleanSpace(text string) string { return strings.TrimSpace(spaceRe.ReplaceAllString(text, " ")) }
