// The cross-device claim, kept out of bookStore for its size.

import { InspectBookLock, RequestBookTakeover, BookTakeoverStatus, WithdrawBookTakeover } from '../../wailsjs/go/main/App'
import type { types } from '../../wailsjs/go/models'

export interface BookLockWarning {
  path: string
  info: types.BookLockInfo
}

// A stale claim means the other machine stopped; asking about those trains the
// author to click through. A failed check never blocks opening.
export async function deviceHoldingBook(path: string): Promise<types.BookLockInfo | null> {
  try {
    const info = await InspectBookLock(path)
    return info?.held && !info.stale ? info : null
  } catch {
    return null
  }
}

/** Why a wait for another device ended. */
export type HandoverOutcome =
  | { status: 'ready' }
  | { status: 'unverifiable'; message: string }
  | { status: 'declined'; message: string }
  | { status: 'cancelled' }
  | { status: 'error'; message: string }

export interface HandoverWatch {
  /** Called with the sentence to show while the wait runs. */
  onProgress: (message: string) => void
  /** Polled between looks, so Cancel takes effect within a second. */
  isCancelled: () => boolean
}

const POLL_EVERY = 1000
// PATIENCE is how long the wait runs before it admits the other machine has not
// answered. It does NOT stop waiting: the request has crossed a sync folder and
// the answer has to cross back, which can take longer than anybody will sit and
// watch. It keeps polling until the writer cancels, so an answer that arrives
// late is still acted on.
//
// It has to clear a whole round trip or it calls a working handover a failure:
// the request syncs across, the countdown over there runs its fifteen seconds,
// and the answer syncs back. Saying "no answer" at twenty seconds would be
// saying it while somebody is still reading the question.
const PATIENCE = 45_000

const sleep = (ms: number) => new Promise(resolve => setTimeout(resolve, ms))

/** Ask the device holding a book to hand it over. */
export async function askForBook(path: string): Promise<types.BookTakeoverStatus | null> {
  try {
    return await RequestBookTakeover(path)
  } catch {
    return null
  }
}

export function stopAskingForBook(path: string): void {
  void Promise.resolve(WithdrawBookTakeover(path)).catch(() => {})
}

// waitForHandover polls until the book beside us is provably the one the other
// machine wrote, or until the writer gives up.
//
// It resolves on 'ready' only when the bytes hash to what the grant recorded.
// Nothing else counts: a file that merely exists, or is merely the right size,
// is as likely to be the copy the sync client has not replaced yet, and that
// copy opens perfectly and is yesterday's book.
export async function waitForHandover(path: string, watch: HandoverWatch): Promise<HandoverOutcome> {
  const started = Date.now()
  for (;;) {
    if (watch.isCancelled()) return { status: 'cancelled' }

    let status: types.BookTakeoverStatus
    try {
      status = await BookTakeoverStatus(path)
    } catch (e) {
      return { status: 'error', message: `${e}` }
    }

    if (status.declined) return { status: 'declined', message: status.message }
    if (status.granted && status.arrived) return { status: 'ready' }
    if (status.granted && status.unverifiable) {
      return { status: 'unverifiable', message: status.message }
    }
    if (!status.asked) {
      // The request is gone and nothing answered it, which means something
      // else removed it. Opening on that basis would be opening on a guess.
      return { status: 'error', message: 'the handover request is no longer there' }
    }

    const waited = Date.now() - started
    watch.onProgress(waited < PATIENCE ? status.message : stillWaiting(status))
    await sleep(POLL_EVERY)
  }
}

// stillWaiting is what the overlay says once the countdown has run out. It has
// to read as "this is slow", not as "this has failed": the answer may well be
// on its way through the sync folder.
function stillWaiting(status: types.BookTakeoverStatus): string {
  if (status.granted) return status.message
  return `${status.message} It has not answered yet.`
}
