# Character Detection and Resolution

Draftline's character Codex is built locally. Prose v3 supplies statistical named-entity evidence, while deterministic rules control candidate admission, identity resolution, canonical names, and review status. No manuscript text leaves the process.

## Pipeline

1. **Select narrative material.** Covers, contents, copyright, acknowledgments, glossaries, indexes, navigation, and headings are excluded. The combined chapter index remains stable so heatmaps and mention navigation still address the correct section.
2. **Extract mention candidates.** Capitalization, honorifics, possessives, sentence position, local false-positive contexts, and Prose labels contribute evidence. Common sentence grammar is trimmed from candidate edges.
3. **Resolve identity conservatively.** Exact names, ordered first/surname components, title compatibility, and full-name nickname/typo rules can join mentions. Ambiguous multi-entity matches remain separate; a new mention cannot transitively glue existing people together.
4. **Choose a canonical name.** Repeated body-prose forms outrank one-off, plural, repeated-token, heading-like, or grammar-contaminated forms.
5. **Classify and triage.** Candidates receive an entity kind, evidence score, and status. High-confidence people enter Characters; uncertain people and story actors enter Needs Review; obvious noise remains in persisted analysis without becoming a Codex card.

## Entity Kinds and Statuses

Kinds currently include `person`, `group`, `organization`, `place`, `object`, and `unknown`. A ship, command structure, or institution may be a meaningful story actor even when it is not a person, so those candidates are reviewable rather than blindly discarded.

Statuses are:

- `accepted` - strong enough for the normal Characters view or explicitly confirmed by the writer.
- `review` - plausible but ambiguous; shown in Needs Review.
- `rejected` - obvious low-value noise; retained only in analysis for diagnostics.

Confirming a review candidate is author curation and survives re-detection.
Marking an auto-detected card as **Not a character** stores a rejection rule and removes the card. Decisions use the canonical name plus known aliases instead of volatile entity IDs, and are re-applied only when they identify one unambiguous entity. Clearing all character analysis also clears these decisions and provides a clean reset.

The **Same as...** correction opens a searchable merge dialog across detected names and aliases. The selected target is treated as the correct character: its display name is retained, the duplicate becomes an alias, and the resulting name-based merge rule is remembered on future detection runs.

## Archive Persistence

Archive format 2.1 stores derived analysis in `analysis.json` separately from `story_bible.json`. Entity mentions, evidence, relationships, events, merge rules, and split rules therefore survive close/reopen. A legacy archive with `is_indexed=true` but no analysis is marked for re-detection instead of presenting stale controls.

Analysis is rebuildable. Author-curated roles, descriptions, appearance, personality, motivation, notes, and confirmed/rejected character decisions are preserved when a re-detected entity matches uniquely by name or alias.

## Evaluation Corpus

Real manuscripts may be kept locally as ignored `.draftline` fixtures. They are evaluation material, not training data and must not be committed or redistributed. Record aggregate results and create small synthetic regression cases for each failure mode.

Track at minimum:

- Main/supporting-character recall
- Default-Codex precision
- False cards per 10,000 words
- Merge purity
- Canonical-name accuracy
- Needs Review volume
- Chapter-map correctness
- Runtime and peak memory

The current regression suite covers publishing-page contamination, headings, multi-chapter EPUB spine files, invented names, common-word bridges, bare-word fuzzy collisions, malformed canonical names, and curation persistence.
