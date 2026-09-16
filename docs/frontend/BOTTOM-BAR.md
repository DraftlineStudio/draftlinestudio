# Bottom Bar

The bottom tool window (`components/StorySearchToolWindow.tsx`, mounted in
`App.tsx` above the status bar, resizable 170–560px) is where the manuscript
is questioned rather than written.

The window is called **Plot Inspections**. It holds three tabs, **Continuity**,
**Ask Draftline** and **Scratchpad**, plus an icon button for the **Evidence
index**. It opens on Continuity, the tab that finds things without being asked.
Scratchpad is the Planner's own notes, brought beside the manuscript so an
outline or a loose idea can be read while writing; it edits the same notes in
`planner.json` that the Planner edits.

The status bar carries the buttons that open it: **Plot Inspections**,
**Scratchpad**, and **Character Map**. Each lands on its own tab. The shell owns the 38px header
(tabs with an accent underline, a per-tab subtitle, continuity badges, and the
evidence/expand/close buttons); each panel lives under
`components/storysearch/` and takes
`{ book, onNavigate(section, index, evidenceQuery) }`. Triggers: the TitleBar
menu and Ctrl+Shift+F.

All CSS lives in `styles/global.css` (single-file convention) in the
bottom-tool section: `.story-search-*` for the shell, then `.continuity-*`,
`.ask-panel-*`, and `.evidence-*` for the three panels.

## Data flow

Read: `book.analysis.evidence.records` for the source quotations,
`BuildContinuityReport` for the Continuity tab, `SearchStory` for Ask.
Write path: `bookStore.setContinuityDecision` (reviewed/dismissed) and
`bookStore.updateEvidenceRecord` (evidence review).

## Continuity

The Continuity tab (`storysearch/ContinuityPanel.tsx`, 215 lines) is a
master-detail view over `BuildContinuityReport`. The report is rebuilt on
demand from the evidence index, the character index, and the source-backed
timeline; only the writer's reviewed/dismissed decisions are stored.

Five signal families are reported: a confirmed character with no established
given name, knowledge states that arrive out of order, conflicting physical
facts, clock references that run backwards, and thin chapters.

The left column (320px) lists every signal with a severity dot (error =
paired-source conflict, warning = single-source prompt, muted = observation),
an ellipsized title, and a `kind · chapter range` meta line; chips filter
All/Review/Observations/Decided plus free text. The right column shows the
selected signal's kind chip, location, description, and the evidence
comparison mapped from `sources[]` — "The claim" (error) vs "Established"
(success), a lone source as a full-width card — each navigating to its
passage. Actions: "Go to source", Mark reviewed, Dismiss (auto-advances).
Pure derivations live in `continuityModel.ts` (tested).

## Ask Draftline and the Evidence index

`AskPanel` (352 lines, `SearchStory`-backed detail trails) and
`EvidenceIndexPanel` (163 lines, the author-reviewable evidence archive backed
by `evidenceReview.ts`, tested) are the other two surfaces. The evidence
button toggles the index in and out of the Ask tab's slot and shows the record
count in its tooltip.
