# Story Structure backend

Story Structure is a rebuildable semantic projection over Draftline's lossless
Story Fingerprint. Atomic evidence is never deleted because it is mundane or
excluded from an author-facing view.

## Current gate: significant events

Version 2 has completed only its significant-event gate. A significant event is
an explainable narrative occurrence with a nucleus such as an action,
discovery, decision, transition, interaction, obligation, contradiction, or
persistent state change. Nearby descriptive observations may be attached as
supporting evidence. Incidental descriptions and standalone temporal mentions
remain in the fingerprint without being mislabeled as significant events.

Each occurrence retains its source evidence IDs, quotations, coordinates,
confidence, review state, fingerprint corrections, author decisions, temporal
placement reason, salience components, creation reason, membership reasons, and
rejected boundary decisions.

Author decisions are stored separately from prose. An author may confirm,
correct, reject as irrelevant, leave unresolved, or mark an interpretation as
intentionally ambiguous. Decisions contain source-block dependency snapshots.
If later prose changes or disappears, reanalysis marks the decision conflicted
or orphaned and emits a review diagnostic rather than silently applying it.

## Pending gates

Semantic scenes, sequences, and narrative threads still use the provisional v1
algorithms and must not yet be treated as trustworthy literary analysis. Each
will be revised and validated in order. Arithmetic StoryArc partitions are not
story-arc inference and will be removed from author-facing semantic output.

Fixture acceptance is relationship-based. Tests ask whether named pieces of
evidence group, remain separate, retain their source, or produce a specific
uncertainty reason. They never require a preferred number of events, scenes,
sequences, threads, or arcs.
