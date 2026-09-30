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

export const DEFAULT_MODELS: Record<string, string> = {
  claude: 'claude-sonnet-5',
  openai: 'gpt-5.6-terra',
}
