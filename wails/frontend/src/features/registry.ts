import type { AppSettings } from '../store/appStore'

export type FeatureSettingKey =
  | 'spell_check_enabled'
  | 'grammar_check_enabled'
  | 'cast_enabled'
  | 'story_bible_enabled'
  | 'plot_walker_enabled'
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
}

export const FEATURE_REGISTRY: DraftlineFeature[] = [
  {
    id: 'draftline.spelling', settingKey: 'spell_check_enabled', name: 'Spelling',
    vendor: 'Draftline', version: '1.0', category: 'Writing', monogram: 'Sp', accent: '#d65454', bundled: true,
    description: 'Offline spelling diagnostics, suggestions, and a personal dictionary.',
  },
  {
    id: 'draftline.grammar', settingKey: 'grammar_check_enabled', name: 'Grammar Check',
    vendor: 'Draftline', version: '1.0', category: 'Writing', monogram: 'Gr', accent: '#4f83db', bundled: true,
    description: 'Fast, privacy-friendly checks for high-confidence grammar and usage issues.',
  },
  {
    id: 'draftline.cast', settingKey: 'cast_enabled', name: 'Cast',
    vendor: 'Draftline', version: '1.0', category: 'Story tools', monogram: 'Ca', accent: '#ba6ed6', bundled: true,
    description: 'Character quick reference, entity indexing, and the visual cast workspace.',
  },
  {
    id: 'draftline.story-bible', settingKey: 'story_bible_enabled', name: 'Story Bible',
    vendor: 'Draftline', version: '1.0', category: 'Story tools', monogram: 'Sb', accent: '#4fa879', bundled: true,
    description: 'Plot notes and a timeline that stay beside the manuscript.',
  },
  {
    id: 'draftline.plot-walker', settingKey: 'plot_walker_enabled', name: 'Plot Walker',
    vendor: 'Draftline', version: '1.0', category: 'Story tools', monogram: 'Pw', accent: '#d39a45', bundled: true,
    description: 'Beat sheets, foreshadowing, knowledge tracking, and story issue review.',
  },
  {
    id: 'draftline.ai-studio', settingKey: 'ai_enabled', name: 'AI Studio',
    vendor: 'Draftline', version: '1.0', category: 'Intelligence', monogram: 'Ai', accent: '#7c6ee6', bundled: true,
    description: 'Optional assisted rewriting and inline generation using your selected provider.',
  },
]

export function isFeatureEnabled(settings: AppSettings, key: FeatureSettingKey): boolean {
  return settings[key]
}
