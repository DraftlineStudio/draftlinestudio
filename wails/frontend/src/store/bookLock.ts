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
  /** Nothing answered, so the book is this machine's to take. */
  | { status: 'unanswered' }
  /** The claim went away while we waited: the book is nobody's now. */
  | { status: 'released' }
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
// GIVE_UP is how long the other machine gets before this one takes the book.
//
// Thirty seconds is a whole round trip with room to spare: the request syncs
// across, the fifteen-second countdown runs there, and the answer syncs back. A
// machine that is running answers inside that. One that does not answer is
// almost always switched off or asleep — which is also why taking the book from
// it is safe, since a machine that is not running is not writing either.
//
// The writer is told this number before the wait starts, because a wait that
// ends by doing something has to say what it is going to do.
const GIVE_UP = 30_000

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
  // Consecutive looks that found no claim at all. Two, because a sidecar that
  // is briefly unreadable — a sync client mid-write — looks exactly like one
  // that is gone, and one flicker should not take a book off somebody.
  let released = 0
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

    // The claim went away while we were waiting. Nobody holds the book, so
    // there is nothing left to wait for and nothing to take it from.
    //
    // This is the closed-a-moment-ago case: Draftline was shut over there, its
    // claim was deleted, and the deletion is only now arriving. Nothing is
    // running to answer the request or to stamp what it has, so without this
    // the wait would sit there until the writer gave up — even though the book
    // has been free the whole time.
    if (!status.answered && !status.held_elsewhere) {
      if (++released >= 2) return { status: 'released' }
    } else {
      released = 0
    }

    const waited = Date.now() - started
    // Silence runs out; an answer does not. A refusal is an answer and ends the
    // wait on its own, and a grant means the book is coming, however long its
    // sync client takes over a hundred megabytes. Only nothing at all expires.
    //
    // And running out of silence is not enough on its own. A machine that never
    // answers is a machine that is RUNNING with nobody in front of it — an
    // unattended one stops heartbeating and its claim goes stale, which raises
    // no question in the first place. So it was recently writing, and the copy
    // here has to be shown to be the one it last wrote before taking the book
    // can be safe. can_take_over is that check, made against the fingerprint the
    // holder stamps into its claim the moment it sees the request.
    if (!status.answered && waited >= GIVE_UP && status.can_take_over) {
      return { status: 'unanswered' }
    }

    watch.onProgress(status.answered ? status.message : countdown(status, waited))
    await sleep(POLL_EVERY)
  }
}

// countdown says how long the other machine has left. A wait that ends by doing
// something has to say what it is going to do, and when.
//
// Once the clock has run out the sentence stops promising a takeover and starts
// describing what is still outstanding, which by then is the sync: the message
// from the backend already names it.
function countdown(status: types.BookTakeoverStatus, waited: number): string {
  if (waited >= GIVE_UP) return status.message
  const left = Math.max(1, Math.ceil((GIVE_UP - waited) / 1000))
  return `${status.message} Taking it in ${left} second${left === 1 ? '' : 's'} if there is no answer.`
}
