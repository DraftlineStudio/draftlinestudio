// Settings Dialog Constants

export const CLAUDE_MODELS = [
  { value: 'claude-opus-4-6',           label: 'Claude Opus 4.6 (most capable)' },
  { value: 'claude-sonnet-4-6',         label: 'Claude Sonnet 4.6 (recommended)' },
  { value: 'claude-haiku-4-5-20251001', label: 'Claude Haiku 4.5 (fastest)' },
]

export const OPENAI_MODELS = [
  { value: 'gpt-4o',      label: 'GPT-4o (recommended)' },
  { value: 'gpt-4o-mini', label: 'GPT-4o Mini (faster)' },
  { value: 'o3',          label: 'o3 (reasoning)' },
]

export const GEMINI_MODELS = [
  { value: 'gemini-1.5-pro',   label: 'Gemini 1.5 Pro (recommended)' },
  { value: 'gemini-1.5-flash', label: 'Gemini 1.5 Flash (faster)' },
  { value: 'gemini-2.0-flash', label: 'Gemini 2.0 Flash (latest)' },
]

export const GROK_MODELS = [
  { value: 'grok-2',      label: 'Grok 2 (recommended)' },
  { value: 'grok-beta',   label: 'Grok Beta' },
]

export const BOOK_FONTS = [
  'Merriweather', 'EB Garamond', 'Lora', 'Palatino Linotype', 'Georgia', 'Times New Roman',
]

export const TRIM_SIZES = [
  { value: '6x9',     label: '6″ × 9″ — Standard trade paperback' },
  { value: '5.5x8.5', label: '5.5″ × 8.5″ — Digest / literary fiction' },
  { value: '5x8',     label: '5″ × 8″ — Compact trade' },
  { value: '7x10',    label: '7″ × 10″ — Textbook / reference' },
  { value: 'A5',      label: 'A5 — 148 × 210 mm' },
  { value: 'A4',      label: 'A4 — 210 × 297 mm' },
]

export const DEFAULT_MODELS: Record<string, string> = {
  claude: 'claude-sonnet-4-6',
  openai: 'gpt-4o',
  gemini: 'gemini-1.5-pro',
  grok: 'grok-2',
}
