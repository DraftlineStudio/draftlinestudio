# Manuscript Memory and Narrative Development

Draftline's deterministic story model separates memory from significance:

```text
manuscript prose
  -> evidence atoms
  -> normalized assertions
  -> manuscript fingerprint corpus
     -> state histories and continuity inspections
     -> contextual NarrativeDevelopments
        -> future roadmap, thread, arc, and graph projections
```

Evidence extraction favors recall. Each atom retains exact source text and coordinates, detected participants, named terms, actions, time expressions, confidence, and author-review provenance.

A manuscript fingerprint is normalized memory, not a story beat. Routine locations, custody, physical condition, beliefs, claims, transient actions, descriptions, and other continuity-useful details remain in the corpus even when they never affect the plot. Equivalent assertions collect all supporting passages. No target count or narrative-importance threshold limits this layer.

State histories independently track how properties change over manuscript and story time. Possession, physical custody, location, knowledge, belief, relationship, identity, condition, goals, and commitments can therefore support continuity inspection without becoming roadmap nodes.

A `NarrativeDevelopment` is a contextual change in the course of the story. It may synthesize several fingerprints across a local window or connect distant setup and consequence. Current deterministic development kinds include objectives, obligations, relationships, persistent state, consequential knowledge, obstacles, threats, enablement, fulfillment, corroboration, contradiction, and supersession. A concrete finding can gain retrospective significance when later manuscript knowledge depends on it. Action or grammatical eventhood alone never makes a development.

Roadmap, thread, arc, and graph generation remains disabled for schema 5. Those projections must eventually consume `NarrativeDevelopment`; they must never treat the fingerprint corpus as a node list.

## Epistemic and reality safety

Character speech remains an attributed claim. Belief, suspicion, inference, and identified deception remain attached to their source. Dreams, memories, flashbacks, hypotheticals, stories-within-stories, simulations, and uncertain contexts retain distinct scopes. Conflicting accounts remain separate properties on a likely shared event identity rather than being averaged into false certainty.

Every fingerprint and development retains the evidence IDs and exact manuscript spans that support it. Stable IDs derive from semantic content, scope, epistemic posture, and attribution rather than raw offsets. Author corrections remain separate from prose; evidence-dependent corrections become orphaned when their source no longer exists.

## Three textual diagnostics

The Wails binding `GetFingerprintTextDiagnostics` returns three independently selectable reports without filesystem writes or external models:

- `corpus`: all normalized manuscript memory, state histories, same-event identities, relationships, epistemic status, scope, and source quotations.
- `developments`: contextual changes, before/after state, affected concern, dependencies, reasons, and complete supporting evidence.
- `inspections`: concrete contradictions, supersessions, conflicting event accounts, stable-state conflicts, and open obligations with evidence from every side.

Maintainers can print the same reports from an analyzed `.draftline` archive:

```powershell
cd wails
go run ./cmd/fingerprintdiag -summary "path\to\manuscript.draftline"
go run ./cmd/fingerprintdiag -report corpus "path\to\manuscript.draftline"
go run ./cmd/fingerprintdiag -report developments "path\to\manuscript.draftline"
go run ./cmd/fingerprintdiag -report inspections "path\to\manuscript.draftline"
go run ./cmd/fingerprintdiag -report all "path\to\manuscript.draftline"
```

Reports are regenerated from the canonical evidence in `analysis.json`; formatted reports are never persisted back into the archive.

## Conservative assumptions

- Ambiguous significance remains corpus memory rather than an invented development.
- Relative time phrases provide temporal evidence but do not by themselves relocate a containing scene.
- Claims in incompatible reality scopes are not treated as ordinary continuity contradictions.
- A cross-scope dependency may exist without asserting that a dream, memory, or hypothetical physically occurred.
- Same-event identity is provisional when participants, event type, time, consequence, and context do not provide enough agreement.
- No production rule contains manuscript titles, character names, supplied synopses, benchmark counts, or benchmark passages.
