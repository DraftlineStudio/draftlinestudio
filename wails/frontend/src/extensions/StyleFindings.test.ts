import { describe, expect, it } from 'vitest'
import { getSchema } from '@tiptap/core'
import StarterKit from '@tiptap/starter-kit'
import type { StyleFinding } from '../types/draftline'
import { placeStyleFindings } from './StyleFindings'

const schema = getSchema([StarterKit])

function docOf(...paragraphs: string[]) {
  return schema.nodeFromJSON({
    type: 'doc',
    content: paragraphs.map(text => ({
      type: 'paragraph',
      content: text ? [{ type: 'text', text }] : [],
    })),
  })
}

function finding(partial: Partial<StyleFinding>): StyleFinding {
  return {
    id: 'f1', rule: 'echo-word', severity: 'suggestion', chapter_index: 0,
    start_offset: 0, end_offset: 0, message: 'test', snippet: '',
    ...partial,
  }
}

describe('placeStyleFindings', () => {
  it('places a finding by its context and covers only the snippet', () => {
    const doc = docOf('The lantern guttered. A second lantern answered.')
    const placed = placeStyleFindings(doc, [finding({
      snippet: 'lantern', prefix: 'second ', suffix: ' answered',
    })])
    expect(placed).toHaveLength(1)
    expect(doc.textBetween(placed[0].from, placed[0].to)).toBe('lantern')
    const before = doc.textBetween(placed[0].from - 7, placed[0].from)
    expect(before).toBe('second ')
  })

  it('drops a finding whose context matches nowhere or more than once', () => {
    const doc = docOf('A lantern here.', 'A lantern here.')
    expect(placeStyleFindings(doc, [finding({ snippet: 'lantern', prefix: 'A ', suffix: ' here' })]))
      .toHaveLength(0)
    expect(placeStyleFindings(doc, [finding({ snippet: 'beacon', prefix: 'A ', suffix: ' here' })]))
      .toHaveLength(0)
  })

  it('places findings in different paragraphs independently', () => {
    const doc = docOf('The rope held fast.', 'The rope frayed by noon.')
    const placed = placeStyleFindings(doc, [
      finding({ id: 'a', snippet: 'rope', prefix: 'The ', suffix: ' held' }),
      finding({ id: 'b', snippet: 'rope', prefix: 'The ', suffix: ' frayed' }),
    ])
    expect(placed).toHaveLength(2)
    expect(placed.map(p => doc.textBetween(p.from, p.to))).toEqual(['rope', 'rope'])
    expect(placed[0].from).toBeLessThan(placed[1].from)
  })
})
