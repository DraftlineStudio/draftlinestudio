# Import (EPUB / DOCX)

Import logic lives in `wails/import.go` (pipeline + routing) and
`wails/import_sanitize.go` (decoder, sanitizer, chaptering). Both bound entry
points — `ImportEPUB`/`ImportEPUBDialog` and `ImportDOCX`/`ImportDOCXDialog` —
run inside `safeImport`, a recover wrapper that turns any panic into a failed
`ImportResult` instead of killing the Wails process (there is no other
recovery on the Wails binding surface). A successful import always clears the session's current file
so Ctrl+S routes through Save As.

`ImportResult` (in `internal/types/results.go`) carries `warnings []string`
for non-fatal issues — skipped documents, dropped images, truncation. The
NewBookWizard import preview renders them.

## EPUB pipeline

1. **Zip guards** — `ziputil.CheckArchive` (entry/total/ratio caps) and
   per-entry `ReadEntry` limits, as everywhere else zips are read.
2. **OPF** — located via `META-INF/container.xml`; metadata (first
   title/creator/publisher), manifest (with the `properties` attribute), and
   spine are parsed namespace-insensitively.
3. **Spine filtering** — only `application/xhtml+xml` and `text/html` items
   import; `properties~="nav"` (EPUB 3 navigation docs) are skipped; raw
   documents over **8 MB** are skipped with a warning.
4. **Decoding** — `decodeToUTF8`: BOM sniff (UTF-16LE/BE, UTF-8), then
   declared xml/meta encoding (Latin-1 family → Windows-1252, UTF-16), then
   UTF-8 validity, then a Windows-1252 fallback that never fails.
5. **Sanitizing** — `parseSpineDoc` parses with the fault-tolerant
   `golang.org/x/net/html` parser (a missing `</body>` can never leak
   `<head>`/`<style>` content) and walks the body emitting `importedBlock`s
   in the editor's dialect only.
6. **Chaptering** — `assembleChapters` (rules below).
7. **Routing** — `routeImportedSection` places each section in
   FrontMatter / Body / BackMatter / the Copyright field, or skips it.

## Sanitizer whitelist (the editor dialect)

| Source | Result |
|---|---|
| `p` | `<p>` (keeps only `text-align: left/right/center/justify` style) |
| `h1–h3` / `h4–h6` | kept / clamped to `<h3>` (chapter/part/scene convention) |
| `blockquote`, `ul`/`ol`/`li`, `hr`, `br` | kept (`hr` renders as the ⁂ scene break) |
| `pre` | `<pre><code>` with internal whitespace preserved |
| `b`/`i`/`strike`/`del`/`cite`/`q` | `strong`/`em`/`s`/`s`/`em`/`em` |
| `strong`/`em`/`u`/`s`/`sub`/`sup`/`code` | kept |
| `div`, `section`, `article`, `figure`, `td`… | transparent: paragraph boundaries; loose text becomes `<p>` (table cells → one `<p>` per cell) |
| `a`, `span`, `font`, unknown inline | unwrapped to their text |
| `img`/`svg`/`picture`, `script`/`style`/`head`/`nav`/media | dropped with subtree; dropped images are counted and reported as a warning |

Text entities decode and re-escape; whitespace collapses per block (never
across blocks), so verse and `<br>` layouts survive. Class/id/epub:*
attributes never survive (TipTap would drop them on first edit anyway — the
sanitizer just makes the loss deterministic and pre-save).

## Chaptering rules

1. A spine document is **one chapter by default**.
2. It splits only at **multiple `<h1>`s**, or multiple `<h2>`s when it has no
   `<h1>` (same philosophy as the DOCX importer's heading1-only splitting).
3. The splitting heading (or a document-leading h1/h2) becomes the chapter
   title and is **removed from content** — the app renders titles itself and
   the EPUB exporter re-adds `<h1>`, so this is what prevents double titles
   on round trips. Intra-chapter h2/h3 stay as part/scene headings; a leading
   h3 is a scene heading, not a title.
4. Content before the first split heading merges into the first chapter.
5. Caps (all warn, never fail): 2 MB per chapter (truncated at a block
   boundary with a visible notice), 40 MB per book, 500 chapters.

## Section routing

Matched on the normalized title first, then leading-content heuristics
(`acknowledgments…`, `all rights reserved`, publisher/trademark lines):

| Section | Destination (Type) |
|---|---|
| cover, contents/TOC, index, landmarks, guide, nav | **skipped** |
| copyright | `book.Copyright` field (later copies → FrontMatter `Copyright`) |
| title page, half title | FrontMatter `Title Page` |
| dedication / epigraph / foreword / preface / introduction / prologue | FrontMatter (canonical Type) |
| author's note, also by … | FrontMatter `Author's Note` / `Also By` |
| epilogue / afterword / appendix / notes / endnotes | BackMatter (`Appendix` for notes) |
| acknowledgments / about the author / glossary / colophon | BackMatter (canonical Type) |
| everything else | Body `Chapter` |

Type labels use the frontend's canonical vocabulary
(`frontend/src/types/draftline.ts` FRONT/BODY/BACK_MATTER_TYPES) so analysis
exclusion (`ShouldAnalyzeChapter`) and section chrome work automatically.
Untitled body sections number as "Chapter N" counting body chapters only.

## DOCX

DOCX import (`parseDOCXDocument`) is unchanged by the EPUB rework apart from
gaining the `safeImport` recover wrapper: paragraphs with `heading1`/`title`
styles start chapters; bold/italic/underline runs map to marks.

## Tests

`wails/import_test.go` — recover wrapper, nav/SVG skip, oversized-doc
warning, conservative-split table, prefix merging, export→import round trip
(no duplicated title heading), routing fixture, section-route table, save
target clearing. `wails/import_sanitize_test.go` — sanitizer whitelist table
(divs, missing `</body>`, entities, pre/verse, images, alignment, tables,
lists) and `decodeToUTF8` encoding table.
