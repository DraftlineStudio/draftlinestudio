# Book Package

`internal/book/` handles reading and writing `.draftline` project files.

## File Format

`.draftline` files are ZIP archives containing:
- `manifest.json` - Project metadata and chapter index
- `copyright.html` - Copyright page content
- `story_bible.json` - Character and world data
- `beat_sheet.json` - Plot structure beats (optional)
- `foreshadowing.json` - Setup/payoff tracking (optional)
- `knowledge_matrix.json` - Character knowledge states (optional)
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

### Write(path string, book types.BookData, appVersion string) types.SaveResult
Writes a BookData structure to a `.draftline` file.

**Behavior:**
- Creates backup before overwriting (via backup.Create)
- Updates `metadata.modified` timestamp
- Writes all chapters and optional data files
- Returns SaveResult with success/error status

### ReadZipEntry(r *zip.ReadCloser, name string) ([]byte, error)
Helper to read a named entry from an open ZIP archive.

## Version History

- **2.0** - Current format with front_matter/body/back_matter sections
- **1.0** - Legacy format with single `chapters/` directory
