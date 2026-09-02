# Bottom Bar (v7)

The bottom tool window (`components/StorySearchToolWindow.tsx`, mounted in
`App.tsx` above the status bar, resizable 170–560px) is the UI surface for the
Story Fingerprint v2 engine. Design reference:
`reference-assets/Bottom bar timeline redesign/Draftline Bottom Bar v7.dc.html`.

Five tabs — **Story Map · Threads · Review · Continuity · Ask Draftline** —
plus the fingerprint icon-button for the Evidence index. The shell owns the
38px header (tabs with accent underline, per-tab subtitle, Review badge with
undecided-detection count, evidence/expand/close buttons); each tab is a
self-contained panel under `components/storysearch/` taking
`{ book, onNavigate(section, index, evidenceQuery) }`. Triggers: status-bar
"Ask Draftline" button, TitleBar menu, Ctrl+Shift+F. The old Story Graph tab
was retired — the Story Map supersedes it (built from the fingerprint rather
than the timeline projection); `storyGraph.ts` and its panel were deleted.

All CSS lives in `styles/global.css` (single-file convention) in the
bottom-tool section, class families: `.story-search-*` (shell), `.smap-*`,
`.bbthreads-*`, `.bbreview-*`, `.continuity-*`, `.ask-panel-*`,
`.evidence-*`.

## Data flow

Read: `book.analysis.fingerprint` (typed on `AnalysisData` via the generated
`types.StoryFingerprint`), `book.analysis.evidence.records` for quotes,
`BuildContinuityReport` for the Continuity tab, `SearchStory` for Ask.
Write paths: `UpdateStoryAuthorModel(book, model)` (corrections, author
contexts, canon — rebuilds the fingerprint only, result flows through
`updateBook`) and `bookStore.setContinuityDecision` (reviewed/dismissed).

## Story Map

The Story Map tab (`storysearch/StoryMapPanel.tsx`) renders the story
fingerprint as a horizontally scrollable story-time timeline plus a 298px
event-detail pane. All derivation is pure and unit-tested in
`storysearch/storyMapModel.ts`: `buildStoryMapLayout(fingerprint, opts)`
groups events into era bands (one per `StoryContext`, with the primary
context split into contiguous day segments where events carry
`day_offset`/`earliest_day`; unanchored flashback frames sort left, unplaced
remainders right), assigns up to three lanes (primary on top, nested contexts
below), places nodes inside their era ordered by story day then narrative
order, and builds the manuscript-order cubic path. It also emits day ticks
(weekday extracted from `story_time.label`), checkpoint flags from
`author_model.checkpoints` (fulfilled/partial, with satisfied/total ratio),
dashed dangle curves for up to two open threads, and a loop-back curve when
an undecided `near_duplicate_chapter` diagnostic spans the first and last
chapters. Nodes cap at the 120 highest-importance events. `describeEvent`
joins evidence, states, and threads into the detail-pane model;
`appendStoryDayCorrection` pins a weakly-placed event's day as a `story_day`
correction (status `active`) submitted via `UpdateStoryAuthorModel`.

## Threads

The Threads tab (`storysearch/ThreadsPanel.tsx`) renders
`fingerprint.threads` as horizontal obligation rows in a fixed-width
(1260px, horizontally scrollable) SVG, 30px per row, capped at the 40 most
relevant threads with a "+N more" footer. Geometry comes from
`threadsModel.ts`: x positions normalize the thread events' coordinates
across the manuscript, using story days when ≥70% of events are anchored and
manuscript order otherwise (the caption states which). Row states map to
labels/colors — OPEN·seeded, ESCALATING·n/m, DORMANT n CH, CONVERGING,
RESOLVED (ring at the resolution event), INTENTIONALLY DEFERRED — plus a
derived REOPENED state when a resolved thread has events after its
resolution. Dormancy gaps (≥3 chapters) render as sparse dash; converging
parent threads get a CONVERGE connector. Row click navigates to the opening
scene. Line colors rotate through six tokens keyed by a stable thread-id
hash. No writes.

## Review

The Review tab (`ReviewDeskPanel` + pure `reviewDeskModel`) turns
`fingerprint.diagnostics` into ordered decision cards: undecided first, then
by kind urgency (near-duplicate → canon → presence → direction → attribute →
identity → orphaned correction → checkpoint → forgotten intro → dormant
thread). The top undecided card takes a wide lead column; decided cards sink
to the bottom at 45% opacity. Decisions are written through
`setContinuityDecision` under both the diagnostic id and the derived
continuity-signal id (the model reimplements the backend's sha256
`stableID`/signal derivations, test-verified against Go-generated vectors),
so a Review decision also dims the matching Continuity question.
Author-model writes: "It's a reset ↺" appends an author
`StoryContext{kind:simulation}`; "Discard" removes the matched orphaned
correction — both via `UpdateStoryAuthorModel`. "Reattach…" is disabled
pending a backend affordance. Dormant-thread cards deep-link to the Threads
tab via the shell-wired `onOpenThreads`.

## Continuity

The Continuity tab (`storysearch/ContinuityPanel.tsx`) is a master-detail
view over `BuildContinuityReport`. The left column (320px) lists every
signal with a severity dot (error = paired-source conflict, warning =
single-source prompt, muted = observation), an ellipsized title, and a
`kind · chapter range` meta line; chips filter All/Review/Observations/
Decided plus free text. The right column shows the selected signal's kind
chip, location, description, and the evidence comparison mapped from
`sources[]` — "The claim" (error) vs "Established" (success), lone source as
a full-width card — each navigating to its passage. Actions: "Go to source",
Mark reviewed, Dismiss (auto-advances). Pure derivations live in
`continuityModel.ts`.

## Ask Draftline & Evidence

`AskPanel` (SearchStory-backed trails) and `EvidenceIndexPanel` carried over
unchanged from the previous bottom tool.

## Known gaps (backend)

- `StoryThread.state = "intentionally_deferred"` and
  `FingerprintDiagnostic.status` have no engine writer yet — "Intentional"
  decisions live as continuity dismissals until a first-class field lands.
- Reattaching an orphaned correction to a new passage needs a picker +
  backend affordance; the button is disabled with an explanatory title.
