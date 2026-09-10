# Book Package

`internal/book/` handles reading and writing `.draftline` project files.

## File Format

`.draftline` files are ZIP archives containing:
- `manifest.json` - Project metadata and chapter index
- `copyright.html` - Copyright page content
- `story_bible.json` - Character and world data
- `analysis.json` - Entity and relationship analysis (optional/rebuildable)
- `history/index.json` - Chapter snapshot metadata (optional, format 2.2+)
- `history/snapshots/*.html` - Deduplicated chapter versions (optional)
- `beat_sheet.json` - Plot structure beats (optional)
- `foreshadowing.json` - Setup/payoff tracking (optional)
- `knowledge_matrix.json` - Character knowledge states (optional)
- `read_aloud_cast.json` - Per-book Read Aloud voice casting (optional)
- `front_matter/NNN.html` - Front matter chapter content
- `body/NNN.html` - Main chapter content
- `back_matter/NNN.html` - Back matter chapter content

## Functions

### Open(path string) (types.BookData, error)
Reads a `.draftline` file and returns the complete BookData structure.

**Handles:**
- v1.0 → v2.0 migration (old `chapters/` → new `body/`)
- Missing optional files (story_bible, beat_sheet, etc.)
- Character array initialization
- Stable chapter-ID assignment for legacy archives

### Write(path string, book types.BookData, appVersion string) types.SaveResult
Writes a BookData structure to a `.draftline` file.

**Behavior:**
- Creates backup before overwriting (via backup.Create)
- Updates `metadata.modified` timestamp
- Writes all chapters and optional data files
- Preserves embedded chapter history as compressed ZIP entries
- Returns SaveResult with success/error status

### ReadZipEntry(r *zip.ReadCloser, name string) ([]byte, error)
Helper to read a named entry from an open ZIP archive.

### RefreshWordCount(b *types.BookData)
(`wordcount.go`) Recomputes the manuscript word count into `metadata.word_count`. Runs on open and save so the frontend never re-counts the whole manuscript on its main thread.

### WriteWithSnapshots(path string, book types.BookData, appVersion string, snapshots []types.ChapterSnapshotRequest) types.SaveResult

Atomically saves the book and adds deduplicated, bounded chapter snapshots to the archive. AI review application supplies its before/after chapter states together so both recovery boundaries and the resulting manuscript are committed in one archive replacement. `ListChapterHistory` reads metadata for a stable chapter ID; `GetChapterHistorySnapshot` resolves one indexed snapshot and returns its HTML.

## Archive Format Versions

- **2.2** - Stable chapter IDs and embedded chapter version history
- **2.1** - Persisted entity and relationship analysis
- **2.0** - Front-matter/body/back-matter sections
- **1.0** - Legacy format with single `chapters/` directory
