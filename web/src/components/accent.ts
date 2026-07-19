export type AccentColor = 'teal' | 'blue' | 'gold' | 'rose'

export const ACCENTS: AccentColor[] = ['teal', 'blue', 'gold', 'rose']

/** CSS-variable reference for an accent, for inline `style` (e.g. accent bars). */
export function accentVar(accent: AccentColor): string {
  return `var(--accent-${accent})`
}

/**
 * Canonical (theme-independent) hex for each accent, stored in `Topic.color`.
 * These are the dark-theme accent values; the UI renders the stored hex directly
 * so a chapter keeps its colour across themes (matching the B3 chapter rows).
 */
export const ACCENT_HEX: Record<AccentColor, string> = {
  teal: '#3ec5a0',
  blue: '#5b8def',
  gold: '#e8b765',
  rose: '#c2738f',
}

/** Nearest accent name for a stored hex; defaults to teal for anything unknown. */
export function accentFromHex(hex: string): AccentColor {
  const match = (Object.entries(ACCENT_HEX) as [AccentColor, string][]).find(
    ([, value]) => value.toLowerCase() === hex.toLowerCase(),
  )
  return match ? match[0] : 'teal'
}
