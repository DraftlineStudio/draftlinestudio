// Dragging a card to the edge of the timeline or board scrolls it, so a card
// can go from Later to an early chapter in one motion instead of a
// drop / scroll / pick up again cycle.
//
// Two things make this less obvious than it looks. The scroll runs from a
// frame loop off the last known pointer position rather than from the drag
// events, because dragover stops firing while the pointer is held still at
// the edge -- which is exactly when the scroll still needs to be going. And
// the position is tracked on the document, not on the scroller, so the loop
// knows when the pointer has left the scroller entirely and should stop
// rather than run on a stale point.

import { useCallback, useEffect, useRef } from 'react'

// How close to an edge starts the scroll, and the fastest it goes at the edge
// itself. The speed ramps across the zone so a small overshoot nudges and the
// far edge moves properly.
const EDGE = 64
const MAX_STEP = 24

/** Pixels to scroll this frame for one axis, signed towards the near edge.
 *
 *  `pos` is the pointer's offset inside the scroller, `size` the scroller's
 *  length on that axis. `gutter` is the width of anything held over the start
 *  edge by position: sticky -- the timeline parks its story-line column
 *  there, so the zone that scrolls back towards chapter one has to begin
 *  inside it, or the column becomes a wall you cannot drag past. */
export function edgeScrollDelta(pos: number, size: number, gutter = 0): number {
  const ramp = (depth: number) => Math.ceil((Math.min(depth, EDGE) / EDGE) * MAX_STEP)
  const fromStart = pos - gutter
  if (fromStart < EDGE) return -ramp(EDGE - fromStart)
  const fromEnd = size - pos
  if (fromEnd < EDGE) return ramp(EDGE - fromEnd)
  return 0
}

/** Attaches edge auto-scrolling to a scroller for the length of a card drag.
 *
 *  `active` should be true only while a card is in hand, so an unrelated drag
 *  over the window (a dropped file, say) never scrolls the Planner. */
export function useDragAutoScroll(active: boolean, gutter = 0) {
  const ref = useRef<HTMLDivElement | null>(null)
  const point = useRef<{ x: number; y: number } | null>(null)
  const frame = useRef(0)

  const stop = useCallback(() => {
    point.current = null
    if (frame.current) {
      cancelAnimationFrame(frame.current)
      frame.current = 0
    }
  }, [])

  const tick = useCallback(() => {
    frame.current = 0
    const el = ref.current
    const at = point.current
    if (!el || !at) return
    const box = el.getBoundingClientRect()
    // Outside the scroller the loop idles rather than ending, so coming back
    // over an edge picks the scroll up again without a new drag.
    const inside = at.x >= box.left && at.x <= box.right && at.y >= box.top && at.y <= box.bottom
    if (inside) {
      el.scrollLeft += edgeScrollDelta(at.x - box.left, box.width, gutter)
      el.scrollTop += edgeScrollDelta(at.y - box.top, box.height)
    }
    frame.current = requestAnimationFrame(tick)
  }, [gutter])

  useEffect(() => {
    if (!active) {
      stop()
      return
    }
    const track = (e: DragEvent) => {
      point.current = { x: e.clientX, y: e.clientY }
      if (!frame.current) frame.current = requestAnimationFrame(tick)
    }
    document.addEventListener('dragover', track)
    return () => {
      document.removeEventListener('dragover', track)
      stop()
    }
  }, [active, tick, stop])

  return ref
}
