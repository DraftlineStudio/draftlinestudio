// The export flow's line art, in one place because the same five shapes are
// drawn on four different screens. Every path is stroked in currentColor; the
// stylesheet decides the weight and the colour.

import type { FlowOutput } from './exportFlow'

const PATHS: Record<FlowOutput, string> = {
  epub: 'M4 5.5A2.5 2.5 0 016.5 3H11v16H6.5A2.5 2.5 0 004 21.5v-16zM20 5.5A2.5 2.5 0 0017.5 3H13v16h4.5a2.5 2.5 0 012.5 2.5v-16z',
  docx: 'M6 2.75h8l4 4V21.25H6zM14 3v4h4M8.5 11l1.25 5 1.4-3.7 1.35 3.7 1.25-5',
  pdf: 'M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8zM14 2v6h6M8 13h8M8 17h5',
  'print-pdf': 'M6 9V2h12v7M6 18H4a2 2 0 01-2-2v-5a2 2 0 012-2h16a2 2 0 012 2v5a2 2 0 01-2 2h-2M6 14h12v8H6z',
  hc: 'M6 9V2h12v7M6 18H4a2 2 0 01-2-2v-5a2 2 0 012-2h16a2 2 0 012 2v5a2 2 0 01-2 2h-2M6 14h12v8H6z',
  audio: 'M12 2a3 3 0 00-3 3v7a3 3 0 006 0V5a3 3 0 00-3-3zM19 10v2a7 7 0 01-14 0v-2M12 19v3M8 22h8',
}

export function FormatGlyph({ output }: { output: FlowOutput }) {
  return <svg className="export-glyph" viewBox="0 0 24 24" aria-hidden="true"><path d={PATHS[output]} /></svg>
}

export function TickGlyph() {
  return <svg className="export-glyph tick" viewBox="0 0 20 20" aria-hidden="true"><path d="M4 10.5l3.5 3.5L16 5.5" /></svg>
}

export function LockGlyph() {
  return <svg className="export-glyph" viewBox="0 0 24 24" aria-hidden="true">
    <path d="M6 10.5h12v10H6zM8.5 10.5V7a3.5 3.5 0 017 0v3.5" />
  </svg>
}

export function CloseGlyph() {
  return <svg className="export-glyph" viewBox="0 0 24 24" aria-hidden="true"><path d="M6 6l12 12M18 6L6 18" /></svg>
}

export function SparkGlyph() {
  return <svg className="export-glyph" viewBox="0 0 24 24" aria-hidden="true">
    <path d="M12 3v4M12 17v4M4.9 7.5l2.9 2.9M16.2 13.6l2.9 2.9M3 14h4M17 10h4M7.8 16.5l-2.9 2.9M18.1 7.4l-2.9 2.9" />
  </svg>
}

export function StackGlyph() {
  return <svg className="export-glyph" viewBox="0 0 24 24" aria-hidden="true">
    <path d="M12 3l9 5-9 5-9-5 9-5zM3 13l9 5 9-5M3 17l9 5 9-5" />
  </svg>
}

export function InfoGlyph() {
  return <svg className="export-glyph" viewBox="0 0 24 24" aria-hidden="true">
    <circle cx="12" cy="12" r="9" /><path d="M12 11v5M12 8h.01" />
  </svg>
}
