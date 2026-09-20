// Settings Dialog Constants

export const CLAUDE_MODELS = [
  { value: 'claude-opus-5',             label: 'Claude Opus 5 (most capable)' },
  { value: 'claude-sonnet-5',           label: 'Claude Sonnet 5 (recommended)' },
  { value: 'claude-haiku-4-5-20251001', label: 'Claude Haiku 4.5 (fastest)' },
]

export const OPENAI_MODELS = [
  { value: 'gpt-6-astra',   label: 'GPT-6 Astra (most capable)' },
  { value: 'gpt-5.6-sol',   label: 'GPT-5.6 Sol' },
  { value: 'gpt-5.6-terra', label: 'GPT-5.6 Terra (recommended)' },
  { value: 'gpt-5.6-luna',  label: 'GPT-5.6 Luna (fastest)' },
  { value: 'gpt-5.5',       label: 'GPT-5.5 (previous generation)' },
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
  claude: 'claude-sonnet-5',
  openai: 'gpt-5.6-terra',
}
