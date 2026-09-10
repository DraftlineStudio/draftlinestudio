# Bottom Bar

The bottom tool window (`components/StorySearchToolWindow.tsx`, mounted in
`App.tsx` above the status bar, resizable 170–560px) is the UI surface for
the deterministic story-analysis engine.

Five tabs — **Story Map · Threads · Review · Continuity · Ask Draftline** —
plus the fingerprint icon-button for the Evidence index. The shell owns the
38px header (tabs with accent underline, per-tab subtitle, tab badges,
evidence/expand/close buttons); each tab is a self-contained panel under
`components/storysearch/` taking
`{ book, onNavigate(section, index, evidenceQuery) }`. Triggers: TitleBar
menu and Ctrl+Shift+F.

> **Current status: three of the five tabs are placeholder stubs.**
> `StoryMapPanel.tsx`, `ThreadsPanel.tsx`, and `ReviewDeskPanel.tsx` are
> ~25-line components that render a "rebuilding" notice while the
> manuscript-memory engine (v5) is rebuilt; their model modules
> (`storyStructureMapModel.ts`, `storyMapModel.ts`, `threadsModel.ts`,
> `reviewDeskModel.ts`) were deleted with the previous implementations, and
> the stubbed tabs' badges stay quiet. The live panels are **Continuity**,
> **Ask Draftline**, and the **Evidence index**. The retired panels'
> designs (semantic-zoom story map, obligation-row thread swimlanes,
> ordered review decision cards) are preserved in the roadmap material as
> reference for the developments-driven rebuild.

All CSS lives in `styles/global.css` (single-file convention) in the
bottom-tool section, class families: `.story-search-*` (shell),
`.continuity-*`, `.ask-panel-*`, `.evidence-*` (live), and `.smap-*`,
`.bbthreads-*`, `.bbreview-*` (currently unreferenced — retained pending a
decision on whether the rebuilt panels reuse them; see the technical-debt
ledger).

## Data flow

Read: `book.analysis.fingerprint` (typed on `AnalysisData` via the generated
`types.StoryFingerprint`), `book.analysis.evidence.records` for quotes,
`BuildContinuityReport` for the Continuity tab, `SearchStory` for Ask.
Write path: `bookStore.setContinuityDecision` (reviewed/dismissed).

## Continuity

The Continuity tab (`storysearch/ContinuityPanel.tsx`, 215 lines) is a
master-detail view over `BuildContinuityReport`. The left column (320px)
lists every signal with a severity dot (error = paired-source conflict,
warning = single-source prompt, muted = observation), an ellipsized title,
and a `kind · chapter range` meta line; chips filter
All/Review/Observations/Decided plus free text. The right column shows the
selected signal's kind chip, location, description, and the evidence
comparison mapped from `sources[]` — "The claim" (error) vs "Established"
(success), lone source as a full-width card — each navigating to its
passage. Actions: "Go to source", Mark reviewed, Dismiss (auto-advances).
Pure derivations live in `continuityModel.ts` (tested).

## Ask Draftline & Evidence

`AskPanel` (352 lines, SearchStory-backed trails) and `EvidenceIndexPanel`
(163 lines, author-reviewable evidence archive backed by
`evidenceReview.ts`, tested) are the other live surfaces.

## Stubbed tabs

`StoryMapPanel`, `ThreadsPanel`, and `ReviewDeskPanel` render an
intentional placeholder ("rebuilding") until the v5 engine's narrative
developments are ready to drive their visuals. Do not build against the
old models — they no longer exist.
