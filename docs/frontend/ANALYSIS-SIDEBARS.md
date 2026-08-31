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

## Pacing panel (`PacingPanel.tsx`, `pacing.css`) — design ref 3d

Tempo-by-chapter view of the manuscript.

**What it shows, top to bottom:**
1. **The whole book** — a heat strip with one cell per analyzed chapter, background `var(--app-accent)` at opacity `0.35 + (tempo_score/100)*0.65` (brighter = faster). Chapters with `tempo_score` 0 or under 20 words render as `var(--text-faint)`. Each cell has a `"{n} · {title} — {tempo_score}"` tooltip. Below it, an insight footnote: the panel finds the contiguous window of 3+ analyzable chapters with the lowest mean tempo (prefix-sum scan) and, if that window sits at least 12 points below the book mean, appends "The pace sags around chapters {start}–{end}." to "Brighter is faster."
2. **Metric tabs** — Tempo / Dialogue / Ease / Length switch which `ChapterAnalysis` field the list plots (`tempo_score`, `dialogue_percent`, `reading_ease`, `word_count`). Local `useState`, not persisted.
3. **Column header** — 9px uppercase micro-labels: blank 18px / Chapter / metric name 76px / Brk 22px.
4. **Chapter rows** (`.an-chapter-row` buttons) — 1-based number, title (fallback "Chapter {n}"), 76px bar normalized to the metric max (min 2% width), and `scene_break_count` (em dash for sub-20-word chapters). Bar turns `var(--status-warning)` when the chapter's metric is more than 1.5σ from the mean of analyzable chapters (sub-20-word chapters excluded from mean/σ and never flagged; their bars render faint). Click navigates via `goToChapter(chapter.chapter_index)` from `Analysis/shared.ts`.
5. **Footer** — "Brk = scene breaks · {total} across the manuscript · amber bars are outliers".

**Data:** `useBookStore(s => s.book).analysis?.story.chapters` (`ChapterAnalysis[]`). Insight and per-metric stats (max/mean/σ) are memoized on `chapters`/active metric. No whole-book HTML parsing, no debounced live analysis, no localStorage keys.

**Empty states:** no book → `tool-empty-state` "Open a project to see analysis."; book without analysis (or zero analyzed chapters) → `.an-empty` explainer with an "Analyze now" `.an-run-btn` wired to `useAnalysisStore().run`, disabled while running or when `settings.analysis_enabled` is off.

## Chapters panel (`ChaptersPanel.tsx`, `chapters-panel.css`)

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

**CSS**: panel-specific rules in `review.css` (`rv-` prefix); shared vocabulary from
`analysis.css`. No hardcoded colors — all values are theme custom properties.
