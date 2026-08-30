// Constants for ToolsPanel and its sub-components

import type { WritingStyleOptions, BeatType } from '../../types/draftline'
import type { GlyphSection, AIMode } from './types'

// AI mode options for the AI Studio
export const AI_MODES: { id: AIMode; label: string; desc: string }[] = [
  { id: 'line_edit', label: 'Line Edit', desc: 'Prose rhythm and sentence variety' },
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
  { id: 'plot', label: 'Plot', tooltip: 'Plot Notes' },
  { id: 'timeline', label: 'Timeline', tooltip: 'Story Timeline' },
  { id: 'beats', label: 'Beats', tooltip: 'Beat Sheet' },
  { id: 'foreshadow', label: 'Foreshadow', tooltip: 'Foreshadowing Ledger' },
  { id: 'knowledge', label: 'Knowledge', tooltip: 'Knowledge Matrix' },
  { id: 'issues', label: 'Issues', tooltip: 'Story Analysis' },
  { id: 'ai', label: 'AI', tooltip: 'AI Studio' },
]

// Beat types for the Beat Sheet (Save the Cat structure)
export const BEAT_TYPES: { value: BeatType | string; label: string }[] = [
  { value: 'opening_image', label: 'Opening Image' },
  { value: 'theme_stated', label: 'Theme Stated' },
  { value: 'setup', label: 'Setup' },
  { value: 'catalyst', label: 'Catalyst' },
  { value: 'debate', label: 'Debate' },
  { value: 'break_into_two', label: 'Break into Two' },
  { value: 'b_story', label: 'B Story' },
  { value: 'fun_and_games', label: 'Fun & Games' },
  { value: 'midpoint', label: 'Midpoint' },
  { value: 'bad_guys_close_in', label: 'Bad Guys Close In' },
  { value: 'all_is_lost', label: 'All Is Lost' },
  { value: 'dark_night', label: 'Dark Night' },
  { value: 'break_into_three', label: 'Break into Three' },
  { value: 'finale', label: 'Finale' },
  { value: 'final_image', label: 'Final Image' },
  { value: 'custom', label: 'Custom' },
]
