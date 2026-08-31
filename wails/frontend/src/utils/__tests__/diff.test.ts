import { describe, expect, it } from 'vitest'
import {
  diffContent,
  extractChanges,
  assembleFromChanges,
  type DiffChange,
} from '../diff'

/** Decide every change the same way (accept vs reject) and assemble. */
function assembleAll(originalHtml: string, revisedHtml: string, accept: boolean): string {
  const diffs = diffContent(originalHtml, revisedHtml)
  const changes = extractChanges(diffs).map<DiffChange>(c => ({ ...c, accepted: accept, decided: true }))
  return assembleFromChanges(diffs, changes)
}

describe('diffContent formatting preservation', () => {
  it('keeps untouched paragraphs verbatim (incl. <em>/<strong>/<a>) when one paragraph changes', () => {
    const original =
      '<p>The <strong>first</strong> paragraph.</p>' +
      '<p>A <em>second</em> with a <a href="https://x.test">link</a>.</p>' +
      '<p>The third paragraph is quiet.</p>'
    const revised =
      '<p>The first paragraph.</p>' + // AI rewrote paragraph 1 (plain text)
      '<p>A second with a link.</p>' +
      '<p>The third paragraph is calm.</p>' // word change in paragraph 3

    const diffs = diffContent(original, revised)
    // Accept ONLY the change in the third paragraph; leave the first alone.
    const changes = extractChanges(diffs)
    // Paragraph 1 stripped==revised stripped? No: <strong>first</strong> -> "first"
    // stripped text equals revised "first", so paragraph 1 has NO change at all.
    // Paragraph 2 likewise stripped equals revised -> no change.
    // Only paragraph 3 ("quiet" -> "calm") is a real change.
    expect(changes.length).toBe(1)

    const accepted = changes.map<DiffChange>(c => ({ ...c, accepted: true, decided: true }))
    const result = assembleFromChanges(diffs, accepted)

    // Untouched paragraphs 1 and 2 keep their EXACT original inline HTML.
    expect(result).toContain('<p>The <strong>first</strong> paragraph.</p>')
    expect(result).toContain('<p>A <em>second</em> with a <a href="https://x.test">link</a>.</p>')
    // Paragraph 3 got the accepted revision.
    expect(result).toContain('calm')
    expect(result).not.toContain('quiet')
  })

  it('reject-all yields byte-identical original HTML even with inserts and deletes', () => {
    const original =
      '<p>Alpha with <strong>bold</strong>.</p>' +
      '<p>Beta paragraph.</p>' +
      '<p>Gamma <em>italic</em> line.</p>'
    // Revised: rewords Beta, inserts a new paragraph, deletes Gamma.
    const revised =
      '<p>Alpha with bold.</p>' +
      '<p>Beta paragraph reworded.</p>' +
      '<p>Brand new inserted paragraph.</p>'

    const result = assembleAll(original, revised, /* accept */ false)
    expect(result).toBe(original)
  })

  it('reject-all with only a word change reproduces the original exactly', () => {
    const original = '<p>Keep <em>this</em> exact.</p><p>And <a href="/l">links</a> too.</p>'
    const revised = '<p>Keep this precise.</p><p>And links too.</p>'
    expect(assembleAll(original, revised, false)).toBe(original)
  })

  it('does not mark following paragraphs changed when a paragraph is inserted', () => {
    const original =
      '<p>One.</p>' +
      '<p>Two.</p>' +
      '<p>Three.</p>'
    const revised =
      '<p>One.</p>' +
      '<p>Inserted between one and two.</p>' +
      '<p>Two.</p>' +
      '<p>Three.</p>'

    const diffs = diffContent(original, revised)
    const changed = diffs.filter(d => d.hasChanges)
    // Exactly one change row: the inserted paragraph. "Two." and "Three." stay equal.
    expect(changed.length).toBe(1)
    expect(changed[0].originalHtml).toBe('') // pure insertion
    expect(changed[0].revisedHtml).toBe('<p>Inserted between one and two.</p>')

    // Only one accept/reject change should be surfaced to the user.
    expect(extractChanges(diffs).length).toBe(1)
  })

  it('accept-all preserves the revised HTML verbatim for whole-paragraph rewrites', () => {
    const original = '<p>Old one.</p><p>Old two.</p>'
    const revised = '<p>New <strong>one</strong>.</p><p>New two.</p>'
    const result = assembleAll(original, revised, /* accept */ true)
    // Whole-paragraph rewrites accepted -> revised HTML kept verbatim, formatting intact.
    expect(result).toContain('<p>New <strong>one</strong>.</p>')
  })

  it('rejecting an inserted paragraph drops it, accepting keeps its verbatim HTML', () => {
    const original = '<p>Only.</p>'
    const revised = '<p>Only.</p><p>Added <em>tail</em>.</p>'

    // Reject -> original only.
    expect(assembleAll(original, revised, false)).toBe('<p>Only.</p>')
    // Accept -> inserted paragraph kept verbatim (formatting intact).
    expect(assembleAll(original, revised, true)).toBe('<p>Only.</p><p>Added <em>tail</em>.</p>')
  })
})
