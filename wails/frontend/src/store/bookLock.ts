// The cross-device claim, lifted out of bookStore so the store keeps its size.
//
// A book in a synced folder can be open on two machines at once, and no file
// syncer can merge two versions of a ZIP: whichever device saves last wins and
// the other one's afternoon is replaced. The backend leaves a claim beside the
// book that travels with it, and this is what the app does with it.
//
// Everything here treats the claim as a hint. It reaches this machine through
// whatever syncs the author's folder, which can be minutes behind, so the
// wording says a book MAY be open elsewhere and never that it is. See
// wails/internal/booklock.

import { InspectBookLock } from '../../wailsjs/go/main/App'
import type { types } from '../../wailsjs/go/models'

/** A book the author is being asked about, and what is claiming it. */
export interface BookLockWarning {
  path: string
  info: types.BookLockInfo
}

/**
 * The device holding a book, if the claim is worth interrupting for.
 *
 * A stale claim means the other machine stopped without releasing it, which is
 * the ordinary result of closing a laptop lid. Asking about those would train
 * the author to click through the warning, and then the one that mattered gets
 * clicked through too.
 *
 * Any failure is swallowed. A warning that cannot be produced must never stand
 * between somebody and their own manuscript — without this feature they had no
 * warning at all.
 */
export async function deviceHoldingBook(path: string): Promise<types.BookLockInfo | null> {
  try {
    const info = await InspectBookLock(path)
    return info?.held && !info.stale ? info : null
  } catch {
    return null
  }
}
