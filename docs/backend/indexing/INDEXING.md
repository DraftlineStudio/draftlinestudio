# Indexing Package

`internal/indexing/` provides character detection and content analysis for the Story Bible.

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
