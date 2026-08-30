# Characters View — Design Gaps & Deferred Work

The Characters workspace (0.16.02394–02396) implements the "Hybrid: swimlanes + ego view" design from `reference-assets/Draftline character relationship redesign/`. Everything the current data model supports is in. This documents what the design implies but the app cannot honestly deliver yet, for separate work.

## 1. Sentence-quality mention excerpts (backend)

**Design:** the Mentions section shows full-sentence manuscript excerpts in Merriweather italic.
**Reality:** `MentionRecord` carries only the mention text itself (`text`), `chapter`, `char_offset`, and a `sentence_id` — but the sentence store the ID points into is never persisted to the frontend.
**Current behavior:** the pane derives context client-side — it re-strips the chapter HTML, locates the mention text near its recorded offset, and slices ~90 chars to word boundaries (`mentionExcerpt` in `CharactersView.tsx`). Works well in practice but is a heuristic: the offset is treated as a hint, and an unlocatable mention falls back to the bare mention text.
**Proper fix:** have `IndexBook` return the sentence text per mention (the Go pipeline already segments sentences — `sentence_id` proves it), and render that verbatim. Backend change + type change in `entities.go` / `draftline.ts`.

## 2. Precise jump-to-mention (frontend, non-trivial)

**Design:** "Jump to mention →" lands on the exact spot in the manuscript.
**Reality:** no mechanism maps a stripped-text `char_offset` to a ProseMirror document position.
**Current behavior:** "Jump to chapter →" — navigates to the mention's chapter (`setCurrentChapter`) and returns to the editor.
**Proper fix:** build stripped-offset → ProseMirror position mapping (walk text nodes accumulating stripped length), then `scrollIntoView` + a transient highlight. Building blocks already exist: the `ChapterSearch` extension (scroll-to-match) and `CharacterHighlight` (name decoration). Worth doing together with the in-editor sidebar redesign since the sidebar wants the same jump.

## 3. Tier taxonomy: design's 3 tiers vs app's 5 roles

**Design:** a Major / Supporting / Minor tier dropdown per row.
**Decision taken:** the app's existing `CharacterRole` taxonomy (protagonist / antagonist / supporting / minor / other) is used instead — it's the persisted metric, and collapsing it to 3 tiers would lose information. If the 3-tier view is wanted later, add a *computed* tier (e.g. by mention-share percentile) displayed alongside the role, not replacing it.

## 4. Role auto-suggestion (new metric)

The design's demo auto-ranks characters into tiers by mention volume. Detection currently assigns every auto-detected character `minor`. A rank-based suggestion (top-N by mentions → supporting, etc.) would need product decisions (thresholds, whether detection may overwrite a hand-set role) — deferred.

## 5. Light-theme pass

The lane cells use hard-coded dark-tuned empties (`rgba(255,255,255,0.03/0.05/0.07)`), matching the design (authored dark-only). In the light theme these near-white overlays are almost invisible against white panels. Needs a `--chars-cell-empty` custom property defined per theme in `global.css`. Cosmetic; the view is functional in light theme.

## 6. In-editor sidebar — ~~out of scope~~ RESOLVED (0.16.02413)

The sidebar was redesigned per "Editor Sidebar Options" design 2a (chapter-first + inline detail): `CharacterQuickRef.tsx` replaced `CastQuickRef.tsx`, the glyph label reads "Characters", and `cast.css` and the whole `components/cast/` directory are deleted. "Jump to first mention" is chapter-level, same as the codex (see gap 2).

## 7. Internal identifiers keep the old name

`cast_enabled` (settings key), `viewMode: 'cast'`, and the `draftline.cast` feature id are unchanged — renaming them would break existing settings.json files and buy nothing user-visible. If ever renamed, migrate the settings key in Go's `loadSettingsFromDisk`.
