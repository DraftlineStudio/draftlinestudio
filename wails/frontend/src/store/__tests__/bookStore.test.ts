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
import { diffContent } from '../../utils/diff'

// Stable mock fns that survive vi.resetModules() (the factory re-runs on
// re-import but returns this same hoisted object, so references stay valid).
// Every binding imported by bookStore.ts and appStore.ts must be stubbed.
const mocks = vi.hoisted(() => ({
  // bookStore.ts imports
  NewBook: vi.fn(),
  OpenBookDialog: vi.fn(),
  PickBookPath: vi.fn(),
  SaveBook: vi.fn(),
  SaveBookAs: vi.fn(),
  SaveBookSnapshots: vi.fn(),
  OpenRecentProject: vi.fn(),
  AddRecentProject: vi.fn(),
  IndexBook: vi.fn(),
  MergeEntities: vi.fn(),
  SplitEntity: vi.fn(),
  ImportEPUB: vi.fn(),
  ImportDOCX: vi.fn(),
  ShowInfoDialog: vi.fn(),
  AnalyzeBook: vi.fn(),
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
  it('disables autosave without disabling independent chapter history', async () => {
    mocks.SaveBookSnapshots.mockResolvedValue(okSave())
    appStoreMod.useAppStore.setState(s => ({ settings: { ...s.settings, activity_autosave_enabled: false } }))
    const book = makeBook({ body: [{ id: 'ch-one', title: 'Chapter 1', type: 'chapter', content: '<p>original</p>' }] })
    bookStoreMod.useBookStore.setState({ book, isDirty: false })

    store().updateCurrentContent('<p>manual only</p>')
    await vi.advanceTimersByTimeAsync(15 * 60 * 1000)
    await flushMicrotasks()

    expect(mocks.SaveBook).not.toHaveBeenCalled()
    expect(mocks.SaveBookSnapshots).toHaveBeenCalledTimes(1)
    expect(mocks.SaveBookSnapshots.mock.calls[0][1]).toEqual([
      expect.objectContaining({ chapter_id: 'ch-one', content: '<p>manual only</p>' }),
    ])
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

  it('atomically snapshots both sides of an accepted AI selection edit', async () => {
    mocks.SaveBookSnapshots.mockResolvedValue(okSave())
    const before = '<p>Before old after.</p>'
    const after = '<p>Before new after.</p>'
    const book = makeBook({ body: [{ id: 'ch-one', title: 'Chapter 1', type: 'chapter', content: before }] })
    bookStoreMod.useBookStore.setState({ book, currentSection: 'body', currentIndex: 0, isDirty: false })

    const editorStoreMod = await import('../editorStore')
    let editorHtml = before
    editorStoreMod.useEditorStore.setState({
      editorRef: {
        getHTML: () => editorHtml,
        commands: {
          insertContentAt: () => {
            editorHtml = after
            return true
          },
        },
      } as any,
    })
    await editorStoreMod.useEditorStore.getState().setPendingDiff({
      diffs: diffContent('<p>old</p>', '<p>new</p>'),
      originalHtml: '<p>old</p>',
      target: { kind: 'selection', from: 8, to: 11, sourceDocumentHtml: before },
      historyReason: 'AI Line Edit',
    })
    editorStoreMod.useEditorStore.getState().acceptAllDiff()

    await store().applyPendingDiff()

    expect(store().book?.body[0].content).toBe(after)
    expect(mocks.SaveBookSnapshots).toHaveBeenCalledTimes(1)
    expect(mocks.SaveBookSnapshots.mock.calls[0][0].body[0].content).toBe(after)
    expect(mocks.SaveBookSnapshots.mock.calls[0][1]).toEqual([
      expect.objectContaining({ chapter_id: 'ch-one', content: before, reason: 'Before AI Line Edit' }),
      expect.objectContaining({ chapter_id: 'ch-one', content: after, reason: 'After AI Line Edit' }),
    ])
  })
})

describe('background analysis revision safety', () => {
  it('discards analysis results that finish after newer prose edits', async () => {
    const original = makeBook()
    const analyzed = makeBook({ analysis: { story: {
      content_hash: 'old', engine: 'prose-v3', last_analyzed: 'now', version: 1,
      overview: {
        chapter_count: 1, word_count: 1, sentence_count: 1, paragraph_count: 1,
        average_chapter_words: 1, average_sentence_words: 1, dialogue_percent: 0,
        reading_ease: 1, mean_grade_level: 1, tempo_score: 50,
      },
      chapters: [], observations: [],
    } } })
    const inFlight = deferred<{ success: boolean; error?: string; book: BookData }>()
    mocks.AnalyzeBook.mockImplementationOnce(() => inFlight.promise)
    bookStoreMod.useBookStore.setState({ book: original, currentSection: 'body', currentIndex: 0, analysisRevision: 0 })
    const analysisStoreMod = await import('../analysisStore')

    const run = analysisStoreMod.useAnalysisStore.getState().run()
    await flushMicrotasks()
    store().updateCurrentContent('<p>newer prose</p>')
    inFlight.resolve({ success: true, book: analyzed })
    await run

    expect(store().book?.body[0].content).toBe('<p>newer prose</p>')
    expect(store().book?.analysis?.story).toBeUndefined()
    expect(analysisStoreMod.useAnalysisStore.getState().state).toBe('stale')
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

    expect(appStoreMod.useAppStore.getState().statusMessage).toContain('Save failed: disk full')
    expect(store().dialogs.showUnsavedWarning).toBe(true)
    expect(store().dialogs.pendingAction).toBe('open')
    expect(mocks.PickBookPath).not.toHaveBeenCalled()
    expect(store().isDirty).toBe(true)
  })

  it('saveAndProceed on a cancelled SaveAs picker returns to the intact dialog', async () => {
    openDirtyBookWithPendingOpen()
    // Go returns error 'cancelled' when the user dismisses the file picker.
    mocks.SaveBook.mockResolvedValue({ success: false, file_path: '', error: 'cancelled' })

    await store().saveAndProceed()

    expect(store().dialogs.showUnsavedWarning).toBe(true)
    expect(store().dialogs.pendingAction).toBe('open')
    expect(mocks.PickBookPath).not.toHaveBeenCalled()
    expect(store().isDirty).toBe(true)
    expect(appStoreMod.useAppStore.getState().statusMessage).not.toContain('Save failed')
  })

  it('saveAndProceed on a successful save closes the dialog and proceeds to open', async () => {
    openDirtyBookWithPendingOpen()
    mocks.SaveBook.mockResolvedValue(okSave())
    const opened = makeBook({ file_path: 'C:/tmp/other.draftline' })
    opened.metadata.title = 'Opened Book'
    mocks.PickBookPath.mockResolvedValue('C:/tmp/other.draftline')
    mocks.OpenRecentProject.mockResolvedValue(opened)

    await store().saveAndProceed()

    expect(store().dialogs.showUnsavedWarning).toBe(false)
    expect(store().dialogs.pendingAction).toBe(null)
    expect(mocks.PickBookPath).toHaveBeenCalledTimes(1)
    expect(mocks.OpenRecentProject).toHaveBeenCalledWith('C:/tmp/other.draftline')
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
    expect(appStoreMod.useAppStore.getState().statusMessage).toContain('not closed')
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
    expect(mocks.PickBookPath).not.toHaveBeenCalled()
    expect(appStoreMod.useAppStore.getState().statusMessage).toContain('Newer edits')
  })

  it('an old autosave cannot mutate a replacement project', async () => {
    const inFlight = deferred<{ success: boolean; file_path: string }>()
    mocks.SaveBook.mockImplementationOnce(() => inFlight.promise)
    const replacement = makeBook({ file_path: 'C:/tmp/replacement.draftline' })
    replacement.metadata.title = 'Replacement'
    mocks.PickBookPath.mockResolvedValue('C:/tmp/replacement.draftline')
    mocks.OpenRecentProject.mockResolvedValue(replacement)

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
    expect(appStoreMod.useAppStore.getState().statusMessage).toBe('Save failed: permission denied')

    appStoreMod.useAppStore.setState({ statusMessage: 'Ready' })
    mocks.SaveBookAs.mockResolvedValue({ success: false, file_path: '', error: 'cancelled' })

    await store().saveBookAs()
    expect(appStoreMod.useAppStore.getState().statusMessage).toBe('Ready') // user dismissed the picker: no error banner
    expect(store().isDirty).toBe(true)
  })
})

// OS file associations (0.16.02475): a specific file path must survive the
// unsaved-changes dialog instead of degrading to the generic file picker.
describe('external file opens', () => {
  it('a dirty book defers the .draftline path and opens it after discard', async () => {
    bookStoreMod.useBookStore.setState({ book: makeBook(), isDirty: true })
    const target = makeBook({ file_path: 'C:/books/other.draftline', metadata: { title: 'Other', author: '', publisher: '', created: '', modified: '' } as any })
    mocks.OpenRecentProject.mockResolvedValue(target)

    await store().openExternalFile('C:/books/other.draftline')
    expect(store().dialogs.showUnsavedWarning).toBe(true)
    expect(store().dialogs.pendingAction).toEqual({ openPath: 'C:/books/other.draftline' })
    expect(mocks.OpenRecentProject).not.toHaveBeenCalled()

    await store().discardAndProceed()
    expect(mocks.OpenRecentProject).toHaveBeenCalledWith('C:/books/other.draftline')
    expect(mocks.PickBookPath).not.toHaveBeenCalled() // no picker fallback
    expect(store().book?.file_path).toBe('C:/books/other.draftline')
  })

  it('an .epub routes through the importer and loads as an unsaved project', async () => {
    const imported = makeBook({ file_path: '', metadata: { title: 'Imported Epub', author: '', publisher: '', created: '', modified: '' } as any })
    mocks.ImportEPUB.mockResolvedValue({ success: true, book: imported, warnings: ['1 image was removed'] })

    await store().openExternalFile('C:/books/story.epub')
    expect(mocks.ImportEPUB).toHaveBeenCalledWith('C:/books/story.epub')
    expect(store().book?.metadata.title).toBe('Imported Epub')
    expect(store().isDirty).toBe(true) // imported books are new, unsaved projects
    expect(appStoreMod.useAppStore.getState().statusMessage).toContain('1 import warning')
  })

  it('unsupported extensions are ignored', async () => {
    await store().openExternalFile('C:/books/story.pdf')
    expect(mocks.ImportEPUB).not.toHaveBeenCalled()
    expect(mocks.OpenRecentProject).not.toHaveBeenCalled()
    expect(store().dialogs.showUnsavedWarning).toBe(false)
  })

  it('a .storiverse shows the not-yet-supported notice instead of opening', async () => {
    mocks.ShowInfoDialog.mockResolvedValue(undefined)
    await store().openExternalFile('C:/books/saga.storiverse')
    expect(mocks.ShowInfoDialog).toHaveBeenCalledWith(
      'Storiverse universe',
      expect.stringContaining('later version'),
    )
    expect(mocks.OpenRecentProject).not.toHaveBeenCalled()
    expect(store().book).toBe(null)
  })
})
