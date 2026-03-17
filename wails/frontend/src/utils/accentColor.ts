export const AVATAR_PALETTE = [
  '#E06C75','#E5A25A','#E5C07B','#98C379','#56B6C2',
  '#61AFEF','#C678DD','#F28B82','#73C991','#78B7D0',
  '#FABF73','#A78BFA','#F472B6','#34D399','#60A5FA',
  '#E879F9','#FACC15','#2DD4BF','#818CF8','#FB923C',
  '#4ADE80','#38BDF8','#F87171','#A3E635','#C084FC','#FDD663',
]

export function avatarColor(title: string): string {
  const code = title.toUpperCase().charCodeAt(0)
  return AVATAR_PALETTE[Math.abs(code - 32) % AVATAR_PALETTE.length]
}

export function hexToRgba(hex: string, alpha: number): string {
  const r = parseInt(hex.slice(1, 3), 16)
  const g = parseInt(hex.slice(3, 5), 16)
  const b = parseInt(hex.slice(5, 7), 16)
  return `rgba(${r},${g},${b},${alpha})`
}

export function applyAccent(title: string) {
  const color = avatarColor(title)
  const el = document.documentElement
  el.style.setProperty('--accent',        color)
  el.style.setProperty('--accent-dim',    hexToRgba(color, 0.15))
  el.style.setProperty('--accent-subtle', hexToRgba(color, 0.08))
  el.style.setProperty('--border-focus',  color)
}

export function clearAccent() {
  const el = document.documentElement
  el.style.removeProperty('--accent')
  el.style.removeProperty('--accent-dim')
  el.style.removeProperty('--accent-subtle')
  el.style.removeProperty('--border-focus')
}
