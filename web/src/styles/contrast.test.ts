import { describe, expect, it } from 'vitest'
import { ACCENT_HEX } from '../components/accent'
import tokensCss from './tokens.css?raw'

/*
 * WCAG AA contrast gate for the design tokens. The axe suite runs in jsdom,
 * which can't compute colour contrast, so palette regressions would otherwise
 * pass silently. This reads the hex values straight from tokens.css.
 */

type Theme = 'dark' | 'light'

function parseTheme(theme: Theme): Record<string, string> {
  const block = new RegExp(`:root\\[data-theme='${theme}'\\]\\s*\\{([^}]*)\\}`).exec(tokensCss)
  if (!block) throw new Error(`no ${theme} block in tokens.css`)
  const tokens: Record<string, string> = {}
  for (const [, name, hex] of block[1].matchAll(/--([\w-]+):\s*(#[0-9a-fA-F]{6})\b/g)) {
    tokens[name] = hex
  }
  return tokens
}

function luminance(hex: string): number {
  const [r, g, b] = [1, 3, 5].map((i) => {
    const c = parseInt(hex.slice(i, i + 2), 16) / 255
    return c <= 0.04045 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4
  })
  return 0.2126 * r + 0.7152 * g + 0.0722 * b
}

function contrast(a: string, b: string): number {
  const [hi, lo] = [luminance(a), luminance(b)].sort((x, y) => y - x)
  return (hi + 0.05) / (lo + 0.05)
}

/** Every surface readable text sits on (`surface` is also the page itself). */
const SURFACES = ['surface', 'surface-elevated', 'bar', 'input-bg', 'fill-muted']

/** Tokens used for readable text. (`text-faint` is decorative-only, so absent.) */
const TEXT = [
  'text',
  'text-2',
  'text-muted',
  'accent-teal-ink',
  'accent-teal-ink-hover',
  'accent-blue-ink',
  'accent-gold-ink',
  'accent-rose',
]

/** Ink on accent fills: [ink, fill]. */
const FILL_INK: [string, string][] = [
  ['btn-ink', 'accent-teal'],
  ['btn-ink', 'accent-teal-hover'],
  ['btn-ink', 'accent-gold'],
  ['btn-ink', 'accent-gold-hover'],
  ['danger-ink', 'accent-rose'],
  ['accent-gold-ink', 'fill-raised'], // active theme-toggle option
]

describe.each<Theme>(['dark', 'light'])('%s theme contrast (WCAG AA)', (theme) => {
  const t = parseTheme(theme)
  const hex = (name: string) => {
    const value = t[name]
    if (!value) throw new Error(`token --${name} missing in ${theme}`)
    return value
  }

  it.each(TEXT.flatMap((fg) => SURFACES.map((bg) => [fg, bg])))(
    'text --%s on --%s is at least 4.5:1',
    (fg, bg) => {
      expect(contrast(hex(fg), hex(bg))).toBeGreaterThanOrEqual(4.5)
    },
  )

  it.each(FILL_INK)('--%s on --%s is at least 4.5:1', (ink, fill) => {
    expect(contrast(hex(ink), hex(fill))).toBeGreaterThanOrEqual(4.5)
  })

  // The done-milestone check glyph sits on the topic's stored accent hex (the
  // same in both themes); icons need 3:1.
  it.each(Object.entries(ACCENT_HEX))(
    'done check on the %s topic accent is at least 3:1',
    (_, fill) => {
      expect(contrast(hex('btn-ink'), fill)).toBeGreaterThanOrEqual(3)
    },
  )
})
