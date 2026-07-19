export type Theme = 'light' | 'dark'

export const THEME_STORAGE_KEY = 'tiyy-theme'

/**
 * Resolve the theme to use on first render: a valid stored choice wins,
 * otherwise fall back to the OS preference, otherwise dark. Kept in sync with
 * the inline no-flash script in index.html.
 */
export function resolveInitialTheme(): Theme {
  try {
    const stored = localStorage.getItem(THEME_STORAGE_KEY)
    if (stored === 'light' || stored === 'dark') return stored
    if (window.matchMedia('(prefers-color-scheme: light)').matches) return 'light'
  } catch {
    // localStorage/matchMedia unavailable (e.g. SSR/tests) — fall through.
  }
  return 'dark'
}

/** Apply the theme to <html> so the token overrides take effect. */
export function applyTheme(theme: Theme): void {
  document.documentElement.setAttribute('data-theme', theme)
}
