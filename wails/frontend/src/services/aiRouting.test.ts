import { describe, expect, it } from 'vitest'
import { resolveTaskProvider, setTaskProvider } from './aiRouting'

describe('AI task routing', () => {
  it('uses the default provider when a task has no override', () => {
    expect(resolveTaskProvider('codex', {}, 'line_edit')).toBe('codex')
  })

  it('routes each editing task independently', () => {
    const routes = { line_edit: 'claudecode', expand: 'api', smooth: 'local' } as const
    expect(resolveTaskProvider('codex', routes, 'line_edit')).toBe('claudecode')
    expect(resolveTaskProvider('codex', routes, 'copy_edit')).toBe('codex')
    expect(resolveTaskProvider('codex', routes, 'expand')).toBe('api')
    expect(resolveTaskProvider('codex', routes, 'smooth')).toBe('local')
  })

  it('can restore a task to the default without disturbing other routes', () => {
    const routes = setTaskProvider({ line_edit: 'claudecode', expand: 'api' }, 'line_edit', null)
    expect(routes).toEqual({ expand: 'api' })
  })
})
