# Indexing Package

`internal/indexing/` provides character detection and content analysis for the Story Bible.

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

### IndexBook(book types.BookData) types.IndexResult
Scans all chapters and detects character names based on capitalization patterns, dialogue attribution, and frequency.

**Returns:**
- Detected character names
- Per-chapter word counts
- Character mention counts

### IndexChapter(content string) ([]string, int)
Indexes a single chapter's content, returning detected names and word count.

### DetectCharacterNames(content string) []string
Extracts potential character names from text using pattern matching.

**Detection rules:**
- Capitalized words not at sentence start
- Words following dialogue attribution ("said X", "X replied")
- Excludes common words (the, and, but, etc.)
- Excludes single letters and short words

### ExtractAttributes(content string, name string) map[string]string
Extracts character attributes mentioned near a name (physical descriptions, titles, relationships).

## Patterns

### commonWords
Map of ~100 common English words to exclude from character detection.

### Regex Patterns
- Dialogue attribution: `(?:said|asked|replied|shouted|whispered|muttered)\s+([A-Z][a-z]+)`
- Sentence boundaries: `[.!?]\s+`
- Capitalized words: `\b[A-Z][a-z]{2,}\b`

## Usage

Called when user clicks "Index Book" in the Story Bible panel. Results populate the character list with auto-detected names.
