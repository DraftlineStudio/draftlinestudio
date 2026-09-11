// RangeSpotlight plugin-state tests. The vitest environment has no DOM, so
// these exercise createSpotlightPlugin() against a bare ProseMirror
// EditorState — decorations, meta updates, and range remapping across edits.

import { describe, expect, it, vi } from 'vitest'
import { Schema } from '@tiptap/pm/model'
import { EditorState } from '@tiptap/pm/state'
import {
  createSpotlightPlugin,
  findRangeIndex,
  RangeSpotlightKey,
  setSpotlightHandlers,
} from './RangeSpotlight'

const schema = new Schema({
  nodes: {
    doc: { content: 'paragraph+' },
    paragraph: { content: 'text*' },
    text: {},
  },
})

function makeState(text: string): EditorState {
  const doc = schema.node('doc', null, [schema.node('paragraph', null, [schema.text(text)])])
  return EditorState.create({ doc, plugins: [createSpotlightPlugin()] })
}

const RANGES = [
  { from: 1, to: 12 },
  { from: 12, to: 24 },
]

describe('findRangeIndex', () => {
  it('maps positions to their containing range', () => {
    expect(findRangeIndex(RANGES, 1)).toBe(0)
    expect(findRangeIndex(RANGES, 11)).toBe(0)
    expect(findRangeIndex(RANGES, 12)).toBe(1)
    expect(findRangeIndex(RANGES, 23)).toBe(1)
    expect(findRangeIndex(RANGES, 24)).toBe(-1)
    expect(findRangeIndex([], 5)).toBe(-1)
  })
})

describe('spotlight plugin state', () => {
  it('starts inactive with no decorations', () => {
    const state = makeState('The quick brown fox jumps over the dog.')
    const ps = RangeSpotlightKey.getState(state)!
    expect(ps.active).toBe(false)
    expect(ps.decorations.find()).toHaveLength(0)
  })

  it('decorates exactly the active range on a meta update', () => {
    let state = makeState('The quick brown fox jumps over the dog.')
    state = state.apply(state.tr.setMeta(RangeSpotlightKey, { active: true, ranges: RANGES, activeIndex: 0 }))
    const decos = RangeSpotlightKey.getState(state)!.decorations.find()
    expect(decos).toHaveLength(1)
    expect(decos[0].from).toBe(1)
    expect(decos[0].to).toBe(12)
  })

  it('renders nothing for an out-of-bounds active index', () => {
    let state = makeState('The quick brown fox jumps over the dog.')
    state = state.apply(state.tr.setMeta(RangeSpotlightKey, { active: true, ranges: RANGES, activeIndex: 7 }))
    expect(RangeSpotlightKey.getState(state)!.decorations.find()).toHaveLength(0)
  })

  it('remaps ranges across a document edit and reports the edit', async () => {
    const onDocEdited = vi.fn()
    setSpotlightHandlers({ onDocEdited })
    let state = makeState('The quick brown fox jumps over the dog.')
    state = state.apply(state.tr.setMeta(RangeSpotlightKey, { active: true, ranges: RANGES, activeIndex: 1 }))
    // Insert five characters at the very start of the paragraph text.
    state = state.apply(state.tr.insertText('HELLO', 1))
    const ps = RangeSpotlightKey.getState(state)!
    expect(ps.ranges[0].to).toBe(17)
    expect(ps.ranges[1].from).toBe(17)
    expect(ps.ranges[1].to).toBe(29)
    await Promise.resolve() // the edit callback fires via queueMicrotask
    expect(onDocEdited).toHaveBeenCalledTimes(1)
    setSpotlightHandlers({})
  })

  it('clears via the inactive meta update', () => {
    let state = makeState('The quick brown fox jumps over the dog.')
    state = state.apply(state.tr.setMeta(RangeSpotlightKey, { active: true, ranges: RANGES, activeIndex: 0 }))
    state = state.apply(state.tr.setMeta(RangeSpotlightKey, { active: false, ranges: [], activeIndex: -1 }))
    const ps = RangeSpotlightKey.getState(state)!
    expect(ps.active).toBe(false)
    expect(ps.decorations.find()).toHaveLength(0)
  })
})
