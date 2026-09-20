// Built-in routes. Anything else is the id of a provider the writer configured.
export const BUILT_IN_MODES = ['claudecode', 'codex', 'api'] as const

export type AIProviderMode = string
export type AIEditingTask = 'line_edit' | 'copy_edit' | 'expand' | 'smooth' | 'custom'
export type AITaskRoutes = Partial<Record<AIEditingTask, AIProviderMode>>

// A route pointing at a provider that has since been deleted falls back to the
// default rather than failing the task.
export function resolveTaskProvider(
  defaultProvider: AIProviderMode,
  routes: AITaskRoutes | null | undefined,
  task: AIEditingTask,
  providerIds: readonly string[] = [],
): AIProviderMode {
  const route = routes?.[task]
  if (!route) return defaultProvider
  const known: readonly string[] = [...BUILT_IN_MODES, ...providerIds]
  return known.includes(route) ? route : defaultProvider
}

export function setTaskProvider(
  routes: AITaskRoutes | null | undefined,
  task: AIEditingTask,
  provider: AIProviderMode | null,
): AITaskRoutes {
  const next = { ...(routes ?? {}) }
  if (provider) next[task] = provider
  else delete next[task]
  return next
}
