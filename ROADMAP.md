# Draftline Roadmap

This is the short, current product roadmap. Detailed architecture and older
design exploration live in [PlotWalker](docs/roadmaps/PLOTWALKER.md) and the
[Storiverse roadmap](docs/roadmaps/ROADMAP-STORIVERSE.md).

Draftline's default story tools remain local and deterministic. Optional model
plugins may enhance ambiguous interpretation later, but core writing,
retrieval, indexing, review, and refactoring must not depend on generative AI.

## Shipped foundations

- [x] Chapter-scoped find and replace.
- [x] Character detection, entity resolution, aliases, merge/split correction,
      appearances, notable quotes, and confirmed-character relationships.
- [x] Activity-aware on-open/background analysis with visible progress and
      stale/current status.
- [x] Local structure, pacing, readability, dialogue, keyword, and extractive
      summary analysis.
- [x] Resizable bottom Story Search with multi-term scene search, quoted
      phrases, confirmed-character alias expansion, source excerpts, and
      editor navigation (`0.16.02469`).
- [x] Persistent local fact/event evidence index with stable content-derived
      IDs, exact source sentences, chapter/paragraph coordinates, character
      and named-entity links, time expressions, explainable cue strength, and
      an inspectable Evidence Index (`0.16.02470`).

## Next: make evidence editorially useful

- [ ] Author review controls: confirm, reject, edit, annotate, or pin an
      inferred fact/event; preserve decisions through reanalysis.
- [ ] Evidence-backed detail search: first establishment, later references,
      missing given names, singleton named people, discoveries, locations,
      objects, and who knew what when.
- [ ] Automatic timeline draft assembled from confirmed events and explicit
      time expressions, with ambiguity shown instead of silently guessed.
- [ ] Continuity checks over confirmed facts: conflicting attributes,
      impossible knowledge, reintroductions, unexplained singleton characters,
      chronology conflicts, and unresolved plants/payoffs.
- [ ] Full-screen Storyboard for chapter/event density, plot threads, timeline,
      thin chapters, evidence review, and source navigation.
- [ ] Chapter-history provenance where snapshots can prove when a passage
      changed; never invent edit history that predates available snapshots.

## Refactoring engine

### Character refactoring

- [ ] Rename a character across manuscript prose with a previewable change
      set that preserves capitalization, possessives, punctuation, and rich
      text.
- [ ] Understand canonical name, surname, given name, title, nickname, and
      author-confirmed aliases without replacing unrelated words or people.
- [ ] Update structured references by stable character ID: relationships,
      evidence, timeline, beat sheets, notes, and future universe indexes.
- [ ] Review each change, apply atomically, checkpoint chapter history first,
      and make rollback straightforward.

### Storyline and detail refactoring

- [ ] Change a story fact, event, location, object, relationship, or timeline
      detail and build an impact set from the persistent evidence graph.
- [ ] Separate direct textual references, structured references, likely causal
      consequences, and ambiguous passages requiring author judgment.
- [ ] Present affected passages side by side, allow accept/reject/edit per
      passage, and never rewrite the manuscript invisibly.
- [ ] Support deterministic direct-reference changes first; optional local or
      provider plugins may propose repairs for genuinely semantic consequences.

## Plugin expansion

- [ ] Signed/versioned plugin marketplace metadata and safe installation.
- [ ] Optional lightweight analysis packs for coreference, temporal parsing,
      semantic retrieval, contradiction ranking, genre-specific beat systems,
      and cross-book continuity.
- [ ] Clear model size, memory, privacy, license, and network disclosures before
      an optional plugin downloads anything.
