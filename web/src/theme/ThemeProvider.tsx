import { useCallback, useEffect, useMemo, useState } from 'react'
import type { ReactNode } from 'react'
import { applyTheme, resolveInitialTheme, THEME_STORAGE_KEY, type Theme } from './theme'
import { ThemeContext } from './ThemeContext'

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [theme, setThemeState] = useState<Theme>(resolveInitialTheme)

  // Reflect the choice onto <html> and persist it whenever it changes.
  useEffect(() => {
    applyTheme(theme)
    try {
      localStorage.setItem(THEME_STORAGE_KEY, theme)
    } catch {
      // Persistence is best-effort; ignore storage failures.
    }
  }, [theme])

  const setTheme = useCallback((next: Theme) => setThemeState(next), [])
  const toggle = useCallback(() => setThemeState((t) => (t === 'dark' ? 'light' : 'dark')), [])

  const value = useMemo(() => ({ theme, setTheme, toggle }), [theme, setTheme, toggle])

  return <ThemeContext value={value}>{children}</ThemeContext>
}
