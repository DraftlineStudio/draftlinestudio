// The cross-device claim, kept out of bookStore for its size.

import { InspectBookLock } from '../../wailsjs/go/main/App'
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
