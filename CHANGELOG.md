# Changelog

All notable changes to Draftline will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),

## Versioning System

This project does NOT use Semantic Versioning (SemVer).

Instead, it uses a Linear Build Versioning system.
Format: MAJOR.MINOR.BUILD

* **MAJOR / MINOR:** Incremented manually when major feature sets or milestones are reached.
* **BUILD:** A monotonic counter that increments by 1 for every single change/commit to the main branch. One fix = one number.

If a package manager or strict SemVer parsing is a hard requirement for your workflow, this project may not be a fit for you.

If this versioning system has a formal name, I am unaware of it, feel free to raise an issue in Github and hit me with a "WeLl AcTuAlLy" if you know the name.


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
