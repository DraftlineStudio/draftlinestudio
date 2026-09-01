import { describe, expect, it } from 'vitest'
import { buildSignalSummary } from './SignalsPanel'

describe('Signals summary copy', () => {
  it('describes measured surface patterns without inferring an audience', () => {
    const summary = buildSignalSummary(9.6, 24.7, 'brisk')

    expect(summary).toBe('Short sentences, moderate dialogue, brisk prose tempo.')
    expect(summary).not.toMatch(/grade|child|young adult|audience|easy|difficult/i)
  })
})
