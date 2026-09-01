# Changelog

All notable changes to Draftline will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),

## [0.16.02473] - 2026-09-01

### Changed
- Simplified the status bar's right side to just the three analysis module chips, now labeled **Characters / Plot / Prose** (previously Characters / Story / Pacing; internal module keys are unchanged).
- Added a **Character Map** button next to Story Search that opens the full-screen character codex.

### Removed
- Removed the status-bar AI detection meter (AI analysis lives in its own sidebar panel) and the whole-book word count (already shown at the bottom of the chapter list and in the Writing Dashboard), along with their now-dead styles.

---

## [0.16.02472] - 2026-09-01

### Added
- Defined Storiverse as a persistent universe context around the existing Draftline editor. The manuscript/navigation pane gains Manuscript and Storiverse tabs; the latter begins with linked book titles, and clicking an available title opens its original `.draftline` for editing while keeping the universe loaded.
- Specified live book-to-universe propagation: edits mark the book fingerprint and dependent universe views stale immediately, the existing idle pipeline refreshes only that book, and affected cross-book joins update after the fingerprint is atomically committed.
- Added synchronization safeguards for obsolete analysis revisions, unavailable-source fingerprint views, per-book analysis state, and normal unsaved-change handling while switching linked books.

### Changed
- Removed the obsolete trial/upgrade lock from the planned Storiverse tab. Manuscript and Storiverse are two views of the same open-source Draftline workspace, not separate products or mutually exclusive file modes.

---

## [0.16.02471] - 2026-09-01

### Added
- Defined the Story Fingerprint as Draftline's versioned, evidence-backed representation of a book: entities, relationships, facts, events, time and knowledge changes, recurring details, structural signals, author review provenance, and the exact excerpts required to justify its conclusions.
- Preserved the complete raw evidence browser in the roadmap as the Evidence Archive, opened from later editorial screens through a small archive-folder action while ranked views keep routine writing interfaces uncluttered.
- Made connected Story Fingerprints the core Storiverse architecture. A portable universe folder owns per-book fingerprint snapshots under `.storiverse/fingerprints/`, enabling cross-book continuity, chronology, relationships, knowledge tracking, and search when a linked manuscript is moved, offline, or intentionally absent.

### Changed
- Established that Storiverse never keeps a second full manuscript copy. Adding a book creates a live per-device link to the original `.draftline`; saves and universe reindexing refresh its stable-ID fingerprint, while another device can relink any updated copy without creating a duplicate. DOCX/EPUB conversion creates one author-owned `.draftline`, and portable universe exports retain fingerprints by default.

---

## [0.16.02470] - 2026-09-01

### Added
- Added a persistent, local fact/event evidence index to the normal on-open and idle analysis pipeline. Conservative prose/v3-assisted rules identify introductions, discoveries, interactions, transitions, stated facts, and explicit time references while retaining the exact source sentence, chapter/paragraph coordinates, linked confirmed characters, named terms, cue strength, and a stable content-derived ID.
- Added an Evidence Index tab to the resizable Story Search tool window. Writers can browse event/fact counts, filter the records, inspect why each passage qualified, and jump directly to the exact sentence; Story Search results also display any persisted evidence supporting the matched scene.
- Added the current product roadmap at `ROADMAP.md`, including evidence-backed detail search and timelines, continuity checks, a full-screen Storyboard, previewable character refactoring, and storyline/detail refactoring.
- Added regression coverage for exact evidence provenance, stable IDs after unrelated edits, preserved author review decisions, singleton introductions, false-positive restraint, archive round trips, and Story Search evidence attachment.

### Changed
- Replaced the manuscript-specific Story Search placeholder with neutral guidance applicable to every project.
- Bounded evidence analysis to four concurrent chapter workers, cutting the reference manuscript pass from roughly 11.7 seconds to 3.2 seconds while preserving manuscript-order output and the 50,000-record safety cap.
- Moved full-analysis orchestration out of `app.go` into `analysis.go` and ratcheted the facade from 1,511 to 1,468 lines, preserving the 02468 monolith cleanup as the analysis pipeline grows.
- Advanced the rebuildable `analysis.json` schema to version 3. Older projects remain compatible and acquire evidence during their next normal analysis; character reindex, merge, split, and clear operations invalidate stale evidence before rebuilding it.

---

## [0.16.02469] - 2026-08-31

### Added
- Added a JetBrains-style Story Search tool window along the bottom of the editor. It opens from the project menu, the status bar, or `Ctrl+Shift+F`, resizes vertically without displacing the analysis sidebars, and keeps supporting excerpts visible beside one-click navigation that opens the source chapter and highlights the matching evidence.
- Added deterministic whole-manuscript evidence search in a dedicated `internal/storysearch` package. Multi-term queries find scenes where details converge, quoted text performs phrase search, results stay in manuscript order with first/last occurrence controls, and confirmed characters expand to their known aliases while Needs Review candidates do not.
- Added focused regression coverage for alias expansion, review-candidate exclusion, exact phrases, scene boundaries, result limits, and manuscript ordering.

### Changed
- Story Search runs only on explicit submission rather than every keystroke, avoiding repeated full-manuscript transfer across the Wails bridge. It is entirely local, stores no duplicate manuscript data, and does not invoke AI.
- Preserved the 02468 backend refactor: the search engine and its shared result types live under `internal/`, while the new Wails method is isolated in `story_search.go`; no search logic or duplicated helpers were added back to `app.go`.

---

## [0.16.02468] - 2026-08-31

### Changed
- Extracted the AI HTTP transport layer from `app.go` into a new `internal/ai/providers` package (2,065 → 1,511 lines): Anthropic (with streaming), Gemini, and a single OpenAI-compatible implementation replacing the three near-verbatim OpenAI/Grok/local copies. Providers receive an injected emit closure, so the package never touches the Wails runtime; cancellation, the keyring, and all bound methods stay on the app. Event names and payloads (`ai:token`, `ai:log`) are unchanged.
- The pure Claude Code / Codex CLI helpers (argument building, model resolution, prompt-safe failure messages) moved to the new package with their tests, including the security invariants that prompts never echo into error messages. The CLI drivers themselves deliberately remain in `app.go` next to setup's exec helpers (recorded as backlog).

### Added
- Added an automated size guardrail (`debt_guardrail_test.go`): any source file over 800 lines fails `go test` unless it has a ratchet allowlist entry recorded at its current size — allowlisted files may shrink but never grow. CSS is exempt (`global.css` is an intentional single-file stylesheet).
- Rebuilt `docs/TECHNICAL-DEBT.md` from measured counts: all three active regressions resolved (builds 02465–02468), the CSS co-location recommendation withdrawn per owner direction, and the CLI-driver deferral recorded.

---

## [0.16.02467] - 2026-08-31

### Removed
- Pruned 743 lines of dead CSS from `global.css` (5,014 → 4,271): the retired PlotTree visualization and Issues panel blocks, legacy tools-panel/tools-tab chrome, the pre-redesign dashboard and diff-view rules, PlotWalker status-bar styles, and orphaned one-off selectors — every deletion verified against a full unused-selector audit of the frontend source.

### Changed
- `global.css` stays a single stylesheet by design (per workflow: one home per rule, greppable CSS bugs); a header comment now records that convention and the cross-cutting classes to know before pruning. The mislabeled "New Universe Wizard" banner — which actually holds the app-wide dialog/button/form primitives used by ~26 components — is renamed "Shared dialog + form primitives" so nobody deletes it by mistake.

---

## [0.16.02466] - 2026-08-31

### Changed
- Moved app-level UI state out of the book store into the app store where the rest of it already lived: the status-bar message, the left-panel toggle, and the four self-contained dialogs (book info, new chapter, export wizard, chapter history). Only the save-entangled dialogs (unsaved-changes warning, new-book wizard) remain with the book data.
- The book store now owns exactly its domain — book data, file I/O, autosave, chapter history, characters, and the editor bridge — completing the store decoupling flagged by the technical-debt audit (956 → 825 lines across the two builds).

---

## [0.16.02465] - 2026-08-31

### Removed
- Deleted the dead frontend planning-transform layer flagged by the technical-debt audit: `plotStore` (beats, foreshadowing, secrets, knowledge-matrix wrappers) and the story-bible text wrapper had zero callers since the planning sidebars were retired in 0.16.02449. The `.draftline` file-format fields are untouched — only unused code was removed.
- Removed vestigial UI state that was written but never read: the book store's `darkMode` mirror (theming is driven by settings) and the right-panel toggle, whose Ctrl+] shortcut had silently done nothing.

### Added
- Added test coverage for the live character store path (add/update/delete, highlighting, and the entity-resolution accept/reject rewrite), which the audit found completely uncovered.

---

## [0.16.02464] - 2026-08-31

### Changed
- Rebuilt `docs/TECHNICAL-DEBT.md` from measured line counts instead of remembered ones. The previous version reported `app.go` as resolved at 1,259 lines and test coverage at 0%, both of which had been wrong for some time, and never tracked `global.css` at all.
- Recorded three active regressions: `app.go` regrew to 2,065 lines (+66% since the March refactor, ~800 of them added across two days), `styles/global.css` is a 5,014-line untracked monolith and the largest file in the repo, and the `bookStore.ts` split is a facade — `plotStore` and `storyBibleStore` are stateless transform bags that `bookStore` re-exports through 39 delegating wrappers, so no component imports them directly and the coupling never changed.
- Documented that `callOpenAI`, `callGrok`, and `callLocalAI` are near-verbatim copies of the same OpenAI-compatible request, and that the stated reason for keeping AI providers in `app.go` no longer holds now that `dispatchAI` threads context explicitly.
- Marked the parts of the refactor that held — 11 internal Go packages, the six Analysis panels (all under 320 lines), the settings and editor module splits, `ToolsPanel.tsx` still a slim router, and 31 test files where the doc claimed zero — and named `Analysis/` as the reference shape for the components still to be split.
- Removed the stale `tools/StoryBible/` and `tools/PlotWalker/` module listings; both sidebars were retired in `1b3587c` and `416d64e`.
- Added a guardrail: refresh counts with the reproducible commands in the doc rather than from memory, and treat any file crossing 800 lines as needing a deliberate split-or-justify decision.

---

## [0.16.02463] - 2026-08-31

### Changed
- Removed mechanical readability and grade-level estimates from the Signals overview. The neutral replacement reports manuscript word count, and the one-line summary now describes only measured sentence length, dialogue share, and prose tempo without comparing the manuscript to an age group, school grade, genre, or audience.
- Moved traditional readability formulas into a collapsed details section in Prose. When deliberately opened, they are explicitly described as sentence-and-word-shape calculations that do not measure subject knowledge, terminology, narrative complexity, intended audience, or comprehension.

---

## [0.16.02462] - 2026-08-31

### Fixed
- Imported chapters no longer inherit the book's title: EPUBs that repeat the book name in every spine item's `<title>` (common when chapter numbers are images) now fall back to clean "Chapter N" names instead of naming all 100+ chapters after the book.
- Untitled sections routed to front or back matter now take their section name (Dedication, Epigraph, Acknowledgments, …) instead of a generic chapter number.
- Verified against real commercial and self-published EPUBs: numbered chapters, dedication/epigraph front matter, epilogue/acknowledgments back matter, and copyright pages all land correctly with no crashes.

---

## [0.16.02461] - 2026-08-31

### Added
- EPUB imports now route sections where they belong instead of dumping everything into the body: dedications, epigraphs, title pages, forewords, prefaces, introductions, prologues, and author's notes land in Front Matter; epilogues, afterwords, appendices, acknowledgments, about-the-author pages, glossaries, and colophons land in Back Matter — all with the app's canonical section types.
- The copyright page now fills the book's dedicated Copyright section, and cover, table-of-contents, and index pages are skipped entirely (Draftline generates its own navigation on export).
- The import preview now lists any import warnings (skipped documents, dropped images, truncation) and notes how many front/back matter sections were detected alongside the chapter list.
- Documented the import pipeline, sanitizer whitelist, chaptering rules, and routing table in `docs/backend/import/IMPORT.md`.

### Changed
- Untitled imported sections are numbered "Chapter N" counting body chapters only, so front and back matter no longer shift the numbering.

---

## [0.16.02460] - 2026-08-31

### Fixed
- EPUB chapter layouts are no longer butchered by over-eager splitting: a spine document is now one chapter by default and splits only at multiple `<h1>`s (or multiple `<h2>`s when the document has no `<h1>`). Books that use h2/h3 for part titles and scene headings keep them inline instead of being shattered into dozens of bogus chapters.
- The chapter-opening heading now becomes the chapter title and is removed from the content, so exporting and re-importing no longer stacks a duplicate title heading at the top of every chapter.
- Content before the first chapter heading merges into the first chapter instead of becoming a spurious extra chapter.

### Changed
- Import size guards: a single chapter truncates at 2 MB of content with a visible notice, and imports stop with a warning at 40 MB of total content or 500 chapters — well before the editor could be frozen by a pathological file.

---

## [0.16.02459] - 2026-08-31

### Fixed
- Replaced the EPUB importer's regex HTML munging with a real, fault-tolerant XHTML parser and a whitelist sanitizer that emits only the editor's dialect. Imported chapters no longer lose structure invisibly: divs and sections become clean paragraph boundaries, legacy `<b>/<i>/<strike>` map to proper marks, spans/anchors/classes unwrap to their text, `h4–h6` clamp to `h3`, tables flatten to one paragraph per cell, and text alignment survives.
- A missing `</body>` can no longer leak `<head>`/`<style>`/script content into chapter text, and entities (`&mdash;`, numeric references) now decode correctly.
- Verse and preformatted text survive import: whitespace is collapsed per block instead of across the whole document, `<br>` line breaks are kept, and `<pre>` content imports as a code block with its spacing intact.
- Non-UTF-8 EPUBs no longer import as mojibake: UTF-16 (BOM or declared), Latin-1/Windows-1252, and invalid-UTF-8 documents are detected and decoded to clean UTF-8.

### Changed
- Embedded images and vector art are removed during import (the editor cannot represent them); the import now reports how many were dropped instead of losing them silently.

---

## [0.16.02458] - 2026-08-31

### Fixed
- EPUB and DOCX imports can no longer crash Draftline: a panic anywhere inside an importer now returns a normal import error instead of killing the process (the Wails bridge previously had no recovery, so one bad file took the whole app down).
- The editor pane is now wrapped in the error boundary, so a rendering failure on pathological imported content shows a retry fallback instead of a blank window.

### Changed
- EPUB spine hygiene: only real chapter documents (`application/xhtml+xml`, `text/html`) are imported — SVG covers and other XML resources no longer appear as garbage chapters — and EPUB 3 navigation documents (`properties="nav"`) are skipped.
- Spine documents larger than 8 MB are skipped with a warning instead of being pushed through the importer and the JSON bridge; import results now carry a warnings list for non-fatal issues.

---

## [0.16.02457] - 2026-08-31

### Changed
- Redesigned Pacing as an explainable prose-tempo panel: an explicit overall score and definition replace the ambiguous brightness scale, the chapter strip now uses labeled measured/balanced/brisk bands, and the panel states clearly that tempo describes prose rhythm rather than plot urgency or story quality.
- Added plain-language chapter-to-chapter transition findings, with the three largest tempo rises or drops linked directly to their destination chapters.
- Reworked chapter rows to show exact values and supporting context for tempo, dialogue share, and chapter length. Length now compares against the median chapter instead of whichever chapter happens to be longest, and Reading Ease has moved out of Pacing's controls because it belongs to Prose.
- Relabeled the readability result as a composite grade estimate and added an explanation that it averages five formulas rather than representing a definitive reading age; Signals no longer makes unsupported audience claims from that estimate.

### Added
- Added focused tests for tempo boundaries, median chapter-length comparison, and detection of noticeable neighboring tempo changes.

### Fixed
- Author's Note front matter is no longer included in story, readability, pacing, or character analysis.

---

## [0.16.02456] - 2026-08-31

### Changed
- Redesigned the Writing Dashboard around writing activity only: renamed Manuscript and Today sections with clickable "target {n}k" / "goal {n}" hints, "{pct}%" / "{remaining} to go" progress labels, and a This session row with words written, time writing, and words / min.
- Chapter breakdown now spans all sections: front and back matter collapse to single aggregate rows around numbered body chapters, with a "Show all {N}" / "Show fewer" toggle after 10 rows.

### Added
- Added a 7-day goal-streak row to the Today section, backed by a per-book daily word history kept locally for 60 days — green squares for days with words, accent for today.

### Removed
- Removed the dashboard's AI Detection gauge and AI Anti-Patterns sections (and their now-dead styles) in favor of a compact jump row showing the current chapter's debounced detection score and opening the dedicated AI Analysis panel.

---

## [0.16.02455] - 2026-08-31

### Added
- Added the AI Analysis panel to the analysis sidebar: a semicircular gauge scores the current chapter's AI signal with a plain verdict (Reads human / Mixed signals / Likely AI), refreshed instantly on chapter switch and two seconds after edits.
- Added a whole-book AI scan that scores every chapter a few at a time in the background — the UI never blocks — with a book-order heat strip, a Highest signal top-5 list that jumps to the chapter, and completed scans cached so reopening the panel is instant.
- Added Flagged passages: the two highest-signal chapters at or above the mixed threshold surface their single most AI-reading paragraph as a serif excerpt with a score badge and a Go-to-chapter link.
- Added a per-chapter AI anti-pattern list (overused prose tics such as "eyes widened") with counts against per-chapter limits, plus an explicit disclaimer that the score is a probabilistic heuristic, not a verdict.

---

## [0.16.02454] - 2026-08-31

### Added
- Added the Signals analysis panel as the redesigned hub of Story Analysis: freshness row with status dot and Run again, six manuscript stat tiles (chapters, avg. chapter/sentence words, dialogue %, reading ease, tempo) with since-last-run deltas, a one-line plain-language read of the manuscript, and jump rows into the Prose, Pacing, Chapters, and Worth Reviewing panels.
- Stat tiles show "first run" until a second analysis run exists; the Worth Reviewing jump row carries a live count badge that hides at zero.

### Changed
- Analysis freshness now reads simply "Analyzed …" — engine names no longer appear anywhere in the UI.

### Removed
- Removed the old single Story Analysis pane and its styles; its overview metrics live in Signals, its observations in Worth Reviewing, and its per-chapter cards in the Chapters panel. A saved sidebar state pointing at the old pane falls back to the Writing Dashboard.

---

## [0.16.02453] - 2026-08-31

### Added
- Added the Worth Reviewing analysis sidebar panel: dismissable observation cards (structure, pacing, and other kinds) with per-kind filter tabs and counts, so the list can be worked down to zero.
- Observation cards deep-link into the editor via "Go to chapter" and can be dismissed per book; dismissals persist across analysis runs until the observation's content changes, and a "Restore dismissed" action brings them all back.
- The Worth Reviewing rail glyph now carries a live count badge of undismissed observations.

---

## [0.16.02452] - 2026-08-31

### Added
- Added a Chapters analysis sidebar panel: per-chapter keyword chips and one-line extractive summaries from the last local analysis, presented as a clickable card feed — a skimmable recap of the book so far.
- Chapter cards show word count, scene count, and scene-break count, and clicking a card jumps the editor to that chapter.
- The feed shows the first 8 chapters with a "Show all N chapters" / "Show fewer" toggle; chapters with neither keywords nor a summary (e.g. part dividers) are skipped without disturbing chapter numbering.

---

## [0.16.02451] - 2026-08-31

### Added
- Added a Pacing panel to the analysis sidebar suite: a whole-book tempo heat strip (brighter = faster) with an auto-generated sag note when a stretch of 3+ chapters runs at least 12 tempo points below the book average.
- Added a per-chapter pacing scan list switchable between Tempo, Dialogue, Ease, and Length, with a scene-break column; clicking a row jumps the editor to that chapter.
- Pacing bars highlight outlier chapters (more than 1.5 standard deviations from the mean) in amber; chapters under 20 words render faint and are excluded from the statistics.

---

## [0.16.02450] - 2026-08-31

### Added
- Added the analysis sidebar suite foundation from the design-center layout directions: shared cross-panel navigation (any panel or the dashboard can deep-link to another sidebar pane), per-book since-last-run analysis snapshots, dismissable-observation storage with a live rail badge, semantic status color tokens for both themes, and a shared `.an-*` visual vocabulary for tiles, cards, chips, heat strips, and jump rows.
- Added the Prose analysis sidebar panel: a live sentence-rhythm strip for the current chapter (amber bars flag sentences past 25 words), a whole-manuscript sentence-length histogram, short/long/paragraph stats, readability scores, and a word-class breakdown bar.
- The rhythm strip and manuscript histogram recompute at most once per 2 seconds of typing, so the panel never adds keystroke lag; the rhythm strip works even before the first analysis run, while the remaining blocks show an Analyze-now empty state until analysis exists.

---

## [0.16.02449] - 2026-08-31

### Removed
- Retired the unused planning sidebars ahead of the analysis sidebar redesign: Plot Notes, Story Timeline, Beat Sheet, Foreshadowing Ledger, and Knowledge Matrix no longer appear on the tools rail, and the Story Bible / Plot Walker plugin cards were removed from Settings → Plugins. The Story Analysis pane remains and is now gated by the Story Analysis plugin instead of Plot Walker.
- Removed the retired panels' components and dead CSS. The `.draftline` file-format fields (`beat_sheet`, `foreshadowing`, `knowledge_matrix`, `story_bible` plot notes/timeline), their store reducers, and the Go `story_bible_enabled`/`plot_walker_enabled` settings keys are all retained, so existing project and settings files round-trip unchanged.
- A saved sidebar state pointing at a retired section falls back to the Writing Dashboard on next launch.

---

## [0.16.02448] - 2026-08-31

### Added
- Added bundled local Story Analysis powered by the existing Prose v3 runtime. Draftline now persists per-chapter word, sentence, paragraph, scene-break, dialogue, readability, part-of-speech, keyword, extractive-summary, and descriptive tempo measurements in `analysis.json` without downloading another model or overwriting author-owned planning data.
- Added an activity-aware analysis coordinator. Opening a project schedules analysis, manuscript edits mark Characters, Story, and Pacing as stale, and 15 seconds without another edit starts a consolidated character, relationship, story-structure, and pacing pass. Results from a run that overlaps newer edits are discarded instead of replacing the newer manuscript.
- Added JetBrains-style analysis feedback to the bottom status bar: the active phase/chapter and progress track appear in the center, while compact green/yellow/red module indicators report current, stale, or failed results and can be clicked to run immediately.
- Replaced Plot Walker's placeholder Story Analysis pane with manuscript overview metrics, evidence-based review observations, and per-chapter tempo, keywords, and extractive summaries.
- Added `draftline.story-analysis` to the bundled plugin catalog and introduced capability/resource-profile metadata as the client-side contract for future analysis packs. The Marketplace preview and `docs/architecture/PLUGIN-SYSTEM.md` specify signed, data-only model packs, immutable Hugging Face artifacts, checksums, resource disclosure, atomic rollback, and a no-arbitrary-code boundary.

### Changed
- Automatic analysis now owns automatic character indexing on project open, preventing the old standalone index request from racing the consolidated pipeline.
- Story-analysis work runs only after inactivity or an explicit request; manuscript-scale NLP remains off the typing path.

---

## [0.16.02447] - 2026-08-31

### Added
- Added chapter version history inside `.draftline` archives (format 2.2). Chapters now carry stable IDs so history follows them through renames and reordering; changed chapter content is snapshotted after ten minutes of actual writing activity, deduplicated by content hash, and retained at up to 50 versions per chapter / 1,000 per project.
- Added File → Chapter History, with a version timeline, side-by-side word-level comparison against the current chapter, and explicit restore. Restoring first checkpoints the current chapter, making the operation reversible.
- Added Settings → Application → Activity-based saving. Disabling it stops both the five-second paused-edit save and automatic ten-minute history snapshots; manual save remains available.

### Changed
- Ordinary saves preserve unchanged history as already-compressed ZIP entries instead of inflating every snapshot into memory and recompressing it. The comparison dialog and diff engine are lazy-loaded so version history does not increase editor startup cost.
- Older `.draftline` projects receive stable chapter IDs in memory when opened and persist them on their next save; archives without history remain fully compatible.

### Fixed
- Activity-driven history creates no duplicate or idle snapshots: only chapters changed during the active window are considered, and an unchanged content hash is skipped.

---

## [0.16.02446] - 2026-08-31

### Security
- Converted Claude Code and Codex editing subprocesses into text-only execution paths. Claude receives an explicitly empty tool list with safe mode, slash commands and MCP discovery disabled. Codex ignores user configuration and project rules, disables its shell plus browser, filesystem-adjacent, application, image, skill, hook, and sub-agent capabilities, inherits no shell environment, and retains a disposable empty workspace with a read-only fallback sandbox.
- Prevented raw Claude stdout/stderr or fallback stream JSON from reaching the visible result and error panes. Failures now return only fixed, allowlisted authentication, rate-limit, model, or generic diagnostics.
- Draftline now ignores arbitrary global Claude/Codex executables on `PATH` and runs only exact-version managed installations (`Claude Code 2.1.251`, `Codex CLI 0.151.0`). Managed runtimes live separately in the platform-local `draftline/ai/claude` and `draftline/ai/codex` directories, with shared pinned Node tooling under `draftline/ai/node`.

### Documentation
- Reclassified automatic character indexing on open as intended, advertised product behavior rather than an audit defect.

---

## [0.16.02445] - 2026-08-31

### Added
- Licensed Draftline under the MIT License.
- Added a root project README covering Draftline's purpose and features, local-versus-AI privacy boundaries, source-build and test commands, repository layout, `.draftline` files, linear build versioning, and the official `draftline.ink` website.

---

## [0.16.02444] - 2026-08-31

### Changed
- Reorganized `docs/ai-audit/`: moved the five historical audit reports (and the gitignored 182 MB Qodana SARIF) into `resolved/`, and added `RESOLUTION-2026-08-31.md` — a landing report of the 19 fixes landed in builds 02425–02443, known residuals, deferred items, and a manual-test checklist.

---

## [0.16.02443] - 2026-08-31

### Added
- Structural round-trip tests for the `internal/export` package, which
  previously had only `helpers_test.go` while `docx.go`, `epub.go`, `pdf.go`,
  and `print.go` were untested. New `export_test.go` exports a small book to
  each format and validates: the DOCX archive is a valid zip carrying the
  mandatory OOXML parts, with chapter text and metadata escaped exactly once
  (no double-escaping, no raw `&`); the EPUB archive leads with a stored
  `mimetype` entry and its `content.opf` / chapter XHTML carry correctly
  escaped titles and verbatim body entities; and both the PDF and print PDF
  exports produce a parseable `%PDF...%%EOF` envelope whose startxref, xref
  object count, trailer `/Size`, and actual object markers all agree.
  Audit ref: consensus finding J / Sol QA-001.

## [0.16.02442] - 2026-08-31

### Changed
- Stopped the per-keystroke whole-book word recount on the renderer thread.
  `StatusBar`, `ChapterPanel`, and the writing `Dashboard` previously re-parsed
  and re-counted every chapter's HTML on every render, and `ChapterPanel` /
  `Dashboard` re-rendered on every unrelated store update (isDirty,
  statusMessage, isAutoSaving toggles) via bare `useBookStore()` subscriptions.
  Those two components now subscribe through narrow `useShallow` selectors, and
  all three memoize `countBookWords` / per-chapter counts on the book reference
  so the scan only runs when content actually changes. Counts still update after
  every edit; behavior is otherwise identical.
  Audit ref: consensus finding E/F "no-wholebook-recount-on-typing" / Sol PERF-001.

## [0.16.02441] - 2026-08-31

### Changed
- Reduced the algorithmic complexity of the relationship-analysis path
  (`internal/indexing`) without changing output. `AnalyzeBook` now buckets
  mentions by chapter in a single O(M) pass instead of rescanning every mention
  once per chapter (O(C×M)); `detectCharacterEvents` resolves endpoint names
  through a prebuilt entityID→canonical map instead of scanning all entities per
  relationship (~O(R×E)); and `scenes.go` compiles its scene-break / paragraph
  regexes once at package level instead of recompiling them for every chapter.
  These are hygiene refactors: on realistic books the analysis pass was already
  fast (tens of ms), so the change is not measurable end to end — it removes the
  quadratic shapes so the cost stays linear as manuscripts grow. Existing
  indexing, entityresolution, and relationship determinism tests remain green.
  (audit ref: Fable5 C3)

## [0.16.02440] - 2026-08-31

### Fixed
- Print-ready PDF export (`internal/export/print.go`) now writes a correct
  cross-reference table. Previously object offsets were fabricated as `i*100`
  and `startxref` as `buf.Len()-20`, producing structurally invalid PDFs that
  strict readers and print shops could reject. Object bodies are now buffered in
  ID order and their true byte offsets recorded as each is written, mirroring
  `pdf.go`, and `startxref` points at the real `xref` keyword. Added
  `TestPrintPDFXrefOffsets`, which parses the emitted xref/startxref and asserts
  every entry offset lands on its `N 0 obj` marker.
  (audit ref: Fable5 M7)

## [0.16.02439] - 2026-08-31

### Fixed
- AI diff-apply no longer flattens the whole chapter to plain text. Paragraphs the
  user did not change (untouched or rejected) are now preserved as their exact
  original `<p>` HTML — bold, italic, links, font-size and color survive. Whole-
  paragraph accepts keep the revised HTML verbatim; only a paragraph with mixed
  accept/reject decisions inside it is reconstructed from plain-text word chunks.
  (audit ref: Fable5 C1)

### Changed
- Paragraphs are now aligned by an LCS edit script over their text instead of by
  positional index, so inserting or deleting a paragraph no longer cascades into
  spurious full-paragraph diffs on every following paragraph. (audit ref: Fable5 M3)


## Versioning System

This project does NOT use Semantic Versioning (SemVer).

Instead, it uses a Linear Build Versioning system.
Format: MAJOR.MINOR.BUILD

* **MAJOR / MINOR:** Incremented manually when major feature sets or milestones are reached.
* **BUILD:** A monotonic counter that increments by 1 for every single change/commit to the main branch. One fix = one number.

If a package manager or strict SemVer parsing is a hard requirement for your workflow, this project may not be a fit for you.

If this versioning system has a formal name, I am unaware of it, feel free to raise an issue in Github and hit me with a "WeLl AcTuAlLy" if you know the name.


## [0.16.02438] - 2026-08-31

### Fixed
- Re-indexing a book now invalidates the stored relationship graph so it can no longer reference stale entity IDs. Entity IDs (`entity-N`) are positional and get reassigned every time `IndexBook` resolves entities, but `IndexBook` never cleared `book.Analysis.Relationships` — unlike `MergeCharacterEntities` and `SplitCharacterEntity`, which both null it. As a result, after a plain re-index (e.g. after the manuscript text changed) the previously analyzed relationship edges, scenes, interactions, and events still pointed at the old numbering, silently mapping to the wrong characters or dangling entirely. `IndexBook` now sets `book.Analysis.Relationships = nil` after entity resolution, matching the merge/split behavior, so stale edges can't be read and callers re-run relationship analysis against the fresh entity IDs. Added `relationships_test.go` regression asserting that analyzing relationships, mutating the text, and re-indexing leaves `Relationships` cleared.
- Audit ref: fable5-2026-08-30 — Fable C2 (invalidate-relationships-on-reindex).

---

## [0.16.02437] - 2026-08-31

### Fixed
- Dialogue detection now works on real (curly-quoted) manuscripts. `findDialogueRanges` in the relationship co-occurrence indexer had a broken quote table: its first two entries were duplicate straight double-quotes `"`, the curly double pair U+201C/U+201D was absent entirely, and straight single-quote `'` was treated as a dialogue delimiter. On any manuscript typeset with curly quotes, dialogue detection therefore found nothing and produced no directed dialogue interactions, while apostrophes in possessives/contractions ("Kira's") fabricated phantom dialogue spans. The table now recognizes straight double `"`, curly double `“ ”`, and guillemets `« »`, aligned with the mention scanner's `isQuoteInitial`; single quotes are excluded so apostrophes can no longer be mistaken for dialogue delimiters. Added `relationships_test.go` cases asserting a directed dialogue interaction is produced from curly-quoted dialogue and that apostrophes yield no dialogue ranges.
- Audit ref: fable5-2026-08-30 — Fable C4 (curly-quote-dialogue).

---

## [0.16.02436] - 2026-08-30

### Fixed
- AI subprocess (Codex/Claude) stdout/stderr draining no longer stalls the request for the full 180 s deadline on an over-long output line. The pipes were drained with a `bufio.Scanner` capped at 1–2 MB; a single line longer than the buffer made `Scan()` stop permanently, so the reader goroutine exited, the pipe stopped draining, the child blocked on write once the OS buffer filled, and `cmd.Wait()` hung until the deadline. All three subprocess readers now use a new `drainLines` helper built on `bufio.Reader.ReadString` that truncates the retained/logged portion of any line to `maxDrainLine` (2 MB) while consuming the remainder, so draining continues for the full lifetime of the pipe and the child can never block. Added `drain_test.go` covering the over-long-line regression, CRLF/empty-line parity, partial reads, and EOF without trailing newline.
- Audit ref: fable5-2026-08-30 — Fable M5 (subprocess-longline).

---

## [0.16.02435] - 2026-08-30

### Security
- Cap manifest reference amplification when opening a `.draftline` archive. `ziputil` bounds the ZIP's entry count and per-entry/aggregate declared size, but a manifest could reference the same valid entry an unbounded number of times, loading its decompressed content into memory on each reference and bypassing the intended aggregate memory bound (a small archive amplifying to gigabytes). `Open` in `wails/internal/book/open.go` now rejects manifests declaring more than `MaxManifestRefs` (5000) combined chapter/section references before loading any content, and tracks cumulative decompressed bytes across the whole open operation via a running budget, erroring once the total exceeds `MaxTotalLoadedBytes` (500 MB). Added `book_test.go` coverage for a manifest with an absurd number of references.
- Audit ref: master-audit-report-2026-08-30 — Sol REL-003 (cap-manifest-refs).

---

## [0.16.02434] - 2026-08-30

### Fixed
- Surface chapter read errors instead of silently opening empty chapters. `Open` in `wails/internal/book/open.go` read each manifest-referenced chapter with `content, _ := ReadZipEntry(...)`, so a missing, oversized, or corrupt entry became an empty chapter — which a subsequent autosave could then persist over the still-intact source, causing data loss. `Open` now fails with a precise error naming the offending chapter title and file, distinguishing an absent entry from a size/read failure (new `ziputil.ErrEntryNotFound` sentinel + `errors.Is`). Genuinely-optional, rebuildable files (`analysis.json`, `story_bible.json`, beat sheet, etc.) remain optional; `copyright.html` is tolerated when absent but no longer silently dropped on a read error. Added `book_test.go` cases covering a manifest that references a missing entry and one that references an oversized entry.
- Audit ref: master-audit-report-2026-08-30 — Consensus D / Sol REL-001 (surface-chapter-read-errors).

---

## [0.16.02433] - 2026-08-30

### Fixed
- Serialized backend settings writes. `saveSettings` in `wails/frontend/src/store/appStore.ts` fired `SaveSettings(next)` with no ordering, so concurrent callers (sidebar width, `sidebar_active_section`, lane view, `ai_mode`/`ai_model`, `custom_dictionary`) could have their backend writes land out of order and drop a setting (lost update). Each call now merges into in-memory state synchronously and chains the backend `SaveSettings` through a module-scoped `saveChain` promise (mirroring `bookStore`'s save serialization), snapshotting the newest merged state inside each chained write so writes are ordered and no patch is lost. `saveSettings` now returns the chained promise; the previously-floating callers in `ToolsPanel.tsx` and `CharactersView.tsx` explicitly discard it.
- Audit ref: fable5-2026-08-30 — M2 (serialize-savesettings).

---

## [0.16.02432] - 2026-08-30

### Fixed
- Synchronized shared backend state across Wails goroutines. Wails dispatches each bound method on its own goroutine, and `a.settings` / `a.currentFile` were read (AI dispatch, save, getters) and written (`startup`, `SaveSettings`, open/import/write paths) in `wails/app.go` and `wails/import.go` with no lock, and `a.legacyAPIKey` was written outside `apiKeyMu` in `SetAPIKey`/`ClearAPIKey`/`startup` — allowing torn reads and lost updates (worst case a Save racing an import that just cleared the save target). Added a `sync.RWMutex` (`a.stateMu`) guarding `settings` and `currentFile` behind small `get`/`set` accessors, moved the `legacyAPIKey` writes under `apiKeyMu`, and made every AI path snapshot the settings it needs under the lock before doing any HTTP/subprocess work so no lock is ever held across I/O.
- Audit ref: fable5-2026-08-30 — M1 (backend-state-mutex).

---

## [0.16.02431] - 2026-08-30

### Security
- Bounded AI provider response reads. The OpenAI, Gemini, Grok, Anthropic error-body, and local-endpoint paths in `wails/app.go` previously read provider HTTP response bodies with unbounded `io.ReadAll` (and the local endpoint decoded an unbounded JSON body), allowing a hostile or malfunctioning endpoint to exhaust memory with an arbitrarily large body. Each read now goes through a new `readAIResponseBody` helper that caps the body at 16 MB (`maxAIResponseBytes`, comfortably above max-output-tokens payloads) via `io.LimitReader` and returns a clear "response too large" error on overflow. The streaming Anthropic success path already used a bounded `bufio.Scanner` and is unchanged.
- Audit ref: sol5.6-2026-08-30 — SEC-007 (bound-ai-reads).

---

## [0.16.02430] - 2026-08-30

### Security
- Redacted AI activity-log output for the Claude Code CLI path (`callClaudeCodeCLI` in `wails/app.go`). The subprocess stdout stream-json lines and raw stderr lines were forwarded verbatim to the `ai:log` runtime event (visible in the UI and screenshots), leaking prompt/manuscript-derived content. The path now parses the structured stream and emits only allowlisted status strings (`Streaming…`, `Done`, `Error`) to `ai:log`, at parity with the Codex path; raw stream/stderr content is kept only in the opt-in debug log via `logging.AIContent` (gated by `ai_debug_logging`). Result extraction and error handling are unchanged.
- Audit ref: sol5.6-2026-08-30 — SEC-002 (claude-log-redaction).

---

## [0.16.02429] - 2026-08-30

### Security
- Tightened filesystem permissions on locally cached user data. Rolling backups (`internal/backup/backup.go`) now create the per-project backup directory `0700` and write `backup.N.draftline` and `info.json` as `0600` (was `0755`/`0644`), so a full manuscript copy is no longer world/group-readable on multi-user systems. Recent-projects writes (`AddRecentProject`/`RemoveRecentProject`/`ClearRecentProjects` in `app.go`) now use `fsutil.WriteFileAtomic` at `0600` (was `os.WriteFile` `0644`). Opening a project best-effort tightens any pre-existing loose directory/file via `os.Chmod` (errors ignored; Unix bits are a no-op on Windows).
- Audit ref: sol5.6-2026-08-30 — SEC-001 / SEC-004 (backup-perms).

---

## [0.16.02428] - 2026-08-30

### Changed
- Removed dead CSS from `frontend/src/styles/global.css`: the replaced character-card story-bible UI (`.character-card*`, `.character-role*`, `.char-field*`, `.char-attr`, `.character-attributes`, `.character-auto-badge`, `.character-aliases*`, `.alias-tag`/`.alias-more`, `.aliases-label`, `.character-badges`, `.character-stats*`/`.character-stat`, `.chapter-mention*`, `.characters-header/stats/count/actions/sort`, `.sort-label`, `.sort-btn`, `.character-form*`, `.character-highlight-btn`, `.character-name/-desc-preview/-expanded/-card-actions`), the old diff-view block (`.diff-view`, `.diff-controls`, `.diff-count`, `.diff-para*`, `.diff-insert`, `.diff-delete`, `.diff-apply-row`, `.diff-nav-controls`), `.scope-toggle`/`.scope-btn`, `.voice-findings`, `.story-bible-subnav`/`.bible-nav-btn`, and `.book-analysis-modal`/`.analysis-finding*`. Each selector was grep-confirmed to have zero references across all `.tsx` files before removal; still-live neighbors (`.bible-add-btn`, `.beat-card`, `.foreshadow-card`, `.wizard-choice-tile`, `.tool-card`, `.ai-run-btn`, `.knowledge-matrix`, `.character-highlight`, `.chars-sort-label`) were preserved.
- Audit ref: master-audit-report-2026-08-30 — finding H (Dead CSS in global.css, CssUnusedSymbol ×196).

---

## [0.16.02427] - 2026-08-30

### Fixed
- Floating promises: added explicit `void` handling to fire-and-forget async calls so promise rejections are no longer silently swallowed. Covers `App.tsx` (settings/recent-project init, Ctrl+N/O/S keyboard shortcuts), `main.tsx` (dictionary load), `WelcomeScreen.tsx` (remove recent project), `editor/InlinePrompt.tsx` (Ctrl+Enter submit), `dialogs/NewBookWizard.tsx` (Enter to create), and `dialogs/settings/index.tsx` (CLI status checks). Behavior preserved; only rejection is made non-silent.
- Audit ref: master-audit-report-2026-08-30 — Qodana floating-promises (fire-and-forget, non-settings).

---

## [0.16.02426] - 2026-08-30

### Changed
- Go idiom nits: `ziputil.ReadEntry` now uses `defer func() { _ = rc.Close() }()` to match the codebase's explicit-ignore idiom, and `app_test.go` compares the cancelled context error with `errors.Is(..., context.Canceled)` instead of `==`.
- Audit ref: master-audit-report-2026-08-30 — Qodana `GoUnhandledErrorResult` (`ziputil/ziputil.go:58`) + `GoDirectComparisonOfErrors` (`app_test.go:30`).

---

## [0.16.02425] - 2026-08-30

### Removed
- Dead Go code: unused exported functions `NormalizeToken` and `GetNameHead` (`entityresolution/honorifics.go`), `IsTypo` (`entityresolution/levenshtein.go`), `GetNicknames` (`entityresolution/nicknames.go`), and the unused global var `CommonWords` (`indexing/patterns.go`). All verified unreferenced across the `wails/` tree including tests. Character-pipeline leftovers from the prose/v3 rewrite.
- Audit ref: master-audit-report-2026-08-30 — Qodana `GoUnusedExportedFunction` ×4 + `GoUnusedGlobalVariable` ×1 (dead Go code).

---

## [0.16.02424] - 2026-08-30

### Added
- Three code-audit reports under `docs/ai-audit/`: an independent Fable 5 pass, a consolidated master report cross-referencing Fable 5 + GPT-5.6 Sol + a Qodana static-analysis run, and Sol's own report. The 182 MB Qodana SARIF is intentionally not tracked (history bloat); its findings are distilled in the master report

---

## [0.16.02423] - 2026-08-30

### Added
- The tools sidebar's open/closed state and active pane are now saved with the rest of the app preferences: it opens on the Writing Dashboard by default, restores whichever pane it was left on, and no longer resets to closed after opening the Characters codex or restarting — writers who prefer a single sidebar can close it once and it stays closed

### Fixed
- If a saved sidebar pane's plugin has been disabled, the sidebar falls back to the Dashboard instead of opening a dead pane

---

## [0.16.02422] - 2026-08-30

### Changed
- Line Edit is now a restrained, selective prose pass: it changes only sentences with a concrete phrasing, repetition, syntax, readability, or rhythm problem and explicitly preserves effective prose instead of rewriting for variety
- Copy Edit is now strictly mechanical—spelling, grammar, punctuation, syntax, capitalization, hyphenation, and number-format consistency—and no longer receives the prose guide or attempts continuity/style changes
- Line Edit and Copy Edit automatically use lightweight models while Expand, Smooth, Custom, and inline generation retain the configured full model: Claude Haiku, Codex Luna/available Mini at low reasoning, OpenAI Mini, and Gemini Flash
- AI Studio mode descriptions now state the narrower responsibilities of Line Edit and Copy Edit

### Performance
- Codex lightweight passes reuse its local model-availability cache, run ephemerally, and avoid maximum reasoning; Claude CLI lightweight passes use low effort and skip session persistence

### Tests
- Added prompt-restraint, mechanical-copy-edit, lightweight-routing, provider-model, and Codex fast-tier regression coverage

---

## [0.16.02421] - 2026-08-30

### Fixed
- Switching AI Studio to Codex no longer passes a leftover Claude, Gemini, Grok, Llama, or Mistral model name to the Codex CLI; existing settings are also sanitized in the backend so Codex falls back to its supported account default
- Codex failures now show a concise actionable message instead of dumping CLI session diagnostics and submitted manuscript text into the AI Studio error panel
- Provider rows no longer display the active provider's shared model name under every inactive route

### Changed
- Codex CLI output is requested as uncolored JSON and drained privately; only safe lifecycle messages are sent to the AI Studio activity log

### Tests
- Added regression coverage for provider-model isolation, Codex command arguments, and prompt-safe failure messages

---

## [0.16.02420] - 2026-08-30

### Added
- `docs/frontend/AI-STUDIO-GAPS.md` — what the AI sidebar redesign shows but was deliberately deferred: Pass Strength (needs backend intensity support), per-provider API keys (the single key slot limits the switcher to one API row), and per-route model memory

---

## [0.16.02419] - 2026-08-30

### Changed
- Rebuilt the AI Studio sidebar per the redesign: a provider quick-switcher bar sits under the header showing the active route (Claude Code / Codex / configured API provider / Local) with a live ready dot, and its dropdown switches routes in one click — unconfigured routes appear dimmed as "Not configured" and route to Settings, with "Manage providers…" alongside
- Editing modes are now icon cards (including the new Copy Edit); the Style Options panel is a collapsible card with four-stop dot tracks and an "N of 8 on" count replacing the range sliders; the custom-prompt area keeps the `@ai` in-text tip; the Run button lives in a pinned footer
- Both CLI statuses are checked on sidebar mount so the switcher is accurate without opening Settings

### Removed
- ~300 lines of dead AI-panel CSS from two superseded layouts (the pre-redesign mode grid and the flat list this replaces); the new styles live in a component-scoped stylesheet

---

## [0.16.02418] - 2026-08-30

### Added
- A Copy Edit AI mode: a conservative correctness pass that fixes grammar, punctuation, misspellings, doubled words, wrong homophones, and tense/continuity slips — and never rephrases for style. Intentional fragments, pacing comma-splices, and character-voice dialogue are explicitly off-limits; the author's voice stays verbatim except where an error is corrected
- Copy Edit uses the targeted per-paragraph diff format (like Line Edit and Smooth), so only changed paragraphs come back for review

### Tests
- Prompt coverage: copy_edit carries the correctness instructions and none of the enrichment/restyle language; unknown modes still fall back to line_edit

---

## [0.16.02417] - 2026-08-30

### Fixed
- The spell-check regression test's fetch mock now satisfies strict TypeScript checking (`as unknown as typeof fetch`), unblocking `tsc --noEmit` and the production build

---

## [0.16.02416] - 2026-08-30

### Fixed
- Spell checking now canonicalizes curly and alternate apostrophes before dictionary lookup, so valid contractions such as `couldn’t`, `wouldn’t`, and `isn’t` are no longer marked as misspelled
- Suggestion deduplication compares canonical apostrophe forms, preventing a typographic contraction from receiving the visually identical ASCII-apostrophe word as its correction
- Replacement suggestions preserve the manuscript's apostrophe style
- Confirmed character names and aliases are now supplied to spell checking as a transient per-project lexicon, preventing names such as Hanlon, Mara, or Ionescu from receiving red underlines without polluting the personal custom dictionary
- Character possessives inherit the ignored root-name behavior automatically

### Tests
- Added regression coverage for four apostrophe variants, live dictionary lookup, possessive dictionary roots, duplicate custom entries, suggestion deduplication, and multi-word character-name lexicon expansion

---

## [0.16.02415] - 2026-08-30

### Added
- A "Codex" option in Settings › AI Studio beside Claude Code: the same guided setup wizard (install runtime & CLI → sign in) now serves both CLIs, with a "Sign in with ChatGPT…" button that runs the browser OAuth flow in-app — no terminal `/login` required for either assistant
- AI Studio recognizes Codex mode: configured-state detection, setup guidance panes (not installed / sign-in required), and run labeling
- Codex model field is a free-text override, left blank by default so the CLI's own current model is used

---

## [0.16.02414] - 2026-08-30

_Developed in parallel with builds 02409–02413 and renumbered at merge time (originally 02410–02411 on its branch)._

### Added
- OpenAI Codex CLI as a third AI mode ("codex"), letting ChatGPT Plus/Pro/Team accounts power AI features the same way Claude.ai accounts do: installed through the identical bundled-Node pipeline (`@openai/codex` pinned to 0.151.0), invoked non-interactively with the prompt piped via stdin (never on the command line), running read-only sandboxed in an isolated home with only the ChatGPT auth credentials — the user's own Codex config and MCP servers can never hang a rewrite
- In-app sign-in intercept for Codex: `codex login` runs as a hidden background process that opens the browser OAuth flow and reports completion back to the app — no terminal, no `/login` incantations
- The final assistant message is captured via `--output-last-message` rather than scraped from the progress stream; no default model is hardcoded, so the CLI's own current default is used unless the user sets an override

### Tests
- Codex argv construction (stdin placeholder always last, model flag only with override, git-check/sandbox flags present) and graceful not-installed error

---

## [0.16.02413] - 2026-08-30

### Changed
- Redesigned the in-editor character sidebar per the "chapter-first + inline detail" design: characters present in the chapter being written come first with their local mention counts, everyone else follows with a mini per-chapter presence strip, and each row click-expands an inline card (meta, presence strip, top-3 ties, jump links) without anything else moving
- The sidebar can pre-select a character in the Characters codex ("Open in Characters"), jump to a character's first-appearance chapter, and still toggle in-text highlighting
- Sidebar glyph and panel now say "Characters" — the last user-visible "Cast" is gone, and the old CastQuickRef component and cast.css are deleted

---

## [0.16.02412] - 2026-08-30

### Fixed
- The tools sidebar header now matches the 38px chrome height of the editor toolbar and manuscript panel header — it previously sized itself from padding and sat ~6px taller, breaking the horizontal chrome line across the top of the workspace

---

## [0.16.02411] - 2026-08-30

### Fixed
- The compact Characters sidebar now lists only manually created or confirmed characters; Needs Review candidates remain available only in the review workspace
- Strongest-tie associations and generated character events are hidden whenever either endpoint is unconfirmed, preventing output such as a detective meeting a street or neighborhood fragment
- Fresh relationship analysis now excludes Needs Review mentions before building interactions, relationships, and meeting events
- Visible relationship totals now count only confirmed-character relationships
- Legacy auto-detected cards without an admission status no longer leak into precision-first sidebars; re-detection classifies them before display

### Tests
- Added confirmed-character scope coverage for manual, accepted, review, rejected, and legacy candidates, including mixed relationship and event participants

---

## [0.16.02410] - 2026-08-30

### Changed
- Replaced the clipped click-a-row character merge mode with a dedicated searchable **Merge Character** dialog
- Merge targets can be searched by canonical name or alias and show their mention counts for disambiguation
- The dialog previews the duplicate-to-canonical direction before confirmation; the searched-for target keeps its display name and the duplicate becomes an alias

### Fixed
- Character merging no longer appears to be a dead button when toolbar space clips its prior instruction banner
- Escape and clicking outside the merge dialog now cancel the merge without closing the Characters workspace

---

## [0.16.02409] - 2026-08-30

### Added
- Confirmed and rejected character-candidate decisions are now stored in entity analysis by canonical name and known aliases, allowing them to survive rebuilt entity IDs
- Auto-detected cards now use a clearer **Not a character** action that permanently suppresses the false detection; manually created cards retain the normal Delete action

### Fixed
- Re-detecting a manuscript no longer resurrects false-positive candidates that the writer already rejected
- Ambiguous saved decisions are skipped when their names match multiple newly resolved entities, preventing an old rule from silently changing the wrong character

### Tests
- Added coverage for accepted/rejected decision restoration across changed entity IDs and safe handling of ambiguous shared aliases

---

## [0.16.02408] - 2026-08-30

### Added
- Character candidates now carry persisted person/non-person evidence, entity kind, detection score, and an `accepted`, `review`, or `rejected` status
- Added a Needs Review view to the character Codex for uncertain people and story actors such as organizations, groups, ships, and places, keeping the default Characters view precision-first without permanently discarding ambiguity
- Added Confirm Character for review candidates; confirmation is author curation and survives future re-detection
- Added `docs/architecture/CHARACTER-DETECTION.md`, documenting the local pipeline, archive persistence, triage model, and real-manuscript evaluation gates

### Fixed
- Obvious low-value object/noise candidates are retained for diagnostics in analysis but no longer become character cards
- Rejected diagnostic candidates are excluded from scene, relationship, and event maps so hidden noise cannot inflate Codex relationship counts
- EPUB import now prefers visible section headings over repeated book-level `<title>` metadata and recognizes acknowledgment/copyright content markers, allowing story-only indexing to exclude those pages after import

### Tests
- Added entity-kind/status triage, rejected-card exclusion, visible EPUB heading, and author-review workflow coverage

---

## [0.16.02407] - 2026-08-30

### Fixed
- Entity resolution no longer transitively combines established characters through a shared word; ambiguous multi-entity matches remain separate instead of creating super-characters such as the corpus's Aelin/Rowan and Maeve/Yrene clusters
- Bare single words no longer fuzzy-merge on one edit, preventing unrelated pairs such as Mart/Mark, Bank/Bonk, and Chris/Christ from contaminating one another
- Name-component matching is now ordered and limited to name edges rather than arbitrary token subsets, reducing family-name and faction-name collisions
- Canonical names are selected using occurrence frequency and penalties for common grammar, repeated tokens, and accidental plurals; `Daniel Hanlon` now outranks the one-off `Daniel Hanlons`
- Expanded sentence-grammar filtering for modal verbs, determiners, and inflected verbs that previously attached themselves to nearby names

### Tests
- Added regression cases for transitive bridge merges, bare-word fuzzy collisions, plural canonical names, and sentence grammar leaking into names

---

## [0.16.02406] - 2026-08-30

### Fixed
- Character and relationship analysis now excludes chapter headings, navigation, scripts, styles, and known non-story sections such as covers, contents, copyright, acknowledgments, glossaries, and indexes
- EPUB files that package many chapters inside one spine document are now split at internal H1-H3 boundaries, restoring real chapter-level heatmaps, appearances, relationship timing, and navigation
- Imported non-story sections retain a semantic type so future re-detection does not turn editors, publishers, glossary terms, or contents entries into cast members

### Tests
- Added regression coverage for heading-derived aliases, acknowledgment-page names, non-story classification, and multi-chapter EPUB spine files

---

## [0.16.02405] - 2026-08-30

### Fixed
- `.draftline` archive format 2.1 now persists entity mentions, relationships, manual events, and character merge/split corrections in `analysis.json` instead of silently losing them when a project is closed
- Legacy files that claim to be indexed but contain no analysis are marked for re-detection when opened, preventing a stale character Codex with unavailable merge, split, relationship, and mention controls
- Re-detecting characters now preserves author-curated roles, descriptions, appearance, personality, motivation, and notes when the character can be matched uniquely by name or alias

### Tests
- Added archive round-trip coverage for analysis, correction rules, manual events, and legacy indexed-state migration

---

## [0.16.02404] - 2026-08-30

### Fixed
- **Critical data-loss bug:** importing an EPUB or DOCX while a project was open left the backend's save target pointing at the previous project's file, so Ctrl+S silently overwrote it with the imported book instead of opening the Save As dialog. Imports now clear the session save target — an imported book is a new, unsaved project and Save always routes through Save As
- Added regression tests: a successful import clears the save target (with a real minimal EPUB fixture); a failed import leaves the open project's save target untouched
- If this bit you: the overwritten file's pre-overwrite content is in the rolling backups (`%AppData%\draftline\backups\`, folder identified by `info.json` → `original_path`, newest is `backup.1.draftline`)

---

## [0.16.02403] - 2026-08-30

### Fixed
- Header and pane buttons in the Characters view no longer inherit the tool-card button's stacked-layout quirks (full width, zero side padding, 4px sibling offset) — Re-Detect and Close now sit level with proper padding

---

## [0.16.02402] - 2026-08-30

### Fixed
- The Characters view now stretches to fill the workspace: it was missing the flex-grow the old view had, leaving a phantom empty strip along the right edge
- "Re-Detect" and "SORTED BY" no longer wrap onto two lines; header and toolbar controls keep their width and the filter input shrinks first when space is tight

---

## [0.16.02401] - 2026-08-30

### Added
- The codex lane style (Grid vs Heatmap) is now remembered in app settings alongside the theme and other preferences — pick a favorite once and the Characters view opens that way every time (reopening in Heatmap also restores its ranked sort)

---

## [0.16.02400] - 2026-08-30

### Changed
- Switching to Heatmap view auto-selects "Mentions (descending)" sorting — the heatmap reads as a ranked order, so the biggest presences go on top; the sort remains freely changeable afterwards

---

## [0.16.02399] - 2026-08-30

### Changed
- The tools sidebar (including the character quick reference) now closes while the Characters codex is open, giving the swimlane grid the full width; it returns when the codex closes

---

## [0.16.02398] - 2026-08-30

### Fixed
- Long names, aliases, mention excerpts, and event text no longer overflow their containers in the Characters view: the header count, merge banner, detail pane, split-mention checklist, and excerpt cards all clamp or wrap correctly

---

## [0.16.02397] - 2026-08-30

### Added
- `docs/frontend/CHARACTERS-VIEW-GAPS.md` — what the character redesign implies but the current data model can't honestly deliver yet: sentence-quality mention excerpts (needs backend sentence text), precise jump-to-mention (needs offset→editor position mapping), computed tiers, role auto-suggestion, and a light-theme cell-color pass

---

## [0.16.02396] - 2026-08-30

### Added
- An inline role selector on each row in Heatmap view (Protagonist / Antagonist / Supporting / Minor / Other — Draftline's existing role taxonomy), so roles can be triaged for the whole cast without opening each character's edit form

---

## [0.16.02395] - 2026-08-30

### Added
- An ego mini-map in the character detail pane: the selected character centered with their six strongest ties at fixed spoke positions — the relationship-graph feel without ever laying out the whole web
- An "Appears in" per-chapter presence strip and up to three mention excerpts (drawn from the manuscript around each recorded mention) with "Jump to chapter" navigation back into the editor

---

## [0.16.02394] - 2026-08-30

### Changed
- Replaced the force-directed relationship graph with a swimlane character grid: one row per character, one column per chapter, cell intensity showing mention density — readable at 8 characters or 800, where the old graph collapsed into a hairball
- The new Characters workspace keeps everything else: filter, detection, add/edit/delete, split, merge ("Same as…"), relationship strengths, and key events, now in a persistent right-hand detail pane
- Two lane styles: Grid view (discrete chapter cells with numbered columns) and Heatmap view (continuous per-chapter intensity), with sorting by first appearance or mention count

### Removed
- The D3 force-graph dependency (~62 kB of minified bundle); character visuals are now pure CSS

---

## [0.16.02393] - 2026-08-30

### Changed
- Advanced the feature line to 0.16 for the character codex redesign
- Renamed the "Cast" plugin to "Characters" in the Plugins settings page — this is a character codex, not a theater production; the full view rename lands with the redesigned workspace (in-editor sidebar rename deferred to its own redesign)

---

## [0.15.02392] - 2026-08-30

### Fixed
- The root App component now selects only book existence, title, and file path instead of the entire mutable book object, so ordinary editor keystrokes no longer rerender the whole application shell
- Corrected the welcome-screen effect dependencies while narrowing the subscription

---

## [0.15.02391] - 2026-08-30

### Fixed
- Clearing an AI diff while its lazily loaded engine is in flight no longer allows the obsolete diff to reappear after navigation
- Applying a diff now uses the latest accept/reject decisions made while the diff engine loads instead of an earlier snapshot
- Added two editor-store regression tests (12 frontend tests total)

---

## [0.15.02390] - 2026-08-30

### Fixed
- Save-and-proceed and project close now stop if the saved snapshot became stale because the manuscript changed while the save was in flight
- A late autosave from a discarded project can no longer update the replacement project's file path, dirty flag, or autosave status
- Added regression coverage for both stale-transition and cross-project autosave races (10 frontend store tests total)

---

## [0.15.02389] - 2026-08-30

### Changed
- Manuscript files (`*.draftline`) are now gitignored so a book saved into the repo directory can never be committed

---

## [0.15.02388] - 2026-08-30

### Added
- `docs/ai-audit/overnight-run-report.md` — morning report for the overnight audit-fix run (13 fixes, builds 0.15.02375–0.15.02387): what landed per fix with verification evidence, open items, final gate results, and the merge procedure

### Changed
- `docs/ai-audit/gpt-sol.md` — refreshed the stale "Testing and release readiness" annotation (exports, AI request coordination, and frontend-store tests landed during the run; only imports and settings remain untested), unified the finding 12/13 annotation markers with the rest of the file, and linked the run report from the scoreboard

---

## [0.15.02387] - 2026-08-30

### Changed
- `App.tsx`, `EditorPanel.tsx`, and `StatusBar.tsx` now subscribe to `useBookStore` through narrow `useShallow` selectors instead of bare `useBookStore()` calls, so they only re-render when a field they actually read changes
- Purely mechanical performance change — no behavior differences

---

## [0.15.02386] - 2026-08-30

### Changed
- `editorStore.ts` now lazy-loads the diff engine (`utils/diff.ts`) via dynamic `import()` instead of a static import, so Vite can code-split it as AIStudio already intended
- The ineffective-dynamic-import build warning for `diff.ts` is gone; the module ships as its own 2.29 kB chunk and the main bundle shrinks from 833.39 kB to 831.28 kB (gzip 245.75 kB to 244.83 kB)

---

## [0.15.02385] - 2026-08-30

### Changed
- Added a Vitest test harness to the frontend (`npm test` in `wails/frontend`, node environment, Wails bindings mocked at the module boundary)
- Added 8 regression tests for the save pipeline proving the 0.15.02383/0.15.02384 data-loss fixes: an edit during an in-flight autosave stays dirty and persists the newer content next cycle; failed or cancelled saves block save-and-proceed and close-project (dialog stays open, book retained); Save As surfaces errors but stays silent on a dismissed picker; saves are strictly serialized
- Tests were mutation-checked against the pre-fix code: reverting 0.15.02383 fails 6 of 8, reverting only 0.15.02384 fails the 4 transition tests

---

## [0.15.02384] - 2026-08-30

### Fixed
- A failed save no longer lets "Save and proceed" continue into New/Open and discard the unsaved manuscript: the unsaved-changes dialog now stays open (pending action retained) on a save error, and cancelling the Save As picker returns to the dialog instead of being treated as consent to discard
- Closing a project with unsaved changes now aborts (book and dirty state untouched, welcome screen not shown) if the save fails or is cancelled, with an explanatory status message
- Save As now surfaces failures via a "Save failed" status message instead of failing silently

---

## [0.15.02383] - 2026-08-30

### Fixed
- Editing while an autosave is in flight no longer marks the newer edits as saved: a document revision counter makes a completing save clear the dirty flag only if nothing changed since its snapshot, so the newer state persists on the next autosave cycle
- All saves (manual, autosave, Save As, close, save-and-proceed) are serialized through one queue — a queued save always writes the newest book state and two saves can never interleave — and every successful save now writes the file path back into the book (autosave previously dropped it)
- Books imported through the New Project wizard now arm the autosave timer immediately instead of waiting for the first manual edit

---

## [0.15.02382] - 2026-08-30

### Fixed
- Inline prompt (generate content) Cancel button is no longer disabled while a generation is in flight — the exact moment cancellation matters most
- Cancelling the inline prompt via the Cancel button or the Escape key now aborts the in-flight AI request through `CancelRewrite` instead of letting it run to completion in the background

---

## [0.15.02381] - 2026-08-30

### Fixed
- AI requests are now serialized through a single request slot: a second concurrent request gets a clean "an AI request is already in progress" error instead of silently overwriting the first request's cancel handle and interleaving streamed tokens
- Cancel now actually works for the Local AI, OpenAI, Gemini, and Grok providers — their HTTP calls are context-aware (`http.NewRequestWithContext`) with the private 120-second client timeouts removed so the request's 180-second deadline governs; cancellation surfaces as a friendly "cancelled" message
- A stale cleanup from a finished/cancelled request can no longer clear a successor request's cancel registration (generation-counter guard); custom-prompt rewrites now route through the same guarded dispatch path as all other AI calls

---

## [0.15.02380] - 2026-08-30

### Security
- Claude Code CLI prompts are now piped via stdin instead of being placed on the command line, keeping manuscript-derived content out of every exec path (and lifting the ~32K Windows command-line length limit on prompt size)
- `BatchCommand` (Windows .cmd/.bat launcher) now rejects cmd.exe metacharacters (`& | < > ^ "`, CR/LF/NUL) and any script path or argument containing two or more `%` signs — on the /S /C command line cmd.exe expands `%VAR%` and even `%%VAR%%` (percent-doubling only escapes inside batch-file bodies), so multi-`%` strings are refused outright while a lone `%` (e.g. "100% done") stays allowed — refusing to run instead of assembling a hostile command line

---

## [0.15.02379] - 2026-08-30

### Fixed
- Exports (PDF, DOCX, print) now decode HTML entities — `&amp;`, `&nbsp;`, `&mdash;`, `&#8217;`, etc. — instead of emitting them literally in output text or double-escaping them in DOCX
- Non-breaking spaces from `&nbsp;` are normalized to regular spaces in exported plain text

---

## [0.15.02378] - 2026-08-30

### Security
- Auto theme now defaults to manual dawn/dusk times (no network call); IP-based geolocation is strictly opt-in
- When location-based times are enabled, geolocation now uses HTTPS (ipwho.is) instead of plaintext HTTP (ip-api.com)
- Settings dialog now discloses that the location option sends the device IP to ipwho.is and derived coordinates to sunrise-sunset.org

---

## [0.15.02377] - 2026-08-30

### Security
- Gemini API requests no longer place the API key in the URL (`?key=`), where it could leak into proxy and server logs; the key is now sent via the `x-goog-api-key` request header

---

## [0.15.02376] - 2026-08-30

### Security
- Cleared both high-severity npm dev-dependency advisories via `npm audit fix`: nanoid 3.3.15 → 3.3.18 (GHSA-28wg-ghj8-5hjv, GHSA-2v37-7h3g-55p8) and postcss 8.5.16 → 8.5.26 (GHSA-fxqj-rqcc-2cmp, GHSA-r28c-9q8g-f849)
- `npm audit --audit-level=high` now reports zero vulnerabilities; lockfile diff confined to the two version/integrity bumps

---

## [0.15.02375] - 2026-08-30

### Fixed
- Fixed the AI-detection "Uniform sentence structure" flag never firing: the condition `sentenceVariety < 30 && sentenceVariety > 70` was always false; it now correctly triggers on `sentenceVariety > 70` (high score = uniform, AI-like structure), matching the sibling flag thresholds

---

## [0.15.02374] - 2026-08-30

### Added
- Integrated the local pure-Go prose/v3 statistical NLP pipeline to distinguish people from places, facilities, and organizations without generative AI or network access
- Added real-prose regression coverage for Chicago street addresses, cross-chapter place/person name collisions, possessives, titled names, and invented science-fiction names

### Changed
- Combined NLP classifications with Draftline's deterministic honorific, possessive, alias, and address rules instead of treating either system as an infallible classifier
- Made non-person evidence and corroboration book-wide while allowing strong person evidence to resolve genuine name/location collisions
- Batched NLP work across a bounded concurrent worker pool so explicit full-book detection remains responsive on large manuscripts

### Fixed
- Prevented `Chicago`, `Hubbard Street`, `Ontario`, `Wells Ave`, `Lower Wacker Drive`, and their component tokens from being promoted into the cast while correctly retaining `Detective Daniel Hanlon`, `Hanlon's`, and `Mara Ionescu`

---

## [0.15.02373] - 2026-08-30

### Changed
- Annotated the Sol 5.6 security audit (`docs/ai-audit/gpt-sol.md`) with a per-finding status scoreboard: findings 3–6 fixed by builds 02351–02366, findings 1–2 (frontend data-loss flows) remain the open release blockers

---

## [0.15.02372] - 2026-08-30

### Added
- Added regression tests for backup rotation and restore, book save/open round-trips, atomic file writes, archive decompression limits, and Node archive extraction safety

---

## [0.15.02371] - 2026-08-30

### Changed
- Tidied Go module metadata; promoted go-keyring to a direct dependency

---

## [0.15.02370] - 2026-08-30

### Fixed
- A transient OS keyring read failure no longer permanently caches an empty API key until restart; the next AI request retries the keyring

---

## [0.15.02369] - 2026-08-30

### Fixed
- Clearing the API key now removes the plaintext fallback copy from settings.json even when the OS keyring is unavailable — previously the one machine that stored the key on disk was the one where clearing silently failed

---

## [0.15.02368] - 2026-08-30

### Changed
- AI Studio determines API configuration from the backend's has-key flag instead of reading a key value from frontend state

---

## [0.15.02367] - 2026-08-30

### Changed
- The settings dialog no longer displays or round-trips the stored API key: the field only accepts a replacement key (submitted through the dedicated keyring binding on save) and gains a Clear button

---

## [0.15.02366] - 2026-08-30

### Security
- Opening .draftline files and importing EPUB/DOCX now enforce archive limits (entry count, per-entry size, total size, compression ratio) with bounded decompression reads, preventing crafted archives from exhausting memory

---

## [0.15.02365] - 2026-08-30

### Fixed
- Backup restores write the manuscript atomically, and a failed pre-restore safety backup is logged instead of silently ignored

---

## [0.15.02364] - 2026-08-30

### Fixed
- Backup copies and metadata are written atomically, and backup-rotation failures are now surfaced in the log instead of swallowed

---

## [0.15.02363] - 2026-08-30

### Fixed
- Book saves are crash-safe: the archive is written to a same-directory temporary file and atomically renamed over the manuscript, and the produced archive is validated (manifest readable) before it replaces the user's file — a disk-full or crash mid-save can no longer truncate the manuscript

---

## [0.15.02362] - 2026-08-30

### Security
- The Claude Code CLI install is pinned to an explicit version instead of latest, keeping the setup supply chain auditable

---

## [0.15.02361] - 2026-08-30

### Security
- Node.js archive extraction caps entry count and per-entry/total decompressed sizes, counted on actual bytes written rather than declared headers

---

## [0.15.02360] - 2026-08-30

### Security
- Archive-supplied file modes are masked during extraction, stripping setuid/setgid/sticky and non-permission bits

---

## [0.15.02359] - 2026-08-30

### Security
- Tar symlinks in the Node.js archive must resolve inside the destination directory and hardlink entries are rejected outright

---

## [0.15.02358] - 2026-08-30

### Security
- Node.js archive entry paths are confined to the destination directory, rejecting absolute, drive-prefixed, and traversal paths (zip-slip)

---

## [0.15.02357] - 2026-08-30

### Security
- The Node.js bootstrap download is verified against pinned per-platform SHA-256 checksums (matching the official nodejs.org SHASUMS256.txt) with a download size cap; unlisted platforms are refused rather than installed unverified

---

## [0.15.02356] - 2026-08-30

### Security
- AI debug log files are created user-only (0600) in a user-only (0700) directory

---

## [0.15.02355] - 2026-08-30

### Security
- AI debug logging — which can include manuscript text and prompts — is now explicitly opt-in via a new setting with a toggle in AI Studio settings, and is off by default

---

## [0.15.02354] - 2026-08-30

### Security
- settings.json is written atomically with user-only (0600) permissions

---

## [0.15.02353] - 2026-08-30

### Fixed
- On machines without an OS keyring, the API key is retained in user-only settings.json instead of being lost, and saving settings no longer wipes the stored key

---

## [0.15.02352] - 2026-08-30

### Security
- The API key never crosses the frontend/backend bridge in either direction: settings expose only a has-key flag, saved settings discard any key field, and new SetAPIKey / HasAPIKey / ClearAPIKey bindings handle the key directly

---

## [0.15.02351] - 2026-08-30

### Security
- AI provider API keys are stored in the OS keyring (Windows Credential Manager / macOS Keychain / Secret Service) instead of plaintext settings.json, with a one-time migration that strips the plaintext copy only after the keyring accepts the key

---

## [0.15.02350] - 2026-08-30

### Added
- Added a JetBrains-inspired Find/Replace bar scoped to the current chapter, opened with `Ctrl+F` or `Ctrl+H`
- Added live match highlighting, current-match navigation, match counts, case-sensitive search, and whole-word search
- Added formatting-safe Replace and single-transaction Replace All operations that can be undone in one step
- Added roadmap designs for character refactoring and AI-assisted story-detail/event refactoring with evidence-based human review

### Changed
- Advanced the feature line to 0.15 for chapter editing and future manuscript-refactoring work
- Defined fixture-based accuracy gates and a stable-ID entity-resolution foundation to replace the current prototype character-detection heuristics before refactoring ships

---

## [0.14.02349] - 2026-08-30

### Changed
- Replaced typo-js's exhaustive edit-distance-two suggestion generator with a prewarmed, length-indexed bounded Damerau-Levenshtein search
- Moved dictionary parsing and suggestion-index construction to application startup in the existing worker, eliminating first-use context-menu initialization

### Fixed
- Reduced representative uncached spelling-suggestion latency from 1.3–3.3 seconds to roughly 130–185 milliseconds while preserving five ranked corrections

---

## [0.14.02348] - 2026-08-30

### Fixed
- Normalized possessive spellings to their lexical root so adding `FleetCom's` or `FleetCom’s` stores `FleetCom`, and all possessive uses are then recognized automatically
- Migrated existing custom-dictionary possessive entries to deduplicated root words

---

## [0.14.02347] - 2026-08-30

### Added
- Offline grammar checking with distinct grammar and style diagnostics, contextual explanations, and one-click corrections
- Persistent spelling diagnostics with asynchronous suggestions and a personal custom dictionary
- A searchable, JetBrains-inspired Plugins settings page for enabling or disabling bundled features
- Feature controls for Spelling, Grammar Check, Cast, Story Bible, Plot Walker, and AI Studio
- A registry-based feature model designed to accommodate community plugins in a future release

### Changed
- Moved expensive spelling suggestions into a dedicated web worker, keeping editor input and context menus responsive
- Routed Claude Code AI work through a dedicated subprocess flow so long-running generation does not block the main application process
- Improved Claude Code discovery on Windows by launching the native executable or Node entry point directly instead of relying on fragile command-shim invocation
- Preserved existing user settings while adding backwards-compatible defaults for newly introduced feature flags
- Normalized backend and frontend version reporting to `0.14.02347`

### Fixed
- Fixed Claude Code integration failures that surfaced as `not recognized as an internal or external command` in the AI sidebar
- Fixed persistent spell-check lag caused by synchronous suggestion generation on the editor thread
- Fixed reactive editor-store subscriptions that caused excessive rerenders, instability, and intermittent crashes during editing and AI review
- Fixed spelling markers disappearing when the custom context menu took focus
- Fixed misspelling suggestions being calculated for the keyboard cursor instead of the word that was right-clicked
- Fixed sidebar width and feature settings being discarded when settings were persisted

### Removed
- Removed legacy trial, Indie, and paid-edition branding from the welcome screen for the open-source release

---

## [0.2.02330]

### Added
- Comprehensive documentation structure in `/docs`
- This changelog for tracking changes going forward

### Changed
- Reorganized documentation into topic-based folders
- Moved legacy docs to `/docs/deprecated`
- **Major refactor**: `ToolsPanel.tsx` split from 2,131 lines to 121 lines (94% reduction)
  - Extracted `tools/constants.ts` - shared configuration
  - Extracted `tools/types.ts` - shared type definitions
  - Extracted `tools/GlyphIcon.tsx` - icon component
  - Extracted `tools/Dashboard/` - word counts, goals, session stats
  - Extracted `tools/AIStudio/` - AI modes, style mixer, streaming
  - Extracted `tools/StoryBible/` - characters, plot notes, timeline
  - Extracted `tools/PlotWalker/` - beats, foreshadowing, knowledge matrix
- **Store refactor**: `bookStore.ts` split into domain-specific stores
  - Extracted `editorStore.ts` - editor ref, selection, inline prompt, diff/review
  - Extracted `storyBibleStore.ts` - characters CRUD, merging, highlighting
  - Extracted `plotStore.ts` - beats, foreshadowing, knowledge matrix
  - `bookStore.ts` now delegates to specialized stores (947 → 713 lines)
- **Dialog refactor**: `AppSettingsDialog.tsx` split from 683 lines to modular components
  - Extracted `settings/constants.ts` - model options, fonts, trim sizes
  - Extracted `settings/types.ts` - TypeScript interfaces
  - Extracted `settings/ApplicationSection.tsx` - identity, theme, save location
  - Extracted `settings/AIStudioSection.tsx` - Claude Code, API keys, local model
  - Extracted `settings/BookSection.tsx` - editor display, export settings

### Fixed
- All Qodana code quality issues resolved (0 remaining)
- Fixed defer-in-loop warnings in import.go
- Fixed unhandled error returns throughout codebase
- Updated Go dependencies to resolve security CVEs:
  - golang.org/x/crypto v0.53.0
  - golang.org/x/net v0.56.0
  - golang.org/x/sys v0.46.0
  - golang.org/x/text v0.38.0
- Updated Vite to resolve npm audit vulnerabilities

### Removed
- Unused constants: `AppVersionMajor`, `AppVersionMinor`, `AppVersionBuild`

---

## [0.12.02325] - 2026-07-08

### Added
- Beat Sheet system for Save the Cat! story structure tracking
- Foreshadowing Ledger for plant/reinforce/payoff tracking
- Knowledge Matrix for tracking who knows what secrets
- Character auto-detection with dialogue pattern recognition
- Attribute extraction (eye color, hair color, age)
- Character highlighting in editor

### Changed
- PlotWalker tools moved to Bible tab in sidebar
- Improved character mention counting

### Fixed
- Various UI polish and bug fixes

---

## [0.11.x] - Previous Releases

### Features Added Over Time
- AI rewriting with multiple providers (Claude Code, Anthropic, OpenAI, Gemini, Grok, Local)
- Style Mixer system (8 dimensions for controlling AI output)
- Prose guide feature for style matching
- EPUB import and export
- DOCX import and export
- PDF export (ebook and print-ready formats)
- Custom title bar with Windows integration
- Auto-save functionality
- Writing goals tracking (daily word count, manuscript progress)
- Session statistics
- AI content detection
- Spell checking
- Dark/light theme with auto-switching
- Chapter management (add, delete, reorder, types)
- Front matter and back matter support
- Story Bible (characters, plot notes, timeline)

---

## Version Format

`MAJOR.MINOR.BUILD`

- **MAJOR**: Significant milestones (1.x, 2.x, 3.x)
- **MINOR**: Feature chunks within major (x.1, x.2, x.3)
- **BUILD**: Always incrementing 5-digit build number (never resets)

Example progression: 0.8.02313 → 0.8.02314 (bug fix) → 0.9.02315 (new feature set)
