# Analysis Sidebars

The analysis sidebar suite replaces the retired Story Bible / Plot Walker planning
panes (removed in 0.16.02449) and the old single Story Analysis pane. Each tool has
one glyph on the rail and one focused panel, following the suite's design reference
(sections 3a–3g). Panels: Signals (hub overview), Prose, Pacing, Chapters,
Worth Reviewing, and AI Analysis — plus the redesigned Writing Dashboard (documented
here because the design treats the seven as one suite).

## Shared infrastructure

- `components/tools/Analysis/shared.ts` — logic shared by the suite:
  - `openToolsSection(section)` dispatches the `draftline:open-tools-section`
    CustomEvent; ToolsPanel listens and switches the active pane (deep links
    between panels and from the dashboard).
  - `goToChapter(globalIndex)` maps a combined front+body+back chapter index
    (the index space used by `ChapterAnalysis.chapter_index`) to the editor via
    `chapterLocation` and `setCurrentChapter`.
  - `formatAnalyzedStamp` / `formatDelta` — freshness and since-last-run copy.
  - `usePreviousOverview(book)` — persists the two most recent analysis overviews
    per book under localStorage key `draftline.analysis.overview-history.<bookKey>`,
    rolling forward when `content_hash` changes; powers the Signals deltas.
  - Dismissable observations: `observationKey`, `dismissObservation`,
    `restoreAllObservations`, `useDismissedSet`, `activeObservations`,
    `useReviewCount` (rail badge). Stored under
    `draftline.analysis.dismissed.<bookKey>`; keys are content-based so unchanged
    observations stay dismissed across runs.
  - `aggregateWordClasses(chapters)` — manuscript word-class percentages weighted
    by chapter word count.
- The ANALYSIS SUITE section of `styles/global.css` — shared visual vocabulary
  (`.an-*`): panel scaffold, micro-labels, stat tiles, cards, chips, badges, jump
  rows, bars, heat strips, tabs, freshness row, footnotes, empty states. (All
  styles live in global.css; each panel has its own bannered section there.)
- Semantic status tokens `--status-success/warning/error/issue/complete` and
  `--status-badge-text` were added to `global.css` for both themes.
- ToolsPanel renders a `.glyph-badge` count on the Worth Reviewing rail glyph.

## Prose panel (`ProsePanel.tsx`, prefix `prose-`)

Design section 3c "Prose expanded". Five `.an-block` sections, top to bottom:

1. **Rhythm** — a 30px flex strip of up to the last 60 sentences of the *current editor chapter* (`currentSection`/`currentIndex` + `getCurrentContent` + `htmlToText`, split on `[.!?…]`). Bar height is proportional to sentence word count (normalized to the longest, min 8%); bars over 25 words render in `var(--status-warning)`, the rest in `var(--app-accent)`. Footnote: "Each bar is a sentence; amber runs past 25 words." A muted note replaces the strip when the chapter has no prose. Works with no analysis run.
2. **Sentence lengths** — whole-manuscript histogram (buckets 1–5 / 6–10 / 11–15 / 16–20 / 21+) computed client-side from every chapter in front_matter/body/back_matter; fill width is bucket% relative to the largest bucket, right-aligned integer percentages; hint shows the total sentence count.
3. **Sentences** — stat rows (12px, 26px tall, hairline separators, per the mock's StatRow): Short (< 8 words) %, Long (> 25 words) % (both from the same client-side pass), Avg. paragraph (`overview.sentence_count / overview.paragraph_count`, 1 decimal, "—" if no paragraphs), Paragraphs (`toLocaleString`).
4. **Traditional readability formulas** — collapsed by default. Writers may deliberately expand the section to inspect Prose v3's reading-ease score and mean of Flesch–Kincaid, Automated Readability, Gunning Fog, SMOG, and Coleman–Liau. The UI states that these are sentence-and-word-shape calculations, not measurements of subject knowledge, terminology, narrative complexity, audience, or comprehension.
5. **Word classes** — 10px stacked bar (verbs `--app-accent`, adjectives `--section-body`, adverbs `--section-back`, remainder `--bg-surface-alt` via flex) from `aggregateWordClasses(analysis.chapters)` (shared.ts), with a 6px-swatch legend ("Verbs 18.4%" style).

**Performance:** the store replaces the `book` object on every editor flush, so both heavy passes (rhythm parse and whole-manuscript parse) read a 2s-debounced book snapshot held in component state; a project change (`bookKey` from shared.ts) swaps the snapshot immediately so a newly opened book never shows the previous book's stats. Chapter switches refresh the rhythm strip instantly because section/index are not debounced.

**Empty states:** no book → standard `tool-empty-state`; book but no `analysis.story` → rhythm block stays live, followed by the standard `.an-empty` + `.an-run-btn` "Analyze now" block (disabled while running or when the analysis plugin is off).

No localStorage keys. Shared helpers used: `aggregateWordClasses`, `bookKey`.

## Pacing panel (`PacingPanel.tsx`, prefix `pace-`)

An explainable prose-tempo view. Tempo describes how quickly the writing reads
from sentence shape and dialogue; the panel explicitly avoids presenting it as
a measurement of plot urgency or story quality.

**What it shows, top to bottom:**

1. **Prose tempo** — the word-weighted manuscript score, its measured/balanced/brisk band, and a concise definition of what the score does and does not mean. Bands use the same 42/68 boundaries as the backend.
2. **Flow through the book** — one clickable cell per chapter, colored categorically rather than by ambiguous brightness. A legend gives the number of measured, balanced, and brisk chapters. The accompanying read reports the number of neighboring changes of at least 10 points and names the largest transition.
3. **Largest transitions** — up to three chapter-to-chapter rises or drops, sorted by magnitude and linked to the destination chapter. Sub-20-word dividers are excluded.
4. **Metric picker** — Tempo, Dialogue, and Length, each with a plain-language explanation. Optional mechanical readability formulas remain in Prose rather than mixing them into pacing.
5. **Chapter rows** — exact value, an absolute-scale bar, and supporting context. Tempo includes average sentence length and dialogue share; Dialogue includes sentence and scene-break counts; Length compares word count against the median analyzed chapter. Every row opens its chapter.

Median length is intentionally resistant to unusually long chapters. Pure helper
tests lock the tempo boundaries, median calculation, and neighboring-transition
threshold. The panel performs no HTML parsing, uses no localStorage, and reads
only the stored `StoryAnalysisData` metrics.

**Empty states:** no book → standard tool empty state; no analysis → local-analysis
explanation and an Analyze now button, disabled while running or when Story
Analysis is switched off.

## Chapters panel (`ChaptersPanel.tsx`, prefix `chp-`)

Design section 3e. A skimmable per-chapter recap of the manuscript: for each analyzed chapter, a clickable card with its number and title, a meta line (`{word_count} words · {scenes} scenes · {breaks} breaks`, pluralized, scenes = scene_break_count + 1), the extractive summary in Merriweather italic (`.an-serif`), and up to 6 keyword chips (`.an-chip`).

**Data**: `useBookStore(s => s.book).analysis?.story.chapters` (`ChapterAnalysis[]`). Chapters with neither keywords nor an extractive summary (part dividers) are filtered out *after* numbering, so card numbers always match a chapter's 1-based position among analyzable chapters. Keywords come from `ChapterAnalysis.keywords[].term`; summaries from `extractive_summary`. All derivation is memoized on the analysis object; no live text parsing occurs in this panel.

**Navigation**: card click calls `goToChapter(chapter.chapter_index)` from `Analysis/shared.ts` (combined front/body/back index space).

**Behaviors**: first 8 cards shown, then an `.an-text-action` toggling "Show all {N} chapters" / "Show fewer" (component state, not persisted). No localStorage keys.

**Empty states**: no book → standard `tool-empty-state`; no analysis → `.an-empty` with an "Analyze now" `.an-run-btn` wired to `useAnalysisStore().run` (disabled while running or when `settings.analysis_enabled` is off, showing "Enable in Plugins"); analysis present but no chapter has keywords or a summary → a dedicated `.an-empty` note.

**CSS**: only `chp-*` rules; overrides `.an-card-list` gap to 10px inside this panel to match the mock's feed spacing; cards are `<button>` elements normalized on top of `.an-card`.

## Worth Reviewing panel (`ReviewPanel.tsx`, design 3f)

Dismissable observation cards sourced from `book.analysis.story.observations`
(`StoryAnalysisObservation[]`), filtered through `activeObservations(analysis, useDismissedSet(book))`
from `Analysis/shared.ts`.

**Layout** (top to bottom):
- Filter block (bottom-bordered, 10px 12px): sentence-case `.an-tab` buttons — "All {n}"
  plus one tab per observation kind present among *active* (non-dismissed) observations,
  each labeled Capitalized with its count (e.g. "Structure 6", "Pacing 2"). Below the tabs,
  the `.an-footnote` "Measured differences, not errors or prescriptions." The block is hidden
  when no active observations remain.
- `.an-card-list` of cards, each with: a 9px bold uppercase kind micro-label
  (`--status-issue` for `structure`, `--section-back` for `pacing`, `--text-section` otherwise),
  a 12px semibold title, an 11px secondary detail line, and an action row containing
  "Go to chapter ›" (`.an-link`, only when `chapter_index` is defined; calls `goToChapter`)
  and a muted "Dismiss" button.

**Filter behavior**: the selected kind falls back to "All" automatically when its last card
is dismissed (derived `effectiveFilter`, no state write during render).

**Dismissals**: keyed by `observationKey(o)` (kind + title + detail), stored in
localStorage `draftline.analysis.dismissed.{bookKey}` via shared.ts's `dismissObservation` /
`restoreAllObservations` / `useDismissedSet` external store — so the rail badge
(`useReviewCount`) updates live in other panels.

**Empty states**:
- No book: standard `tool-empty-state`.
- No `analysis.story`: `.an-empty` explainer + `.an-run-btn` (Enable in Plugins / Analyzing… /
  Analyze now, matching the other suite panels), wired to `useAnalysisStore().run`.
- Observations exist but all dismissed: `.an-empty` "All observations dismissed." with an
  `.an-run-btn` "Restore dismissed".
- Analysis has zero observations: `.an-empty` "Nothing worth flagging — the manuscript reads evenly."

**CSS**: panel-specific rules (`rv-` prefix) and the shared `.an-*` vocabulary both
live in `styles/global.css`. No hardcoded colors — all values are theme custom
properties.

## Signals panel (`SignalsPanel.tsx`, prefix `sg-`)

Hub overview of the local manuscript analysis (design 3b).

**Layout (top to bottom)**
1. **Freshness row** (`.an-freshness`) — status dot (success when `useAnalysisStore().state === 'current'`, error class on `'error'`, stale class otherwise), text from `formatAnalyzedStamp(analysis.last_analyzed)` ("Analyzing…" while running, "Analysis out of date" when stale), and a "Run again" `.an-run-btn` wired to `useAnalysisStore().run`, disabled while running or when `settings.analysis_enabled` is off.
2. **Six stat tiles** (`.an-tiles`) from `book.analysis.story.overview`: Chapters, Avg. chapter words, Avg. sentence words, Dialogue %, total manuscript words, and Tempo /100. The first five carry since-last-run deltas via `usePreviousOverview(book)` + `formatDelta`; Tempo keeps its measured/balanced/brisk descriptor.
3. **"In one line"** (`.an-label` + `.an-serif`) — a strictly descriptive sentence containing only the measured sentence-length clause (<10 short, <18 varied, else long), dialogue clause (<15% sparse, <35% moderate, else dialogue-heavy), and prose-tempo band. It makes no comparison to an age group, school grade, genre, audience, or prose quality.
4. **Jump list** — four `.an-jump-row` buttons calling `openToolsSection('prose' | 'pacing' | 'chapters' | 'review')`; the review row shows the `useReviewCount()` count in an `.an-badge` before the chevron, hidden when 0.

**Data sources** — `useBookStore(s => s.book)` (`book.analysis.story`), `useAnalysisStore` (run state + `run()`), `useAppStore` (`settings.analysis_enabled`), and shared helpers from `Analysis/shared.ts`. `usePreviousOverview` persists per-book overview history under `draftline.analysis.overview-history.<bookKey>` in localStorage (managed by shared.ts, not this panel).

**Empty states** — no book: `tool-empty-state` "Open a project to see analysis."; book without analysis: `.an-empty` explaining local analysis after 15 s of inactivity with an "Analyze now" run button (same disabled rules).

**Behaviors** — no whole-book HTML parsing or timers; all values come precomputed from `analysis.overview`, so the panel is cheap to re-render. No engine names are shown.

## AI Analysis panel (`AIDetectPanel.tsx`, design 3g)

Heuristic AI-detection for the whole book. Unlike the other analysis panels it does **not** read `book.analysis.story` and needs no analysis run — everything comes from the local frontend heuristics in `services/aiDetection.ts` (`analyzeText`, `analyzeAntiPatterns`, `getScoreColor`, `getAntiPatternColor`) applied to chapter HTML.

**Blocks, top to bottom**
1. **Gauge** — semicircular SVG gauge (`getScoreColor` arc) for the current editor chapter, with a 26px `{score}%` readout, an "AI signal" micro-label, a verdict line (`< 40` Reads human / `40–70` Mixed signals / `> 70` Likely AI in the matching `--status-*` color), a "{title} · current chapter" sub-line, and a three-dot threshold legend. Analysis is instant on chapter/book switch and debounced 2s while typing (the switch detector keys on book identity + section + index, so opening another book at the same location re-analyzes immediately). Chapters under 100 plain-text chars show "Not enough text in this chapter to analyze." instead of the gauge.
2. **By chapter** — heat strip from `useBookAIScan` in book order; cell color by the same thresholds, opacity `0.35 + score/100 × 0.65`, tooltip "{title} — {score}%". Footnote counts chapters ≥ 40 ("Two chapters cross the mixed threshold." with word-number pluralization); shows "Scanning chapters…" while the scan runs.
3. **Highest signal** — top 5 scores as `.an-chapter-row` buttons (1-based book-order position among scanned chapters, title, 64px threshold-colored bar, `{score}%`); click navigates via `goToChapter(globalIndex)` from `Analysis/shared.ts`.
4. **Flagged passages** — for up to the two highest-scoring chapters at ≥ 40 (only once the scan is done), the single best paragraph from `scanPassages` as an `.an-card`: "Chapter {n} · {title}", score `.an-badge` (`error` > 70, else `warning`), curly-quoted `.an-serif` excerpt truncated at ~160 chars on a word boundary, "Go to chapter ›" link. Empty: "No passages currently read as AI-assisted."
5. **AI anti-patterns** — `analyzeAntiPatterns` on the current chapter (same debounce): displayName left, "{count}× (limit {n}/ch)" right, 2px left border in `getAntiPatternColor(severity)` for over-limit rows. Empty: "No overused AI patterns detected."
6. Closing footnote stating the score is probabilistic, not a verdict.

**`services/aiScan.ts`**
- `useBookAIScan(book)` → `{ scores: ChapterAIScore[], done }`. Scans all chapters (front_matter + body + back_matter, combined global index) in batches of 3 per `setTimeout(0)` tick; eligibility (≥ 100 plain-text chars) is checked *inside* the tick so no whole-book HTML parse ever happens synchronously. Partial results stream out per tick. Caches: a `WeakMap` per book object (reopening the panel after a completed scan is instant; a remount mid-scan restarts immediately), plus a per-identity map (`file_path` + `metadata.created`) that keeps the last complete scan on screen while a 2s-debounced re-scan runs after edits. First scan of a book identity starts immediately; re-scans (new book object, same identity) are debounced 2s.
- `scanPassages(html, analyzeThreshold = 150)` → `[{ excerpt, score }]` sorted descending: paragraphs from `<p>` elements (fallback: blank-line split), scored with `analyzeText` when ≥ 150 chars.

No localStorage keys. No new dependencies. Panel-specific styles (`.ai-*` prefix) and the shared `.an-*` vocabulary live in `styles/global.css`.

## Writing Dashboard (redesigned)

**Component:** `wails/frontend/src/components/tools/Dashboard/index.tsx` (+ `history.ts`). Root classes `dashboard-content dash-panel`; reuses the existing `dashboard-*` / `big-number` / `progress-bar*` / `chapter-bar-*` / `goal-input*` classes from global.css, with panel-specific styles prefixed `dash-` in the WRITING DASHBOARD section of the same file.

**What it shows (top to bottom):**
- **Manuscript** — total book words (memoized `countBookWords` keyed on the `book` reference), hint button "target {n}k" when a target is set, progress bar with "{pct}%" left and "{remaining} to go" right (both hint and right label open the target editor; the set/edit input flows are unchanged from the old dashboard).
- **Today** — hint "goal {n}", big number "{today} / of {goal} words", small progress bar, plus a goal-streak row: seven 8px squares (last 7 days, oldest to newest; `--status-complete` for days with words, `--app-accent` for today, bordered `--bg-surface-alt` otherwise) captioned "{count} of last 7 days". `words_today` lives in `book.writing_goals` and is treated as reset on a new day via `last_writing_date`; on a new day the session word delta stands in for it.
- **This session** — hint "started {h:MM AM/PM}"; three stats: words written, time writing, words / min (`sessionWords / max(1, sessionMinutes)`, 1 decimal). The session baseline (word count + start time) is captured when a book first appears in the panel and re-captured per `bookKey` when a different book is opened; a 30s interval tick keeps the time-based stats fresh.
- **Chapter stats** — unchanged (Chapters / Avg. length / Shortest / Longest, body chapters only).
- **Chapter breakdown** — hint "all sections"; aggregate muted "Front matter" row, then "{n} · {title}" per body chapter, then aggregate "Back matter" row; bars normalized to the longest body chapter; first 10 rows then a "Show all {N}" / "Show fewer" toggle.
- **AI detection jump row** — borderless bottom row "AI detection — current chapter" with the current chapter's score ("{score}%", colored by `getScoreColor`), computed by `analyzeText` 2s after the chapter content stops changing and omitted when the chapter has under 100 characters of plain text (`htmlToText` length, not HTML length). Clicking dispatches `openToolsSection('aidetect')` from `Analysis/shared.ts`.

**localStorage:** `draftline.writing-history.{bookKey(book)}` — a `{ "YYYY-MM-DD": words }` map maintained by `history.ts` (`recordTodayWords` upserts today's max and prunes entries older than 60 days; `getLastNDays` returns the last N days oldest-first with missing days as 0). Dates are LOCAL calendar days, not UTC, so streak squares flip at the writer's midnight. Entries are never lowered within a day.

**Empty state:** no book → `tool-empty-state` "Open or create a project to see your writing dashboard." (dashboard-specific wording kept from the previous component; the generic "…to see analysis." copy does not apply here).
