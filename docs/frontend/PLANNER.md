# Planner

The Planner is Draftline's story-line timeline: lanes for the main plot,
subplots, and character arcs, crossed with the manuscript's chapters, with plot
cards where a line meets a chapter. It is opened from the **Planner** tab at
the top of the left panel and closed from the **Manuscript** tab.

It is dual-facet by design. A card can be typed by hand, proposed from an
outline, or adopted from a development the narrative engine found in the text;
however it arrived, it is the same kind of card with the same fields, so
anything the engine can suggest a writer can also plan.

## Views

The glyph rail on the right switches between four views; clicking the active
glyph hides the tools panel.

- **Timeline** — lanes down the left, chapters across, cards in the cells.
  Drag a card to another chapter or lane; double-click a cell to add a card
  there. A card on several lines is drawn once, on its first line, with a
  bracket to a small diamond on each crossing lane. Empty chapters are dimmed.
  The last column, *Later*, holds cards not yet pinned to a chapter, and the
  *Add Chapter* column adds an empty chapter to the manuscript. Beat marks
  (Three-Act or Save the Cat) sit at manuscript percentages across the chapter
  axis when a template is chosen.
- **Board** — the same cards as index cards, grouped by chapter or by story
  line. Drag onto a column to move; drop onto a card to order before it.
- **Scratchpad** — free-text notes. The *Dead ideas* note is automatic: every
  card deleted from the timeline or board, and every dismissed unplanned
  development, is written there as an outline entry so *Propose Cards* can
  bring it back. A note can be excluded from analysis.
- **Synopsis** — one paragraph per chapter, generated from that chapter's card
  synopses in timeline order. Paragraphs are editable; *Rebuild from Cards*
  discards edits. Chapters without cards stay blank rather than being
  invented. Export copies Markdown or saves it as a Scratchpad note.

## Cards

A card has a title, synopsis, one or more lines (the first is where it sits),
who (confirmed codex characters), what changes, stakes, a chapter position,
and an optional scene link. Linking a card to a scene marks it **drafted**;
otherwise it is **planned**. *What changes* and *stakes* are kept as separate
fields so that a future reconciliation against extracted developments needs no
data migration.

## Import Outline

*Import Outline* takes pasted text or a Markdown or text file and proposes
cards from deterministic rules; nothing is committed until the proposals are
accepted:

- `#` headings and "Chapter N" / "Part N" / "Act N" lines set the chapter
  position for everything under them and may name the chapter.
- List items and paragraphs become cards: first sentence as the title, the
  rest as the synopsis. Nested list items join their parent's synopsis.
- A recognised beat name (from the Three-Act and Save the Cat templates) lands
  on the main line at its manuscript percentage.
- Names the codex knows pick the card's *who*; the first of them with a
  character lane picks the lane, otherwise the main line.
- Unstructured text with many paragraphs, a short story for instance, is
  clustered into scene-sized proposals across the drafted chapters.

The proposal view lets each card be retitled, re-laned, re-chaptered, or
dropped. Accepting adds planned cards, creates any chapters the outline reaches
past the end of the manuscript (empty, titled from the outline), and keeps the
pasted text as a Scratchpad note.

## Plot Walker

With **Reconcile cards with the text** on, the narrative engine reads the
manuscript and its proposals appear as dashed **unplanned** cards on the
timeline, positioned by chapter and scene with the sentence that nominated them
as evidence. *Adopt as card* turns one into an ordinary card that keeps its
evidence; *Dismiss* sends it to Dead ideas. Detection runs when the Planner is
opened with the toggle on and from the toolbar's *Refresh*, never per
keystroke. Matching existing cards to developments (kept and drifted statuses)
is the next Plot Walker milestone; the statuses are reserved in the model.

## Storage

Planner data is stored in the `.draftline` archive as `planner.json`, written
once the Planner has been used for a book: lanes, cards, notes, per-chapter
synopsis edits, the beat template, hidden lanes, dismissed proposals, and the
Plot Walker and density settings. Cards reference chapters by their stable
chapter IDs, so reordering or renaming chapters does not move cards. See
`docs/data-model/DATA-MODEL.md`.

## Code

- `components/planner/plannerModel.ts` — pure model: outline rules, layout,
  synopsis generation, Dead ideas entries. Unit-tested.
- `store/plannerStore.ts` — view state and every mutation of the persisted
  data, written through the book store so changes ride the normal save and
  autosave path.
- `components/planner/PlannerView.tsx`, `PlannerSidebar.tsx`,
  `PlannerPanel.tsx`, `PlannerDialogs.tsx` — the views, the left panel body,
  the tools panel with card inspector and glyph rail, and the dialogs.
- Backend: `wails/planner.go` (detection binding over the narrative engine),
  `internal/book` (archive read/write), `internal/types` (data types).
