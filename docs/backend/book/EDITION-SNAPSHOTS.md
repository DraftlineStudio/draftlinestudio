# Frozen manuscripts

An edition record says which ISBN a book was sold under. It does not, on its
own, say which words went out under it — and those are not the same thing. An
author who publishes a first edition, spends a year revising, and then re-exports
the first edition for a new retailer would otherwise ship the second edition's
text under the first edition's number.

A snapshot is the manuscript as it stood at one moment: every section's HTML,
the order and titles the manifest gave them, the author's own copyright page,
and the metadata. Anything less and "as it was" is not true.

Code: `wails/internal/book/snapshot.go` (build, decode, retention),
`wails/snapshot.go` (the cache, the freeze binding, the export resolution),
`wails/internal/types/snapshot.go` (the record),
`wails/frontend/src/components/dialogs/SnapshotCard.tsx` and `snapshotModel.ts`
(the panel and the transforms).

## Where the bytes are

| Where | What |
|---|---|
| `editions/snapshots/<sha256>/index.json` | version, metadata as it stood, section order and titles |
| `editions/snapshots/<sha256>/copyright.html` | the author's own copyright page, when the book had one |
| `editions/snapshots/<sha256>/NNN.html` | section text, numbered across front matter, body and back matter in manuscript order |
| `editions/index.json` → `snapshots[]` | the catalogue: when frozen, word count, section count, members, bytes |
| `editions/index.json` → `editions[].formats[].snapshot_id` | which text this ISBN stands for |

Members are written stored rather than deflated, through the byte path added in
02645, and survive a save by the `editions/` entry in
`preservedArchivePrefixes`. `index.json` inside a snapshot folder carries its
own version and is refused by `requireArchiveVersion` if a later Draftline
wrote it — the same hard gate `planner.json` and `editions/index.json` apply.

## Content addressing

The identifier is the SHA-256 of the frozen content: the snapshot's own
`index.json` and every section body, each length-prefixed so that no
rearrangement of the same bytes across different members can collide.

That is why the same unchanged manuscript exported as a paperback and as an
ebook is stored **once** and referenced twice. Per-edition folders could not do
that, and a sixty-section novel frozen for six formats is three hundred and
sixty archive members that did not need to exist.

Two things are deliberately **not** in the hash:

- the moment of freezing, which lives only in the catalogue record;
- `metadata.modified`, which records when the project file was last written.
  If it counted, exporting twice in one afternoon would freeze two copies of an
  identical book.

The word count in the snapshot is recomputed from the frozen text rather than
copied from `metadata.word_count`, so the record describes the words it holds.

## Retention

Retention is by reference count, held in `editions/index.json` and never
inferred from the archive. `EditionIndex.SnapshotReferences` counts the formats
whose `snapshot_id` names a hash.

- A catalogue record nothing points at is dropped at the next save, and its
  bytes are simply not carried across. Otherwise a novel's worth of HTML that
  no ISBN refers to would be copied into the project file and into every
  rolling backup on every autosave, for the life of the book.
- A format that still points at a record the catalogue has lost **fails the
  save, by name**. The alternative is dropping the text a published ISBN stands
  for on the strength of a bookkeeping mistake.

Both halves live in `prepareSnapshotCatalogue`. On the frontend the same rule
is `withReleasedSnapshot`: releasing one format never removes a record another
format still points at.

`editions/snapshots/` is exempt from the edition reaper (`editionFolder`
returns nothing for it), because frozen manuscripts are shared between editions
by design and outlive any one of them. `snapshots` is therefore a reserved
edition identifier and `prepareEditionsData` refuses it.

## Budget

Frozen manuscripts are the first thing Draftline has written that can approach
`ziputil.MaxEntries` (10,000) and `ziputil.MaxTotalSize` (500 MB). The archive
writer counts what goes in, attributes the share belonging to
`editions/snapshots/`, and refuses a save that would exceed either with a
message naming the frozen manuscripts and saying where to release one — rather
than the generic limit error on a file that opened yesterday. The format panel
reports the same footprint while the project is still comfortable.

`backup.Create` copies the whole project file before every save. It streams the
copy (`fsutil.CopyFileAtomic`) rather than reading it into memory first, which
mattered much less when a `.draftline` was a few hundred kilobytes of HTML.

## Freezing and exporting

Freezing happens **at export**, not at publish, plus an explicit *Freeze now*
on the format panel.

1. The wizard calls `FreezeSnapshot(book, formatID)`. Go builds the snapshot,
   keeps the bytes in `snapshotCache` — they never cross the bridge — and
   returns the catalogue record.
2. The store applies `withFrozenSnapshot`, which stamps the format and
   catalogues the record in one change. The ordinary five-second autosave
   writes the words.
3. The export is handed the book with that stamp already on it, so the file and
   the record cannot disagree even though the save is still seconds away.
4. `App.exportSource` resolves the book every exporter actually renders: a
   format with a `snapshot_id` is rebuilt from the frozen members through
   `DecodeSnapshot`, keeping the *live* publishing record, file path and story
   bible; a format without one exports the working draft.

Reconstruction hands a plain `types.BookData` to the same exporter. Every
exporter already takes one, so this is a different book down one path, not a
second export path to keep in step.

An unreadable frozen manuscript is an **error**, not a quiet fallback. Handing
over today's words under a published ISBN, having told the author they were
getting the book as it was, is the one outcome worth refusing an export for.

## Not in scope

Diffing a snapshot against the working draft, restoring one into the editor,
per-chapter snapshot browsing (chapter history already does that), and any
pruning policy beyond reference counts.
