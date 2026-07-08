// Types for ToolsPanel and its sub-components

export type GlyphSection = 'dashboard' | 'characters' | 'plot' | 'timeline' | 'beats' | 'foreshadow' | 'knowledge' | 'issues' | 'ai' | null

export type AIMode = 'line_edit' | 'expand' | 'smooth' | 'custom'

export type AIState = 'idle' | 'loading' | 'voice' | 'error'

// Utility function for generating unique IDs
export function genId(): string {
  return Math.random().toString(36).slice(2) + Date.now().toString(36)
}
