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

## Story Fingerprint and Evidence Archive

- [ ] Keep the complete raw evidence browser as an **Evidence Archive**. It is
      intentionally not a primary writing surface; a small archive-folder
      action in Detail Search, Timeline, Continuity, Storyboard, and other
      derived views opens everything Draftline's heuristics know about the
      book, including mundane and low-confidence records.
- [ ] Derive a versioned **Story Fingerprint** from the manuscript's reviewed
      entities, aliases, relationships, appearances, interactions, notable
      quotes, facts, events, explicit times, character knowledge, locations,
      objects, recurring concepts, plot threads, pacing, and structural
      metrics. Every conclusion remains traceable to source evidence.
- [ ] Present the useful fingerprint through ranked editorial views while
      retaining the full archive for transparency, debugging, author review,
      and the possibility that an apparently mundane detail matters later.
- [ ] Track fingerprint schema, analysis-engine version, source content hash,
      generation time, and review provenance so Draftline can distinguish
      current, stale, partially upgraded, and source-unavailable fingerprints.
- [ ] Treat fingerprints as local derived intellectual property: portable and
      user-owned, never uploaded or submitted to AI without explicit action.

## Storiverse: connect book fingerprints

- [ ] Storiverse is a universe context inside the Draftline editor, not an
      alternative to it. Opening a universe adds **Manuscript** and
      **Storiverse** tabs to the manuscript/navigation pane while retaining the
      same editor, commands, history, spellcheck, and analysis tools.
- [ ] The Storiverse tab begins with the linked book titles. Clicking an
      available title opens that original `.draftline` in the editor and
      switches to Manuscript, while the universe remains loaded around it.
- [ ] If a source is unavailable, open its fingerprint overview with a clear
      Locate/Relink action rather than pretending the manuscript is editable.
- [ ] A Storiverse is a normal portable folder containing its manifest and a
      hidden `.storiverse/` data directory. Source manuscripts may be dropped
      beside it or selected from anywhere, but are inputs—not files Storiverse
      secretly copies into its own storage.
- [ ] Adding a `.draftline` indexes it in place. Importing DOCX/EPUB creates one
      standalone `.draftline` at an author-chosen location, then indexes that
      source; Storiverse retains the fingerprint, never another manuscript.
- [ ] Treat an added book as a live link to its original `.draftline`. Saving
      that book in Draftline schedules its Storiverse fingerprint refresh;
      opening or reindexing the universe also compares source hashes so edits
      made outside universe context cannot leave a silently stale fingerprint.
- [ ] Mark the active fingerprint and its dependent universe views stale as
      soon as manuscript content changes. After the existing idle delay,
      reanalyze only the changed book, atomically refresh its fingerprint, then
      recompute only affected cross-book joins. Never block typing or rescan
      every book for one chapter edit.
- [ ] Use analysis revision tokens so a result produced from older text cannot
      overwrite a newer edit; discard it and queue the current revision.
- [ ] Store an independent snapshot of every book's Story Fingerprint under
      `.storiverse/fingerprints/`. The universe can therefore retain character,
      timeline, continuity, and knowledge context when a linked manuscript is
      offline, moved, deliberately omitted from a shared copy, or temporarily
      unavailable.
- [ ] Join stable identities across fingerprints without destroying book-local
      identity: one universe character may map to different book entity IDs,
      aliases, roles, ages, and knowledge states over time.
- [ ] Build cross-book continuity, chronology, relationship evolution,
      knowledge tracking, recurring details, universe rules, and series-wide
      search from fingerprints rather than repeatedly opening every manuscript.
- [ ] Give every `.draftline` a stable book ID. Dropping an updated copy into a
      Storiverse and choosing reindex must find the existing fingerprint by ID,
      compare its content hash, refresh it atomically, preserve compatible
      author decisions, and show conflicts instead of creating a duplicate
      book or silently replacing universe-level corrections.
- [ ] Keep absolute source locations in per-device local app state, not inside
      the portable universe. On another device, dragging any copy with the same
      book ID relinks it without importing a duplicate or losing the cached
      fingerprint.
- [ ] A fingerprint is a continuity index, not a manuscript substitute: retain
      only the evidence excerpts and coordinates required to justify its
      conclusions, and degrade source navigation honestly when the book is
      unavailable.
- [ ] Exporting or copying a Storiverse includes fingerprints by default—not
      the source manuscripts—so it remains small and portable. Authors may
      explicitly choose to bundle sources as a separate export operation.

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
