// Showing a passage is one job with one implementation: leave whatever
// workspace is open, go to the chapter, then ask the find bar to locate the
// text there. Ask Draftline and the character sidebar both go through this, so
// a jump that stops working stops working in one place, not two.
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { goToManuscript, findInManuscript, FIND_IN_MANUSCRIPT_EVENT } from '../findInManuscript'
import type { Section } from '../../types/draftline'

const listeners: Array<(e: Event) => void> = []

beforeEach(() => {
  vi.useFakeTimers()
  // Node has no window; the helper only needs these three.
  ;(globalThis as any).window = {
    setTimeout: (fn: () => void, ms: number) => setTimeout(fn, ms),
    dispatchEvent: (e: Event) => { listeners.forEach(l => l(e)); return true },
  }
  ;(globalThis as any).CustomEvent = class {
    type: string; detail: unknown
    constructor(type: string, init?: { detail?: unknown }) { this.type = type; this.detail = init?.detail }
  }
  listeners.length = 0
})

afterEach(() => { vi.useRealTimers() })

function capture() {
  const seen: Array<{ type: string; query?: string }> = []
  listeners.push((e: any) => seen.push({ type: e.type, query: e.detail?.query }))
  return seen
}

describe('taking the writer to a passage', () => {
  it('opens the editor, the chapter, and then finds the text in it', () => {
    const seen = capture()
    const setViewMode = vi.fn()
    const setCurrentChapter = vi.fn()

    goToManuscript({ section: 'body' as Section, index: 3, find: 'Ruiz' }, { setViewMode, setCurrentChapter })

    // The chapter has to be open before the find bar can search it.
    expect(setViewMode).toHaveBeenCalledWith('editor')
    expect(setCurrentChapter).toHaveBeenCalledWith('body', 3)
    expect(seen).toEqual([])

    vi.runAllTimers()
    expect(seen).toEqual([{ type: FIND_IN_MANUSCRIPT_EVENT, query: 'Ruiz' }])
  })

  it('jumps to the chapter alone when there is nothing to find', () => {
    const seen = capture()
    const setCurrentChapter = vi.fn()

    goToManuscript({ section: 'front_matter' as Section, index: 0 }, { setViewMode: vi.fn(), setCurrentChapter })
    vi.runAllTimers()

    expect(setCurrentChapter).toHaveBeenCalledWith('front_matter', 0)
    expect(seen).toEqual([])
  })

  it('ignores an empty or blank query rather than clearing the find bar', () => {
    const seen = capture()
    findInManuscript('')
    findInManuscript('   ')
    vi.runAllTimers()
    expect(seen).toEqual([])
  })

  it('trims the query, so a padded name still matches', () => {
    const seen = capture()
    findInManuscript('  Officer Ruiz  ')
    vi.runAllTimers()
    expect(seen).toEqual([{ type: FIND_IN_MANUSCRIPT_EVENT, query: 'Officer Ruiz' }])
  })
})
