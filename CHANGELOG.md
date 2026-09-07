# Changelog

All notable changes to Draftline will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),

## [0.17.02561] - 2026-09-07

### Added
- Scene-scoped narrative development synthesis for the v5 engine. The synthesizer now walks scene/chapter units (scene breaks detected in the evidence coordinate space) carrying forward what is true or unresolved — open questions, goals, obstacles, firmly known facts — and emits a development whenever a unit's frames change that state: mysteries introduced/narrowed/reframed/resolved, investigation progress (several small frames aggregating into one scene-level change; acquisitions, events, and claims tied to open goals and questions by content-word overlap), obstacles introduced/overcome, goal establishment and changes of course, decisions causally linked to same-scene discoveries, threat escalation (active-character deaths, shrinking countdowns), corroboration/disconfirmation/cross-reality revelation from event identities, and setup/payoff. Every development records its supporting frames, exact evidence spans, before→after state, the goal or question it advances, causal predecessor/successor, and confidence; the narrative-developments diagnostic renders all of it grouped by chapter and scene.

### Fixed
- Content-word overlap ignores auxiliaries, interrogatives, discourse connectives, contraction remnants, epistemic verbs, and the subject's own name, so "But X didn't know that yet" can no longer open or narrow anything; anaphoric non-questions ("X didn't know.") never open mysteries; recurring-discovery developments require a real acquisition, not a recalled attempt. Extraction: "must be/mean" reads as epistemic modality like "must have"; obligation phrases inside relative clauses ("what he was supposed to look like") abstain; determiner-led "shot" (the noun) is no longer an injury; "shot and killed" now registers as a death.

---

## [0.17.02560] - 2026-09-07

### Fixed
- Precision fixes from running the v5 engine against a full manuscript: bare knowledge-state cues now carry the whole sentence instead of a lone verb; "must have" reads as epistemic modality, never an obligation; possession and transfer items require a real determiner-led noun phrase and reject idioms ("lost his mind") and pronoun objects ("gave me just enough autonomy"); routine motion verbs ("walked", "started walking") can never anchor a same-event identity; backward-story-time findings are limited to nearby chapters; and one open question per character per chapter. Diagnostic outputs are gitignored so manuscript-quoting reports never land in the repository.

---

## [0.17.02559] - 2026-09-06

### Added
- Continuity inspections for the v5 engine, with narrative-scope judgment: characters acting after a narrated death, one entity in two places at the same story time, contradictory route directions to the same destination, items changing hands without a hand-off, knowledge referenced before its acquisition, conflicting retellings of one event, countdown reversals and backward story time, unresolved obligations, and items relinquished before ever being held. Same-scope conflicts read as likely errors; divergence across dreams, simulations, and flashbacks is reported as informational scope divergence — deliberate devices are never flattened into continuity errors.
- A hard acceptance suite: a synthetic manuscript with planted contradictions (route direction, inventory hand-off, countdown reversal, knowledge-before-acquisition, differing retellings, simulation divergence) runs the full pipeline and must catch every plant, keep every frame verbatim and well-formed, and keep the simulation divergence out of the error column.

---

## [0.17.02558] - 2026-09-06

### Added
- Narrative development synthesis for the v5 engine: major discoveries (facts that recur across chapters), open questions created and resolved, goal and relationship changes, new obstacles from opened obligations, same-scope life-status reversals, corroborations and reveals from mixed claim/narration accounts of one event, item/access payoffs, and explicit decisions. Every development is chronological, grounded in frame and evidence IDs, states its mechanical basis, and quotes only verbatim source text. These developments — not raw frames — are the future PlotWalker input.

---

## [0.17.02557] - 2026-09-06

### Added
- Structured event identity for the v5 manuscript-memory engine: accounts join one underlying event only through typed anchors (a person's life-status outcome, an injury condition, a transfer's item and recipient, an actor's action head) — never through shared vocabulary. Retellings keep every account as a property value, conflicting details mark the identity conflicted, and different subjects can never merge. Tests plant a twice-told death, a retold discovery, and two vocabulary-identical sentences with different actors.

---

## [0.17.02556] - 2026-09-06

### Added
- Canonical state ledgers for the v5 manuscript-memory engine: per-entity histories for location, possession (with both sides of every transfer), condition, life status, knowledge, goals, obligations, relationships, access, and a per-scope timeline. Conflicting accounts are preserved as separate entries with their own scope and epistemic posture, never averaged; item qualifiers normalize noun phrases so "picked up the brass key" and "handed the brass key to Mira" meet in one history.

---

## [0.17.02555] - 2026-09-06

### Added
- Typed frame extraction for the v5 manuscript-memory engine: life status, location, possession, transfer, injury, knowledge, belief, claim, goal, decision, obligation, relationship, access, and witnessed-event frames, each anchored on a sentence-initial canonical entity and carrying only verbatim slices of the source sentence. Attributed quotes become character claims that never masquerade as narrator truth; transfers without a resolvable distinct recipient abstain instead of inventing one; unanchored sentences produce nothing. Tests lock the verbatim invariant, the no-self-reference rule, abstention behavior, and the per-sentence frame cap.

---

## [0.17.02554] - 2026-09-06

### Changed
- Replaced the narrative-fingerprint engine with the v5 manuscript-memory architecture (stage 1: the reset). The freeform subject/predicate/object assertion generator, lexical same-event resolver, verb-driven state transitions, promoted-fingerprint corpus, and the dead structure/thread/arc builders are deleted along with their types; the engine now carries typed frames (constrained vocabulary, verbatim payloads, explicit abstention), state ledgers, structured event identities, developments, and scope-aware inspections as its model, with extraction landing in the next builds. Story timeline now always builds from source evidence; continuity projects the new inspections. The Story Graph, Threads, and Worth Reviewing panels are disconnected with clear rebuilding notices; Continuity, Ask Draftline, and the Evidence index are unaffected. The fingerprintdiag tool now writes fingerprints-v5.txt, narrative-developments-v5.txt, and inspections-v5.txt in UTF-8.

---

## [0.17.02553] - 2026-09-06

### Changed
- The file-size guardrail asks for raise rationale in the commit message or PR description instead of pointing at a maintainers-only ledger, and two code comments no longer cite untracked design-asset paths. (Restores changes lost in a branch reset, together with the private-to-public sync tooling.)

---

## [0.17.02552] - 2026-09-06

### Added
- Added three independently selectable, read-only textual quality gates for the fingerprint corpus, NarrativeDevelopments, and continuity inspections. The `fingerprintdiag` CLI supports `-report corpus|developments|inspections|all` and `-summary`; the Wails contract exposes the same reports through `GetFingerprintTextDiagnostics`.
- Added generic regression coverage preventing idioms, scores, inanimate grammatical subjects, and bare intransitive returns from becoming object-custody history while retaining concrete acquisition, carrying, surrender, and transfer.

### Changed
- Removed the retired promoted-fingerprint schema, relations, statistics, report binding, compatibility projection, and implementation. All normalized assertions now enter the manuscript fingerprint corpus; contextual significance exists only as `NarrativeDevelopment`.
- Renamed the remaining assertion-relation implementation around its actual corpus-memory role, and updated author correction reconciliation to recognize corpus fingerprints, same-event identities, and NarrativeDevelopments.
- Limited the new inspection report to source-backed continuity categories. Noisy character-presence guesses and evidence-free orphan records remain available through their legacy diagnostics without masquerading as two-sided fingerprint conflicts.
- Updated backend and roadmap documentation to make the evidence → assertion → fingerprint corpus → state/inspection and development boundaries explicit. Scenes, sequences, threads, arcs, and graphs remain disabled until they can consume trustworthy NarrativeDevelopments.

---

## [0.17.02551] - 2026-09-06

### Added
- Added contextual `NarrativeDevelopment` synthesis as a distinct layer above comprehensive manuscript fingerprints. Developments express objectives, obligations, relationship and persistent-state changes, discoveries, obstacles, threats, causal enablement, fulfillment, corroboration, contradiction, and supersession without treating every action as story movement.
- Added retrospective significance for concrete findings: an otherwise ordinary discovery remains manuscript memory until later prose depends on the same semantic result. Each development retains its complete fingerprint and evidence dependencies, affected entities, reality scope, story-time placement, before/after state, reasons, confidence, and stable identity.
- Added source-semantics metadata to each manuscript fingerprint, including original evidence types, actions, named entities, and time expressions, so contextual synthesis and downstream continuity tools do not have to reverse-engineer those details from prose.
- Added a plain-text narrative-development diagnostic with normalized changes, synthesis reasons, epistemic and reality status, confidence, and every exact supporting quotation.

### Changed
- Expanded general objective and relationship recognition to cover determined or necessary courses of action and present-tense trust changes while requiring stable actors, substantive objectives, and actual relationship participants before creating a development.
- Kept isolated object findings, routine movement, generic information exchange, and action-heavy passages in the fingerprint corpus unless later context establishes a meaningful consequence.

---

## [0.17.02550] - 2026-09-06

### Added
- Added corpus-derived state histories for possession/custody, location, knowledge, belief, relationships, identity, attributes, conditions, goals, and commitments. State entries preserve operation, value, attribution, epistemic status, reality scope, story time, manuscript position, and source evidence without becoming roadmap events.
- Added fingerprint inspections as a separate corpus query layer. Conflicting assertions, incompatible properties on a shared event, stable-state conflicts, open obligations, and the existing source-backed continuity checks now produce inspectable findings with evidence from every side and an explicit reality/epistemic assessment.
- Added separate plain-text state-history and inspection diagnostics plus generic fixtures for custody transitions, location and knowledge memory, conflicting attributes, competing event accounts, cross-reality state separation, and unresolved obligations.

### Changed
- Removed promoted narrative fingerprints from the build path. Story events, threads, and structure remain empty until they can be derived from contextual NarrativeDevelopments; local story questions now query an ephemeral projection of the complete fingerprint corpus instead of roadmap importance.
- Distinguished carrying an object from acquiring it, preserving custody progression instead of collapsing both statements into one state.

### Fixed
- Prevented legacy attribute inspection from comparing persistent details across incompatible reality scopes.

---

## [0.17.02549] - 2026-09-06

### Added
- Added schema-5 manuscript-memory fingerprints as a comprehensive normalized corpus rather than a story-importance filter. Every retained assertion now carries semantic subject/predicate/object detail, entities, attribution, epistemic status, reality scope, temporal placement, persistence, state changes, confidence, stable identity, and complete source spans.
- Added conservative same-event identity resolution using event type, semantic detail, participants, reality scope, temporal compatibility, and nearby narrative context. Multiple accounts retain separate properties and evidence; incompatible properties mark the shared event identity as conflicted instead of being averaged away.
- Added an in-memory fingerprint-corpus diagnostic that enumerates all retained manuscript knowledge, relationships, evidence, and same-event identities without persisting duplicate quotations in the archive.
- Added manuscript-independent fixtures proving that routine continuity details remain fingerprints, epistemic and dream scopes survive normalization, repeated assertions accumulate evidence, and conflicting accounts remain attached to one probable underlying event.

---

## [0.17.02548] - 2026-09-06

### Added
- Added a read-only `GetNarrativeFingerprintDiagnostic` Wails binding and `fingerprintdiag` maintainer CLI. Both rebuild a plain-text semantic quality report from persisted evidence, including promoted assertions, epistemic status, attribution, reality scope, promotion reasons, exact supporting spans, and identified relationships; summary mode exposes only pipeline counts.
- Added maintainer documentation for the evidence → assertion/state → narrative fingerprint boundary and the textual quality gate that must pass before higher-level story structure returns.

### Changed
- Kept formatted diagnostic reports transient instead of persisting duplicate manuscript quotations inside `analysis.json`; `.draftline` archives retain the canonical evidence once, and reports are regenerated locally on demand.

### Fixed
- Made the diagnostic CLI state plainly when an archive has no persisted evidence instead of printing an empty summary, and aligned author-model-only rebuilds with analysis schema 6.

---

## [0.17.02547] - 2026-09-06

### Changed
- Replaced the quadratic all-assertion relationship scan with semantic indexes for comparable durable states, commitments, potential setup, and uncertain claims. Retrospective promotion now evaluates only explainable candidates that share substantive terms, keeping large evidence sets responsive without introducing an event-count target.
- Tightened narrative promotion around objective, durable state changes and explicit causal language. Conditional decisions, attenuated outcomes, routine logistics, ordinary object handling, and low-information discoveries remain lossless evidence unless later prose establishes their significance.

### Fixed
- Prevented dialogue, second-person address, pronouns, and deictic or existential subjects such as “this” and “there” from silently becoming objective world truth or fabricated contradiction anchors.
- Canonicalized equivalent persistent outcomes such as death, injury, capture, escape, and destruction so alternate wording does not create false state conflicts.
- Added generic regression fixtures for conditional decisions, near misses, deictic and existential claims, and 1,500 routine evidence atoms that must remain free of invented narrative structure.

---

## [0.17.02546] - 2026-09-06

### Changed
- Reset the deterministic story-fingerprint pipeline around explicit semantic boundaries: ProseV3 sentence records remain lossless evidence atoms, richer assertions preserve attribution, epistemic posture, reality scope, persistence, state transitions, temporal placement, and stable evidence provenance, and only independently justified assertions become narrative fingerprints.
- Replaced the evidence-per-event build path with precision-first promotion plus retrospective dependency analysis. Routine movement, incidental activity, descriptive state, and ordinary dialogue remain queryable evidence; durable state changes, goals, commitments, consequential discoveries, relationship changes, identifiable deception, contradictions, fulfillment, corroboration, and later-used setup can be promoted with inspectable reasons.
- Disabled legacy automatic thread and Story Structure generation until the promoted narrative layer is semantically trustworthy. Continuity state remains evidence-complete through an internal compatibility join, without exposing those support records as author-facing events.
- Added manuscript-independent behavioral fixtures for routine motion, object setup and payoff, lies, beliefs, contradictory accounts, correcting reveals, dreams, flashbacks, false leads, promises, knowledge transfer, repeated evidence, retrospective significance, action-heavy but unchanged scenes, and quiet durable decisions.

### Fixed
- Prevented incidental relative phrases such as “two years ago” from relocating the containing scene or all later prose into a fabricated flashback context.

---

## [0.17.02545] - 2026-09-06

### Changed
- Aligned the shared documentation with the public repository word for word: design-asset folder references removed, the AI Studio and Characters gap docs describe the shipped designs directly, VERSION.md drops the roadmap row from the bump table (the roadmap no longer carries a version line), and the root ROADMAP.md intro no longer links maintainers-only planning notes. Added the CONTRIBUTING.md guide. From here the docs trees diff clean between the two repositories.

---

## [0.17.02544] - 2026-09-06

### Fixed
- Repaired stale documentation references: the doc index and architecture overview now link the guides that exist (BACKEND.md, FRONTEND.md, DATA-MODEL.md, the AI docs) instead of never-written per-directory README files, and the analysis-suite/frontend docs reflect the single-stylesheet consolidation (panel styles live in bannered global.css sections, not per-panel .css files). Trimmed a speculative note from VERSION.md.

---

## [0.17.02543] - 2026-09-05

### Fixed
- Prevented selected-text AI edits from leaving empty paragraphs at selection boundaries. Single-paragraph replacements now remain inline, while fully selected paragraph wrappers are consumed for multi-paragraph replacements.
- Restored Claude Code requests by supplying the pinned CLI with its required empty `mcpServers` record instead of an invalid bare MCP configuration object.

### Changed
- Removed the duplicate Read Aloud control from the right sidebar rail. The floating editor control remains the single reopen affordance.

---

## [0.17.02542] - 2026-09-05

### Changed
- Replaced the alarming full-page presentation for selected-text AI reviews with a contained review dialog over the still-visible, muted manuscript. The dialog identifies the AI pass and chapter and explicitly states that only the selection can change; true whole-chapter passes retain the full-page diff workspace.

---

## [0.17.02541] - 2026-09-05

### Fixed
- Fixed a critical selected-text AI data-loss path where applying the review treated the selected excerpt as a complete chapter and deleted all surrounding prose. Selection reviews now retain their exact TipTap range, refuse stale results, replace only that range through one undoable editor transaction, and never fall back to whole-chapter replacement.
- Kept the live editor mounted behind AI review so an applied AI change remains available to TipTap's normal `Ctrl+Z` undo history.
- Added immediate atomic Chapter History boundaries before and after every applied AI or comparison pass. Periodic Chapter History capture is now independent of the activity-based autosave toggle, which previously allowed disabling autosave to silently disable all version-history snapshots too.

---

## [0.17.02540] - 2026-09-04

### Changed
- Renamed the focused AI-diff actions from the ambiguous `Keep` and `Accept` labels to `Reject change` and `Accept change`, matching the established bulk-action language without changing their behavior or keyboard shortcuts.

---

## [0.17.02539] - 2026-09-04

### Fixed
- Fixed every Codex AI Studio request failing before execution with `argument '--ask-for-approval' not found`. Codex CLI 0.151 still supports the policy, but it is a root option; Draftline now places `--ask-for-approval never` before the `exec` subcommand and locks that ordering with a regression test. The read-only sandbox, ignored user configuration, disabled tools, ephemeral session, and isolated working directory remain unchanged.

---

## [0.17.02538] - 2026-09-04

### Fixed
- Rewrote three literal NUL bytes in tools/Analysis/shared.ts (used as cache-key separators inside template strings) as `\u0000` escapes. Runtime values are byte-identical, but the raw NULs made git and grep treat the file as binary, hiding it from text search and diffs.

---

## [0.17.02537] - 2026-09-04

### Changed
- Consolidated all co-located component stylesheets back into the single global.css: the ten fragment files under components/characters, tools/AIStudio, tools/Analysis, and tools/Dashboard (~2,300 lines that had accumulated during the late-August sidebar redesigns) now live as bannered feature sections at the end of global.css, in the same cascade order they previously loaded, and the fragment files and their component imports are gone. global.css's header now states the one-stylesheet rule explicitly for future contributors and agents.

### Fixed
- Pruned the last dead rules from global.css (orphaned diff-view button styles) and corrected the stale "AI MODES GRID" section banner, which had outlived its rules and was mislabeling unrelated tool-panel styles — the mislabeling that made AI Studio's actual styles hard to find.

---

## [0.17.02536] - 2026-09-04

### Added
- AI Studio now supports explicit provider routing by editing task. Claude Code, Codex, the configured API provider, or a local model can independently handle Line Edit, Copy Edit, Expand, Smooth, and Custom Prompt; unassigned tasks inherit a user-selected default. Settings exposes the complete routing table, while the sidebar provider menu changes the route for the currently selected task.

### Changed
- Provider routing never falls through to another service. An unavailable assigned provider produces its own setup guidance instead of silently sending manuscript text to a different provider. Clearly incompatible values from the legacy shared model override are ignored in favor of the selected provider's default model.
- A fresh installation with no CLI, API key, or configured local model now says **Set Up an AI Provider** and opens Settings directly on AI Studio instead of steering everyone toward Claude Code.

### Fixed
- The AI sidebar now refreshes Claude Code and Codex readiness after Settings closes and when authentication completes. The sidebar previously retained its mount-time Codex result while the Settings dialog independently showed a successful installation, trapping users in a setup loop.
- Made the Codex-not-installed regression independent of the workstation's real managed Codex installation, preventing installed developer tooling from invalidating or accidentally executing during the test.

---

## [0.17.02535] - 2026-09-03

### Fixed
- Corrected Read Aloud character casting for dialogue tags segmented into standalone narration, including the exact `“Officer Hanlon.” She said flatly. “Thank you for waiting.”` failure. The intervening tag now anchors both adjacent quotations while remaining in the narrator voice, and pronoun-led action beats can anchor the quotation that follows.
- Cast-roster ambiguity is now scoped to characters who can occur in the active chapter. A same-surname character elsewhere in the manuscript no longer strips a valid local surname alias, as `Dr. Renee Alvarez` previously did to `Detective Renee Alvarez` in Chapter 1 of the benchmark manuscript.
- Reordered fallback evidence so a previous paragraph's action beat cannot override established two-person turn-taking. Added a manuscript-derived interrogation fixture covering named and pronoun tags, action beats, same-paragraph continuation, and untagged replies, plus a cross-chapter namesake regression.

---

## [0.17.02534] - 2026-09-03

### Fixed
- Cast attribution now recognizes dialogue tags that use part of a character's name. The speaker roster derives surname and given-name aliases from every recorded name ("Renee Alvarez" also matches "Alvarez said" and "Renee said"), so a clean tag can no longer be invisible and hand the line to another character; honorifics never identify anyone, and a surname shared by two characters is dropped from both as ambiguous. The roster is also recall-first for playback: auto-detected characters still awaiting review are included (only explicit rejections are excluded), because a name attribution can't see corrupts other characters' lines.

---

## [0.17.02533] - 2026-09-03

### Fixed
- Cast mode no longer speaks narrator asides in the character's voice. Sentences now split at quote boundaries for synthesis: quoted speech plays in the assigned character voice while dialogue tags and interrupting asides ("…," she said, checking the hallway, "…") stay with the narrator, audiobook-style. Fragments too short to speak alone merge into a neighboring piece, and highlighting/navigation still work on whole sentences.

---

## [0.17.02532] - 2026-09-03

### Fixed
- Read Aloud cast attribution is far more consistent. A gender-evidence pre-pass now learns each character's pronoun from narration co-reference before any dialogue is attributed, and pronoun tags ("she said") only resolve to a character with positive matching evidence — a nearby male name can no longer steal a she-tag and be remembered in the wrong voice from then on. Dialogue within a paragraph follows fiction's one-speaker-per-paragraph convention (forward and backward), and untagged paragraph hand-offs now alternate between the two most recent speakers even in multi-party scenes instead of dropping whole exchanges into the narrator's voice.

---

## [0.17.02531] - 2026-09-03

### Changed
- Read Aloud polish: the glow-accent toggle lives in Settings → Read Aloud → Player, the settings speed list now reaches 2.0×, and docs/frontend/READ-ALOUD.md documents the new player, voice cast, its accuracy expectations, and the v1 exclusions. readAloudStore joined the size-guardrail ratchet with its reason recorded in TECHNICAL-DEBT.md.

---

## [0.17.02530] - 2026-09-03

### Added
- Full Voice Cast panel: one row per detected chapter speaker with line counts, a voice picker, and a preview button that speaks a short sample (available while paused or stopped; the sample routes through the same serialized local engine and the player's volume, never interrupting the gapless schedule). Uncast speakers get a one-click "Cast voice"; "Auto-cast from Characters" fills every uncast speaker deterministically; the unknown bucket shows how many lines fall back to the narrator. Rows dim while cast mode is off.

---

## [0.17.02529] - 2026-09-03

### Added
- Expanded Read Aloud panel: the bar's chevron (or voice chip) opens a popover with a "Now reading" strip showing the current sentence in the speaker's color, the Voice Cast section (cast-mode toggle plus the narrator voice, moved out of the compact bar), and a Chapter Progress column with the large seek bar, speaker legend, time-left/elapsed/dialogue-line stats, and shortcut hints. The panel collapses on outside click; an engine badge notes the local Kokoro 82M · 24 kHz pipeline. The compact bar gained the current-speaker voice chip and a one-click cast-mode toggle.

---

## [0.17.02528] - 2026-09-03

### Added
- Cast mode is live in Read Aloud playback: with cast mode on, each sentence synthesizes in its attributed character's assigned voice (narrator, unknown, and uncast speakers fall back to the narrator voice). The seek bar shows speaker-colored dialogue ticks, and the bar's equalizer and speaker label follow the character being read in their app-wide character color. Reassigning a voice or toggling cast mode mid-play lets the audible sentence finish and re-synthesizes everything after it; a warmed-but-idle buffer silently refills. With cast mode off, playback and display are exactly the single-narrator behavior.

---

## [0.17.02527] - 2026-09-03

### Changed
- Rebuilt the Read Aloud player bar to the new player design: equalizer with current-speaker readout, prev/play–pause/next/stop transport with an accent play button, a click-to-seek progress bar with a diamond playhead plus a 2px progress line along the bar's top edge (glow accent included), chapter/sentence/time-remaining readout, one-click speed cycling (0.8×–2×), mute and a draggable volume slider, an expand chevron for the upcoming panel, and a compact layout for narrow windows. Voice model setup, "Read chapter", the prefill warm-up, and the existing Ctrl+Shift shortcuts all carry over.

### Added
- A floating Read Aloud reopen button appears at the bottom-right of the editor when the plugin is enabled but the player is closed.

---

## [0.17.02526] - 2026-09-03

### Added
- Per-book Read Aloud voice-cast persistence: a new optional `read_aloud_cast.json` archive member stores cast mode and each character's assigned voice inside the .draftline file (written only when casting is configured; absent files and older readers are unaffected). Round-trip covered by Go tests.
- Voice-cast model: chapter speaker rosters built from the book's confirmed characters, saved-assignment resolution, and deterministic gender-aware auto-casting over the eight Kokoro voices (never overwrites manual choices; reuses the app-wide character colors).
- Read Aloud playback time accounting: exact durations for synthesized audio plus a self-calibrating seconds-per-character estimate for the rest, feeding a change-gated elapsed/remaining/playhead progress snapshot.
- New settings `read_aloud_volume` (master gain) and `read_aloud_glow` (player glow accents); playback speed range widened to 2.0× with a shared speed-cycle helper. Store actions for seek-by-fraction, speed cycling, volume, session mute, and the expanded-panel flag.

---

## [0.17.02525] - 2026-09-03

### Added
- Deterministic per-sentence speaker attribution for Read Aloud (foundation for cast mode): a quote-span tracker that survives mid-quote sentence splits and multi-paragraph speech, plus tag parsing ("…," Marcus said / said Marcus / she said with pronoun resolution), action beats, nearby-mention lookup, two-speaker alternation, and single-speaker monologue continuation. Untagged multi-party dialogue lands in an explicit unknown bucket; em-dash and single-quote dialogue are out of scope for now and read as narration. Fully local — no AI calls.
- Sentences collected for Read Aloud now carry their paragraph index, which the attributor uses for paragraph-boundary conventions and scene-break resets.

---

## [0.17.02524] - 2026-09-03

### Added
- Read Aloud pipeline groundwork for the redesigned player: a master gain node (volume/mute with click-free ramps), one-shot preview playback outside the gapless schedule, per-unit voice overrides in the playback controller (the cast-mode seam; single-voice behavior is untouched), and unit-duration reporting for upcoming time estimates.

### Fixed
- A voice or speed change while the Read Aloud player sat open but idle silently kept the old pre-buffered audio; pressing play could speak the first seconds in the previous voice or speed. The prepared buffer now refills with the new parameters.
- Starting Read Aloud playback at a specific sentence while a pre-filled buffer was standing by always began at the buffer's start instead of the requested sentence. (Latent today — no caller passes a non-zero start yet — but the upcoming seek bar does.)

---

## [0.17.02523] - 2026-09-03

### Changed
- Changed Read Aloud's default playback speed from 1.2× to 1.1× for more natural narration. Existing saved speed choices remain unchanged; the new value applies to fresh settings and missing-value fallback behavior.

---

## [0.17.02522] - 2026-09-03

### Fixed
- Restored natural sentence pacing in Read Aloud. Ordinary sentence buffers now end with a deliberate 220 ms rest and questions or exclamations receive 280 ms, preventing clean gapless scheduling from sounding like one run-on sentence.
- Kept long-sentence clause splitting natural: synthesis units ending in a comma, semicolon, colon, or dash do not receive the full sentence pause. Regression coverage validates both behaviors at the exact sample level.

---

## [0.17.02521] - 2026-09-03

### Fixed
- Removed the metallic high-frequency whine from native Read Aloud by replacing the quantized INT8 Kokoro vocoder with its checksum-pinned FP32 model. Real-bundle validation now rejects non-finite, out-of-range, or misaligned PCM and reports peak, RMS, and boundary-silence measurements.
- Eliminated unsafe parallel eSpeak phonemization. Lookahead requests now execute in manuscript order through one hot native session; cancellation cannot poison the queue, and regression tests prove that requests never overlap.
- Restored duration-aware startup and underrun buffering that was accidentally disabled by a store-level override. Playback now builds up to six seconds of contiguous audio before starting, while short final selections still begin as soon as their only unit is ready.

### Changed
- Upgraded the native voice bundle to provenance-pinned bundle format v4. A successful repair/download removes the obsolete INT8 model only after the FP32 replacement verifies. The platform-dependent download is now approximately 370–385 MiB.
- Auto performance now uses one process-safe native engine with up to six bounded inference threads instead of three concurrently phonemizing sessions. Across the 12-thread Windows reference runs, the FP32 engine generated test sentences at 0.62–0.82 real-time factor, versus roughly 1.55 for one INT8 session, so one clean engine stays ahead of playback without corrupting eSpeak state.

---

## [0.17.02520] - 2026-09-03

### Fixed
- Fixed native Read Aloud failing with “native interface returned no sample rate.” The loopback response now exposes `X-Draftline-Sample-Rate` through CORS, and regression coverage verifies both the POST preflight and exposed header contract.

### Changed
- Removed browser-based Read Aloud completely. Native sherpa-onnx is now the only synthesis protocol: there is no WASM/WebGPU inference worker, browser model, device selector, performance race, model-serving asset route, or silent fallback.
- Replaced the multi-bundle installer with one checksum-pinned native bundle selected for Windows x64, Linux x64/ARM64, or macOS Intel/Apple Silicon. Install, verification, repair, removal, status, and diagnostics now describe that single backend honestly.
- Removed `kokoro-js`, Transformers.js, and the browser phonemizer from the frontend dependency and production bundles.

---

## [0.17.02519] - 2026-09-03

### Added
- Added an optional native Kokoro playback backend built on the official sherpa-onnx v1.13.7 C API. One CGO-free PureGo bridge loads pinned native libraries for Windows x64, macOS x64/arm64, and Linux x64/arm64; unsupported platforms retain the existing browser backend without downloading unusable files.
- Added a checksum-pinned native bundle (~168 MB, platform dependent): the sherpa-compatible Kokoro int8 model, voice table, English lexicon, safely extracted eSpeak data, and only the current platform's ONNX/sherpa libraries. Native files remain optional downloads and all manuscript text and PCM stay on loopback/offline paths.
- Added a gated real-model native test that verifies installation, dynamic loading, callback PCM, consecutive-sentence latency, and aggregate throughput. CGO-disabled cross-compiles validate every supported macOS and Linux target in addition to the Windows runtime test.

### Fixed
- Replaced the single slower-than-realtime CPU synthesis lane with an adaptive bounded pool of independent hot native sessions. On the 12-thread reference machine, three two-thread sessions generated 16.81 seconds of speech in 13.46 seconds (aggregate RTF 0.801), allowing the existing five-sentence lookahead to stay ahead instead of producing multi-second sentence gaps. Smaller machines receive proportionally fewer sessions.
- Native playback starts as soon as its first sentence is ready instead of applying the browser backend's eight-second reserve; subsequent sentences synthesize concurrently while preserving sentence-level highlighting and navigation.
- Native cancellation propagates from Fetch through the request context into sherpa's audio callback. A single process-lifetime callback trampoline avoids leaking callback registrations across long chapters, and model/DLL handles are released before uninstall so Windows can remove them cleanly.
- Native synthesis preflight now explicitly permits POST requests, preventing embedded browsers from rejecting native playback and silently returning to slower WASM generation. The loopback model and synthesis server also uses a fresh 128-bit capability path each launch so unrelated local web content cannot invoke it.

### Changed
- Native CPU playback is preferred automatically once its optional bundle is installed. Browser WebGPU and WASM remain intact as fallbacks, and the settings screen reports which backend is active.

---

## [0.17.02518] - 2026-09-03

### Fixed
- Removed the 02517 twenty-second/two-paragraph playback gate. On a backend slower than real time, requiring twenty seconds of completed audio translated into roughly a minute of silence before playback; buffering amplified the latency instead of solving synthesis throughput.

### Changed
- Restored the bounded five-unit producer window and eight-second/two-unit startup reserve while the Read Aloud engine moves toward a genuinely streaming native runtime. The current kokoro-js `stream()` method is not waveform streaming: it awaits one complete generated sentence before yielding it, so replacing `generate()` with that API would preserve the same blocking boundary.
- Documented the architectural limit honestly: paragraph-sized browser generation cannot retain accurate sentence highlighting or sentence navigation because kokoro-js exposes neither sentence timestamps nor partial PCM. Native sherpa-onnx Kokoro supports generated-audio callbacks and is the viable path to incremental playback rather than additional queue inflation.

---

## [0.17.02517] - 2026-09-03

### Fixed
- Read Aloud's duration reserve can no longer stall behind its own fixed lookahead. While starting, silently prefilling, or recovering, the producer may queue up to 32 upcoming sentences until it has a genuine 20-second audio runway; normal playback then maintains a rolling 12-unit lookahead. Runs of tiny sentences therefore continue filling instead of stopping below the admission threshold.
- Playback buffering now understands ProseMirror text-block boundaries. It finishes the current paragraph and the following paragraph before beginning whenever both exist, preventing a short final sentence in one paragraph from exposing a long first sentence in the next while preserving sentence-level highlighting, skipping, and click-to-jump. A 32-unit defensive ceiling prevents malformed or imported mega-paragraphs from creating an unbounded queue.
- Starting an already prepared queue now clears its preparation marker. Previously that marker could keep the controller in its deep-prefill policy for the entire playback session.

### Changed
- Paragraph buffering deliberately remains a sequence of separately generated sentence chunks. Kokoro does not return sentence timestamps for a paragraph-sized generation call; treating one opaque paragraph buffer as several sentences would make highlighting dishonest and sentence navigation restart or cut the wrong speech. The controller instead pre-generates whole paragraph runways and schedules their sentence buffers continuously.

---

## [0.17.02516] - 2026-09-03

### Fixed
- Read Aloud now buffers a contiguous reserve of at least two generation units and eight seconds of finished audio before starting or recovering from an underrun. This specifically prevents a short sentence from finishing while a substantially longer successor is still synthesizing; the final remaining sentence is allowed to start without waiting for an impossible reserve.
- Ordinary sentences remain whole synthesis utterances again so Kokoro preserves their sentence-level rhythm, inflection, and clause transitions. Only pathological sentences over 80 words are split at natural clause boundaries as a latency and tokenizer safeguard; highlighting and navigation remain sentence-based.
- Closing Read Aloud during silent prefill now cancels its pending synthesis and releases the prepared queue even though the player intentionally reports an idle public state before Play is pressed.

### Changed
- The five-unit producer lookahead remains in place, but playback admission is now based on measured generated-audio duration rather than unit count alone. This recognizes that one short sentence and one long sentence are not equivalent buffering capacity.
- Updated the Read Aloud architecture documentation with the duration-aware pipeline and measured local backend behavior: the tested WebGPU path sustains real-time generation after warm-up, while CPU/WASM does not reliably do so on the same machine.

---

## [0.17.02515] - 2026-09-03

### Fixed
- Read Aloud's per-sentence gaps. Playback is now a true producer/consumer pipeline: the producer keeps a **five-unit lookahead buffer** filled and never waits on playback (every completed unit immediately requests the next uncovered one; the whole window fires at start), while the consumer only ever waits on generation when the buffer is empty. Previously the effective buffer was too shallow to absorb generation running near real-time, so each sentence waited on its own synthesis.
- Long sentences no longer synthesize as one block: sentences over ~25 words are split at clause boundaries (comma, semicolon, colon, dash) into **generation units** — audio starts on the first clause and later clauses generate while earlier ones play. The highlight still covers the full sentence, the sentence counter counts sentences, and skip/click-to-jump operate on sentences (mapped to their first unit). Audio chunks stay scheduled gaplessly on the single AudioContext timeline via precomputed start times; 'ended' events only drive bookkeeping.
- The pipeline now **pre-fills while the player bar sits open**: the queue from the cursor is built and synthesized before play is pressed, and pressing play arms the prepared buffer instantly when the cursor hasn't moved.
- Version-file sync: docs/VERSION.md and the roadmap current-version line had fallen behind at 02512 while builds 02513–02514 shipped; realigned at 02515.

### Added
- Timeline diagnostics for the gap investigation: `gen[id] start/end` per unit with RTF, `play-start unit k (sentence n)` with main-thread timestamps, the audio port's per-chunk handoff-gap measurement, and an explicit **SharedArrayBuffer → effective ORT thread count** line (without SAB the runtime is single-threaded regardless of the numThreads setting).

### Internal
- kokoro-js's `stream()` was evaluated for intra-sentence chunking and deliberately not adopted: its TextSplitterStream splits at sentence granularity, which is coarser than the clause units now fed to `generate()`, so a multi-chunk protocol would not produce earlier audio.

---

## [0.17.02514] - 2026-09-03

### Fixed
- Repaired the Character Center after the Story Map merge removed the legacy `storyGraph` module while `CharacterInterweave` and its tests still imported three of its display helpers. Beat importance, color, and within-chapter placement now belong to the character weave itself, preserving the upgraded character timeline without restoring obsolete Story Graph code.

---

## [0.17.02513] - 2026-09-03

### Fixed
- Whole-book analysis no longer changes process-global `GOMAXPROCS`. ProseV3 linguistic extraction and evidence extraction now run in separate stage-local, semaphore-bounded worker pools, leaving the Go scheduler available to Wails bindings, the asset server, file operations, Read Aloud, and unrelated backend work.
- The one-analysis-at-a-time backend guard remains intact across automatic full analysis and manual character/relationship indexing.

### Changed
- Gentle, Balanced, Fast, and Adaptive now size analysis workers rather than scheduler threads. Adaptive keeps the Balanced worker count for large books; the 750,000-byte tier is explicitly a memory safeguard, using a 4 MiB weighted in-flight payload gate and 512 KiB linguistic batches instead of silently reducing thread parallelism.
- Settings now describe worker limits accurately. The backend guide documents pool behavior and the measured Reset/12-thread stress results, including the observed Read Aloud contention rather than claiming zero finite-CPU impact.

### Added
- Regression coverage proves analysis acquisition does not change the scheduler, the single-flight lock still rejects overlapping analyses, large-book batching preserves every chapter and respects its byte boundary, and the weighted memory gate bounds concurrent Prose payloads.
- Read Aloud audio diagnostics now include each sentence's measured AudioContext handoff gap, allowing analysis/TTS contention to be quantified without changing playback behavior.

---

## [0.17.02512] - 2026-09-03

### Fixed
- Read Aloud no longer balloons system memory to the machine limit. Root cause: the threaded ONNX runtime creates its WebAssembly memory as `{initial:256, maximum:65536, shared:true}` — a **4 GB maximum** — and growable shared memories cannot relocate, so the engine reserves/commits the full maximum eagerly and every extra instantiation (failed threaded attempt, benchmark backend, jsep/non-jsep) stacked another 4 GB that never shrank. onnxruntime-web exposes no memory knob, so the synthesis worker now wraps the `WebAssembly.Memory` constructor and caps every large/shared wasm memory at **1 GiB** (Kokoro q8 needs a few hundred MB; growth past the cap surfaces as a normal generate error, not a dead machine).
- A WebGPU candidate that fails its audio smoke test now has its ORT session explicitly released before the CPU fallback (previously its whole wasm memory stayed pinned), the dispose path releases the session before the worker terminates, and the editor-watch subscription is bound once per app lifetime instead of accumulating across plugin enable/disable cycles.
- Long plays hold a bounded set of audio buffers: the controller's chunk cache now evicts sentences more than 4 behind the playhead (short skip-backs stay instant) instead of retaining every synthesized Float32Array for the session.

### Added
- Memory instrumentation across the stack, all on the shared diagnostics stream: every worker line now carries a timestamp and worker-start counter; heap + live-wasm-memory snapshots log at pipeline construction, dispose, the first three sentences and every tenth after; each audio enqueue logs buffer size, session total, and scheduled count; and Go samples WebView2/app process RSS every 2 s while playback runs (auto-stops after 5 min).

---

## [0.17.02511] - 2026-09-03

### Fixed
- Read Aloud's runtime and model files are now served from a real loopback HTTP origin — a Go server on 127.0.0.1 (ephemeral port, `ReadAloudServerURL()` binding) with CORS, `Cross-Origin-Resource-Policy: cross-origin`, COEP, and explicit MIME types — because WebView2's asset-scheme handler does not reliably intercept requests initiated inside the ONNX runtime's nested pthread workers. The same-origin asset-handler path remains as fallback. The CPU path additionally pins the exact non-JSEP runtime pair (`ort-wasm-simd-threaded.mjs/.wasm`); JSEP exists for WebGPU and is unnecessary for pure-CPU inference.

### Added
- A complete install/verify/repair lifecycle for the voice model. Installs now write a `manifest.json` beside the files (model id + pinned revision, bundle version, installed-at, per-file sizes and SHA-256). New `VerifyReadAloudModel()` re-hashes everything against the compiled pins on plugin enable, on opening the settings section, and before first playback — playback refuses to load an unverified model. Settings show Not installed / Installed vX (N MB, verified) / Corrupt (N files) with a Repair button that re-downloads only failing files; removal unloads the pipeline, clears the kokoro-voices browser cache, deletes everything including the manifest, and confirms the directory is gone.
- `TestFreshInstallLifecycle` (gated, real downloads) proves the whole cycle: fresh → install+manifest → verified (129,337,456 bytes) → deliberate corruption detected → repair re-downloads only the damaged file → verified → removed and confirmed empty.

### Internal
- Embed audit: the executable contains no model bytes — no `.onnx`, `.bin`, or `.wasm` in `frontend/dist` (6.6 MB total; the only Kokoro artifact is the kokoro-js JavaScript library as a lazy chunk). The model on this machine had been installed by the 02505 pin-verification test, which is why no download prompt ever appeared; the cache was reset so the fresh-install flow now shows.

---

## [0.17.02510] - 2026-09-03

### Fixed
- Read Aloud's threaded runtime failure ("Uncaught [object Event]") and the pipeline being rebuilt 2–3 times per play. Root cause: the ONNX runtime spawns nested pthread workers from `ort-wasm-simd-threaded.jsep.mjs`, and in a cross-origin-isolated page a worker script is blocked unless its own response carries COEP — the model asset handler didn't send those headers, so every pthread spawn died as a bare error Event, and the old error handler tore the whole worker down, forcing a fresh model load per play. The handler now sends COOP/COEP/Cross-Origin-Resource-Policy (the middleware adds CORP too), worker error events are logged with message/filename/line instead of tearing anything down after ready, and a single construction guard means concurrent callers await one pipeline — a "worker start #N" diagnostic makes any rebuild immediately visible.
- Threaded WASM init that still fails now retries once single-threaded inside the worker before reporting an error, and the benchmark retries CPU single-threaded before declaring the backend broken; benchmark failures now carry real error text.

### Changed
- Playback starts faster: the first spoken sentence is split at its first clause break (comma/dash/semicolon, exact highlight positions preserved) so audio begins on a short chunk, and synthesis lookahead deepened from one to two sentences — still strictly per-sentence, never the whole selection up front.

### Added
- Finer timing diagnostics: pipeline construction time, one-off phonemization time, and per-sentence generate time with RTF (generate seconds ÷ audio seconds; target < 0.5 with threads active).

---

## [0.17.02509] - 2026-09-03

### Changed
- Began the fixture-gated Story Structure v2 semantic correction without changing the Story Map frontend. Significant-event aggregation now promotes actions, discoveries, decisions, transitions, interactions, obligations, contradictions, and persistent state changes while retaining incidental descriptions and standalone time references exclusively in the lossless atomic fingerprint unless they support a narrative occurrence.
- Replaced the previous adjacent-event score and six-child anti-growth penalty with explainable occurrence membership. Independent actions remain separate; related observations can support one occurrence; no desired event count or manuscript-specific threshold is used.

### Added
- Inspectable structure decisions record aggregate creation, child membership, rejected boundaries, temporal placement, salience components, confidence, and the evidence behind every signal. Significant events retain source quotations, exact positions, per-source confidence and review state, fingerprint corrections, and durable author-decision relationships.
- Durable author structure decisions support confirmation, correction, irrelevant-noise rejection, unresolved interpretations, and intentional ambiguity outside the prose. Dependency snapshots now mark decisions as conflicted or orphaned when later edits rewrite or remove their source instead of silently applying stale intent.
- Human-reviewed semantic fixtures cover the benchmark manuscript's interrupted temporal frame plus linear, alternating-viewpoint, quiet character-driven, parallel-at-a-distance, converging/separating, documentary/epistolary, simulation, ambiguous, and incomplete narrative forms. Stage 1 tests validate specific grouping and separation outcomes rather than aggregate totals.

### Internal
- Advanced the Story Fingerprint schema to v3 and Story Structure schema to v2. the benchmark manuscript probe now derives 384 significant occurrence candidates from its stored 843 fingerprint events while preserving all atomic data; the count is an observed result, not an acceptance target.

---

## [0.17.02508] - 2026-09-03

### Changed
- Read Aloud synthesis is dramatically faster: the Wails asset server now sends `Cross-Origin-Opener-Policy`/`Cross-Origin-Embedder-Policy` on every response, making the webview cross-origin isolated so the ONNX runtime gets SharedArrayBuffer and real WASM threads. The new threading default is **Auto** (`hardwareConcurrency − 1`, capped at 8) with Single as the fallback option.
- The GPU option is now the correct fast path: **WebGPU runs the full-precision fp32 model** (the quantized model is what produced corrupted audio there) as an optional ~311 MB pinned download in a new "gpu" artifact group, fetched on demand from Settings → Performance. A load-time smoke synthesis guards it; failure falls back to CPU with a diagnostic line. CPU stays WASM + q8 explicitly.
- Diagnostics are now **selectable and copyable**: log blocks restore `user-select: text` (the app body disables selection for window dragging) and the settings readout gains a Copy button.

### Added
- **Run performance check** in Settings → Read Aloud: warms each installed backend, times a steady-state sentence on both, reports the numbers to diagnostics, and persists the faster device for this machine.
- Per-sentence synthesis timings for the first three sentences (wall time, chars, audio seconds, real-time ratio). A single "model loaded" line before them confirms the model is held for the session — never reloaded per sentence (it is created once behind an init guard and reused).

---

## [0.17.02507] - 2026-09-03

### Changed
- Updated `docs/frontend/READ-ALOUD.md` for the reworked plugin: the docked player bar and rail-icon entry points, the explicit CPU/WASM q8 device policy with the experimental GPU opt-in, and the load-time diagnostic sequence.

---

## [0.17.02506] - 2026-09-03

### Fixed
- Read Aloud no longer produces distorted audio: synthesis now runs on **CPU (WASM) with the q8 model explicitly** on every platform instead of auto-detecting WebGPU, whose output corrupted on real hardware despite passing the load-time smoke test. WebGPU remains available as an explicit "GPU — experimental" opt-in.

### Changed
- The floating Read Aloud widget is gone. Playback lives in a **player bar docked at the bottom of the editor** — the bottom counterpart of the chapter find bar — with transport, sentence counter, voice and speed pickers, and a setup prompt when the voice model is missing.
- The toolbar speaker glyph was removed. Read Aloud is now opened from a **rail icon pinned at the very bottom of the tools sidebar**, above the bottom-bar space. When the plugin needs configuration the icon carries a badge and deep-links straight to its settings section, the same pattern as the AI Studio CLI setup.
- The current-sentence highlight is now clearly visible: a strong accent wash plus an accent underline, in both themes.
- The Read Aloud settings section only exists while the plugin is enabled, and was rebuilt on the AI Studio setup-card pattern: a voice-model card with status badge, guided checksum-verified download with progress/cancel/resume and removal; proper voice and speed pickers; and new **Performance** options — synthesis device (CPU recommended / GPU experimental) and threading (Single recommended / Auto) — for cross-platform troubleshooting. Device and thread changes apply to the next playback session automatically.

### Added
- Every voice-model load now logs a fixed diagnostic sequence to the WebView console and to a Diagnostics readout in settings: `navigator.gpu` presence, the resolved device and dtype, `crossOriginIsolated`, ONNX-runtime wasm `numThreads`/`simd`, every runtime `.wasm` file actually fetched, model load time, and the wall time of the first synthesized sentence.

---

## [0.17.02505] - 2026-09-03

### Added
- Documented Read Aloud in `docs/frontend/READ-ALOUD.md`: the privacy stance (fully local; network only for the one-time pinned download, nothing at playback), the Kokoro-82M model and its Apache 2.0 license, the download contents and cache location, the Go-to-worker architecture, controls and shortcuts, segmentation rules, WebGPU/WASM device selection, and limitations. Cross-referenced from `PLUGIN-SYSTEM.md` (Read Aloud is the first shipped `optional-model` plugin exercising the planned download pipeline) and `FRONTEND.md`.

### Internal
- Added `TestInstallRealBundle`, an env-gated end-to-end test (`READALOUD_E2E=1`) that downloads the real bundle and verifies every manifest pin against the live sources; it passed against Hugging Face and jsdelivr, confirming all sixteen pinned checksums.

---

## [0.17.02504] - 2026-09-03

### Added
- Read Aloud is now usable end-to-end in the editor. A toolbar speaker button and **Ctrl+Shift+L** read the selection (or from the cursor when nothing is selected); the floating player adds Read chapter, play/pause, stop, sentence skipping (**Ctrl+Shift+.** / **Ctrl+Shift+,**), a sentence counter, and voice/speed pickers that persist.
- The sentence being spoken carries an accent-tinted highlight that follows playback and stays scrolled into view. Clicking any sentence while playing jumps there. The highlight is a pure ProseMirror decoration — it never modifies the document or the undo history.
- Editing the manuscript, switching chapters, or closing the player stops playback cleanly and clears the highlight. Disabling the plugin also terminates the synthesis worker, unloading the model from memory.

### Internal
- New `extensions/ReadAloud.ts` follows the ChapterSearch meta-driven decoration pattern with injected click/edit handlers, keeping the extension free of store imports. Player styles live in `global.css` under the `read-aloud-` prefix.

---

## [0.17.02503] - 2026-09-03

### Added
- The complete Read Aloud synthesis pipeline: a lazily created module worker loads Kokoro-82M strictly from the locally installed bundle (remote models disabled, onnxruntime `.wasm` served from the same local path, and kokoro-js's hardcoded Hugging Face voice URLs rewritten to local ones — after the one-time download the pipeline cannot touch the network), tries WebGPU with a smoke-test probe and falls back to WASM, and remembers the working device. Sentences play gaplessly through one AudioContext clock with exactly one sentence of synthesis lookahead.
- Playback lives in a pure, port-injected controller: pause/resume suspend the audio clock; skip/jump cancel in-flight synthesis and restart at the target (replaying cached sentences instantly); voice or speed changes keep the audible sentence and re-synthesize only the lookahead; a generation counter makes stale audio unplayable. `readAloudStore` now drives it and exposes playback state to the UI landing next build.

### Internal
- `@huggingface/transformers` is pinned to the exact version kokoro-js resolves so both share one `env` instance; the worker configures that env directly (kokoro's own `env` re-export only forwards `wasmPaths`). Vite builds workers in ES format for code-splitting, excludes both libraries from pre-bundling, and drops the 21 MB onnxruntime `.wasm` that `import.meta.url` references would otherwise copy into the embedded `dist`. Controller behavior is covered by 15 fake-port state-machine tests.

---

## [0.17.02502] - 2026-09-03

### Added
- Prose-aware sentence segmentation for Read Aloud: sentence-final punctuation inside closing quotes (`"Go away." Then he left.`), `?!` clusters, honorifics and other abbreviations, single-capital initials, decimals, `No.` before numbers, trailing-off ellipses (continue on lowercase, end on a capital), and em dashes that never terminate. A companion walker maps every sentence onto ProseMirror document positions — coalescing across bold/italic mark boundaries the way chapter search does — with range clamping that keeps a straddling sentence whole for play-from-cursor.

### Internal
- Segmentation is a pure text→spans function under `services/readaloud/`, separate from the position walker, with 26 unit tests covering dialogue, abbreviations, ellipses, em dashes, fragment merging, and position mapping.

---

## [0.17.02501] - 2026-09-03

### Added
- The **Read Aloud** plugin now appears in Settings → Plugins as an opt-in card (disabled by default) and gets its own Settings → Read Aloud section: voice picker over eight bundled English voices, a 0.8×–1.6× speed slider (default 1.2×), and voice-model management — download with live progress and cancel, resume after interruption, and "Remove downloaded model".
- New `readAloudStore` tracks model install state over the `readaloud:progress`/`readaloud:done` events; voice and speed persist through the standard settings pipeline.

### Internal
- Added the `kokoro-js` dependency (Kokoro-82M TTS via transformers.js). It is only ever loaded on demand inside the forthcoming synthesis worker — enabling nothing loads while the plugin is off.

---

## [0.17.02500] - 2026-09-03

### Added
- Backend foundation for the forthcoming opt-in **Read Aloud** plugin: a new `internal/readaloud` package downloads the Kokoro-82M voice bundle (model, English voices, and the onnxruntime-web runtime, ~130 MB total) from pinned immutable revisions, verifying every file against a hard-coded SHA-256 and byte count before it is installed under the user cache directory. Downloads stream with byte-level progress events, support cancellation, and resume at file granularity.
- A read-only asset-server fallback handler now serves the installed voice bundle to the webview at `/readaloud-models/`, with traversal rejection and an explicit content-type whitelist. This is the first use of the Wails AssetServer `Handler` slot; embedded frontend assets are never shadowed.
- New settings fields `read_aloud_enabled` (default off), `read_aloud_voice`, and `read_aloud_speed`. The plugin's UI arrives in the next builds; nothing loads while it is disabled.

### Internal
- Bound methods live in a new `wails/readaloud.go` and the package ships with download, checksum-rejection, cancellation, resume, status, and handler-security tests. `app.go` gains only three default lines.

---

## [0.17.02499] - 2026-09-02

### Added
- Introduced the persisted, evidence-preserving **Story Structure** layer: significant events aggregate related fingerprint propositions, semantic scenes preserve continuous context, sequences group sustained action, narrative threads may converge and separate, and high-level arcs provide whole-book containers. Every level retains complete evidence and child links back to the lossless ProseV3 fingerprint.
- Added independent salience components for state changes, movement, discoveries, temporal transitions, plot obligations, later references, contradictions, descriptive-only material, and low-confidence parsing. Overview selection now guarantees structural coverage for arcs, narrative threads, temporal contexts, convergence points, and plot-thread changes before filling remaining capacity by salience.
- Added Story Map semantic zoom with Overview, Sequence, Scene, and Details levels, plus distinct **Story time** and **Manuscript order** projections. Unresolved chronology stays visibly floating rather than receiving invented dates.
- Expanded the Story Map inspector from one representative quotation to multiple supporting facts, confidence and salience, state changes, obligations, source navigation, and the existing author-confirmable temporal placement flow.
- Added persisted source-block hashes, evidence-to-aggregate reverse dependencies, and aggregate parent links as the invalidation foundation for progressively narrower analysis updates.

### Changed
- Replaced the live Story Map's direct rendering of shallow fingerprint events and global top-120 ranking with the Story Structure hierarchy. The existing SVG surface, themed era bands, source navigation, diagnostics, and correction flow remain in place.
- ProseV3 evidence analysis now maintains per-chapter content hashes and reuses unchanged chapter evidence after character resolution when entity identities remain stable. Ordinary edits no longer force unchanged chapters through evidence extraction; schema identity advanced to `prose-v3-evidence-v3`.
- Advanced the rebuildable `analysis.json` container to version 5; older archives remain compatible and acquire Story Structure during their next normal analysis.
- Advanced Draftline to the `0.17` milestone for the first author-facing hierarchical representation of manuscript structure.

### Internal
- Added deterministic aggregation and projection tests covering lossless evidence joins, temporal scene separation, structural overview coverage, and differing fictional-time/manuscript-order results.
- Kept EPUB normalization independent from semantic aggregation; uncertain import boundaries are not promoted into authoritative story structure.

---

## [0.16.02498] - 2026-09-01

### Added
- Redesigned the bottom bar to the v7 layout: five tabs — Story Map, Threads, Review, Continuity, Ask Draftline — over the Story Fingerprint engine, with per-tab subtitles, an undecided-detections badge on Review, and the Evidence index behind the fingerprint button.
- Story Map: the manuscript drawn as a story-time map — era bands per reality/context, day-anchored segments with weekday ticks, and the manuscript-order path weaving across present and recalled lanes. Clicking an event opens a detail pane with its source quote, state changes, thread obligations, cast, prev/next stepping, and one-click open-in-chapter. Weakly-placed events show a dashed ring with a "Confirm Day N / Leave floating" prompt that pins the day through the author model; fulfilled checkpoints get flag lines, open threads trail off as dashed curves, and a first/last-chapter near-duplicate draws a "possible loop — see Review" loop-back. Rendering caps at the 120 most important events.
- Threads: every story obligation plotted across story time — solid while active, sparse-dashed through dormancy gaps, dashed to the edge with OPEN flags while unresolved, ringed at resolution, REOPENED flagged in red when a resolution is contradicted, and CONVERGE bracketing where plotlines merge. Filter chips fade non-matching rows; clicking a row jumps to the opening scene.
- Review: fingerprint detections as decision cards sorted by urgency, the top call in a wide lead column. "It's a reset" records an author simulation context; "Discard" removes an orphaned correction and rebuilds the fingerprint; reviewed/dismissed decisions persist and also dim the matching Continuity question (ids derived from the backend's hashing scheme, test-verified).
- Continuity: redesigned as a master-detail review desk — severity-dotted issue list beside a paired "The claim" / "Established" evidence comparison, with go-to-source and decision actions carried over.

### Removed
- The Story Graph tab: the Story Map supersedes it, built from the fingerprint itself rather than the timeline projection. Its panel, derivation module, and styles were removed.

---

## [0.16.02497] - 2026-09-01

### Added
- Added source-backed **character voice fingerprints** for the forthcoming Character Center flows. Each speaking character can now carry attributed dialogue samples, sample/word counts, average utterance length, contraction/question/exclamation rates, recurring vocabulary, direct-address forms, and confidence.
- Added observable vernacular signals for constructions such as `y'all`/`ain't`, dropped-final-g spellings, negative concord, and habitual `be`. Signals retain representative quotations and counts; Draftline does not infer ethnicity, nationality, or regional identity from those patterns.
- Added durable author voice notes per character: declared dialect, preferred vernacular, speaking traits, avoided wording, and freeform guidance. These survive reanalysis and remain distinct from automatic observations.
- Story Fingerprint questions now understand requests such as “How does Mara speak?” and return the matching voice profile.

### Changed
- Dialogue enters a voice profile only when Draftline finds an explicit nearby name-and-speech attribution. Unattributed dialogue remains unassigned instead of contaminating the wrong character's voice.
- Duplicate quotations in one chapter now retain distinct stable sample identities through their source offsets.

### Internal
- Added isolated voice extraction and aggregation under `internal/fingerprint`, with tests for forward and backward speaker attribution, vernacular signals, address forms, author-note persistence, voice queries, and refusal to guess unattributed speakers.

---

## [0.16.02496] - 2026-09-01

### Added
- Added source-backed Story Fingerprint diagnostics for conflicting persistent attributes and first names, declared-canon conflicts, opposite location directions, dormant obligations, near-duplicate chapters, orphaned manual corrections, one-off character introductions, and unresolved physical presence.
- The presence ledger now catches cases such as introducing Kyle beside Hanlon, never removing Kyle, and later declaring Hanlon alone. The warning retains the exact evidence instead of assuming whether the prose or extraction is wrong.
- Added deterministic story questions over chronology, events, character location/presence, knowledge and belief, open threads, and continuity diagnostics. “After” questions first locate the anchor event, then search only later narrative evidence.
- Added backend author-model updates for contexts, checkpoints, canon, summaries, importance, and explicit story-day corrections. This rebuilds only the semantic model—never prose/v3, manuscript text, or the filesystem.

### Changed
- The existing Story Graph timeline contract now projects consolidated fingerprint events and exposes context, approximate story day, narrative order, and importance. Within a shared context, solvable events default to in-universe chronology while ambiguous events retain manuscript order.
- The existing Continuity report now includes fingerprint diagnostics alongside its earlier character, knowledge, attribute, clock, and thin-chapter checks.
- Manual event corrections can remain attached through stable evidence IDs after surrounding prose changes. Deleted dependencies become reviewable orphan conflicts rather than leaking onto a new event.
- First-open analysis progress now advances monotonically through evidence, chronology/fingerprint construction, and prose metrics instead of jumping backward when the new semantic pass begins.

### Internal
- Regenerated Wails bindings for the new Story Fingerprint model and APIs.
- Obligation resolution uses a per-event token posting index instead of comparing every open thread with every later event. Continuity source projection also uses direct evidence lookup, keeping both passes linear around large novels rather than quadratic.
- Added regression coverage for unresolved room presence, contradictory appearance/name facts, near-duplicate loop chapters, chronological questions, and evidence-backed author time corrections.

---

## [0.16.02495] - 2026-09-01

### Added
- Added deterministic obligation-based plot threads for questions, promises, goals, threats, mysteries, and explicitly separated temporal/reality contexts. Threads retain their opening evidence, subsequent contributing events, entity participants, dormancy, partial progress, convergence, and conservative resolution.
- Added author checkpoints with multiple positive requirements, negative conditions, and optional before/after chapter windows. A matching passage produces `partial` or `proposed` status; Draftline never silently declares the author's milestone fulfilled.
- Added overlapping inferred analysis profiles for domains such as police procedural, mystery/thriller, speculative science fiction, military science fiction, and romance. Author-declared profiles remain authoritative while deterministic profiles stay visibly inferred.

### Changed
- Plot-thread identity is now based on narrative obligations and their evidence trail instead of recurring words or exact cast combinations. A later event can connect multiple prior obligations and expose their convergence.
- Thread resolution is deliberately conservative: related evidence advances an obligation, while explicit result language plus sufficiently matching subject matter is required to resolve it. Long manuscript gaps leave a visible dormant thread rather than dropping it.

### Internal
- Added isolated profile, obligation/thread, and checkpoint evaluators plus regression coverage for promises resolved by later events, partially met multi-criterion checkpoints, and multi-genre manuscripts.

---

## [0.16.02494] - 2026-09-01

### Added
- Story Fingerprint assertions now preserve subject, action, object, polarity, confidence, reality context, and whether a statement is narration, dialogue claim, memory, belief, suspicion, or dream material. Knowledge and lack-of-knowledge claims remain distinct instead of becoming one undifferentiated fact.
- Added typed state intervals for physical presence, durable attributes, physical custody, knowledge, and belief. Persistent facts can span chapters; passage presence closes conservatively at the chapter boundary unless later analysis provides a stronger transition.

### Changed
- Adjacent evidence describing one occurrence now consolidates into a single fingerprint event with all supporting source IDs. This removes the one-sentence/one-node failure mode while retaining drill-down evidence.
- Fingerprint events receive composite identities derived from context, chapter, participants, action, and normalized source content. Compatible prior events retain author-written summaries after reanalysis.
- Event summaries are now mechanically constructed from extracted roles when possible; raw source text remains attached as evidence rather than being treated as the display model.

### Internal
- Split semantic extraction into focused assertion, event-consolidation, persistent-state, chronology, and identity modules under `internal/fingerprint`; no orchestration was added to `app.go`.
- Added regression tests for multi-sentence event consolidation, claim/negation posture, persistent character attributes, and bounded physical presence.

---

## [0.16.02493] - 2026-09-01

### Added
- Introduced the versioned, source-backed **Story Fingerprint v2** archive contract. It has explicit homes for reality contexts, temporal constraints, structured assertions, consolidated events, persistent state, plot obligations, author checkpoints, canon, corrections, and diagnostics without embedding a second manuscript copy.
- Added the first deterministic chronology pass. Manuscript order and in-universe story time are now separate values; weekday and relative-time evidence retain confidence and truth posture, and explicit past, dream, and simulation cues create distinct contexts instead of being flattened into chapter order.
- Author-defined contexts, canon, checkpoints, and corrections survive reanalysis. A correction whose source disappears is retained as an orphan for review rather than silently applied to unrelated prose or discarded.

### Changed
- Whole-book analysis now builds the fingerprint after the shared evidence and prose passes and advances the analysis schema to version 4. Existing `.draftline` archives remain readable and gain the new model on their next analysis.

### Internal
- Added the isolated `internal/fingerprint` package so semantic analysis does not expand `app.go` or couple persistence to presentation code.
- Added chronology regression coverage for narrative-order/story-time separation, explicit weekday placement, inferred past contexts, durable author canon, and orphaned corrections.

---

## [0.16.02492] - 2026-09-01

### Added
- **Background analysis CPU profiles** under Settings → Application: Adaptive, Gentle, Balanced, and Fast. Adaptive is the new default and automatically gives manuscripts of roughly 750 KB or larger the Gentle 1–2 core budget; ordinary books use Balanced's maximum of four analysis cores.

### Changed
- **The Intertwined view is now genuinely the large-format Story Graph**, not a smaller one. It previously plotted only "this character appears in this chapter" dots, which made the full-screen Character Center show *less* than the bottom bar. It now plots the same source-backed story beats the Story Graph does, at the same position inside a chapter, using a placement rule both views share so a beat can never appear in two different places depending on which view you are in.
- **Presence rails show gaps.** A single line was drawn from a character's first to last chapter, so someone absent for six chapters in the middle looked continuously present. Rails are now drawn as contiguous runs, and an absence is visible.
- **Interaction curves no longer stack.** Every crossing was drawn at its chapter's centre, so two pairs meeting in the same chapter overlapped exactly and a busy chapter turned to mush. Crossings are now spread across the middle of the column, one per pair per chapter.
- Beat size reflects importance (decisive beats are larger and ringed in their event-type colour), matching the Story Graph's vocabulary.
- Hovering a beat or a crossing shows a styled card with the source sentence or the pair and interaction count, replacing the browser's native tooltips.
- Selecting a character now dims only the genuinely unrelated cast: the selection and everyone it shares a confirmed relationship with stay lit.
- Added chapter-width zoom (four steps, showing chapter titles at wider settings), and toggles for story beats and interactions so a dense manuscript can be read one layer at a time.
- Clicking a beat opens its chapter; clicking a chapter header opens that chapter.
- Chapter header and character-name columns are now frozen with CSS sticky positioning instead of a scroll-position transform, so they stay aligned without JavaScript bookkeeping.
- All in-process prose/v3 work now runs beneath one cross-platform Go scheduler budget. The limit covers character NER, entity resolution, relationship mapping, fact/event extraction, pacing, readability, keywords, and summaries rather than throttling only one worker pool.
- CPU choices are described as concurrency budgets instead of exact percentages because operating-system scheduling and non-analysis work can vary. Adaptive, Gentle, and Balanced deliberately leave processing capacity for the editor and the rest of the computer.

### Fixed
- Prevented automatic whole-book analysis, manual character detection, and manual relationship analysis from running concurrently and competing for every available core. The backend now enforces one manuscript-scale analysis job at a time, and an automatic job waits and retries when a manual job is finishing.
- Large novels no longer start prose/v3's former eight-worker character pass at full machine concurrency by default. On an eight-core machine, a large manuscript now receives two analysis cores in Adaptive mode.

### Internal
- `positionOfParagraph` extracted from `positionInChapter` in `storyGraph.ts` and shared with the weave, so the two views cannot drift apart on beat placement.
- 10 new tests covering gap detection, crossing spread, out-of-range chapter handling, and the shared placement contract (78 frontend tests, up from 68).
- Verified that `chapter_mentions`, `relationship.chapter_history`, and timeline `chapter_index` all use the same global front+body+back enumeration, so beats and crossings land in the columns they belong to.
- Added deterministic tests for adaptive manuscript thresholds, every CPU profile, low-core machines, full-section size accounting, and the backend analysis single-flight lifecycle.

---

## [0.16.02491] - 2026-09-01

### Added
- **Intertwined character timeline** in the full Character Center. The new view reads left to right in manuscript order, keeps one colored rail per confirmed character, plots chapter presence, and bends source-backed relationship crossings between the characters in the chapters where they interact. Selecting a character dims unrelated rails and crossings without hiding the surrounding cast.
- Story beats may now belong to multiple visible story strands. Crossover curves make the chapters where investigations, locations, organizations, objects, or other named story anchors meet visually explicit.

### Changed
- **Story Graph threads now follow story evidence instead of exact cast combinations.** Recurring non-person terms retained from prose/v3 form stable strands even when characters enter or leave a scene. A recurring `IBM` / `One IBM Plaza` reference can therefore remain one strand while Hanlon and Ruiz move independently through it.
- One-off named terms are not promoted into misleading plot lanes. Beats without a recurring story anchor remain visible in explicitly labelled character-led or unlinked fallback lanes, and older archives remain compatible without migration.
- Duplicate anchors backed by the exact same set of source beats collapse into one rail, and the graph caps named strands at twelve to remain legible.
- Character Center remembers Grid, Heatmap, or Intertwined as the author's preferred large-screen view.

### Internal
- Story timeline engine advanced to `source-evidence-timeline-v2` and now projects filtered non-person `thread_terms` from the existing evidence fingerprint. Person, date, time, numeric, and author-confirmed character terms are excluded before they can become story strands.
- Added regression coverage for term filtering, recurring named strands, one-off fallback behavior, crossover membership, link direction, and backwards-safe empty graphs (68 frontend tests).

---

## [0.16.02490] - 2026-09-01

### Added
- **Continuity decisions persist.** Mark a question reviewed or dismissed and the judgement is saved with the book. Decided questions stay listed but drop out of the outstanding counts, so the header badges describe work left rather than work seen. Clicking an active decision again clears it and returns the question to the queue.
- Continuity signal IDs are derived from a question's kind, title, and source evidence, so a decision survives rebuilds of the report but is deliberately not carried over when the underlying prose changes — an edited passage produces a new question that resurfaces for a fresh look.
- Outstanding review/observation counts now appear as badges in the bottom bar header while Continuity is active.
- A **Decided** filter, so previously handled questions can be found again.

### Changed
- **Continuity rebuilt as queue and detail**, matching the Continuity A reference. A 428px queue on the left — filter box, All/Review/Observations/Decided chips, one row per question with severity dot, category and cue strength, and a footer recording how many chapters were checked. The detail pane on the right carries the severity, category and kind chips, the question, its explanation, the paired source quote, and Open source / Mark reviewed / Dismiss actions with a "Question N of M" position.
- Marking a question reviewed or dismissed advances to the next open question, so the queue keeps moving.
- Continuity tab icon is now the reference's circle-check.
- Previously the view was a flat list of cards with four dropdown filters; long explanations and sources competed for the same column, and there was no way to record that a question had been handled.

### Internal
- New `ContinuityDecision` / `ContinuityData` types persisted under `analysis.continuity`. Only decisions are stored — the report itself is always rebuilt from the current fingerprint, so it can never drift from the manuscript.
- `applyDecisions` stamps stored decisions onto freshly built signals; a decision whose question no longer exists is ignored rather than resurrected. Two Go tests cover application, count exclusion, and rejection of unknown or invalid decisions.
- Raised the `bookStore.ts` ratchet 910 → 930 for `setContinuityDecision`, with the reason and a "watch this" note recorded in `docs/TECHNICAL-DEBT.md`. If a fourth raise is needed, the analysis-decision writers should be extracted to a pure helper module instead.

---

## [0.16.02489] - 2026-09-01

### Removed
- **Timeline view**, superseded by the Story Graph. The graph shows the same source-backed events with more context — lanes, chapter placement, connections between beats, and the same "Open source →" navigation — so keeping a second list of the same data would have meant two places to look for one answer.
- Deleted `StoryTimelinePanel.tsx` and its 66 lines of CSS. The `BuildStoryTimeline` backend is unchanged and still in use: it remains the Story Graph's data source.

### Changed
- The bottom bar is now Ask Draftline · Story Graph · Continuity, plus the detections icon.
- The Story Graph owns its loading and empty states (`story-graph-state`) rather than borrowing the timeline's, so removing the timeline could not strand them.
- Reworded the graph's empty state from "No timeline events are indexed yet" to "No story beats are indexed yet" — it referred to a view that no longer exists.

---

## [0.16.02488] - 2026-09-01

### Added
- **Story Graph** — a new bottom-bar tab that draws the manuscript as lanes and beats instead of a list. Rails run left to right in chapter order; each beat is a node placed at its real paragraph position; curved links connect beats that share a character. Everything is derived from the existing source-backed evidence index — no AI, and nothing is drawn that the prose does not support.
- Story Graph lanes switch between **Threads** and **Characters**. Threads are recurring cast combinations (a pairing that appears at least twice becomes a thread); one-off pairings fold into the lead character's rail rather than inventing a thread from a single scene. Character mode gives each confirmed character a rail and draws a beat's supporting cast as stubs back to their own rails.
- Story Graph zoom with three tiers — key beats only, plus notable, all beats. Tiers are deterministic: author-confirmed or pinned records and decisive evidence types (turning point, conflict, resolution, discovery) are key; introductions, first interactions, explicit time markers and any multi-character scene are notable; the rest are minor.
- Story Graph inspector showing the selected beat's type, chapter, exact source sentence, explicit time label, character chips, and its connected beats with the reason each connection exists. Previous/next navigation and "Open source →" jump into the manuscript.
- **Open Character Codex** button in the Story Graph rail, shown when Characters lanes are selected — once you are reading the graph by character, the cast view is one click away.
- Hover tooltips on graph nodes, and an expand/restore control in the bottom bar header.

### Changed
- Bottom bar redesigned to the Story Graph reference: 38px header, underline-style tabs, a per-view subtitle, and a right-hand icon cluster (all detections, expand, close) replacing the previous mixed button styles.
- **Ask Draftline** rebuilt as two columns. The left column answers — signals, "What Draftline can prove" from the knowledge trail, details traveling with the query, and first/last mention. The right column is the source trail, with per-chapter coverage chips that now filter the results and an Earliest/Latest ordering toggle. Previously all of this stacked vertically and the chapter chips were display-only.
- Bottom bar layout adapts below 1100px and 860px so the graph keeps its canvas on narrow windows.

### Fixed
- Selecting a graph beat highlights only its direct connections. Growing the highlight transitively lit the entire graph, because derived links chain nearly every beat together.

### Internal
- `StorySearchToolWindow.tsx` reduced from 313 to 174 lines; it is now a routing shell. New `storysearch/AskPanel.tsx`, `storysearch/StoryGraphPanel.tsx`, and pure `storysearch/storyGraph.ts` follow the `Analysis/` pattern, all well under the 800-line guardrail.
- 19 unit tests for the graph derivation — tiers, thread grouping, character lanes, link direction, zoom filtering, and layout bounds (67 frontend tests total, up from 48).
- Removed the superseded `DetailInsightPanel.tsx` and pruned its 102 dead CSS lines, plus four orphaned `story-search-*` rules.

---

## [0.16.02487] - 2026-09-01

### Added
- Added a deterministic, source-backed **Continuity** workspace beside Ask Draftline and Timeline. Signals are review questions rather than declarations, explain why they fired, and open one or both supporting manuscript passages.
- Added continuity checks for recurring characters without an established given name, strong one-chapter or late one-off characters, closely matched knowledge reversals, knowledge stated before a later acquisition, conflicting eye/hair/age/handedness facts, nearby clock-time regressions, and unusually thin chapter event coverage.
- Added importance, category, confirmed-character, and free-text filters plus explicit cue-strength labels.
- Added regression coverage for character identity gaps, late one-offs, paired knowledge states, physical-detail conflicts, countdown-safe clock comparisons, missing analysis data, and exact paired sources.

### Changed
- Continuity character checks use only manually created characters or accepted person entities with strong detection evidence. Low-confidence names, organizations, objects, and places are excluded from author-facing warnings.

---

## [0.16.02486] - 2026-09-01

### Added
- Added the first automatic, source-backed Story Timeline as a dedicated bottom-tool tab. It assembles deduplicated events in manuscript order from the persistent evidence index and opens every item at its exact source.
- Added a chapter event-density strip and filters for free-text details, confirmed characters, conservatively detected places, event types, and timing certainty.
- Added explicit chronology labels that distinguish time references, relative expressions, and events known only by manuscript placement. Draftline does not assume that a mentioned date is the current scene date or convert ambiguous prose into an invented calendar.
- Added deterministic timeline regression tests covering source deduplication, ordering, rejected evidence, timed facts, character/place/type facets, conservative place cleanup, and missing indexes.

### Changed
- Multiple evidence classifications attached to the same source sentence now become one timeline event, retaining all source IDs while selecting the most editorially useful event label.
- Renamed the status-bar and Story menu entry to **Ask Draftline** so the feature is discoverable by its current name; the same bottom workspace now contains Ask Draftline, Timeline, and the Evidence Archive.

---

## [0.16.02485] - 2026-09-01

### Added
- Added a manuscript-aware **Ask Draftline** landing screen. Writers can explore confirmed characters, knowledge trails, and discoveries generated from the open book without knowing search syntax or relying on hardcoded example queries.

### Changed
- Renamed Detail Search to **Ask Draftline**, replaced its technical empty state with clickable story-specific starting points, and made the input invite ordinary questions.
- Search answers and source-backed knowledge now lead the results display. Raw scene, chapter, event, and fact counts remain available under a compact **Search coverage** disclosure instead of competing with the answer.

---

## [0.16.02484] - 2026-09-01

### Added
- Added persistent, source-backed character knowledge states to the local evidence engine. Draftline now distinguishes learning, knowing, explicitly not knowing, trying to remember, believing, suspecting, sharing, and withholding without using generative AI.
- Added natural knowledge questions to Detail Search. Searches such as “Who knew about the tunnel?” or “Who told Ruiz about IBM?” return a manuscript-order Knowledge Trail naming directly supported characters and counterparties, with the exact sentence and one-click chapter navigation.
- Added regression coverage for knowledge acquisition, direct communication participants, negative knowledge, belief versus certainty, withholding, missing speakers, query interpretation, and source-backed knowledge answers.

### Changed
- Upgraded the persistent evidence engine to `prose-v3-evidence-v2`. Existing event/fact classifications remain stable so compatible author review decisions retain their IDs while the same records gain knowledge metadata during the automatic on-open analysis pass.
- Knowledge attribution now requires a nearby confirmed character in a conservative grammatical position. Pronoun-only subjects, distant clauses, noun uses such as “report,” and ambiguous recipients are omitted rather than silently guessed.
- Evidence records containing knowledge states receive additional priority in the **Worth reviewing** queue.

---

## [0.16.02483] - 2026-09-01

### Changed
- Double-clicking a .storiverse file now shows a clear notice — universe support is coming in a later version of Draftline; update to a version that supports Storiverse — instead of doing nothing. The message is a native dialog, so old installs will tell future users to update rather than dead-clicking.

---

## [0.16.02482] - 2026-09-01

### Added
- .storiverse files are now registered as their own document type with a slate-gray version of the Draftline mark, so universe files are visually distinct from the blue .draftline project icon in Explorer. Opening universes stays unsupported until Storiverse ships — the launch path deliberately ignores the extension for now — but the files are claimed and identifiable today.

---

## [0.16.02481] - 2026-09-01

### Changed
- .draftline files now carry a dedicated document icon: the Draftline mark on the brand-blue rounded square (generated from the preferred dl-icon asset, which is white-on-transparent and needed a backdrop to stay visible in Explorer's light theme). The multi-size .ico is embedded in the exe, extracted to %LOCALAPPDATA%Draftline at registration, and self-heals if deleted; registration falls back to the app icon if extraction fails.

---

## [0.16.02480] - 2026-09-01

### Fixed
- .draftline files now show the Draftline icon instead of the blank-page icon: association registration notifies the Windows shell (SHChangeNotify) whenever the registry entries actually change, so Explorer refreshes its icon cache instead of waiting for a rebuild. Registration also became write-on-change, keeping ordinary launches free of registry churn.

---

## [0.16.02479] - 2026-09-01

### Fixed
- Double-clicking a .draftline (or .epub/.docx) while Draftline was already running focused the window but discarded the file path: the second-instance handler stripped the first argument assuming it was the executable, which could be the document itself. Arguments are now filtered by extension, not position, so the forwarded file always opens.

---

## [0.16.02478] - 2026-09-01

### Added
- Windows: Draftline now self-registers per-user file associations on launch (HKCU, no admin needed): .draftline becomes Draftline's own document type with the app icon, and .epub/.docx gain an "Open with → Draftline" entry without displacing the default ebook reader or Word. Registration is skipped for dev builds and heals itself if the exe moves; removal instructions are in docs/backend/FILE-ASSOCIATIONS.md.
- macOS/Linux packaging assets: a static Info.plist declaring the document types (.draftline as owner, epub/docx as open-with alternates) and a .desktop file + shared-mime-info definition, ready for when those platforms ship. Documented in docs/backend/FILE-ASSOCIATIONS.md.

---

## [0.16.02477] - 2026-09-01

### Added
- Draftline now handles files handed to it by the operating system: a .draftline passed at launch opens the project, and .epub/.docx imports it — the runtime half of file associations. Launching a second copy (double-clicking a document while the app is running) focuses the existing window and opens the file there instead of starting another instance; macOS open-file events are wired through the same path.
- All OS-initiated opens route through the normal flows, so the unsaved-changes dialog is always respected, and imported books arrive as new unsaved projects exactly as wizard imports do.

### Fixed
- Opening a recent project while the current book had unsaved changes used to degrade to the generic file picker after Save/Discard; the specific file now survives the dialog and opens directly.

### Changed
- bookStore's guardrail ratchet raised 860 → 910 (recorded in docs/TECHNICAL-DEBT.md): legitimate growth from Story Search state and the new file-open routing.

---

## [0.16.02476] - 2026-09-01

### Added
- Added persistent evidence review controls to the Evidence Archive. Authors can confirm, reject, pin, annotate, or supply a clearer interpretation for an inferred fact/event while retaining the immutable source sentence underneath it.
- Added a deterministic **Worth reviewing** queue that prioritizes discoveries, introductions, explicit time references, multi-character interactions, lower-confidence matches, and named details that appear in only one indexed passage. The queue is capped at the 100 highest-ranked candidates so reviewing evidence never becomes an obligation to classify the entire raw archive.
- Added saved review provenance (`author_text`, `author_note`, `pinned`, and `reviewed_at`) to evidence records plus regression coverage for ranked candidates and reanalysis persistence.

### Changed
- Evidence Archive now opens on the ranked author-review queue while **Everything**, **Events**, and **Facts** retain access to the complete source-backed index, including rejected records.
- Editing an interpretation stores the author's meaning separately and confirms the record; it never replaces the manuscript quotation used to justify the evidence.

---

## [0.16.02475] - 2026-09-01

### Added
- Evolved Story Search into evidence-backed Detail Search. Queries now produce a compact Story Trail covering every matching chapter, relevant indexed event/fact counts, chronological source passages, and the named details that repeatedly travel with the searched concept.
- Added deterministic question intent for identity, discovery, research, and first-occurrence wording. Searches such as `Ruiz first name` can report that no confirmed given name exists without inventing one, while singleton details receive an explicit one-scene warning and discovery questions distinguish an indexed discovery from later mentions.
- Added clickable related-detail pivots, allowing an author to move directly from one fingerprint association to the complete trail for a connected character, place, organization, or object.
- Added regression coverage for full-trail counts beyond the display limit, natural research questions, confirmed and missing character names, singleton details, related-term noise, and evidence-backed chapter summaries.

### Changed
- Moved the complete raw index behind a compact Evidence Archive folder action while keeping every record browsable. The primary surface now emphasizes author-facing conclusions and exact source trails.
- Restricted supporting evidence to the paragraphs that actually contain query matches. A chapter-sized scene previously attached hundreds of unrelated facts and events to a common character; Detail Search now reports only evidence that genuinely accompanies the searched wording.
- Themed the Story Trail's horizontal chapter and related-detail scrollbars to match Draftline instead of displaying the native bright Windows scrollbar.

---

## [0.16.02474] - 2026-09-01

### Fixed
- The status bar's Plot and Prose chips no longer mirror each other: Plot now tracks the fact/event evidence-indexing phase and Prose the prose-statistics phase (tempo, readability, keywords). Previously both chips were wired to the same pass and the evidence phase was reflected nowhere.

---

## [0.16.02473] - 2026-09-01

### Changed
- Simplified the status bar's right side to just the three analysis module chips, now labeled **Characters / Plot / Prose** (previously Characters / Story / Pacing; internal module keys are unchanged).
- Added a **Character Map** button next to Story Search that opens the full-screen character codex.

### Removed
- Removed the status-bar AI detection meter (AI analysis lives in its own sidebar panel) and the whole-book word count (already shown at the bottom of the chapter list and in the Writing Dashboard), along with their now-dead styles.

---

## [0.16.02472] - 2026-09-01

### Added
- Defined Storiverse as a persistent universe context around the existing Draftline editor. The manuscript/navigation pane gains Manuscript and Storiverse tabs; the latter begins with linked book titles, and clicking an available title opens its original `.draftline` for editing while keeping the universe loaded.
- Specified live book-to-universe propagation: edits mark the book fingerprint and dependent universe views stale immediately, the existing idle pipeline refreshes only that book, and affected cross-book joins update after the fingerprint is atomically committed.
- Added synchronization safeguards for obsolete analysis revisions, unavailable-source fingerprint views, per-book analysis state, and normal unsaved-change handling while switching linked books.

### Changed
- Removed the obsolete trial/upgrade lock from the planned Storiverse tab. Manuscript and Storiverse are two views of the same open-source Draftline workspace, not separate products or mutually exclusive file modes.

---

## [0.16.02471] - 2026-09-01

### Added
- Defined the Story Fingerprint as Draftline's versioned, evidence-backed representation of a book: entities, relationships, facts, events, time and knowledge changes, recurring details, structural signals, author review provenance, and the exact excerpts required to justify its conclusions.
- Preserved the complete raw evidence browser in the roadmap as the Evidence Archive, opened from later editorial screens through a small archive-folder action while ranked views keep routine writing interfaces uncluttered.
- Made connected Story Fingerprints the core Storiverse architecture. A portable universe folder owns per-book fingerprint snapshots under `.storiverse/fingerprints/`, enabling cross-book continuity, chronology, relationships, knowledge tracking, and search when a linked manuscript is moved, offline, or intentionally absent.

### Changed
- Established that Storiverse never keeps a second full manuscript copy. Adding a book creates a live per-device link to the original `.draftline`; saves and universe reindexing refresh its stable-ID fingerprint, while another device can relink any updated copy without creating a duplicate. DOCX/EPUB conversion creates one author-owned `.draftline`, and portable universe exports retain fingerprints by default.

---

## [0.16.02470] - 2026-09-01

### Added
- Added a persistent, local fact/event evidence index to the normal on-open and idle analysis pipeline. Conservative prose/v3-assisted rules identify introductions, discoveries, interactions, transitions, stated facts, and explicit time references while retaining the exact source sentence, chapter/paragraph coordinates, linked confirmed characters, named terms, cue strength, and a stable content-derived ID.
- Added an Evidence Index tab to the resizable Story Search tool window. Writers can browse event/fact counts, filter the records, inspect why each passage qualified, and jump directly to the exact sentence; Story Search results also display any persisted evidence supporting the matched scene.
- Added the current product roadmap at `ROADMAP.md`, including evidence-backed detail search and timelines, continuity checks, a full-screen Storyboard, previewable character refactoring, and storyline/detail refactoring.
- Added regression coverage for exact evidence provenance, stable IDs after unrelated edits, preserved author review decisions, singleton introductions, false-positive restraint, archive round trips, and Story Search evidence attachment.

### Changed
- Replaced the manuscript-specific Story Search placeholder with neutral guidance applicable to every project.
- Bounded evidence analysis to four concurrent chapter workers, cutting the reference manuscript pass from roughly 11.7 seconds to 3.2 seconds while preserving manuscript-order output and the 50,000-record safety cap.
- Moved full-analysis orchestration out of `app.go` into `analysis.go` and ratcheted the facade from 1,511 to 1,468 lines, preserving the 02468 monolith cleanup as the analysis pipeline grows.
- Advanced the rebuildable `analysis.json` schema to version 3. Older projects remain compatible and acquire evidence during their next normal analysis; character reindex, merge, split, and clear operations invalidate stale evidence before rebuilding it.

---

## [0.16.02469] - 2026-08-31

### Added
- Added a JetBrains-style Story Search tool window along the bottom of the editor. It opens from the project menu, the status bar, or `Ctrl+Shift+F`, resizes vertically without displacing the analysis sidebars, and keeps supporting excerpts visible beside one-click navigation that opens the source chapter and highlights the matching evidence.
- Added deterministic whole-manuscript evidence search in a dedicated `internal/storysearch` package. Multi-term queries find scenes where details converge, quoted text performs phrase search, results stay in manuscript order with first/last occurrence controls, and confirmed characters expand to their known aliases while Needs Review candidates do not.
- Added focused regression coverage for alias expansion, review-candidate exclusion, exact phrases, scene boundaries, result limits, and manuscript ordering.

### Changed
- Story Search runs only on explicit submission rather than every keystroke, avoiding repeated full-manuscript transfer across the Wails bridge. It is entirely local, stores no duplicate manuscript data, and does not invoke AI.
- Preserved the 02468 backend refactor: the search engine and its shared result types live under `internal/`, while the new Wails method is isolated in `story_search.go`; no search logic or duplicated helpers were added back to `app.go`.

---

## [0.16.02468] - 2026-08-31

### Changed
- Extracted the AI HTTP transport layer from `app.go` into a new `internal/ai/providers` package (2,065 → 1,511 lines): Anthropic (with streaming), Gemini, and a single OpenAI-compatible implementation replacing the three near-verbatim OpenAI/Grok/local copies. Providers receive an injected emit closure, so the package never touches the Wails runtime; cancellation, the keyring, and all bound methods stay on the app. Event names and payloads (`ai:token`, `ai:log`) are unchanged.
- The pure Claude Code / Codex CLI helpers (argument building, model resolution, prompt-safe failure messages) moved to the new package with their tests, including the security invariants that prompts never echo into error messages. The CLI drivers themselves deliberately remain in `app.go` next to setup's exec helpers (recorded as backlog).

### Added
- Added an automated size guardrail (`debt_guardrail_test.go`): any source file over 800 lines fails `go test` unless it has a ratchet allowlist entry recorded at its current size — allowlisted files may shrink but never grow. CSS is exempt (`global.css` is an intentional single-file stylesheet).
- Rebuilt `docs/TECHNICAL-DEBT.md` from measured counts: all three active regressions resolved (builds 02465–02468), the CSS co-location recommendation withdrawn per owner direction, and the CLI-driver deferral recorded.

---

## [0.16.02467] - 2026-08-31

### Removed
- Pruned 743 lines of dead CSS from `global.css` (5,014 → 4,271): the retired PlotTree visualization and Issues panel blocks, legacy tools-panel/tools-tab chrome, the pre-redesign dashboard and diff-view rules, PlotWalker status-bar styles, and orphaned one-off selectors — every deletion verified against a full unused-selector audit of the frontend source.

### Changed
- `global.css` stays a single stylesheet by design (per workflow: one home per rule, greppable CSS bugs); a header comment now records that convention and the cross-cutting classes to know before pruning. The mislabeled "New Universe Wizard" banner — which actually holds the app-wide dialog/button/form primitives used by ~26 components — is renamed "Shared dialog + form primitives" so nobody deletes it by mistake.

---

## [0.16.02466] - 2026-08-31

### Changed
- Moved app-level UI state out of the book store into the app store where the rest of it already lived: the status-bar message, the left-panel toggle, and the four self-contained dialogs (book info, new chapter, export wizard, chapter history). Only the save-entangled dialogs (unsaved-changes warning, new-book wizard) remain with the book data.
- The book store now owns exactly its domain — book data, file I/O, autosave, chapter history, characters, and the editor bridge — completing the store decoupling flagged by the technical-debt audit (956 → 825 lines across the two builds).

---

## [0.16.02465] - 2026-08-31

### Removed
- Deleted the dead frontend planning-transform layer flagged by the technical-debt audit: `plotStore` (beats, foreshadowing, secrets, knowledge-matrix wrappers) and the story-bible text wrapper had zero callers since the planning sidebars were retired in 0.16.02449. The `.draftline` file-format fields are untouched — only unused code was removed.
- Removed vestigial UI state that was written but never read: the book store's `darkMode` mirror (theming is driven by settings) and the right-panel toggle, whose Ctrl+] shortcut had silently done nothing.

### Added
- Added test coverage for the live character store path (add/update/delete, highlighting, and the entity-resolution accept/reject rewrite), which the audit found completely uncovered.

---

## [0.16.02464] - 2026-08-31

### Changed
- Rebuilt `docs/TECHNICAL-DEBT.md` from measured line counts instead of remembered ones. The previous version reported `app.go` as resolved at 1,259 lines and test coverage at 0%, both of which had been wrong for some time, and never tracked `global.css` at all.
- Recorded three active regressions: `app.go` regrew to 2,065 lines (+66% since the March refactor, ~800 of them added across two days), `styles/global.css` is a 5,014-line untracked monolith and the largest file in the repo, and the `bookStore.ts` split is a facade — `plotStore` and `storyBibleStore` are stateless transform bags that `bookStore` re-exports through 39 delegating wrappers, so no component imports them directly and the coupling never changed.
- Documented that `callOpenAI`, `callGrok`, and `callLocalAI` are near-verbatim copies of the same OpenAI-compatible request, and that the stated reason for keeping AI providers in `app.go` no longer holds now that `dispatchAI` threads context explicitly.
- Marked the parts of the refactor that held — 11 internal Go packages, the six Analysis panels (all under 320 lines), the settings and editor module splits, `ToolsPanel.tsx` still a slim router, and 31 test files where the doc claimed zero — and named `Analysis/` as the reference shape for the components still to be split.
- Removed the stale `tools/StoryBible/` and `tools/PlotWalker/` module listings; both sidebars were retired in `1b3587c` and `416d64e`.
- Added a guardrail: refresh counts with the reproducible commands in the doc rather than from memory, and treat any file crossing 800 lines as needing a deliberate split-or-justify decision.

---

## [0.16.02463] - 2026-08-31

### Changed
- Removed mechanical readability and grade-level estimates from the Signals overview. The neutral replacement reports manuscript word count, and the one-line summary now describes only measured sentence length, dialogue share, and prose tempo without comparing the manuscript to an age group, school grade, genre, or audience.
- Moved traditional readability formulas into a collapsed details section in Prose. When deliberately opened, they are explicitly described as sentence-and-word-shape calculations that do not measure subject knowledge, terminology, narrative complexity, intended audience, or comprehension.

---

## [0.16.02462] - 2026-08-31

### Fixed
- Imported chapters no longer inherit the book's title: EPUBs that repeat the book name in every spine item's `<title>` (common when chapter numbers are images) now fall back to clean "Chapter N" names instead of naming all 100+ chapters after the book.
- Untitled sections routed to front or back matter now take their section name (Dedication, Epigraph, Acknowledgments, …) instead of a generic chapter number.
- Verified against real commercial and self-published EPUBs: numbered chapters, dedication/epigraph front matter, epilogue/acknowledgments back matter, and copyright pages all land correctly with no crashes.

---

## [0.16.02461] - 2026-08-31

### Added
- EPUB imports now route sections where they belong instead of dumping everything into the body: dedications, epigraphs, title pages, forewords, prefaces, introductions, prologues, and author's notes land in Front Matter; epilogues, afterwords, appendices, acknowledgments, about-the-author pages, glossaries, and colophons land in Back Matter — all with the app's canonical section types.
- The copyright page now fills the book's dedicated Copyright section, and cover, table-of-contents, and index pages are skipped entirely (Draftline generates its own navigation on export).
- The import preview now lists any import warnings (skipped documents, dropped images, truncation) and notes how many front/back matter sections were detected alongside the chapter list.
- Documented the import pipeline, sanitizer whitelist, chaptering rules, and routing table in `docs/backend/import/IMPORT.md`.

### Changed
- Untitled imported sections are numbered "Chapter N" counting body chapters only, so front and back matter no longer shift the numbering.

---

## [0.16.02460] - 2026-08-31

### Fixed
- EPUB chapter layouts are no longer butchered by over-eager splitting: a spine document is now one chapter by default and splits only at multiple `<h1>`s (or multiple `<h2>`s when the document has no `<h1>`). Books that use h2/h3 for part titles and scene headings keep them inline instead of being shattered into dozens of bogus chapters.
- The chapter-opening heading now becomes the chapter title and is removed from the content, so exporting and re-importing no longer stacks a duplicate title heading at the top of every chapter.
- Content before the first chapter heading merges into the first chapter instead of becoming a spurious extra chapter.

### Changed
- Import size guards: a single chapter truncates at 2 MB of content with a visible notice, and imports stop with a warning at 40 MB of total content or 500 chapters — well before the editor could be frozen by a pathological file.

---

## [0.16.02459] - 2026-08-31

### Fixed
- Replaced the EPUB importer's regex HTML munging with a real, fault-tolerant XHTML parser and a whitelist sanitizer that emits only the editor's dialect. Imported chapters no longer lose structure invisibly: divs and sections become clean paragraph boundaries, legacy `<b>/<i>/<strike>` map to proper marks, spans/anchors/classes unwrap to their text, `h4–h6` clamp to `h3`, tables flatten to one paragraph per cell, and text alignment survives.
- A missing `</body>` can no longer leak `<head>`/`<style>`/script content into chapter text, and entities (`&mdash;`, numeric references) now decode correctly.
- Verse and preformatted text survive import: whitespace is collapsed per block instead of across the whole document, `<br>` line breaks are kept, and `<pre>` content imports as a code block with its spacing intact.
- Non-UTF-8 EPUBs no longer import as mojibake: UTF-16 (BOM or declared), Latin-1/Windows-1252, and invalid-UTF-8 documents are detected and decoded to clean UTF-8.

### Changed
- Embedded images and vector art are removed during import (the editor cannot represent them); the import now reports how many were dropped instead of losing them silently.

---

## [0.16.02458] - 2026-08-31

### Fixed
- EPUB and DOCX imports can no longer crash Draftline: a panic anywhere inside an importer now returns a normal import error instead of killing the process (the Wails bridge previously had no recovery, so one bad file took the whole app down).
- The editor pane is now wrapped in the error boundary, so a rendering failure on pathological imported content shows a retry fallback instead of a blank window.

### Changed
- EPUB spine hygiene: only real chapter documents (`application/xhtml+xml`, `text/html`) are imported — SVG covers and other XML resources no longer appear as garbage chapters — and EPUB 3 navigation documents (`properties="nav"`) are skipped.
- Spine documents larger than 8 MB are skipped with a warning instead of being pushed through the importer and the JSON bridge; import results now carry a warnings list for non-fatal issues.

---

## [0.16.02457] - 2026-08-31

### Changed
- Redesigned Pacing as an explainable prose-tempo panel: an explicit overall score and definition replace the ambiguous brightness scale, the chapter strip now uses labeled measured/balanced/brisk bands, and the panel states clearly that tempo describes prose rhythm rather than plot urgency or story quality.
- Added plain-language chapter-to-chapter transition findings, with the three largest tempo rises or drops linked directly to their destination chapters.
- Reworked chapter rows to show exact values and supporting context for tempo, dialogue share, and chapter length. Length now compares against the median chapter instead of whichever chapter happens to be longest, and Reading Ease has moved out of Pacing's controls because it belongs to Prose.
- Relabeled the readability result as a composite grade estimate and added an explanation that it averages five formulas rather than representing a definitive reading age; Signals no longer makes unsupported audience claims from that estimate.

### Added
- Added focused tests for tempo boundaries, median chapter-length comparison, and detection of noticeable neighboring tempo changes.

### Fixed
- Author's Note front matter is no longer included in story, readability, pacing, or character analysis.

---

## [0.16.02456] - 2026-08-31

### Changed
- Redesigned the Writing Dashboard around writing activity only: renamed Manuscript and Today sections with clickable "target {n}k" / "goal {n}" hints, "{pct}%" / "{remaining} to go" progress labels, and a This session row with words written, time writing, and words / min.
- Chapter breakdown now spans all sections: front and back matter collapse to single aggregate rows around numbered body chapters, with a "Show all {N}" / "Show fewer" toggle after 10 rows.

### Added
- Added a 7-day goal-streak row to the Today section, backed by a per-book daily word history kept locally for 60 days — green squares for days with words, accent for today.

### Removed
- Removed the dashboard's AI Detection gauge and AI Anti-Patterns sections (and their now-dead styles) in favor of a compact jump row showing the current chapter's debounced detection score and opening the dedicated AI Analysis panel.

---

## [0.16.02455] - 2026-08-31

### Added
- Added the AI Analysis panel to the analysis sidebar: a semicircular gauge scores the current chapter's AI signal with a plain verdict (Reads human / Mixed signals / Likely AI), refreshed instantly on chapter switch and two seconds after edits.
- Added a whole-book AI scan that scores every chapter a few at a time in the background — the UI never blocks — with a book-order heat strip, a Highest signal top-5 list that jumps to the chapter, and completed scans cached so reopening the panel is instant.
- Added Flagged passages: the two highest-signal chapters at or above the mixed threshold surface their single most AI-reading paragraph as a serif excerpt with a score badge and a Go-to-chapter link.
- Added a per-chapter AI anti-pattern list (overused prose tics such as "eyes widened") with counts against per-chapter limits, plus an explicit disclaimer that the score is a probabilistic heuristic, not a verdict.

---

## [0.16.02454] - 2026-08-31

### Added
- Added the Signals analysis panel as the redesigned hub of Story Analysis: freshness row with status dot and Run again, six manuscript stat tiles (chapters, avg. chapter/sentence words, dialogue %, reading ease, tempo) with since-last-run deltas, a one-line plain-language read of the manuscript, and jump rows into the Prose, Pacing, Chapters, and Worth Reviewing panels.
- Stat tiles show "first run" until a second analysis run exists; the Worth Reviewing jump row carries a live count badge that hides at zero.

### Changed
- Analysis freshness now reads simply "Analyzed …" — engine names no longer appear anywhere in the UI.

### Removed
- Removed the old single Story Analysis pane and its styles; its overview metrics live in Signals, its observations in Worth Reviewing, and its per-chapter cards in the Chapters panel. A saved sidebar state pointing at the old pane falls back to the Writing Dashboard.

---

## [0.16.02453] - 2026-08-31

### Added
- Added the Worth Reviewing analysis sidebar panel: dismissable observation cards (structure, pacing, and other kinds) with per-kind filter tabs and counts, so the list can be worked down to zero.
- Observation cards deep-link into the editor via "Go to chapter" and can be dismissed per book; dismissals persist across analysis runs until the observation's content changes, and a "Restore dismissed" action brings them all back.
- The Worth Reviewing rail glyph now carries a live count badge of undismissed observations.

---

## [0.16.02452] - 2026-08-31

### Added
- Added a Chapters analysis sidebar panel: per-chapter keyword chips and one-line extractive summaries from the last local analysis, presented as a clickable card feed — a skimmable recap of the book so far.
- Chapter cards show word count, scene count, and scene-break count, and clicking a card jumps the editor to that chapter.
- The feed shows the first 8 chapters with a "Show all N chapters" / "Show fewer" toggle; chapters with neither keywords nor a summary (e.g. part dividers) are skipped without disturbing chapter numbering.

---

## [0.16.02451] - 2026-08-31

### Added
- Added a Pacing panel to the analysis sidebar suite: a whole-book tempo heat strip (brighter = faster) with an auto-generated sag note when a stretch of 3+ chapters runs at least 12 tempo points below the book average.
- Added a per-chapter pacing scan list switchable between Tempo, Dialogue, Ease, and Length, with a scene-break column; clicking a row jumps the editor to that chapter.
- Pacing bars highlight outlier chapters (more than 1.5 standard deviations from the mean) in amber; chapters under 20 words render faint and are excluded from the statistics.

---

## [0.16.02450] - 2026-08-31

### Added
- Added the analysis sidebar suite foundation from the design-center layout directions: shared cross-panel navigation (any panel or the dashboard can deep-link to another sidebar pane), per-book since-last-run analysis snapshots, dismissable-observation storage with a live rail badge, semantic status color tokens for both themes, and a shared `.an-*` visual vocabulary for tiles, cards, chips, heat strips, and jump rows.
- Added the Prose analysis sidebar panel: a live sentence-rhythm strip for the current chapter (amber bars flag sentences past 25 words), a whole-manuscript sentence-length histogram, short/long/paragraph stats, readability scores, and a word-class breakdown bar.
- The rhythm strip and manuscript histogram recompute at most once per 2 seconds of typing, so the panel never adds keystroke lag; the rhythm strip works even before the first analysis run, while the remaining blocks show an Analyze-now empty state until analysis exists.

---

## [0.16.02449] - 2026-08-31

### Removed
- Retired the unused planning sidebars ahead of the analysis sidebar redesign: Plot Notes, Story Timeline, Beat Sheet, Foreshadowing Ledger, and Knowledge Matrix no longer appear on the tools rail, and the Story Bible / Plot Walker plugin cards were removed from Settings → Plugins. The Story Analysis pane remains and is now gated by the Story Analysis plugin instead of Plot Walker.
- Removed the retired panels' components and dead CSS. The `.draftline` file-format fields (`beat_sheet`, `foreshadowing`, `knowledge_matrix`, `story_bible` plot notes/timeline), their store reducers, and the Go `story_bible_enabled`/`plot_walker_enabled` settings keys are all retained, so existing project and settings files round-trip unchanged.
- A saved sidebar state pointing at a retired section falls back to the Writing Dashboard on next launch.

---

## [0.16.02448] - 2026-08-31

### Added
- Added bundled local Story Analysis powered by the existing Prose v3 runtime. Draftline now persists per-chapter word, sentence, paragraph, scene-break, dialogue, readability, part-of-speech, keyword, extractive-summary, and descriptive tempo measurements in `analysis.json` without downloading another model or overwriting author-owned planning data.
- Added an activity-aware analysis coordinator. Opening a project schedules analysis, manuscript edits mark Characters, Story, and Pacing as stale, and 15 seconds without another edit starts a consolidated character, relationship, story-structure, and pacing pass. Results from a run that overlaps newer edits are discarded instead of replacing the newer manuscript.
- Added JetBrains-style analysis feedback to the bottom status bar: the active phase/chapter and progress track appear in the center, while compact green/yellow/red module indicators report current, stale, or failed results and can be clicked to run immediately.
- Replaced Plot Walker's placeholder Story Analysis pane with manuscript overview metrics, evidence-based review observations, and per-chapter tempo, keywords, and extractive summaries.
- Added `draftline.story-analysis` to the bundled plugin catalog and introduced capability/resource-profile metadata as the client-side contract for future analysis packs. The Marketplace preview and `docs/architecture/PLUGIN-SYSTEM.md` specify signed, data-only model packs, immutable Hugging Face artifacts, checksums, resource disclosure, atomic rollback, and a no-arbitrary-code boundary.

### Changed
- Automatic analysis now owns automatic character indexing on project open, preventing the old standalone index request from racing the consolidated pipeline.
- Story-analysis work runs only after inactivity or an explicit request; manuscript-scale NLP remains off the typing path.

---

## [0.16.02447] - 2026-08-31

### Added
- Added chapter version history inside `.draftline` archives (format 2.2). Chapters now carry stable IDs so history follows them through renames and reordering; changed chapter content is snapshotted after ten minutes of actual writing activity, deduplicated by content hash, and retained at up to 50 versions per chapter / 1,000 per project.
- Added File → Chapter History, with a version timeline, side-by-side word-level comparison against the current chapter, and explicit restore. Restoring first checkpoints the current chapter, making the operation reversible.
- Added Settings → Application → Activity-based saving. Disabling it stops both the five-second paused-edit save and automatic ten-minute history snapshots; manual save remains available.

### Changed
- Ordinary saves preserve unchanged history as already-compressed ZIP entries instead of inflating every snapshot into memory and recompressing it. The comparison dialog and diff engine are lazy-loaded so version history does not increase editor startup cost.
- Older `.draftline` projects receive stable chapter IDs in memory when opened and persist them on their next save; archives without history remain fully compatible.

### Fixed
- Activity-driven history creates no duplicate or idle snapshots: only chapters changed during the active window are considered, and an unchanged content hash is skipped.

---

## [0.16.02446] - 2026-08-31

### Security
- Converted Claude Code and Codex editing subprocesses into text-only execution paths. Claude receives an explicitly empty tool list with safe mode, slash commands and MCP discovery disabled. Codex ignores user configuration and project rules, disables its shell plus browser, filesystem-adjacent, application, image, skill, hook, and sub-agent capabilities, inherits no shell environment, and retains a disposable empty workspace with a read-only fallback sandbox.
- Prevented raw Claude stdout/stderr or fallback stream JSON from reaching the visible result and error panes. Failures now return only fixed, allowlisted authentication, rate-limit, model, or generic diagnostics.
- Draftline now ignores arbitrary global Claude/Codex executables on `PATH` and runs only exact-version managed installations (`Claude Code 2.1.251`, `Codex CLI 0.151.0`). Managed runtimes live separately in the platform-local `draftline/ai/claude` and `draftline/ai/codex` directories, with shared pinned Node tooling under `draftline/ai/node`.

### Documentation
- Reclassified automatic character indexing on open as intended, advertised product behavior rather than an audit defect.

---

## [0.16.02445] - 2026-08-31

### Added
- Licensed Draftline under the MIT License.
- Added a root project README covering Draftline's purpose and features, local-versus-AI privacy boundaries, source-build and test commands, repository layout, `.draftline` files, linear build versioning, and the official `draftline.ink` website.

---

## [0.16.02444] - 2026-08-31

### Changed
- Reorganized `docs/ai-audit/`: moved the five historical audit reports (and the gitignored 182 MB Qodana SARIF) into `resolved/`, and added `RESOLUTION-2026-08-31.md` — a landing report of the 19 fixes landed in builds 02425–02443, known residuals, deferred items, and a manual-test checklist.

---

## [0.16.02443] - 2026-08-31

### Added
- Structural round-trip tests for the `internal/export` package, which
  previously had only `helpers_test.go` while `docx.go`, `epub.go`, `pdf.go`,
  and `print.go` were untested. New `export_test.go` exports a small book to
  each format and validates: the DOCX archive is a valid zip carrying the
  mandatory OOXML parts, with chapter text and metadata escaped exactly once
  (no double-escaping, no raw `&`); the EPUB archive leads with a stored
  `mimetype` entry and its `content.opf` / chapter XHTML carry correctly
  escaped titles and verbatim body entities; and both the PDF and print PDF
  exports produce a parseable `%PDF...%%EOF` envelope whose startxref, xref
  object count, trailer `/Size`, and actual object markers all agree.
  Audit ref: consensus finding J / Sol QA-001.

## [0.16.02442] - 2026-08-31

### Changed
- Stopped the per-keystroke whole-book word recount on the renderer thread.
  `StatusBar`, `ChapterPanel`, and the writing `Dashboard` previously re-parsed
  and re-counted every chapter's HTML on every render, and `ChapterPanel` /
  `Dashboard` re-rendered on every unrelated store update (isDirty,
  statusMessage, isAutoSaving toggles) via bare `useBookStore()` subscriptions.
  Those two components now subscribe through narrow `useShallow` selectors, and
  all three memoize `countBookWords` / per-chapter counts on the book reference
  so the scan only runs when content actually changes. Counts still update after
  every edit; behavior is otherwise identical.
  Audit ref: consensus finding E/F "no-wholebook-recount-on-typing" / Sol PERF-001.

## [0.16.02441] - 2026-08-31

### Changed
- Reduced the algorithmic complexity of the relationship-analysis path
  (`internal/indexing`) without changing output. `AnalyzeBook` now buckets
  mentions by chapter in a single O(M) pass instead of rescanning every mention
  once per chapter (O(C×M)); `detectCharacterEvents` resolves endpoint names
  through a prebuilt entityID→canonical map instead of scanning all entities per
  relationship (~O(R×E)); and `scenes.go` compiles its scene-break / paragraph
  regexes once at package level instead of recompiling them for every chapter.
  These are hygiene refactors: on realistic books the analysis pass was already
  fast (tens of ms), so the change is not measurable end to end — it removes the
  quadratic shapes so the cost stays linear as manuscripts grow. Existing
  indexing, entityresolution, and relationship determinism tests remain green.
  (audit ref: Fable5 C3)

## [0.16.02440] - 2026-08-31

### Fixed
- Print-ready PDF export (`internal/export/print.go`) now writes a correct
  cross-reference table. Previously object offsets were fabricated as `i*100`
  and `startxref` as `buf.Len()-20`, producing structurally invalid PDFs that
  strict readers and print shops could reject. Object bodies are now buffered in
  ID order and their true byte offsets recorded as each is written, mirroring
  `pdf.go`, and `startxref` points at the real `xref` keyword. Added
  `TestPrintPDFXrefOffsets`, which parses the emitted xref/startxref and asserts
  every entry offset lands on its `N 0 obj` marker.
  (audit ref: Fable5 M7)

## [0.16.02439] - 2026-08-31

### Fixed
- AI diff-apply no longer flattens the whole chapter to plain text. Paragraphs the
  user did not change (untouched or rejected) are now preserved as their exact
  original `<p>` HTML — bold, italic, links, font-size and color survive. Whole-
  paragraph accepts keep the revised HTML verbatim; only a paragraph with mixed
  accept/reject decisions inside it is reconstructed from plain-text word chunks.
  (audit ref: Fable5 C1)

### Changed
- Paragraphs are now aligned by an LCS edit script over their text instead of by
  positional index, so inserting or deleting a paragraph no longer cascades into
  spurious full-paragraph diffs on every following paragraph. (audit ref: Fable5 M3)


## Versioning System

This project does NOT use Semantic Versioning (SemVer).

Instead, it uses a Linear Build Versioning system.
Format: MAJOR.MINOR.BUILD

* **MAJOR / MINOR:** Incremented manually when major feature sets or milestones are reached.
* **BUILD:** A monotonic counter that increments by 1 for every single change/commit to the main branch. One fix = one number.

If a package manager or strict SemVer parsing is a hard requirement for your workflow, this project may not be a fit for you.

If this versioning system has a formal name, I am unaware of it, feel free to raise an issue in Github and hit me with a "WeLl AcTuAlLy" if you know the name.


## [0.16.02438] - 2026-08-31

### Fixed
- Re-indexing a book now invalidates the stored relationship graph so it can no longer reference stale entity IDs. Entity IDs (`entity-N`) are positional and get reassigned every time `IndexBook` resolves entities, but `IndexBook` never cleared `book.Analysis.Relationships` — unlike `MergeCharacterEntities` and `SplitCharacterEntity`, which both null it. As a result, after a plain re-index (e.g. after the manuscript text changed) the previously analyzed relationship edges, scenes, interactions, and events still pointed at the old numbering, silently mapping to the wrong characters or dangling entirely. `IndexBook` now sets `book.Analysis.Relationships = nil` after entity resolution, matching the merge/split behavior, so stale edges can't be read and callers re-run relationship analysis against the fresh entity IDs. Added `relationships_test.go` regression asserting that analyzing relationships, mutating the text, and re-indexing leaves `Relationships` cleared.
- Audit ref: fable5-2026-08-30 — Fable C2 (invalidate-relationships-on-reindex).

---

## [0.16.02437] - 2026-08-31

### Fixed
- Dialogue detection now works on real (curly-quoted) manuscripts. `findDialogueRanges` in the relationship co-occurrence indexer had a broken quote table: its first two entries were duplicate straight double-quotes `"`, the curly double pair U+201C/U+201D was absent entirely, and straight single-quote `'` was treated as a dialogue delimiter. On any manuscript typeset with curly quotes, dialogue detection therefore found nothing and produced no directed dialogue interactions, while apostrophes in possessives/contractions ("Kira's") fabricated phantom dialogue spans. The table now recognizes straight double `"`, curly double `“ ”`, and guillemets `« »`, aligned with the mention scanner's `isQuoteInitial`; single quotes are excluded so apostrophes can no longer be mistaken for dialogue delimiters. Added `relationships_test.go` cases asserting a directed dialogue interaction is produced from curly-quoted dialogue and that apostrophes yield no dialogue ranges.
- Audit ref: fable5-2026-08-30 — Fable C4 (curly-quote-dialogue).

---

## [0.16.02436] - 2026-08-30

### Fixed
- AI subprocess (Codex/Claude) stdout/stderr draining no longer stalls the request for the full 180 s deadline on an over-long output line. The pipes were drained with a `bufio.Scanner` capped at 1–2 MB; a single line longer than the buffer made `Scan()` stop permanently, so the reader goroutine exited, the pipe stopped draining, the child blocked on write once the OS buffer filled, and `cmd.Wait()` hung until the deadline. All three subprocess readers now use a new `drainLines` helper built on `bufio.Reader.ReadString` that truncates the retained/logged portion of any line to `maxDrainLine` (2 MB) while consuming the remainder, so draining continues for the full lifetime of the pipe and the child can never block. Added `drain_test.go` covering the over-long-line regression, CRLF/empty-line parity, partial reads, and EOF without trailing newline.
- Audit ref: fable5-2026-08-30 — Fable M5 (subprocess-longline).

---

## [0.16.02435] - 2026-08-30

### Security
- Cap manifest reference amplification when opening a `.draftline` archive. `ziputil` bounds the ZIP's entry count and per-entry/aggregate declared size, but a manifest could reference the same valid entry an unbounded number of times, loading its decompressed content into memory on each reference and bypassing the intended aggregate memory bound (a small archive amplifying to gigabytes). `Open` in `wails/internal/book/open.go` now rejects manifests declaring more than `MaxManifestRefs` (5000) combined chapter/section references before loading any content, and tracks cumulative decompressed bytes across the whole open operation via a running budget, erroring once the total exceeds `MaxTotalLoadedBytes` (500 MB). Added `book_test.go` coverage for a manifest with an absurd number of references.
- Audit ref: master-audit-report-2026-08-30 — Sol REL-003 (cap-manifest-refs).

---

## [0.16.02434] - 2026-08-30

### Fixed
- Surface chapter read errors instead of silently opening empty chapters. `Open` in `wails/internal/book/open.go` read each manifest-referenced chapter with `content, _ := ReadZipEntry(...)`, so a missing, oversized, or corrupt entry became an empty chapter — which a subsequent autosave could then persist over the still-intact source, causing data loss. `Open` now fails with a precise error naming the offending chapter title and file, distinguishing an absent entry from a size/read failure (new `ziputil.ErrEntryNotFound` sentinel + `errors.Is`). Genuinely-optional, rebuildable files (`analysis.json`, `story_bible.json`, beat sheet, etc.) remain optional; `copyright.html` is tolerated when absent but no longer silently dropped on a read error. Added `book_test.go` cases covering a manifest that references a missing entry and one that references an oversized entry.
- Audit ref: master-audit-report-2026-08-30 — Consensus D / Sol REL-001 (surface-chapter-read-errors).

---

## [0.16.02433] - 2026-08-30

### Fixed
- Serialized backend settings writes. `saveSettings` in `wails/frontend/src/store/appStore.ts` fired `SaveSettings(next)` with no ordering, so concurrent callers (sidebar width, `sidebar_active_section`, lane view, `ai_mode`/`ai_model`, `custom_dictionary`) could have their backend writes land out of order and drop a setting (lost update). Each call now merges into in-memory state synchronously and chains the backend `SaveSettings` through a module-scoped `saveChain` promise (mirroring `bookStore`'s save serialization), snapshotting the newest merged state inside each chained write so writes are ordered and no patch is lost. `saveSettings` now returns the chained promise; the previously-floating callers in `ToolsPanel.tsx` and `CharactersView.tsx` explicitly discard it.
- Audit ref: fable5-2026-08-30 — M2 (serialize-savesettings).

---

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
