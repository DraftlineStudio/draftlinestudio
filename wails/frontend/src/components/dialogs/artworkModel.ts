// Removing, replacing, or stopping keeping a copy of artwork.
//
// Every one of these can be the moment an author loses a file they cannot make
// again, so none of them happens on a single click and none of them happens
// before Draftline knows whether another copy exists.
//
// The rule that matters is this one: a project may hold the ONLY copy. An
// author ticks "keep a copy inside this Draftline file", and two years later
// the original has been moved, renamed, or lost with the machine it was made
// on. Unticking the box would then delete the artwork. So before anything is
// dropped, the original's recorded location is checked, and if it is not there
// the stored copy is written out first — not offered, done — and the removal
// only follows a file actually existing on disk.
//
// The second rule is smaller and follows the rest of the screen: a record
// marked Published asks for its own name to be typed. It stands for an object
// in the world, and its artwork is what went on it.

export type ArtworkAction = 'remove' | 'replace' | 'unstore'

/** Where the author's own original is, as far as Draftline can tell. */
export type SourceState = 'present' | 'missing' | 'unknown'

export interface ArtworkSubject {
  /** 'the wrap' or 'the cover' — what the sentences call it. */
  noun: string
  /** What has to be typed when this is published: the format or edition name. */
  name: string
  published: boolean
  fileName: string
  sourcePath: string
  /** True when a copy of the bytes is inside the project. */
  stored: boolean
}

export interface ArtworkPrompt {
  title: string
  /** The sentences, in order. Never empty. */
  body: string[]
  /**
   * True when the project holds the only copy anyone knows of. The backup is
   * then not an offer — it is done first, and the action follows it.
   */
  forcedBackup: boolean
  /** True when a backup is worth offering but the original is still there. */
  offerBackup: boolean
  /** True when this record is published and the name has to be typed. */
  typeToConfirm: boolean
  confirmLabel: string
}

const article = (noun: string) => noun.replace(/^the /, '')

export function artworkPrompt(
  subject: ArtworkSubject, action: ArtworkAction, source: SourceState,
): ArtworkPrompt {
  const noun = subject.noun
  const thing = article(noun)
  // The only copy: stored here, and the original is not where it was recorded
  // (or was never recorded at all).
  const onlyCopy = subject.stored && source !== 'present'
  const body: string[] = []

  if (action === 'unstore') {
    body.push(`The copy of ${noun} inside this project will be dropped. The record stays, and an export will read the file from disk instead.`)
    if (onlyCopy) {
      body.push(subject.sourcePath
        ? `Draftline cannot find the original at ${subject.sourcePath}, so this project holds the only copy. It will be saved to a file you choose first, and dropped only once that file exists.`
        : `No original was recorded for ${noun}, so this project holds the only copy. It will be saved to a file you choose first, and dropped only once that file exists.`)
    } else {
      body.push(`The original is still at ${subject.sourcePath}. An export needs it to be there.`)
    }
    return {
      title: `Stop keeping ${noun} in this project?`,
      body,
      forcedBackup: onlyCopy,
      offerBackup: !onlyCopy && subject.stored,
      typeToConfirm: subject.published,
      confirmLabel: onlyCopy ? 'Save a copy, then drop it' : 'Drop the copy',
    }
  }

  if (action === 'replace') {
    body.push(`${cap(noun)} on this record will be replaced by a file you choose next.`)
  } else {
    body.push(`${cap(noun)} will be taken off this record.`)
  }
  if (onlyCopy) {
    body.push(`This project holds the only copy of ${subject.fileName || thing}. It will be saved to a file you choose first, and ${action === 'replace' ? 'replaced' : 'removed'} only once that file exists.`)
  } else if (subject.stored) {
    body.push(`The original is still at ${subject.sourcePath}, so nothing is lost. Save a copy of the stored file as well if you would rather be sure.`)
  } else if (subject.sourcePath) {
    body.push(`Only the record goes. Your file at ${subject.sourcePath} is not touched.`)
  } else {
    body.push('Only the record goes. No file on disk is touched.')
  }

  return {
    title: action === 'replace' ? `Replace ${noun}?` : `Remove ${noun}?`,
    body,
    forcedBackup: onlyCopy,
    offerBackup: !onlyCopy && subject.stored,
    typeToConfirm: subject.published,
    confirmLabel: action === 'replace'
      ? (onlyCopy ? 'Save a copy, then replace' : 'Choose a new file')
      : (onlyCopy ? 'Save a copy, then remove' : `Remove ${noun}`),
  }
}

/** How many steps the confirmation takes: two for a published record. */
export function artworkSteps(prompt: ArtworkPrompt | null): number {
  return prompt?.typeToConfirm ? 2 : 1
}

/** Case and surrounding space forgiven; nothing else. Same rule as the rest. */
export function artworkConfirmed(name: string, typed: string): boolean {
  const wanted = (name ?? '').trim()
  if (!wanted) return false
  return typed.trim().toLowerCase() === wanted.toLowerCase()
}

/** What CheckCoverSource's answer means here. */
export function sourceStateOf(status: string | undefined, path: string): SourceState {
  if (!path.trim()) return 'missing'
  const word = (status ?? '').trim().toLowerCase()
  if (!word) return 'unknown'
  // 'changed' means the file is there but is not the one that was attached.
  // It is still a file on disk, which is what this decision turns on.
  return word === 'moved' ? 'missing' : 'present'
}

function cap(text: string): string {
  return text.charAt(0).toUpperCase() + text.slice(1)
}
