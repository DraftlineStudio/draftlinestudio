// Constants for ToolsPanel and its sub-components

import type { WritingStyleOptions } from '../../types/draftline'
import type { GlyphSection, AIMode } from './types'

// AI mode options for the AI Studio
export const AI_MODES: { id: AIMode; label: string; desc: string }[] = [
  { id: 'line_edit', label: 'Line Edit', desc: 'Selective style, rhythm, tone, and readability' },
  { id: 'copy_edit', label: 'Copy Edit', desc: 'Spelling, grammar, punctuation, and consistency' },
  { id: 'expand', label: 'Expand', desc: 'Add detail, texture, show vs. tell' },
  { id: 'smooth', label: 'Smooth', desc: 'Remove repetition, improve flow' },
  { id: 'custom', label: 'Custom', desc: 'Use @ai prompts in your text' },
]

// Style feature definitions for the mixer
export const STYLE_FEATURES: { key: keyof WritingStyleOptions; label: string; desc: string }[] = [
  { key: 'metaphors', label: 'Metaphors', desc: 'Figurative comparisons' },
  { key: 'similes', label: 'Similes', desc: '"Like" and "as" comparisons' },
  { key: 'sensory_detail', label: 'Sensory Detail', desc: 'Sight, sound, smell, touch, taste' },
  { key: 'internal_thought', label: 'Internal Thought', desc: 'Character introspection' },
  { key: 'dialogue', label: 'Dialogue', desc: 'Conversation expansion' },
  { key: 'action', label: 'Action', desc: 'Physical beats, movement' },
  { key: 'description', label: 'Description', desc: 'Setting and atmosphere' },
  { key: 'pacing', label: 'Pacing', desc: 'Sentence rhythm variation' },
]

// Intensity labels for style sliders
export const INTENSITY_LABELS = ['Off', 'Subtle', 'Moderate', 'Heavy']

// Glyph section configuration for the sidebar
export const SECTION_CONFIG: { id: Exclude<GlyphSection, null>; label: string; tooltip: string }[] = [
  { id: 'dashboard', label: 'Dashboard', tooltip: 'Writing Dashboard' },
  { id: 'characters', label: 'Characters', tooltip: 'Characters' },
  { id: 'prose', label: 'Prose', tooltip: 'Prose' },
  { id: 'pacing', label: 'Pacing', tooltip: 'Pacing' },
  { id: 'chapters', label: 'Chapters', tooltip: 'Chapters' },
  { id: 'review', label: 'Review', tooltip: 'Worth Reviewing' },
  { id: 'issues', label: 'Issues', tooltip: 'Story Analysis' },
  { id: 'ai', label: 'AI', tooltip: 'AI Studio' },
]
