# Narrative Fingerprint Pipeline

Draftline's local story model has four deliberately separate layers:

```text
manuscript prose
  -> evidence atoms
  -> semantic assertions and state
  -> promoted narrative fingerprints
  -> future structure, chronology, thread, and continuity analysis
```

Evidence atoms favor recall. They retain exact source text, chapter and paragraph coordinates, detected participants, named terms, actions, time expressions, and confidence. They are useful even when narratively mundane.

Assertions interpret that evidence without claiming it is important. They preserve attribution, epistemic status, reality scope, persistence, state changes, temporal information, and stable evidence IDs. Repeated equivalent assertions collect multiple evidence sources.

Narrative fingerprints favor precision. A fingerprint exists only with an inspectable promotion reason: a durable state change, goal or commitment, consequential knowledge change or transfer, relationship change, explicit deception, contradiction, correction, corroboration, fulfillment, or a later dependency that makes earlier evidence significant. Routine movement and incidental activity remain available in evidence and assertions without becoming fingerprints.

Schema 4 intentionally does not produce narrative threads, scenes, sequences, arcs, or Story Map structure. Those higher layers may be rebuilt only after textual fingerprint diagnostics demonstrate trustworthy semantics across unrelated manuscripts.

## Epistemic safety

Character speech remains an attributed claim. Belief and inference remain attached to the believing or inferring character. Dreams, memories, flashbacks, hypotheticals, stories-within-stories, simulations, and uncertain contexts retain distinct scopes. Contradiction and supersession are explicit relationships rather than silent rewrites of prior evidence.

## Evidence and identity

Every promoted fingerprint contains all supporting evidence IDs and complete source spans. Stable IDs derive from semantic content, scope, epistemic posture, and attribution rather than raw offsets. Later evidence may promote an earlier assertion retrospectively, but it does not alter or delete the source atom.

The formatted diagnostic is generated in memory and is deliberately excluded from `analysis.json`. Persisting it would duplicate every supporting quotation already held by the evidence layer and unnecessarily inflate the archive.

## Textual quality gate

The Wails binding `GetNarrativeFingerprintDiagnostic` builds the report locally from an analyzed book's persisted evidence. It performs no filesystem write and invokes no external model.

Maintainers can inspect archives without saving or modifying them:

```powershell
cd wails
go run ./cmd/fingerprintdiag -summary "path\to\manuscript.draftline"
go run ./cmd/fingerprintdiag "path\to\manuscript.draftline"
```

The full report lists every promoted fingerprint, its epistemic status, attribution, reality scope, persistence, state transition, promotion reasons, exact evidence quotations, and causal, contradiction, supersession, corroboration, setup, or fulfillment relationships.

## Current conservative assumptions

- Uncertain significance remains evidence rather than becoming a fingerprint.
- Relative time phrases provide temporal evidence but do not by themselves relocate a scene.
- Claims in incompatible reality scopes are not treated as contradictions.
- A cross-scope setup link may exist—for example, remembered or dreamed information later used in current reality—without asserting that the remembered or dreamed occurrence physically happened.
- No production rule contains manuscript titles, character names, supplied synopses, benchmark counts, or literal benchmark passages.
