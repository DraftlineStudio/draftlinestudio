import { describe, expect, it } from 'vitest'
import { failureMessage } from '../backendError'

describe('reading a rejected backend call', () => {
  it('takes the message off an Error, which is what Wails rejects with', () => {
    expect(failureMessage(new Error('archive is not a Draftline project'))).toBe(
      'archive is not a Draftline project',
    )
  })

  it('accepts a bare string, and an object carrying message or error', () => {
    expect(failureMessage('that file could not be opened')).toBe('that file could not be opened')
    expect(failureMessage({ message: 'cover art is not an image' })).toBe('cover art is not an image')
    expect(failureMessage({ error: 'the export folder is read only' })).toBe(
      'the export folder is read only',
    )
  })

  // Never show the writer "undefined" or an empty notice. A message that says
  // nothing is the same dead end as no message, which is the bug being fixed.
  it('always returns something readable, however empty the rejection', () => {
    for (const nothing of [undefined, null, '', '   ', new Error(''), {}, { message: 42 }, 0]) {
      expect(failureMessage(nothing)).toBe('The operation failed and gave no reason.')
    }
  })

  it('trims, so a Go error ending in a newline does not stretch the notice', () => {
    expect(failureMessage(new Error('  could not write the snapshot\n'))).toBe(
      'could not write the snapshot',
    )
  })
})
