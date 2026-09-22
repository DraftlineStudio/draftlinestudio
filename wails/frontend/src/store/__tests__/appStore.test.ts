// Regression test for the settings save pipeline (audit finding Fable M2:
// serialize-savesettings).
//
// saveSettings used to fire SaveSettings(next) with no serialization. Two rapid
// callers (e.g. sidebar width + lane view) could land out of order on the Go
// side, so the later-completing backend write would overwrite the other's field
// — a lost update. Fixed by merging into in-memory state synchronously and
// chaining every backend write through a module-scoped saveChain, snapshotting
// the newest merged state inside each chained write.
//
// Environment: plain node, no jsdom. Only the generated Wails binding module is
// mocked.

import { describe, it, expect, vi, beforeEach } from 'vitest'

const mocks = vi.hoisted(() => ({
  LoadSettings: vi.fn(),
  SaveSettings: vi.fn(),
  BrowseForDirectory: vi.fn(),
  GetRecentProjects: vi.fn(),
  AddRecentProject: vi.fn(),
  RemoveRecentProject: vi.fn(),
  ClearRecentProjects: vi.fn(),
}))

vi.mock('../../../wailsjs/go/main/App', () => mocks)

function deferred<T>() {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((res) => { resolve = res })
  return { promise, resolve }
}

async function flushMicrotasks() {
  for (let i = 0; i < 20; i++) await Promise.resolve()
}

let appStoreMod: typeof import('../appStore')
const store = () => appStoreMod.useAppStore.getState()

beforeEach(async () => {
  vi.resetModules() // fresh module-scoped saveChain + fresh zustand state
  for (const fn of Object.values(mocks)) fn.mockReset()
  appStoreMod = await import('../appStore')
})

describe('serialize-savesettings — chained backend writes', () => {
  it('two rapid saveSettings calls both persist with no lost update', async () => {
    const first = deferred<void>()
    mocks.SaveSettings
      .mockImplementationOnce(() => first.promise)
      .mockResolvedValueOnce(undefined)

    // Two callers fire back-to-back, each patching a different field.
    const p1 = store().saveSettings({ sidebar_panel_width: 400 })
    const p2 = store().saveSettings({ characters_lane_view: 'heat' })

    // Both fields are merged into in-memory state synchronously.
    expect(store().settings.sidebar_panel_width).toBe(400)
    expect(store().settings.characters_lane_view).toBe('heat')

    await flushMicrotasks()

    // Writes are serialized: the second backend write must not start until the
    // first SaveSettings resolves.
    expect(mocks.SaveSettings).toHaveBeenCalledTimes(1)

    first.resolve()
    await flushMicrotasks()
    await Promise.all([p1, p2])

    // The second write ran only after the first completed.
    expect(mocks.SaveSettings).toHaveBeenCalledTimes(2)

    // The final backend write carries BOTH fields — neither patch was dropped.
    const lastArg = mocks.SaveSettings.mock.calls[1][0]
    expect(lastArg.sidebar_panel_width).toBe(400)
    expect(lastArg.characters_lane_view).toBe('heat')
  })

  it('saveSettings returns the chained promise that resolves after the backend write', async () => {
    const gate = deferred<void>()
    mocks.SaveSettings.mockImplementationOnce(() => gate.promise)

    let resolved = false
    const p = store().saveSettings({ sidebar_panel_width: 321 }).then(() => { resolved = true })

    await flushMicrotasks()
    expect(resolved).toBe(false) // still awaiting the backend write

    gate.resolve()
    await p
    expect(resolved).toBe(true)
  })
})

describe('analysis CPU profile migration', () => {
  it('defaults older settings to adaptive and preserves a valid explicit profile', async () => {
    mocks.LoadSettings.mockResolvedValueOnce({})
    await store().loadSettings()
    expect(store().settings.analysis_cpu_profile).toBe('adaptive')

    mocks.LoadSettings.mockResolvedValueOnce({ analysis_cpu_profile: 'gentle' })
    await store().loadSettings()
    expect(store().settings.analysis_cpu_profile).toBe('gentle')
  })
})

// A settings write that fails on the Go side used to be caught, logged to the
// console, and resolved as though it had worked. The screen kept showing the
// new value and the writer found out at the next launch, when it was gone.
describe('a settings write that fails says so', () => {
  it('reports the failure to the writer and to the caller', async () => {
    mocks.SaveSettings.mockRejectedValueOnce(new Error('disk is full'))

    const saved = await store().saveSettings({ sidebar_panel_width: 420 })

    expect(saved).toBe(false)
    expect(store().statusMessage).toContain('Could not save settings')
    expect(store().statusMessage).toContain('disk is full')
  })

  it('a successful write reports success and says nothing', async () => {
    mocks.SaveSettings.mockResolvedValueOnce(undefined)

    const saved = await store().saveSettings({ sidebar_panel_width: 420 })

    expect(saved).toBe(true)
    expect(store().statusMessage).not.toContain('Could not save settings')
  })

  // The chain is assigned the same promise saveSettings returns. A rejection
  // left on it would skip every later callback, so one failed write would mean
  // no setting ever reached disk again for the rest of the session.
  it('a later write still reaches the backend after a failure', async () => {
    mocks.SaveSettings
      .mockRejectedValueOnce(new Error('disk is full'))
      .mockResolvedValueOnce(undefined)

    expect(await store().saveSettings({ sidebar_panel_width: 420 })).toBe(false)
    expect(await store().saveSettings({ characters_lane_view: 'heat' })).toBe(true)

    expect(mocks.SaveSettings).toHaveBeenCalledTimes(2)
    expect(store().settings.characters_lane_view).toBe('heat')
  })
})
