// The edge maths behind drag auto-scrolling. The frame loop and the document
// listener need a real drag to exercise, but the decision this makes -- which
// way to scroll and how hard, given where the pointer is -- is arithmetic, and
// it is the part that was wrong before: the timeline's sticky story-line
// column made the left edge unreachable.

import { describe, it, expect } from 'vitest'
import { edgeScrollDelta } from '../useDragAutoScroll'

// A scroller 900px wide, with the timeline's 156px sticky lane column.
const WIDTH = 900
const GUTTER = 156

describe('edgeScrollDelta', () => {
  it('stands still across the middle', () => {
    expect(edgeScrollDelta(450, WIDTH)).toBe(0)
    expect(edgeScrollDelta(450, WIDTH, GUTTER)).toBe(0)
  })

  it('scrolls back and forward near the two edges', () => {
    expect(edgeScrollDelta(10, WIDTH)).toBeLessThan(0)
    expect(edgeScrollDelta(WIDTH - 10, WIDTH)).toBeGreaterThan(0)
  })

  it('ramps up as the pointer closes on an edge', () => {
    const near = Math.abs(edgeScrollDelta(60, WIDTH))
    const nearer = Math.abs(edgeScrollDelta(20, WIDTH))
    const atEdge = Math.abs(edgeScrollDelta(0, WIDTH))
    expect(near).toBeLessThan(nearer)
    expect(nearer).toBeLessThan(atEdge)
  })

  it('never asks for more than the top speed, however far past the edge', () => {
    const atEdge = Math.abs(edgeScrollDelta(0, WIDTH))
    expect(Math.abs(edgeScrollDelta(-500, WIDTH))).toBe(atEdge)
    expect(Math.abs(edgeScrollDelta(WIDTH + 500, WIDTH))).toBe(atEdge)
  })

  // The bug this exists for: over the sticky lane column, the old behaviour
  // was to sit still, so a card could not be dragged from Later back to an
  // early chapter in one motion.
  it('scrolls back while the pointer is over the sticky lane column', () => {
    expect(edgeScrollDelta(80, WIDTH, GUTTER)).toBeLessThan(0)
    expect(edgeScrollDelta(GUTTER, WIDTH, GUTTER)).toBeLessThan(0)
  })

  it('stops scrolling back once the pointer clears the column and its zone', () => {
    expect(edgeScrollDelta(GUTTER + 70, WIDTH, GUTTER)).toBe(0)
  })

  it('leaves the far edge where it is when a gutter is set', () => {
    expect(edgeScrollDelta(WIDTH - 10, WIDTH, GUTTER)).toBe(edgeScrollDelta(WIDTH - 10, WIDTH))
  })
})
