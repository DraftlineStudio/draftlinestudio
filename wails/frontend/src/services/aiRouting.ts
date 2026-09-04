export type AIProviderMode = 'claudecode' | 'codex' | 'api' | 'local'
export type AIEditingTask = 'line_edit' | 'copy_edit' | 'expand' | 'smooth' | 'custom'
export type AITaskRoutes = Partial<Record<AIEditingTask, AIProviderMode>>

const PROVIDER_MODES = new Set<AIProviderMode>(['claudecode', 'codex', 'api', 'local'])

export function resolveTaskProvider(
  defaultProvider: AIProviderMode,
  routes: AITaskRoutes | null | undefined,
  task: AIEditingTask,
): AIProviderMode {
  const route = routes?.[task]
  return route && PROVIDER_MODES.has(route) ? route : defaultProvider
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

