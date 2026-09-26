// Handing a book from one of the writer's machines to another.
//
// The rules under test are the ones that decide whether a manuscript survives
// the handover:
//
//   - A book only opens once the bytes beside us hash to what the other machine
//     recorded. The copy a sync client has not finished replacing is a complete,
//     openable, WRONG book, and size alone cannot tell it apart.
//   - Giving up takes the request back. A request left behind gets granted later
//     by a machine with nobody waiting, and that machine saves, releases and
//     closes a book somebody may be sitting in front of.
//   - Handing over saves first, and refuses if the save is not clean. A 'stale'
//     save means somebody is typing here, which is an answer of no.
//   - Nothing may reach the file after the fingerprint is taken.
//
// Environment: plain node, no jsdom. Only the generated Wails bindings are
// mocked; wailsjs/go/models stays real.

import { describe, it, expect, vi, beforeEach } from 'vitest'
import type { BookData } from '../../types/draftline'

const mocks = vi.hoisted(() => ({
  NewBook: vi.fn(),
  OpenBookDialog: vi.fn(),
  PickBookPath: vi.fn(),
  SaveBook: vi.fn(),
  SaveBookAs: vi.fn(),
  SaveBookSnapshots: vi.fn(),
  OpenRecentProject: vi.fn(),
  OpenBookAsCopy: vi.fn(),
  InspectBookLock: vi.fn(),
  AddRecentProject: vi.fn(),
  IndexBook: vi.fn(),
  MergeEntities: vi.fn(),
  SplitEntity: vi.fn(),
  ImportEPUB: vi.fn(),
  ImportDOCX: vi.fn(),
  ShowInfoDialog: vi.fn(),
  CloseBookFile: vi.fn().mockResolvedValue(undefined),
  AnalyzeBook: vi.fn(),
  RequestBookTakeover: vi.fn(),
  BookTakeoverStatus: vi.fn(),
  WithdrawBookTakeover: vi.fn(),
  GrantBookTakeover: vi.fn(),
  DeclineBookTakeover: vi.fn(),
  PendingBookTakeover: vi.fn(),
  SaveBookAside: vi.fn(),
  LoadSettings: vi.fn(),
  SaveSettings: vi.fn(),
  BrowseForDirectory: vi.fn(),
  GetRecentProjects: vi.fn(),
  RemoveRecentProject: vi.fn(),
  ClearRecentProjects: vi.fn(),
}))

vi.mock('../../../wailsjs/go/main/App', () => mocks)

const PATH = 'C:/books/weather-house.draftline'

function makeBook(): BookData {
  return {
    version: '2.0',
    metadata: { title: 'Test Book', author: 'A. Author', isbn: '', publisher: '', created: '', modified: '', word_count: 2 },
    copyright: '',
    front_matter: [],
    body: [{ id: 'ch-one', title: 'Chapter 1', type: 'chapter', content: '<p>original</p>' }],
    back_matter: [],
    file_path: PATH,
    is_indexed: true,
  }
}

// waiting is an unanswered request whose copy here is NOT yet confirmed to be
// the one the holder last wrote; confirmed is the same request once it is.
const waiting = () => ({ asked: true, held_elsewhere: true, answered: false, granted: false, declined: false, arrived: false, unverifiable: false, can_take_over: false, local_bytes: 0, expected_bytes: 0, message: 'Waiting for STUDIO-DESKTOP to answer…' })
const confirmed = () => ({ ...waiting(), can_take_over: true })
const granted = (over: Record<string, unknown> = {}) => ({ ...waiting(), answered: true, granted: true, held_elsewhere: false, message: 'Waiting for the copy from STUDIO-DESKTOP to sync… 1 MB of 3 MB arrived.', local_bytes: 1, expected_bytes: 3, ...over })

let bookStoreMod: typeof import('../bookStore')
const store = () => bookStoreMod.useBookStore.getState()

async function flush() {
  for (let i = 0; i < 20; i++) await Promise.resolve()
}

beforeEach(async () => {
  vi.resetModules()
  for (const fn of Object.values(mocks)) fn.mockReset()
  mocks.AddRecentProject.mockResolvedValue(undefined)
  mocks.CloseBookFile.mockResolvedValue(undefined)
  vi.useFakeTimers()
  bookStoreMod = await import('../bookStore')
})

describe('asking another device for a book', () => {
  it('waits for the copy to arrive before it opens anything', async () => {
    mocks.RequestBookTakeover.mockResolvedValue(waiting())
    // Granted, then still arriving twice, then proven.
    mocks.BookTakeoverStatus
      .mockResolvedValueOnce(granted())
      .mockResolvedValueOnce(granted({ local_bytes: 2 }))
      .mockResolvedValue(granted({ arrived: true, local_bytes: 3, message: 'STUDIO-DESKTOP handed the book over.' }))
    mocks.OpenRecentProject.mockResolvedValue(makeBook())

    bookStoreMod.useBookStore.setState(s => ({ dialogs: { ...s.dialogs, bookLockWarning: { path: PATH, info: { held: true, stale: false, device: 'STUDIO-DESKTOP', platform: 'windows', app: 'Draftline', last_seen: '', message: '' } } } }))
    const asking = store().askDeviceForBook()

    await flush()
    expect(store().isWaitingForDevice).toBe(true)
    // Nothing is opened on the strength of a grant alone.
    expect(mocks.OpenRecentProject).not.toHaveBeenCalled()
    expect(store().openingLabel).toContain('sync')

    await vi.advanceTimersByTimeAsync(3000)
    await asking

    expect(mocks.OpenRecentProject).toHaveBeenCalledWith(PATH)
    expect(store().book?.metadata.title).toBe('Test Book')
    expect(store().isWaitingForDevice).toBe(false)
    // The receipt is spent once the bytes are proven.
    expect(mocks.WithdrawBookTakeover).toHaveBeenCalledWith(PATH)
  })

  it('never opens a book whose fingerprint could not be checked', async () => {
    mocks.RequestBookTakeover.mockResolvedValue(waiting())
    mocks.BookTakeoverStatus.mockResolvedValue(granted({
      unverifiable: true,
      message: 'STUDIO-DESKTOP handed the book over, but this copy could not be checked against it.',
    }))

    bookStoreMod.useBookStore.setState(s => ({ dialogs: { ...s.dialogs, bookLockWarning: { path: PATH, info: { held: true, stale: false, device: 'STUDIO-DESKTOP', platform: '', app: '', last_seen: '', message: '' } } } }))
    await store().askDeviceForBook()

    expect(mocks.OpenRecentProject).not.toHaveBeenCalled()
    expect(store().book).toBeNull()
  })

  it('takes the request back when the writer stops waiting', async () => {
    mocks.RequestBookTakeover.mockResolvedValue(waiting())
    mocks.BookTakeoverStatus.mockResolvedValue(waiting())

    bookStoreMod.useBookStore.setState(s => ({ dialogs: { ...s.dialogs, bookLockWarning: { path: PATH, info: { held: true, stale: false, device: 'STUDIO-DESKTOP', platform: '', app: '', last_seen: '', message: '' } } } }))
    const asking = store().askDeviceForBook()
    await flush()
    expect(store().isWaitingForDevice).toBe(true)

    store().stopWaitingForDevice()
    await vi.advanceTimersByTimeAsync(2000)
    await asking

    expect(mocks.WithdrawBookTakeover).toHaveBeenCalledWith(PATH)
    expect(mocks.OpenRecentProject).not.toHaveBeenCalled()
    expect(store().isWaitingForDevice).toBe(false)
  })

  it('takes the book when nothing answers and the copy here is confirmed', async () => {
    mocks.RequestBookTakeover.mockResolvedValue(confirmed())
    mocks.BookTakeoverStatus.mockResolvedValue(confirmed())
    mocks.OpenRecentProject.mockResolvedValue(makeBook())

    bookStoreMod.useBookStore.setState(s => ({ dialogs: { ...s.dialogs, bookLockWarning: { path: PATH, info: { held: true, stale: false, device: 'STUDIO-DESKTOP', platform: '', app: '', last_seen: '', message: '' } } } }))
    const asking = store().askDeviceForBook()

    await flush()
    // It says what it is about to do, and when.
    expect(store().openingLabel).toMatch(/Taking it in \d+ seconds?/)
    expect(mocks.OpenRecentProject).not.toHaveBeenCalled()

    await vi.advanceTimersByTimeAsync(31_000)
    await asking

    // Opening claims the book, which is what the other machine stands down on.
    expect(mocks.OpenRecentProject).toHaveBeenCalledWith(PATH)
    expect(mocks.WithdrawBookTakeover).toHaveBeenCalledWith(PATH)
    expect(store().book).not.toBeNull()
  })

  it('will not take a book it cannot confirm, however long it waits', async () => {
    // A machine that never answers is a machine that is RUNNING with nobody in
    // front of it: an unattended one stops heartbeating and goes stale, which
    // raises no question at all. So it was recently writing, and the copy here
    // may be the one from before its last save. The clock running out is not
    // permission on its own.
    mocks.RequestBookTakeover.mockResolvedValue(waiting())
    mocks.BookTakeoverStatus.mockResolvedValue(waiting())

    bookStoreMod.useBookStore.setState(s => ({ dialogs: { ...s.dialogs, bookLockWarning: { path: PATH, info: { held: true, stale: false, device: 'STUDIO-DESKTOP', platform: '', app: '', last_seen: '', message: '' } } } }))
    void store().askDeviceForBook()
    await flush()

    await vi.advanceTimersByTimeAsync(120_000)
    await flush()

    expect(mocks.OpenRecentProject).not.toHaveBeenCalled()
    expect(store().book).toBeNull()
    expect(store().isWaitingForDevice).toBe(true)
    store().stopWaitingForDevice()
    await vi.advanceTimersByTimeAsync(2000)
  })

  it('waits as long as the copy needs once it has been granted', async () => {
    // A grant is an answer, so the thirty seconds does not apply to it: a big
    // book crossing a slow connection takes as long as it takes.
    mocks.RequestBookTakeover.mockResolvedValue(waiting())
    mocks.BookTakeoverStatus.mockResolvedValue(granted())

    bookStoreMod.useBookStore.setState(s => ({ dialogs: { ...s.dialogs, bookLockWarning: { path: PATH, info: { held: true, stale: false, device: 'STUDIO-DESKTOP', platform: '', app: '', last_seen: '', message: '' } } } }))
    void store().askDeviceForBook()
    await flush()

    await vi.advanceTimersByTimeAsync(90_000)
    await flush()

    expect(mocks.OpenRecentProject).not.toHaveBeenCalled()
    expect(store().isWaitingForDevice).toBe(true)
    store().stopWaitingForDevice()
    await vi.advanceTimersByTimeAsync(2000)
  })

  it('puts a refusal on screen, where the status bar is not', async () => {
    mocks.RequestBookTakeover.mockResolvedValue(waiting())
    mocks.BookTakeoverStatus.mockResolvedValue({
      ...waiting(), answered: true, declined: true,
      message: 'Somebody is working on STUDIO-DESKTOP, so it kept the book.',
    })

    bookStoreMod.useBookStore.setState(s => ({ dialogs: { ...s.dialogs, bookLockWarning: { path: PATH, info: { held: true, stale: false, device: 'STUDIO-DESKTOP', platform: '', app: '', last_seen: '', message: '' } } } }))
    await store().askDeviceForBook()

    expect(mocks.OpenRecentProject).not.toHaveBeenCalled()
    expect(store().isWaitingForDevice).toBe(false)
    // The launch screen has no status bar, so a refusal reported there is a
    // refusal nobody sees: the overlay just vanishes.
    const report = store().dialogs.handoverReport
    expect(report?.message).toContain('kept the book')
    expect(report?.offerCopy).toBe(true)
  })

  it('opens a copy from the refusal, which is the way out of it', async () => {
    mocks.RequestBookTakeover.mockResolvedValue(waiting())
    mocks.BookTakeoverStatus.mockResolvedValue({
      ...waiting(), answered: true, declined: true,
      message: 'Somebody is working on STUDIO-DESKTOP, so it kept the book.',
    })
    mocks.OpenBookAsCopy.mockResolvedValue(makeBook())

    bookStoreMod.useBookStore.setState(s => ({ dialogs: { ...s.dialogs, bookLockWarning: { path: PATH, info: { held: true, stale: false, device: 'STUDIO-DESKTOP', platform: '', app: '', last_seen: '', message: '' } } } }))
    await store().askDeviceForBook()
    await store().openCopyAfterHandover()
    await flush()

    expect(mocks.OpenBookAsCopy).toHaveBeenCalledWith(PATH)
    expect(store().dialogs.handoverReport).toBeNull()
  })
})

describe('handing a book to another device', () => {
  it('saves before it hands over, and does not save after', async () => {
    mocks.SaveBook.mockResolvedValue({ success: true, file_path: PATH })
    mocks.GrantBookTakeover.mockResolvedValue({ granted: true, declined: false, withdrawn: false, fingerprinted: true, device: 'LAPTOP' })
    bookStoreMod.useBookStore.setState({ book: makeBook(), isDirty: true })

    await store().grantBookToDevice()
    await flush()

    expect(mocks.SaveBook).toHaveBeenCalledTimes(1)
    expect(mocks.GrantBookTakeover).toHaveBeenCalledTimes(1)
    expect(store().book).toBeNull()
    expect(mocks.CloseBookFile).toHaveBeenCalled()

    // An autosave armed before the handover must not write over the book the
    // other machine has already been told the hash of.
    await vi.advanceTimersByTimeAsync(60_000)
    await flush()
    expect(mocks.SaveBook).toHaveBeenCalledTimes(1)
  })

  it('keeps the book when somebody is typing in it', async () => {
    // A save that completes after another edit comes back stale.
    mocks.SaveBook.mockImplementation(async () => {
      store().updateCurrentContent('<p>still writing</p>')
      return { success: true, file_path: PATH }
    })
    mocks.DeclineBookTakeover.mockResolvedValue({ granted: false, declined: true, withdrawn: false, fingerprinted: false, device: 'LAPTOP' })
    bookStoreMod.useBookStore.setState({ book: makeBook(), isDirty: true })

    await store().grantBookToDevice()
    await flush()

    expect(mocks.GrantBookTakeover).not.toHaveBeenCalled()
    expect(mocks.DeclineBookTakeover).toHaveBeenCalledTimes(1)
    expect(store().book).not.toBeNull()
  })

  it('keeps the book when the other device stopped asking', async () => {
    mocks.GrantBookTakeover.mockResolvedValue({ granted: false, declined: false, withdrawn: true, fingerprinted: false })
    bookStoreMod.useBookStore.setState({ book: makeBook(), isDirty: false })

    await store().grantBookToDevice()
    await flush()

    expect(store().book).not.toBeNull()
    expect(mocks.CloseBookFile).not.toHaveBeenCalled()
  })
})

describe('coming back to a handover already granted', () => {
  it('resumes the wait rather than opening the copy it has', async () => {
    mocks.BookTakeoverStatus
      .mockResolvedValueOnce(granted())
      .mockResolvedValueOnce(granted({ local_bytes: 2 }))
      .mockResolvedValue(granted({ arrived: true, local_bytes: 3 }))
    mocks.OpenRecentProject.mockResolvedValue(makeBook())

    const opening = store().openRecentBook(PATH)
    await flush()
    // The grant beside the book is picked up without asking again.
    expect(mocks.RequestBookTakeover).not.toHaveBeenCalled()
    expect(store().isWaitingForDevice).toBe(true)

    await vi.advanceTimersByTimeAsync(2000)
    await opening
    expect(mocks.OpenRecentProject).toHaveBeenCalledWith(PATH)
  })
})

describe('waking to a book another device has taken', () => {
  const taken = { held: true, stale: false, device: 'STUDIO-DESKTOP', platform: 'windows', app: 'Draftline', last_seen: '', message: 'STUDIO-DESKTOP has this book now, so it was closed here.' }

  it('never writes to the original, and keeps unsaved words in a file of their own', async () => {
    mocks.SaveBookAside.mockResolvedValue({ success: true, file_path: 'C:/books/weather-house (unsaved on LAPTOP).draftline' })
    bookStoreMod.useBookStore.setState({ book: makeBook(), isDirty: true })

    await bookStoreMod.useBookStore.getState().standDownFromBook(taken as never)
    await flush()

    // The original belongs to the other machine now.
    expect(mocks.SaveBook).not.toHaveBeenCalled()
    expect(mocks.SaveBookAside).toHaveBeenCalledTimes(1)
    expect(store().book).toBeNull()
    expect(mocks.CloseBookFile).toHaveBeenCalled()

    // And the autosave that survived the sleep must not fire into the file.
    await vi.advanceTimersByTimeAsync(60_000)
    await flush()
    expect(mocks.SaveBook).not.toHaveBeenCalled()
  })

  it('closes a clean book without leaving a rescue file behind', async () => {
    bookStoreMod.useBookStore.setState({ book: makeBook(), isDirty: false })

    await bookStoreMod.useBookStore.getState().standDownFromBook(taken as never)
    await flush()

    expect(mocks.SaveBookAside).not.toHaveBeenCalled()
    expect(mocks.SaveBook).not.toHaveBeenCalled()
    expect(store().book).toBeNull()
  })

  it('still closes the book when the rescue file cannot be written', async () => {
    mocks.SaveBookAside.mockRejectedValue(new Error('the folder is gone'))
    bookStoreMod.useBookStore.setState({ book: makeBook(), isDirty: true })

    await bookStoreMod.useBookStore.getState().standDownFromBook(taken as never)
    await flush()

    // Staying open would be worse: the next autosave would overwrite the other
    // machine's work.
    expect(store().book).toBeNull()
    expect(mocks.SaveBook).not.toHaveBeenCalled()
  })
})

describe('the other computer closed the book a moment ago', () => {
  // Draftline was shut over there, so its claim is deleted and the deletion is
  // still crossing. Nothing is running to answer, and nothing is running to say
  // what it has — so waiting for either would be waiting for good.
  const letGo = () => ({ ...waiting(), held_elsewhere: false })

  it('opens the book once the claim is gone, without waiting out the clock', async () => {
    mocks.RequestBookTakeover.mockResolvedValue(waiting())
    mocks.BookTakeoverStatus.mockResolvedValue(letGo())
    mocks.OpenRecentProject.mockResolvedValue(makeBook())

    bookStoreMod.useBookStore.setState(s => ({ dialogs: { ...s.dialogs, bookLockWarning: { path: PATH, info: { held: true, stale: false, device: 'STUDIO-DESKTOP', platform: '', app: '', last_seen: '', message: '' } } } }))
    const asking = store().askDeviceForBook()

    // Well inside the thirty seconds, and with nothing ever stamped.
    await vi.advanceTimersByTimeAsync(4000)
    await asking

    expect(mocks.OpenRecentProject).toHaveBeenCalledWith(PATH)
    expect(store().book).not.toBeNull()
    expect(mocks.WithdrawBookTakeover).toHaveBeenCalledWith(PATH)
  })

  it('does not take a book off anybody over one unreadable look', async () => {
    // A sidecar caught mid-write reads as missing. One flicker is not a release.
    mocks.RequestBookTakeover.mockResolvedValue(waiting())
    mocks.BookTakeoverStatus
      .mockResolvedValueOnce(letGo())
      .mockResolvedValue(waiting())

    bookStoreMod.useBookStore.setState(s => ({ dialogs: { ...s.dialogs, bookLockWarning: { path: PATH, info: { held: true, stale: false, device: 'STUDIO-DESKTOP', platform: '', app: '', last_seen: '', message: '' } } } }))
    void store().askDeviceForBook()

    await vi.advanceTimersByTimeAsync(10_000)
    await flush()

    expect(mocks.OpenRecentProject).not.toHaveBeenCalled()
    expect(store().isWaitingForDevice).toBe(true)
    store().stopWaitingForDevice()
    await vi.advanceTimersByTimeAsync(2000)
  })
})
