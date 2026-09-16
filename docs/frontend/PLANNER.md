# Planner

The Planner is Draftline's story-line timeline: lanes for the main plot,
subplots, and character arcs, crossed with the manuscript's chapters, with plot
cards where a line meets a chapter. It is opened from the **Planner** tab at
the top of the left panel and closed from the **Manuscript** tab.

It is a manual outliner. Every card on it was typed by the writer or proposed
from an outline the writer pasted in and accepted. The Planner reads the codex
for the people a card can name and the manuscript for the chapters and scenes
a card can sit in; it does not read the prose and it never writes a card by
itself.

## Views

The glyph rail on the right switches between four views; clicking the active
glyph hides the tools panel.

- **Timeline** — lanes down the left, chapters across, cards in the cells.
  Drag a card to another chapter or lane; double-click a cell to add a card
  there. A card on several lines is drawn once, on its first line, with a
  bracket to a small diamond on each crossing lane. Empty chapters are dimmed.
  The last column, *Later*, holds cards not yet pinned to a chapter. The Planner
  never creates manuscript chapters; chapters added in the manuscript appear
  here automatically. Beat marks (Three-Act or Save the Cat) sit at manuscript
  percentages across the chapter axis when a template is chosen.
- **Board** — the same cards as index cards, grouped by chapter or by story
  line. Drag onto a column to move; drop onto a card to order before it.
- **Scratchpad** — free-text notes. The *Dead ideas* note is automatic: every
  card deleted from the timeline or board is written there as an outline entry
  so *Propose Cards* can bring it back. A note can be excluded from Propose
  Cards.
- **Synopsis** — one paragraph per chapter, made of that chapter's card
  synopses in timeline order. Paragraphs are editable; *Rebuild from Cards*
  discards edits. Chapters without cards stay blank rather than being
  invented. Export copies Markdown or saves it as a Scratchpad note.

## Cards

A card has a title, synopsis, one or more lines (the first is where it sits),
who (confirmed codex characters, stored with their names; the stored name is
what the card goes by, so it still names its people after re-indexing
reassigns codex IDs, and choosing a person again re-binds the card to their
current ID), what changes, stakes, a chapter position, and an optional scene
link.

A card has two statuses and nothing else decides them:

| Status | What it means |
| --- | --- |
| **planned** | No scene link yet. |
| **drafted** | Linked to a scene. |

*What changes* and *stakes* are the promise the card makes. They are the
writer's own note about the card, not something Draftline checks against the
manuscript.

Scene numbers follow the manuscript's scene breaks: a paragraph that is only
`***`, `* * *`, `⁂`, `###`, `# # #`, `---`, `- - -`, `~ ~ ~` or `. . .`, or a
horizontal rule, ends a scene. Every marker starts a scene, so a marker at the
very top of a chapter, or two markers in a row, each open one (the first of
them empty).

## Import Outline

*Import Outline* takes freeform pasted text or a Markdown or text file and
proposes a compact set of substantial cards; formatting is optional and
nothing is committed until the proposals are accepted:

- When text has a heading hierarchy, the larger headings become proposal
  groups and the next heading level becomes cards. Deeper headings, bullets,
  and paragraphs become supporting synopsis text instead of tiny cards.
- Flat lists and loose prose are grouped into adjacent substantial movements;
  a short list can still produce one card per item.
- "Chapter N", "Part N", "Act N", and beat names describe the imported
  outline only. They never create, rename, or assume manuscript chapters.
- Names the codex knows pick the card's *who*; the first of them with a
  character lane picks the lane, otherwise the main line.
- Each detected proposal group starts as a proposed story line named after the
  group. It can instead be moved to the main plot or an existing story line
  before acceptance. The proposal screen can also create and name a different
  plot track for a group; it remains provisional until the import is accepted.
- Explicit Characters, Cast, or Dramatis Personae sections produce reviewable
  codex candidates instead of plot cards. Accepted candidates can create
  character tracks immediately, before any manuscript chapters are drafted.

Import Outline reads nothing from the story analysis. It parses the text you
give it and asks the codex only for accepted character names so it can fill in
a card's *who*. A book that has never been analyzed imports an outline exactly
the same way, minus the name matching.

The proposal view lets each card be retitled, re-laned, or dropped, lets a
group create a named plot track in place, and lets detected characters be
renamed, dropped, or added without a character track. Card synopses are kept
brief; the complete imported outline remains in the Scratchpad note rather
than being packed into every card. Accepting adds planned cards to *Later*.
The writer can attach cards to existing or future manuscript chapters afterward.

Running *Propose Cards* again on the same Scratchpad note is incremental.
Movements already accepted from that note are omitted, even if their supporting
details changed or the resulting card was retitled in the Planner; newly added
movements are the only card proposals shown. Cards imported by older Draftline
versions are matched by title the first time this incremental check runs.

The same notes are reachable from the writing window: the Scratchpad tab in
the bottom Plot Inspections dock reads and edits `planner.notes` directly, so a
note written while drafting is the note Propose Cards reads.

## Storage

Planner data is stored in the `.draftline` archive as `planner.json`, written
once the Planner has been used for a book: lanes, cards, notes, per-chapter
synopsis edits, the beat template, hidden lanes, and the compact view setting.
Cards reference chapters by their stable chapter IDs, so reordering or renaming
chapters does not move cards. Nothing derived from the manuscript is stored
there. See `docs/data-model/DATA-MODEL.md`.

## Code

- `components/planner/plannerModel.ts` — pure model: outline rules, layout,
  card status, lane rows and crossings, Dead ideas entries. Unit-tested.
- `components/planner/plannerSynopsis.ts` — the chapter synopsis a book's own
  cards make, and its Markdown export with one trace line per card.
  Unit-tested.
- `store/plannerStore.ts` — view state and every mutation of the persisted
  data, written through the book store so changes ride the normal save and
  autosave path.
- `components/planner/PlannerView.tsx`, `PlannerSidebar.tsx`,
  `PlannerPanel.tsx`, `PlannerDialogs.tsx` — the views, the left panel body,
  the tools panel with card inspector and glyph rail, and the dialogs.
- Backend: `internal/book/planner.go` (archive read/write and version check),
  `internal/types/book.go` (the Planner records). The Planner has no Wails
  binding of its own; it rides the book save path.
