// Character visual identity: stable color + initials per name.
// Purely local — derived from the name text, no services involved.

import { AVATAR_PALETTE } from './accentColor'

export function characterColor(name: string): string {
  let h = 5381
  for (let i = 0; i < name.length; i++) h = ((h << 5) + h + name.charCodeAt(i)) | 0
  return AVATAR_PALETTE[Math.abs(h) % AVATAR_PALETTE.length]
}

export function characterInitials(name: string): string {
  const words = name.replace(/\./g, '').split(/\s+/).filter(Boolean)
  if (!words.length) return '?'
  if (words.length === 1) return words[0].slice(0, 2)
  return words[0][0] + words[words.length - 1][0]
}
