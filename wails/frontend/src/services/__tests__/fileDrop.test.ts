import { describe, expect, it } from 'vitest'
import { firstProjectPath } from '../fileDrop'

describe('recognising a dropped project', () => {
  it('finds the project among other files, whatever the case of the extension', () => {
    expect(firstProjectPath(['C:/books/The Lantern.draftline'])).toBe('C:/books/The Lantern.draftline')
    expect(firstProjectPath(['C:/books/notes.txt', 'C:/books/x.DRAFTLINE'])).toBe('C:/books/x.DRAFTLINE')
  })

  it('declines a drop with no project in it, so another handler can claim it', () => {
    expect(firstProjectPath([])).toBeNull()
    expect(firstProjectPath(['C:/art/cover.jpg', 'C:/art/wrap.pdf'])).toBeNull()
  })

  // A name that merely contains the word, or ends in something else, is not a
  // project. Claiming it would fail inside the archive reader instead of being
  // declined here where another handler could still take it.
  it('is not fooled by a name that only contains the extension', () => {
    expect(firstProjectPath(['C:/books/draftline-backup.zip'])).toBeNull()
    expect(firstProjectPath(['C:/books/The Lantern.draftline.bak'])).toBeNull()
  })

  it('takes the first project when several are dropped at once', () => {
    expect(firstProjectPath(['C:/a.draftline', 'C:/b.draftline'])).toBe('C:/a.draftline')
  })
})
