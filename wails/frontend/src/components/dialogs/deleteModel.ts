// Deleting an edition or one of its formats.
//
// Nothing on the Book & Editions screen is deleted on a single click, and a
// published record is not deleted on a single confirmation either. The rule is
// the one the release flow already follows: the amount of friction matches how
// hard the thing is to get back.
//
//   anything      one confirmation, listing what goes with it. An edition
//                 takes its formats, its ISBNs and its cover with it, and none
//                 of that is obvious from a button that says Remove.
//   published     a second step that asks for the record's own name, typed.
//                 A published record stands for an object that exists in the
//                 world under a number that cannot be reused; deleting it is
//                 not the same class of mistake as deleting a draft.
//
// The counts and the losses are computed here rather than written into the
// dialog so that what the author is told is what the record actually holds.

import type { Edition, EditionFormat, EditionIndex } from '../../types/draftline'
import { referenceCount, snapshotFor } from './snapshotModel'

export interface Deletion {
  kind: 'edition' | 'format'
  /** What has to be typed on the second step: the record's own name. */
  name: string
  /** The heading: "Delete the paperback of the first edition?" */
  title: string
  published: boolean
  /** What goes with it, itemised. Never empty. */
  losses: string[]
  /** A last line, when there is something reassuring and true to say. */
  note: string
}

const isPublished = (status: string | undefined): boolean =>
  (status ?? '').trim().toLowerCase() === 'published'

export function editionDeletion(index: EditionIndex, edition: Edition): Deletion {
  const name = (edition.label ?? '').trim() || 'this edition'
  const formats = edition.formats
  const numbered = formats.filter(one => (one.isbn13 ?? '').trim())
  const losses: string[] = []

  losses.push(formats.length
    ? `${formats.length} ${formats.length === 1 ? 'format' : 'formats'}: ${formats.map(one => (one.format ?? '').trim() || 'a format').join(', ')}.`
    : 'The edition record itself. It has no formats yet.')
  if (numbered.length) {
    losses.push(`${numbered.length} ${numbered.length === 1 ? 'ISBN' : 'ISBNs'} and the specification each was registered with.`)
  }
  if (edition.cover) losses.push('The cover artwork attached to this edition.')
  const wraps = formats.filter(one => one.wrap).length
  if (wraps) losses.push(`${wraps} print-ready ${wraps === 1 ? 'wrap' : 'wraps'} kept with ${wraps === 1 ? 'its format' : 'their formats'}.`)

  return {
    kind: 'edition',
    name,
    title: `Delete ${name}?`,
    published: isPublished(edition.status),
    losses,
    note: snapshotNote(index, edition.formats),
  }
}

export function formatDeletion(index: EditionIndex, edition: Edition, format: EditionFormat): Deletion {
  const word = (format.format ?? '').trim() || 'this format'
  const label = (edition.label ?? '').trim()
  const isbn = (format.isbn13 ?? '').trim()
  const losses: string[] = [
    isbn
      ? `Its ISBN, ${isbn}, and the specification registered with it.`
      : 'Its export template and specification. It carries no ISBN.',
  ]
  if (format.wrap) losses.push(`The print-ready wrap attached to it: ${format.wrap.file_name}.`)
  losses.push(label ? `Nothing else on ${label} is touched.` : 'Nothing else on this edition is touched.')

  return {
    kind: 'format',
    name: word,
    title: label ? `Delete the ${word.toLowerCase()} of ${label}?` : `Delete ${word}?`,
    published: isPublished(format.status),
    losses,
    note: snapshotNote(index, [format]),
  }
}

// What happens to the locked manuscript. It is the one thing an author is most
// likely to fear losing and most likely to be wrong about: the words are
// stored once and survive as long as anything still stands on them.
function snapshotNote(index: EditionIndex, formats: EditionFormat[]): string {
  const ids = new Set(formats.map(one => (one.snapshot_id ?? '').trim()).filter(Boolean))
  if (!ids.size) return ''
  for (const id of ids) {
    const held = referenceCount(index, id)
    const going = formats.filter(one => one.snapshot_id === id).length
    if (held > going) {
      return 'The locked manuscript stays: another format is still published from those same words.'
    }
  }
  const one = snapshotFor(index, formats.find(f => f.snapshot_id))
  const words = one ? `${one.word_count.toLocaleString()} words` : 'the locked text'
  return `The locked manuscript goes with it. Nothing else stands on ${words}, so the next save drops them from the project file.`
}

/** One step for a draft, two for something published. */
export function deleteSteps(deletion: Deletion | null): number {
  return deletion?.published ? 2 : 1
}

/**
 * Whether what was typed releases the delete.
 *
 * Case and surrounding space are forgiven; nothing else is. It is a function
 * so that the rule guarding an irreversible act can be tested, and it is the
 * same rule the snapshot release uses.
 */
export function deleteConfirmed(deletion: Deletion | null, typed: string): boolean {
  const name = (deletion?.name ?? '').trim()
  if (!name) return false
  return typed.trim().toLowerCase() === name.toLowerCase()
}

export function deleteButtonLabel(deletion: Deletion | null, step: number): string {
  if (deleteSteps(deletion) > step) return 'Yes, continue'
  return deletion?.kind === 'edition' ? 'Delete edition' : 'Delete format'
}
