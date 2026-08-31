// Regression tests for the save pipeline (audit findings 1 and 2).
//
// Finding 2: an edit made while an autosave was in flight used to be wiped
// clean (isDirty=false) when the stale save completed, silently dropping the
// newer content. Fixed by the module-scoped saveRevision counter + serialized
// saveChain in bookStore.ts.
//
// Finding 1: a failed or cancelled save used to fall through in
// saveAndProceed / closeProject, discarding the unsaved book anyway. Fixed by
// performSave outcomes gating every transition.
//
// Environment: plain node, no jsdom. The only boundary mocked is the
// generated Wails binding module; wailsjs/go/models stays real.

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import type { BookData } from '../../types/draftline'

// Stable mock fns that survive vi.resetModules() (the factory re-runs on
// re-import but returns this same hoisted object, so references stay valid).
// Every binding imported by bookStore.ts and appStore.ts must be stubbed.
const mocks = vi.hoisted(() => ({
  // bookStore.ts imports
  NewBook: vi.fn(),
  OpenBookDialog: vi.fn(),
  SaveBook: vi.fn(),
  SaveBookAs: vi.fn(),
  SaveBookSnapshots: vi.fn(),
  OpenRecentProject: vi.fn(),
  AddRecentProject: vi.fn(),
  IndexBook: vi.fn(),
  MergeEntities: vi.fn(),
  SplitEntity: vi.fn(),
  // appStore.ts imports (bookStore imports appStore)
  LoadSettings: vi.fn(),
  SaveSettings: vi.fn(),
  BrowseForDirectory: vi.fn(),
  GetRecentProjects: vi.fn(),
  RemoveRecentProject: vi.fn(),
  ClearRecentProjects: vi.fn(),
}))

vi.mock('../../../wailsjs/go/main/App', () => mocks)

// Holds a mocked binding call in flight until the test resolves it.
function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((res, rej) => { resolve = res; reject = rej })
  return { promise, resolve, reject }
}

// Drain chained microtasks (performSave has several await points).
async function flushMicrotasks() {
  for (let i = 0; i < 20; i++) await Promise.resolve()
}

function makeBook(overrides: Partial<BookData> = {}): BookData {
  return {
    version: '2.0',
    metadata: { title: 'Test Book', author: 'A. Author', isbn: '', publisher: '', created: '', modified: '' },
    copyright: '',
    front_matter: [],
    body: [{ title: 'Chapter 1', type: 'chapter', content: '<p>original</p>' }],
    back_matter: [],
    file_path: 'C:/tmp/test.draftline',
    is_indexed: true,
    ...overrides,
  }
}

const okSave = (path = 'C:/tmp/test.draftline') => ({ success: true, file_path: path })

let bookStoreMod: typeof import('../bookStore')
let appStoreMod: typeof import('../appStore')

const store = () => bookStoreMod.useBookStore.getState()

beforeEach(async () => {
  vi.resetModules() // fresh saveRevision / saveChain / autoSaveTimer + fresh zustand state
  for (const fn of Object.values(mocks)) fn.mockReset()
  mocks.AddRecentProject.mockResolvedValue(undefined)
  vi.useFakeTimers()
  bookStoreMod = await import('../bookStore')
  appStoreMod = await import('../appStore')
})

describe('activity-based saving and chapter history', () => {
  it('does not autosave when activity-based saving is disabled', async () => {
    appStoreMod.useAppStore.setState(s => ({ settings: { ...s.settings, activity_autosave_enabled: false } }))
    bookStoreMod.useBookStore.setState({ book: makeBook(), isDirty: false })

    store().updateCurrentContent('<p>manual only</p>')
    await vi.advanceTimersByTimeAsync(15 * 60 * 1000)

    expect(mocks.SaveBook).not.toHaveBeenCalled()
    expect(mocks.SaveBookSnapshots).not.toHaveBeenCalled()
    expect(store().isDirty).toBe(true)
  })

  it('snapshots only a changed chapter after ten minutes of writing activity', async () => {
    mocks.SaveBook.mockResolvedValue(okSave())
    mocks.SaveBookSnapshots.mockResolvedValue(okSave())
    const book = makeBook({ body: [{ id: 'ch-one', title: 'Chapter 1', type: 'chapter', content: '<p>original</p>' }] })
    bookStoreMod.useBookStore.setState({ book, isDirty: false })

    store().updateCurrentContent('<p>changed</p>')
    await vi.advanceTimersByTimeAsync(5_000)
    expect(mocks.SaveBookSnapshots).not.toHaveBeenCalled()

    await vi.advanceTimersByTimeAsync(9 * 60 * 1000 + 55_000)
    await flushMicrotasks()
    expect(mocks.SaveBookSnapshots).toHaveBeenCalledTimes(1)
    expect(mocks.SaveBookSnapshots.mock.calls[0][1]).toEqual([
      expect.objectContaining({ chapter_id: 'ch-one', content: '<p>changed</p>' }),
    ])
  })
})

afterEach(() => {
  vi.clearAllTimers()
  vi.useRealTimers()
})

describe('finding 2 — revision counter + serialized saves', () => {
  it('an edit during an in-flight autosave keeps isDirty, and the next cycle persists the newer content', async () => {
    const inFlight = deferred<{ success: boolean; file_path: string }>()
    mocks.SaveBook
      .mockImplementationOnce(() => inFlight.promise)
      .mockResolvedValueOnce(okSave())

    bookStoreMod.useBookStore.setState({ book: makeBook(), isDirty: false })
    store().updateCurrentContent('<p>edit one</p>')
    expect(store().isDirty).toBe(true)

    // Autosave debounce fires; SaveBook is now held in flight.
    await vi.advanceTimersByTimeAsync(5000)
    expect(mocks.SaveBook).toHaveBeenCalledTimes(1)
    expect(mocks.SaveBook.mock.calls[0][0].body[0].content).toBe('<p>edit one</p>')

    // User keeps typing while the save is on the wire.
    store().updateCurrentContent('<p>edit two</p>')

    // The stale save completes successfully...
    inFlight.resolve(okSave())
    await flushMicrotasks()

    // ...but must NOT mark the newer edit clean. (This is the data-loss bug:
    // without the revision check, isDirty would be false here and edit two
    // would never be written.)
    expect(store().isDirty).toBe(true)

    // The edit re-armed the debounce; the next cycle writes the newer state.
    await vi.advanceTimersByTimeAsync(5000)
    await flushMicrotasks()
    expect(mocks.SaveBook).toHaveBeenCalledTimes(2)
    expect(mocks.SaveBook.mock.calls[1][0].body[0].content).toBe('<p>edit two</p>')
    expect(store().isDirty).toBe(false)
  })

  it('serializes saves: a second save waits for the first SaveBook call to resolve', async () => {
    const first = deferred<{ success: boolean; file_path: string }>()
    mocks.SaveBook
      .mockImplementationOnce(() => first.promise)
      .mockResolvedValueOnce(okSave())

    bookStoreMod.useBookStore.setState({ book: makeBook(), isDirty: true })

    const p1 = store().saveBook()
    await flushMicrotasks()
    const p2 = store().saveBook()
    await flushMicrotasks()

    // Second SaveBook must not be issued while the first is still in flight.
    expect(mocks.SaveBook).toHaveBeenCalledTimes(1)

    first.resolve(okSave())
    await flushMicrotasks()
    await Promise.all([p1, p2])
    expect(mocks.SaveBook).toHaveBeenCalledTimes(2)
  })
})

describe('finding 1 — failed or cancelled saves block transitions', () => {
  function openDirtyBookWithPendingOpen() {
    bookStoreMod.useBookStore.setState({ book: makeBook(), isDirty: true })
    // Real flow: attempting to open with unsaved changes raises the dialog.
    void store().openBook()
    expect(store().dialogs.showUnsavedWarning).toBe(true)
    expect(store().dialogs.pendingAction).toBe('open')
  }

  it('saveAndProceed on a failed save keeps the dialog open, retains pendingAction, and does not open a book', async () => {
    openDirtyBookWithPendingOpen()
    mocks.SaveBook.mockResolvedValue({ success: false, file_path: '', error: 'disk full' })

    await store().saveAndProceed()

    expect(store().statusMessage).toContain('Save failed: disk full')
    expect(store().dialogs.showUnsavedWarning).toBe(true)
    expect(store().dialogs.pendingAction).toBe('open')
    expect(mocks.OpenBookDialog).not.toHaveBeenCalled()
    expect(store().isDirty).toBe(true)
  })

  it('saveAndProceed on a cancelled SaveAs picker returns to the intact dialog', async () => {
    openDirtyBookWithPendingOpen()
    // Go returns error 'cancelled' when the user dismisses the file picker.
    mocks.SaveBook.mockResolvedValue({ success: false, file_path: '', error: 'cancelled' })

    await store().saveAndProceed()

    expect(store().dialogs.showUnsavedWarning).toBe(true)
    expect(store().dialogs.pendingAction).toBe('open')
    expect(mocks.OpenBookDialog).not.toHaveBeenCalled()
    expect(store().isDirty).toBe(true)
    expect(store().statusMessage).not.toContain('Save failed')
  })

  it('saveAndProceed on a successful save closes the dialog and proceeds to open', async () => {
    openDirtyBookWithPendingOpen()
    mocks.SaveBook.mockResolvedValue(okSave())
    const opened = makeBook({ file_path: 'C:/tmp/other.draftline' })
    opened.metadata.title = 'Opened Book'
    mocks.OpenBookDialog.mockResolvedValue(opened)

    await store().saveAndProceed()

    expect(store().dialogs.showUnsavedWarning).toBe(false)
    expect(store().dialogs.pendingAction).toBe(null)
    expect(mocks.OpenBookDialog).toHaveBeenCalledTimes(1)
    expect(store().book?.metadata.title).toBe('Opened Book')
    expect(store().isDirty).toBe(false)
  })

  it('closeProject aborts the close when the save fails', async () => {
    bookStoreMod.useBookStore.setState({ book: makeBook(), isDirty: true })
    appStoreMod.useAppStore.setState({ showWelcome: false })
    mocks.SaveBook.mockResolvedValue({ success: false, file_path: '', error: 'disk full' })

    await store().closeProject()

    expect(store().book).not.toBe(null)
    expect(store().isDirty).toBe(true)
    expect(appStoreMod.useAppStore.getState().showWelcome).toBe(false)
    expect(store().statusMessage).toContain('not closed')
  })

  it('closeProject proceeds when the save succeeds', async () => {
    bookStoreMod.useBookStore.setState({ book: makeBook(), isDirty: true })
    appStoreMod.useAppStore.setState({ showWelcome: false })
    mocks.SaveBook.mockResolvedValue(okSave())

    await store().closeProject()

    expect(mocks.SaveBook).toHaveBeenCalledTimes(1)
    expect(store().book).toBe(null)
    expect(store().isDirty).toBe(false)
    expect(appStoreMod.useAppStore.getState().showWelcome).toBe(true)
  })

  it('saveAndProceed does not discard an edit made while its save is in flight', async () => {
    openDirtyBookWithPendingOpen()
    const inFlight = deferred<{ success: boolean; file_path: string }>()
    mocks.SaveBook.mockImplementationOnce(() => inFlight.promise)

    const proceed = store().saveAndProceed()
    await flushMicrotasks()
    store().updateCurrentContent('<p>newer edit</p>')
    inFlight.resolve(okSave())
    await proceed

    expect(store().isDirty).toBe(true)
    expect(store().dialogs.showUnsavedWarning).toBe(true)
    expect(store().dialogs.pendingAction).toBe('open')
    expect(mocks.OpenBookDialog).not.toHaveBeenCalled()
    expect(store().statusMessage).toContain('Newer edits')
  })

  it('an old autosave cannot mutate a replacement project', async () => {
    const inFlight = deferred<{ success: boolean; file_path: string }>()
    mocks.SaveBook.mockImplementationOnce(() => inFlight.promise)
    const replacement = makeBook({ file_path: 'C:/tmp/replacement.draftline' })
    replacement.metadata.title = 'Replacement'
    mocks.OpenBookDialog.mockResolvedValue(replacement)

    bookStoreMod.useBookStore.setState({ book: makeBook(), isDirty: false })
    store().updateCurrentContent('<p>discard me</p>')
    await vi.advanceTimersByTimeAsync(5000)
    expect(mocks.SaveBook).toHaveBeenCalledTimes(1)

    bookStoreMod.useBookStore.setState(s => ({
      dialogs: { ...s.dialogs, showUnsavedWarning: true, pendingAction: 'open' },
    }))
    await store().discardAndProceed()
    expect(store().book?.metadata.title).toBe('Replacement')

    inFlight.resolve(okSave('C:/tmp/old-project.draftline'))
    await flushMicrotasks()

    expect(store().book?.metadata.title).toBe('Replacement')
    expect(store().book?.file_path).toBe('C:/tmp/replacement.draftline')
    expect(store().isDirty).toBe(false)
  })

  it('saveBookAs surfaces non-cancelled errors and stays silent on cancelled', async () => {
    bookStoreMod.useBookStore.setState({ book: makeBook(), isDirty: true })
    mocks.SaveBookAs.mockResolvedValue({ success: false, file_path: '', error: 'permission denied' })

    await store().saveBookAs()
    expect(store().statusMessage).toBe('Save failed: permission denied')

    bookStoreMod.useBookStore.setState({ statusMessage: 'Ready' })
    mocks.SaveBookAs.mockResolvedValue({ success: false, file_path: '', error: 'cancelled' })

    await store().saveBookAs()
    expect(store().statusMessage).toBe('Ready') // user dismissed the picker: no error banner
    expect(store().isDirty).toBe(true)
  })
})
