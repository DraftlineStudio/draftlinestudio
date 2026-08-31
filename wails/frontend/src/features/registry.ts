import type { AppSettings } from '../store/appStore'

export type FeatureSettingKey =
  | 'spell_check_enabled'
  | 'grammar_check_enabled'
  | 'cast_enabled'
  | 'analysis_enabled'
  | 'ai_enabled'

export interface DraftlineFeature {
  id: string
  settingKey: FeatureSettingKey
  name: string
  vendor: string
  version: string
  category: 'Writing' | 'Story tools' | 'Intelligence'
  description: string
  monogram: string
  accent: string
  bundled: boolean
  capabilities: string[]
  resourceProfile: 'tiny' | 'light' | 'optional-model'
}

export const FEATURE_REGISTRY: DraftlineFeature[] = [
  {
    id: 'draftline.spelling', settingKey: 'spell_check_enabled', name: 'Spelling',
    vendor: 'Draftline', version: '1.0', category: 'Writing', monogram: 'Sp', accent: '#d65454', bundled: true,
    description: 'Offline spelling diagnostics, suggestions, and a personal dictionary.',
    capabilities: ['editor.diagnostics.spelling'], resourceProfile: 'light',
  },
  {
    id: 'draftline.grammar', settingKey: 'grammar_check_enabled', name: 'Grammar Check',
    vendor: 'Draftline', version: '1.0', category: 'Writing', monogram: 'Gr', accent: '#4f83db', bundled: true,
    description: 'Fast, privacy-friendly checks for high-confidence grammar and usage issues.',
    capabilities: ['editor.diagnostics.grammar'], resourceProfile: 'tiny',
  },
  {
    id: 'draftline.cast', settingKey: 'cast_enabled', name: 'Characters',
    vendor: 'Draftline', version: '1.0', category: 'Story tools', monogram: 'Ch', accent: '#ba6ed6', bundled: true,
    description: 'Character codex: entity indexing, chapter presence, and relationship insight.',
    capabilities: ['analysis.characters', 'analysis.relationships'], resourceProfile: 'light',
  },
  {
    id: 'draftline.story-analysis', settingKey: 'analysis_enabled', name: 'Story Analysis',
    vendor: 'Draftline', version: '1.0', category: 'Intelligence', monogram: 'An', accent: '#4ba39a', bundled: true,
    description: 'Private Prose-powered chapter structure, pacing, readability, dialogue, and keyword analysis.',
    capabilities: ['analysis.structure', 'analysis.pacing', 'analysis.keywords', 'analysis.summary'], resourceProfile: 'light',
  },
  {
    id: 'draftline.ai-studio', settingKey: 'ai_enabled', name: 'AI Studio',
    vendor: 'Draftline', version: '1.0', category: 'Intelligence', monogram: 'Ai', accent: '#7c6ee6', bundled: true,
    description: 'Optional assisted rewriting and inline generation using your selected provider.',
    capabilities: ['editor.rewrite', 'editor.generate'], resourceProfile: 'optional-model',
  },
]

export function isFeatureEnabled(settings: AppSettings, key: FeatureSettingKey): boolean {
  return settings[key]
}
