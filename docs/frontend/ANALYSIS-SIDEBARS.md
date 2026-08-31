# Analysis Sidebars

The analysis sidebar suite replaces the retired Story Bible / Plot Walker planning
panes (removed in 0.16.02449) and the old single Story Analysis pane. Each tool has
one glyph on the rail and one focused panel, following the design-center reference
in `reference-assets/Draftline sidebar layout directions/Story Analysis Sidebars.dc.html`
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
- `components/tools/Analysis/analysis.css` — shared visual vocabulary (`.an-*`):
  panel scaffold, micro-labels, stat tiles, cards, chips, badges, jump rows, bars,
  heat strips, tabs, freshness row, footnotes, empty states.
- Semantic status tokens `--status-success/warning/error/issue/complete` and
  `--status-badge-text` were added to `global.css` for both themes.
- ToolsPanel renders a `.glyph-badge` count on the Worth Reviewing rail glyph.

## Prose panel (`ProsePanel.tsx`, `prose.css`, prefix `prose-`)

Design section 3c "Prose expanded". Five `.an-block` sections, top to bottom:

1. **Rhythm** — a 30px flex strip of up to the last 60 sentences of the *current editor chapter* (`currentSection`/`currentIndex` + `getCurrentContent` + `htmlToText`, split on `[.!?…]`). Bar height is proportional to sentence word count (normalized to the longest, min 8%); bars over 25 words render in `var(--status-warning)`, the rest in `var(--app-accent)`. Footnote: "Each bar is a sentence; amber runs past 25 words." A muted note replaces the strip when the chapter has no prose. Works with no analysis run.
2. **Sentence lengths** — whole-manuscript histogram (buckets 1–5 / 6–10 / 11–15 / 16–20 / 21+) computed client-side from every chapter in front_matter/body/back_matter; fill width is bucket% relative to the largest bucket, right-aligned integer percentages; hint shows the total sentence count.
3. **Sentences** — stat rows (12px, 26px tall, hairline separators, per the mock's StatRow): Short (< 8 words) %, Long (> 25 words) % (both from the same client-side pass), Avg. paragraph (`overview.sentence_count / overview.paragraph_count`, 1 decimal, "—" if no paragraphs), Paragraphs (`toLocaleString`).
4. **Readability** — `overview.reading_ease.toFixed(1)`, `overview.mean_grade_level.toFixed(1)`.
5. **Word classes** — 10px stacked bar (verbs `--app-accent`, adjectives `--section-body`, adverbs `--section-back`, remainder `--bg-surface-alt` via flex) from `aggregateWordClasses(analysis.chapters)` (shared.ts), with a 6px-swatch legend ("Verbs 18.4%" style).

**Performance:** the store replaces the `book` object on every editor flush, so both heavy passes (rhythm parse and whole-manuscript parse) read a 2s-debounced book snapshot held in component state; a project change (`bookKey` from shared.ts) swaps the snapshot immediately so a newly opened book never shows the previous book's stats. Chapter switches refresh the rhythm strip instantly because section/index are not debounced.

**Empty states:** no book → standard `tool-empty-state`; book but no `analysis.story` → rhythm block stays live, followed by the standard `.an-empty` + `.an-run-btn` "Analyze now" block (disabled while running or when the analysis plugin is off).

No localStorage keys. Shared helpers used: `aggregateWordClasses`, `bookKey`.
