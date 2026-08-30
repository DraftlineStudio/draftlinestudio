package indexing

import (
	"fmt"
	"strings"
	"time"

	"draftline/internal/entityresolution"
	"draftline/internal/types"
)

// AllChapters returns every chapter in global index order: front matter,
// body, then back matter. Mention extraction and relationship analysis MUST
// both use this enumeration so chapter indexes agree across the pipeline.
func AllChapters(book *types.BookData) []types.ChapterItem {
	all := make([]types.ChapterItem, 0, len(book.FrontMatter)+len(book.Body)+len(book.BackMatter))
	all = append(all, book.FrontMatter...)
	all = append(all, book.Body...)
	all = append(all, book.BackMatter...)
	return all
}

// IndexBook runs the two-phase character pipeline on the whole book:
// Phase 1 extracts mention spans from every chapter, Phase 2 resolves them
// into entities. The StoryBible characters and Analysis.EntityResolution are
// rebuilt; manually created characters are preserved.
func IndexBook(book *types.BookData) types.IndexResult {
	if book.StoryBible.Characters == nil {
		book.StoryBible.Characters = []types.Character{}
	}
	if book.Analysis.EntityResolution == nil {
		book.Analysis.EntityResolution = &types.EntityData{Version: 1}
	}

	// Phase 1: mention extraction over stripped text, with book-wide
	// corroboration (a name attested in one chapter validates its
	// sentence-initial uses in another).
	var fullText strings.Builder
	chapterTexts := []string{}
	for _, chapter := range AllChapters(book) {
		text := ""
		if ShouldAnalyzeChapter(chapter) {
			text = StripHTMLForAnalysis(chapter.Content)
		}
		fullText.WriteString(" ")
		fullText.WriteString(text)
		chapterTexts = append(chapterTexts, text)
	}
	allMentions := ExtractBookMentions(chapterTexts)

	// Phase 2: entity resolution, honoring manual splits from prior runs.
	resolver := entityresolution.NewResolver()
	for _, pair := range book.Analysis.EntityResolution.SeparatedPairs {
		resolver.SeparatedPairs = append(resolver.SeparatedPairs, entityresolution.SeparatedPair{
			MentionID1: pair.MentionID1,
			MentionID2: pair.MentionID2,
			Reason:     pair.Reason,
		})
	}
	result := resolver.ResolveEntities(allMentions)
	ClassifyResolvedEntities(result.Entities, allMentions)

	// Re-apply user-confirmed "same person" merges from prior sessions.
	mergeRules := book.Analysis.EntityResolution.MergeRules
	decisions := book.Analysis.EntityResolution.Decisions
	entityRecords := applyMergeRules(ConvertEntitiesToRecords(result.Entities), mergeRules)
	ApplyEntityDecisions(entityRecords, decisions)

	book.Analysis.EntityResolution = &types.EntityData{
		Mentions:       ConvertMentionsToRecords(allMentions),
		Entities:       entityRecords,
		SeparatedPairs: ConvertSeparatedPairsToRecords(result.SeparatedPairs),
		MergeRules:     mergeRules,
		Decisions:      decisions,
		LastResolved:   time.Now().Format(time.RFC3339),
		Version:        1,
	}

	// Rebuild StoryBible characters (manual ones preserved), then assign
	// attributes from a single scan of the full text.
	rebuildCharactersFromEntities(book)
	attrsByName := ExtractAllAttributes(fullText.String())
	newCount := 0
	for i := range book.StoryBible.Characters {
		char := &book.StoryBible.Characters[i]
		if char.IsAutoDetected {
			char.Attributes = LookupAttributes(attrsByName, char.Name, char.Aliases)
			newCount++
		}
	}

	book.IsIndexed = true
	book.LastIndexed = time.Now().Format(time.RFC3339)

	return types.IndexResult{
		Success:         true,
		CharactersFound: len(entityRecords),
		NewCharacters:   newCount,
		Characters:      book.StoryBible.Characters,
		Book:            *book,
	}
}

// SplitCharacterEntity splits an entity into two, moving specified mentions
// to a new entity. This is used when auto-merging incorrectly combined two
// different people (e.g., two characters both called "Ruiz").
func SplitCharacterEntity(book *types.BookData, entityID string, mentionIDs []string, newCanonical string) error {
	if book.Analysis.EntityResolution == nil {
		return fmt.Errorf("no entity resolution data found")
	}

	entities := recordsToEntities(book.Analysis.EntityResolution.Entities)

	resolver := entityresolution.NewResolver()
	for _, pair := range book.Analysis.EntityResolution.SeparatedPairs {
		resolver.SeparatedPairs = append(resolver.SeparatedPairs, entityresolution.SeparatedPair{
			MentionID1: pair.MentionID1,
			MentionID2: pair.MentionID2,
			Reason:     pair.Reason,
		})
	}

	newEntities, err := resolver.SplitEntity(entities, entityID, mentionIDs)
	if err != nil {
		return err
	}

	// Name the new entity: explicit name if given, otherwise the text of its
	// first mention.
	mentionText := make(map[string]string)
	for _, m := range book.Analysis.EntityResolution.Mentions {
		mentionText[m.ID] = m.Text
	}
	for i := range newEntities {
		if newEntities[i].Canonical != "" {
			continue
		}
		if newCanonical != "" {
			newEntities[i].Canonical = newCanonical
		} else if len(newEntities[i].MentionIDs) > 0 {
			newEntities[i].Canonical = mentionText[newEntities[i].MentionIDs[0]]
		}
	}

	book.Analysis.EntityResolution.Entities = ConvertEntitiesToRecords(newEntities)
	book.Analysis.EntityResolution.SeparatedPairs = ConvertSeparatedPairsToRecords(resolver.SeparatedPairs)
	book.Analysis.EntityResolution.LastResolved = time.Now().Format(time.RFC3339)

	rebuildCharactersFromEntities(book)

	// Relationship data references the pre-split entity — stale now.
	book.Analysis.Relationships = nil

	return nil
}
