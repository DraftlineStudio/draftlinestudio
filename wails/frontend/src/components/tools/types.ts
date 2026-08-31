// Types for ToolsPanel and its sub-components

export type GlyphSection =
  | 'dashboard'
  | 'characters'
  | 'signals'
  | 'prose'
  | 'pacing'
  | 'chapters'
  | 'review'
  | 'aidetect'
  | 'issues'
  | 'ai'
  | null

export type AIMode = 'line_edit' | 'copy_edit' | 'expand' | 'smooth' | 'custom'

export type AIState = 'idle' | 'loading' | 'voice' | 'error'
