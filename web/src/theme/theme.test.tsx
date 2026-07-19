import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ThemeProvider } from './ThemeProvider'
import { useTheme } from './useTheme'
import { THEME_STORAGE_KEY } from './theme'

function Probe() {
  const { theme, toggle } = useTheme()
  return (
    <button type="button" onClick={toggle}>
      theme:{theme}
    </button>
  )
}

describe('ThemeProvider', () => {
  it('honours a stored theme and applies it to <html>', () => {
    localStorage.setItem(THEME_STORAGE_KEY, 'light')
    render(
      <ThemeProvider>
        <Probe />
      </ThemeProvider>,
    )
    expect(screen.getByRole('button')).toHaveTextContent('theme:light')
    expect(document.documentElement).toHaveAttribute('data-theme', 'light')
  })

  it('falls back to prefers-color-scheme when nothing is stored', () => {
    window.matchMedia = vi.fn().mockReturnValue({
      matches: true, // prefers light
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    }) as unknown as typeof window.matchMedia
    render(
      <ThemeProvider>
        <Probe />
      </ThemeProvider>,
    )
    expect(screen.getByRole('button')).toHaveTextContent('theme:light')
  })

  it('toggles and persists the choice', async () => {
    const user = userEvent.setup()
    localStorage.setItem(THEME_STORAGE_KEY, 'dark')
    render(
      <ThemeProvider>
        <Probe />
      </ThemeProvider>,
    )
    await user.click(screen.getByRole('button'))
    expect(document.documentElement).toHaveAttribute('data-theme', 'light')
    expect(localStorage.getItem(THEME_STORAGE_KEY)).toBe('light')
  })
})
