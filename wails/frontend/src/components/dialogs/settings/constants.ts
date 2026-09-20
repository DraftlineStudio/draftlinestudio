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
}
