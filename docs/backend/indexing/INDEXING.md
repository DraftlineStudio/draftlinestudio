# Indexing Package

`internal/indexing/` provides character detection and content analysis feeding the analysis sidebars and the character pipeline.

## Analysis concurrency

Whole-book analysis is single-flight: Draftline accepts only one manuscript-
scale analysis at a time. Within that run, the ProseV3 linguistic stage and
the evidence stage each use their own bounded worker pool. The configured
Gentle, Balanced, Fast, or Adaptive budget changes those pools only; analysis
never changes the process-wide Go scheduler. Wails bindings, the asset server,
file I/O, and Read Aloud therefore keep the runtime's normal scheduling
capacity while analysis is active.

Adaptive uses the Balanced worker count. Manuscripts at or above 750,000
source bytes additionally use a 4 MiB weighted in-flight payload gate and
512 KiB linguistic batches. This large-manuscript tier exists to bound the
transient Prose document memory multiplied by concurrent work. It is a memory
limit, not a hidden thread-count reduction. A chapter larger than the gate is
kept intact for stable offsets and runs alone.

## Functions

### IndexBook(book *types.BookData) types.IndexResult
Runs the two-phase character pipeline on the whole book: Phase 1 extracts mention spans from every chapter (`ExtractBookMentions`, with book-wide corroboration), Phase 2 resolves them into entities via `internal/entityresolution`, honoring prior manual merge/split decisions. Rebuilds the StoryBible character list (manual characters preserved) and `Analysis.EntityResolution`, and assigns attributes from a single full-text scan.

### IndexBookWithOptions(book *types.BookData, pool AnalysisPoolOptions) types.IndexResult
Same pipeline with an explicit concurrency/memory budget for the model-backed stages.

### SplitCharacterEntity(book *types.BookData, entityID string, mentionIDs []string, newCanonical string) error
Splits an entity into two when auto-merging incorrectly combined different people, moving the given mentions to a new entity.

### ExtractAllAttributes(text string) / LookupAttributes(attrsByName, name, aliases)
Attribute extraction (`text.go`): one scan collects attributes near names across the full text; lookup resolves a character's attributes across its name and aliases. `text.go` also exposes `StripHTML` / `StripHTMLForAnalysis` and `ShouldAnalyzeChapter`.

## Patterns

`patterns.go` (~340 lines) holds `commonWordsLower`, a lowercase map of common English words accessed via `IsCommonWord`, plus false-positive context checks (`IsFalsePositiveContext`, address/street detection). Character detection is no longer regex-driven: the mention pipeline (`mentions.go`) extracts candidate name spans over stripped text, and `classification.go` classifies the resolved entities.

## Usage

`IndexBookWithOptions` runs as part of the consolidated local analysis pipeline (`wails/analysis.go`) and via the `IndexBook` Wails binding. Results populate the character list with auto-detected names.
