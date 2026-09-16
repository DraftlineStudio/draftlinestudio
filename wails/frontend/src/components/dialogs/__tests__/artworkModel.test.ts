import { describe, expect, it } from 'vitest'
import {
  artworkConfirmed, artworkPrompt, artworkSteps, sourceStateOf, type ArtworkSubject,
} from '../artworkModel'

const wrap = (patch: Partial<ArtworkSubject> = {}): ArtworkSubject => ({
  noun: 'the wrap',
  name: 'Paperback',
  published: false,
  fileName: 'reset-wrap.tif',
  sourcePath: 'D:/Art/reset-wrap.tif',
  stored: true,
  ...patch,
})

describe('when the project holds the only copy', () => {
  // The whole point. An author ticks "keep a copy", and two years later the
  // original is gone with the machine it was made on. Unticking must not be
  // the moment the artwork stops existing.
  it('saves the file out first, and does not offer that as a choice', () => {
    const prompt = artworkPrompt(wrap(), 'unstore', 'missing')
    expect(prompt.forcedBackup).toBe(true)
    expect(prompt.offerBackup).toBe(false)
    expect(prompt.confirmLabel).toBe('Save a copy, then drop it')
    expect(prompt.body.join(' ')).toContain('only copy')
    expect(prompt.body.join(' ')).toContain('D:/Art/reset-wrap.tif')
  })

  it('says so for a removal and a replacement too', () => {
    for (const action of ['remove', 'replace'] as const) {
      const prompt = artworkPrompt(wrap(), action, 'missing')
      expect(prompt.forcedBackup).toBe(true)
      expect(prompt.body.join(' ')).toContain('only copy')
    }
  })

  it('treats artwork that never recorded an original as the only copy', () => {
    const prompt = artworkPrompt(wrap({ sourcePath: '' }), 'unstore', 'missing')
    expect(prompt.forcedBackup).toBe(true)
    expect(prompt.body.join(' ')).toContain('No original was recorded')
  })
})

describe('when the original is still on disk', () => {
  it('offers a backup rather than forcing one, and says where the file is', () => {
    const prompt = artworkPrompt(wrap(), 'unstore', 'present')
    expect(prompt.forcedBackup).toBe(false)
    expect(prompt.offerBackup).toBe(true)
    expect(prompt.confirmLabel).toBe('Drop the copy')
    expect(prompt.body.join(' ')).toContain('still at D:/Art/reset-wrap.tif')
  })

  it('offers nothing to back up when nothing is stored', () => {
    const prompt = artworkPrompt(wrap({ stored: false }), 'remove', 'present')
    expect(prompt.forcedBackup).toBe(false)
    expect(prompt.offerBackup).toBe(false)
    expect(prompt.body.join(' ')).toContain('not touched')
  })
})

describe('a published record is asked twice', () => {
  it('takes two steps and the record’s own name', () => {
    const draft = artworkPrompt(wrap(), 'remove', 'present')
    expect(artworkSteps(draft)).toBe(1)

    const live = artworkPrompt(wrap({ published: true }), 'remove', 'present')
    expect(artworkSteps(live)).toBe(2)
    expect(live.typeToConfirm).toBe(true)

    expect(artworkConfirmed('Paperback', 'paperback')).toBe(true)
    expect(artworkConfirmed('Paperback', '  PAPERBACK  ')).toBe(true)
    expect(artworkConfirmed('Paperback', 'yes')).toBe(false)
    expect(artworkConfirmed('', '')).toBe(false)
  })
})

describe('reading the check on the original', () => {
  it('calls a moved file missing and a changed one present', () => {
    // 'changed' means the file is there and is not the one attached. It is
    // still a file on disk, which is what this decision turns on.
    expect(sourceStateOf('present', 'D:/a.tif')).toBe('present')
    expect(sourceStateOf('changed', 'D:/a.tif')).toBe('present')
    expect(sourceStateOf('moved', 'D:/a.tif')).toBe('missing')
    expect(sourceStateOf('present', '')).toBe('missing')
    expect(sourceStateOf(undefined, 'D:/a.tif')).toBe('unknown')
  })

  // An unknown answer is not a reassuring one: a check that could not run is
  // treated the same as a file that is not there.
  it('errs towards keeping a copy when the check could not run', () => {
    expect(artworkPrompt(wrap(), 'unstore', 'unknown').forcedBackup).toBe(true)
  })
})

describe('the wording names the thing, not the feature', () => {
  it('says cover for a cover and wrap for a wrap', () => {
    const cover = artworkPrompt(
      { ...wrap(), noun: 'the cover', name: 'First edition', fileName: 'cover.jpg' },
      'replace', 'present',
    )
    expect(cover.title).toBe('Replace the cover?')
    expect(artworkPrompt(wrap(), 'remove', 'present').title).toBe('Remove the wrap?')
    expect(artworkPrompt(wrap(), 'unstore', 'present').title)
      .toBe('Stop keeping the wrap in this project?')
  })
})
