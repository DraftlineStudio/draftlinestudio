import { describe, expect, it } from 'vitest'
import { getSchema } from '@tiptap/core'
import type { Editor } from '@tiptap/core'
import StarterKit from '@tiptap/starter-kit'
import type { StyleFinding } from '../types/draftline'
import { collectDetections, detectionKey, globalChapterIndex, typographyFixableCount } from './proofread'

const schema = getSchema([StarterKit])

function editorOf(...paragraphs: string[]): Editor {
  const doc = schema.nodeFromJSON({
    type: 'doc',
    content: paragraphs.map(text => ({
      type: 'paragraph',
      content: text ? [{ type: 'text', text }] : [],
    })),
  })
  return { state: { doc } } as unknown as Editor
}

describe('collectDetections', () => {
  it('collects grammar and style issues in reading order with context', () => {
    const editor = editorOf('He said "its been a long day..." and left.')
    const detections = collectDetections(editor, [])
    const rules = detections.map(detection => detection.rule)
    expect(rules).toContain('its-contraction')
    expect(rules).toContain('straight-quote')
    expect(rules).toContain('ellipsis')
    for (let i = 1; i < detections.length; i++) {
      expect(detections[i].from).toBeGreaterThanOrEqual(detections[i - 1].from)
    }
    const its = detections.find(detection => detection.rule === 'its-contraction')!
    expect(its.text).toBe('its')
    expect(its.before.endsWith('"')).toBe(true)
  })

  it('folds placed backend findings into the walk', () => {
    const editor = editorOf('The lantern guttered. A second lantern answered.')
    const finding: StyleFinding = {
      id: 'x1', rule: 'echo-word', severity: 'suggestion', chapter_index: 0,
      start_offset: 0, end_offset: 0, message: 'echo', snippet: 'lantern',
      prefix: 'second ', suffix: ' answered',
    }
    const detections = collectDetections(editor, [finding])
    const echo = detections.find(detection => detection.rule === 'echo-word')
    expect(echo?.finding?.id).toBe('x1')
    expect(echo?.before.endsWith('second ')).toBe(true)
  })

  it('counts what the typography fix would resolve', () => {
    const editor = editorOf('He paused--then "stopped" and waited...')
    const detections = collectDetections(editor, [])
    expect(typographyFixableCount(detections)).toBe(4)
  })
})

describe('detection keys and chapter indexing', () => {
  it('keys are stable across position shifts', () => {
    const [a] = collectDetections(editorOf('Its been a day.'), [])
    const [b] = collectDetections(editorOf('Filler before it. Its been a day.'), [])
    expect(a.rule).toBe('its-contraction')
    expect(detectionKey(a)).not.toBe('')
    // Same flagged text and local context, different positions — same key.
    expect(detectionKey({ ...a, from: 99, to: 102 })).toBe(detectionKey(a))
    expect(b.from).not.toBe(a.from)
  })

  it('maps sections onto the analysis chapter walk', () => {
    expect(globalChapterIndex('front_matter', 1, 2, 5)).toBe(1)
    expect(globalChapterIndex('body', 3, 2, 5)).toBe(5)
    expect(globalChapterIndex('back_matter', 0, 2, 5)).toBe(7)
    expect(globalChapterIndex('copyright', 0, 2, 5)).toBe(-1)
  })
})
