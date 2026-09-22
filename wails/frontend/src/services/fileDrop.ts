// One owner for dropped files.
//
// Wails' OnFileDrop is a single global registration and OnFileDropOff tears
// the listeners out entirely, taking the runtime's preventDefault with them —
// and WebView2 then downloads a dropped file instead of opening it. So the
// registration is made once, here, and never removed. Consumers subscribe to
// this instead of to Wails.
//
// It is registered with useDropTarget false, meaning every drop arrives rather
// than only those over an element carrying --wails-drop-target. That filtering
// is done here instead, by asking what is actually under the drop point, which
// is the same question with the answer in our hands: a consumer that wants
// only its own card says so, and the fallback takes what nothing else claimed.

import { OnFileDrop } from '../../wailsjs/runtime/runtime'

/**
 * Handles a drop, or declines it.
 *
 * Return true to claim the drop. Returning false passes it to the next
 * consumer, which is how "only over my card" is expressed.
 */
export type DropConsumer = (x: number, y: number, paths: string[]) => boolean

interface Registration {
  consumer: DropConsumer
  /** Higher wins. A specific target must be asked before the fallback. */
  priority: number
}

const registrations: Registration[] = []
let installed = false

function dispatch(x: number, y: number, paths: string[]): void {
  const files = (paths ?? []).filter(p => typeof p === 'string' && p.trim() !== '')
  if (files.length === 0) return
  for (const { consumer } of [...registrations].sort((a, b) => b.priority - a.priority)) {
    try {
      if (consumer(x, y, files)) return
    } catch (error) {
      // One broken consumer must not eat the drop for the rest.
      console.error('[fileDrop] consumer threw', error)
    }
  }
}

/**
 * Subscribe to dropped files. Returns an unsubscribe.
 *
 * The Wails registration is installed on the first subscribe and deliberately
 * never removed: taking it away is what let WebView2 download the file.
 */
export function subscribeFileDrop(consumer: DropConsumer, priority = 0): () => void {
  const registration: Registration = { consumer, priority }
  registrations.push(registration)
  if (!installed) {
    installed = true
    OnFileDrop(dispatch, false)
  }
  return () => {
    const at = registrations.indexOf(registration)
    if (at !== -1) registrations.splice(at, 1)
  }
}

/** Is the drop point inside an element matching `selector`? */
export function droppedOn(x: number, y: number, selector: string): boolean {
  const element = document.elementFromPoint(x, y)
  return element ? element.closest(selector) !== null : false
}

/** The first path that looks like a Draftline project, if any. */
export function firstProjectPath(paths: string[]): string | null {
  return paths.find(p => p.toLowerCase().endsWith('.draftline')) ?? null
}
