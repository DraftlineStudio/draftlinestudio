// Frozen manuscripts, as the screen talks about them.
//
// Everything here is a pure function of the publishing record. The text itself
// is never in the frontend: a format record names a snapshot, the catalogue
// says how big it is and when it was frozen, and the words are read out of the
// project file by Go at the moment of an export.
//
// The transforms at the bottom are the only two ways the catalogue ever
// changes, and they are here rather than in the store so that the rule about
// shared snapshots — releasing one format must not take the text another
// format was published from — is a function that can be tested, rather than a
// condition buried in a state update.

import type { Edition, EditionFormat, EditionIndex, EditionSnapshot } from '../../types/draftline'
import { formatTitle } from './editionModel'

// ── Reading the record ─────────────────────────────────────────────────────

/** The frozen manuscript one format was published from, if it has one. */
export function snapshotFor(
  index: EditionIndex | undefined, format: EditionFormat | undefined,
): EditionSnapshot | undefined {
  const id = (format?.snapshot_id ?? '').trim()
  if (!id) return undefined
  return (index?.snapshots ?? []).find(snapshot => snapshot.id === id)
}

/**
 * The other formats published from the same words.
 *
 * This is the reference count made visible. It is why releasing one format's
 * frozen text does not delete the text: the same hash may be the first
 * edition's paperback as well as its ebook, and both went out with it.
 */
export function sharedWith(
  index: EditionIndex | undefined, snapshotID: string, exceptFormatID: string,
): string[] {
  const id = (snapshotID ?? '').trim()
  if (!id || !index) return []
  const names: string[] = []
  for (const edition of index.editions) {
    for (const format of edition.formats) {
      if (format.snapshot_id === id && format.id !== exceptFormatID) {
        names.push(formatTitle(edition, format))
      }
    }
  }
  return names
}

/** Every format published from the given words, itself included. */
export function referenceCount(index: EditionIndex | undefined, snapshotID: string): number {
  const id = (snapshotID ?? '').trim()
  if (!id || !index) return 0
  return index.editions.reduce(
    (total, edition) => total + edition.formats.filter(f => f.snapshot_id === id).length, 0)
}

// ── Saying it on screen ────────────────────────────────────────────────────

export function byteLabel(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return '0 KB'
  if (bytes >= 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  return `${Math.max(1, Math.round(bytes / 1024))} KB`
}

/** A frozen moment as a date the author will recognise, or '' if unreadable. */
export function frozenLabel(frozen: string | undefined): string {
  const stamp = (frozen ?? '').trim()
  if (!stamp) return ''
  const when = new Date(stamp)
  if (Number.isNaN(when.getTime())) return stamp
  return when.toLocaleDateString(undefined, { year: 'numeric', month: 'long', day: 'numeric' })
}

export interface Fact { label: string; value: string }

/** The snapshot row on the format panel: which text this ISBN represents. */
export function snapshotFacts(snapshot: EditionSnapshot | undefined): Fact[] {
  if (!snapshot) return []
  return [
    { label: 'Frozen', value: frozenLabel(snapshot.frozen) || 'unknown' },
    // The frozen title is here because it is the visible edge of the thing an
    // author is most likely to be caught by: the book details were frozen too,
    // so a title corrected since is not the title this edition prints.
    { label: 'Book details', value: snapshot.title ? `“${snapshot.title}”, as they stood then` : 'as they stood then' },
    { label: 'Length', value: `${snapshot.word_count.toLocaleString()} words in ${snapshot.sections} ${snapshot.sections === 1 ? 'section' : 'sections'}` },
    { label: 'In this project file', value: byteLabel(snapshot.bytes) },
    { label: 'Reference', value: snapshot.id.slice(0, 12) },
  ]
}

/** What every frozen manuscript in this book occupies. */
export function snapshotFootprint(index: EditionIndex | undefined): {
  count: number; members: number; bytes: number; label: string
} {
  const snapshots = index?.snapshots ?? []
  const members = snapshots.reduce((total, s) => total + s.members, 0)
  const bytes = snapshots.reduce((total, s) => total + s.bytes, 0)
  const label = snapshots.length === 0
    ? 'No text is frozen in this project yet.'
    : `${snapshots.length} frozen ${snapshots.length === 1 ? 'manuscript' : 'manuscripts'} · ${byteLabel(bytes)} · ${members} ${members === 1 ? 'piece' : 'pieces'} of this project file.`
  return { count: snapshots.length, members, bytes, label }
}

/**
 * What an export of this format will actually put in the file, said plainly
 * enough that nobody has to interpret it.
 *
 * The two sentences are the whole of the feature from the author's side: a
 * format with a frozen manuscript gives you the book as it was, and a format
 * without one gives you the book on screen.
 */
export function exportTextNote(
  index: EditionIndex | undefined, format: EditionFormat | undefined,
): string {
  if (!format) {
    return 'Exports your manuscript as it stands today. Nothing is frozen, because this export is not attached to an edition.'
  }
  const snapshot = snapshotFor(index, format)
  if (!snapshot) {
    return 'No text is frozen for this ISBN yet, so this export uses your manuscript as it stands today — and freezes it once the file is written, so that exporting this edition again after you have carried on writing gives you these same words back.'
  }
  const when = frozenLabel(snapshot.frozen)
  return `Exports the text frozen for this ISBN${when ? ` on ${when}` : ''} — ${snapshot.word_count.toLocaleString()} words — and not the draft you have open. ${SNAPSHOT_DETAILS_NOTE}`
}

/**
 * Which parts of a published edition still follow the record, and which were
 * frozen with the text.
 *
 * This sentence has to be exact, because it is the one an author checks after
 * correcting a misspelled name. The book details — the title, the author, the
 * publisher, the copyright holder — are in the frozen manuscript, so a
 * correction made afterwards does not reach an edition already frozen. The
 * edition's own record is read at export: its ISBN, its cover, the imprint of
 * record and the rights line come out as they stand today.
 */
export const SNAPSHOT_DETAILS_NOTE =
  'The book details went in with it — the title, the author and the publisher printed in this edition are the ones it was frozen with. The cover, the ISBN, the imprint of record and the rights line are read from the edition record each time, so a correction to those does reach the file.'

export const SNAPSHOT_REFREEZE_NOTE =
  'To take up corrected book details, release the frozen text and freeze it again — which also takes up whatever you have written since, so it is a decision about the whole edition rather than about one name.'

export const SNAPSHOT_ABSENT_NOTE =
  'Nothing is frozen for this ISBN. Exporting it will freeze the manuscript as it stands, and from then on this ISBN exports those words however far the book moves on.'

export const SNAPSHOT_SHARED_NOTE =
  'The same words can stand behind several ISBNs. They are stored once, and releasing one of them leaves the others alone.'

export const SNAPSHOT_RELEASE_NOTE =
  'Releasing the frozen text does not change the book, and does not touch a format that shares it. When nothing points at those words any more, the next save drops them and the project file gets smaller.'

// ── Changing the record ────────────────────────────────────────────────────

const withFormats = (
  index: EditionIndex, fn: (edition: Edition, format: EditionFormat) => EditionFormat,
): EditionIndex => ({
  ...index,
  editions: index.editions.map(edition => ({
    ...edition,
    formats: edition.formats.map(format => fn(edition, format)),
  })),
})

/**
 * Stamp one format with the text it was published from, and record the text in
 * the catalogue.
 *
 * Both halves happen together, and they have to: the Go writer refuses an
 * index whose format points at a manuscript the catalogue does not list,
 * precisely so that a half-applied change can never be the thing that decides
 * to drop a published edition's words.
 */
export function withFrozenSnapshot(
  index: EditionIndex, formatID: string, record: EditionSnapshot,
): EditionIndex {
  const snapshots = index.snapshots ?? []
  const already = snapshots.some(snapshot => snapshot.id === record.id)
  return {
    ...withFormats(index, (_edition, format) =>
      (format.id === formatID ? { ...format, snapshot_id: record.id } : format)),
    // An identical text frozen again is the same record; keep the one already
    // held, because it carries the moment this text first went out.
    snapshots: already ? snapshots : [...snapshots, record],
  }
}

/**
 * Let one format go back to exporting the working draft.
 *
 * The catalogue record only goes when nothing else points at it. Anything
 * else would destroy the text behind a published ISBN on the strength of an
 * unrelated format being tidied up.
 */
export function withReleasedSnapshot(index: EditionIndex, formatID: string): EditionIndex {
  const released = (index.editions
    .flatMap(edition => edition.formats)
    .find(format => format.id === formatID)?.snapshot_id ?? '').trim()
  if (!released) return index

  const next = withFormats(index, (_edition, format) => {
    if (format.id !== formatID) return format
    const copy = { ...format }
    delete copy.snapshot_id
    return copy
  })
  const stillUsed = referenceCount(next, released) > 0
  return {
    ...next,
    snapshots: (next.snapshots ?? []).filter(snapshot => stillUsed || snapshot.id !== released),
  }
}

// ── Freezing around an export ──────────────────────────────────────────────
//
// The order here is the whole of the rule, and both halves of it matter.
//
// The manuscript is frozen BEFORE the exporter runs, and the exporter is
// handed the book with the stamp already on it, so the file that comes out is
// rendered from the frozen words rather than from something that merely ought
// to match them.
//
// The publishing record is changed AFTER the file exists. Pressing Export
// opens the system's save dialog inside the exporter, and dismissing that
// dialog is a perfectly ordinary thing to do — an author who changes their
// mind at the file picker has not published anything, and must not come back
// to an ISBN silently bound to the words that happened to be on screen at the
// moment they clicked. Nothing is written to the record, the frozen bytes are
// handed back, and the wizard says so.

/** What FreezeSnapshot returns. */
export interface FreezeOutcome {
  success: boolean
  snapshot?: EditionSnapshot
  reused?: boolean
  error?: string
}

/** What an exporter returns. 'cancelled' is the author dismissing the dialog. */
export interface WriteOutcome {
  success: boolean
  file_path?: string
  error?: string
}

export interface ExportRun<B extends { editions?: EditionIndex }> {
  book: B
  /** The registered format this export is made against, or '' for scratch. */
  formatID: string
  freeze: () => Promise<FreezeOutcome>
  write: (outgoing: B) => Promise<WriteOutcome>
  /** Called only once a file exists. */
  commit: (record: EditionSnapshot) => void
  /** Called when no file was written, to release the frozen bytes. */
  discard: (snapshotID: string) => void
}

export interface ExportRunResult {
  ok: boolean
  cancelled: boolean
  error: string
  filePath: string
  /** What to tell the author about the words, whichever way it went. */
  note: string
}

export const NOTHING_FROZEN_NOTE =
  'No file was written, so nothing was frozen. This ISBN still exports your manuscript as it stands, and the next export of it will freeze the text as it is then.'

const formatRecordIn = (index: EditionIndex | undefined, formatID: string) =>
  (index?.editions ?? []).flatMap(edition => edition.formats).find(format => format.id === formatID)

export async function runExportWithFreeze<B extends { editions?: EditionIndex }>(
  run: ExportRun<B>,
): Promise<ExportRunResult> {
  const index = run.book.editions
  const formatID = (run.formatID ?? '').trim()
  const already = formatID ? snapshotFor(index, formatRecordIn(index, formatID)) : undefined

  const settle = (out: WriteOutcome, note: string): ExportRunResult => {
    if (out.success) return { ok: true, cancelled: false, error: '', filePath: out.file_path || '', note }
    const cancelled = out.error === 'cancelled'
    return {
      ok: false, cancelled, filePath: '', note: '',
      error: cancelled ? '' : (out.error || 'Export failed'),
    }
  }

  // Nothing to freeze: an export from scratch, or one whose ISBN already
  // stands for a text. Both export what they were always going to export.
  if (!index || !formatID || already) {
    let note = ''
    if (already) {
      const when = frozenLabel(already.frozen)
      note = `Exported the text frozen for this ISBN${when ? ` on ${when}` : ''}, not the draft on screen.`
    }
    return settle(await run.write(run.book), note)
  }

  const frozen = await run.freeze()
  if (!frozen.success || !frozen.snapshot) {
    return {
      ok: false, cancelled: false, filePath: '', note: '',
      error: frozen.error || 'The text for this edition could not be frozen.',
    }
  }
  const record = frozen.snapshot
  const editions = withFrozenSnapshot(index, formatID, record)
  const out = await run.write({ ...run.book, editions })
  if (!out.success) {
    run.discard(record.id)
    const result = settle(out, '')
    return { ...result, note: result.cancelled ? NOTHING_FROZEN_NOTE : '' }
  }

  run.commit(record)
  const shared = sharedWith(editions, record.id, formatID)
  return settle(out, frozen.reused && shared.length
    ? `These are the same words already frozen for ${shared.join(' and ')}. Your project file stores them once and both ISBNs point at them.`
    : `Froze ${record.word_count.toLocaleString()} words as the text this ISBN stands for. Exporting it again gives you these words, however far the book moves on.`)
}
