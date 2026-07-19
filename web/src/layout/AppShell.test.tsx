import { describe, expect, it } from 'vitest'
import { http, HttpResponse } from 'msw'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { AppShell } from './AppShell'
import { ThemeProvider } from '../theme/ThemeProvider'
import { AuthProvider } from '../auth/AuthProvider'
import { THEME_STORAGE_KEY, type Theme } from '../theme/theme'
import { server, TEST_USER, tokenResponse } from '../test/msw'

async function renderShell(theme: Theme) {
  localStorage.setItem(THEME_STORAGE_KEY, theme)
  server.use(http.post('/api/v1/auth/refresh', () => HttpResponse.json(tokenResponse())))
  render(
    <ThemeProvider>
      <AuthProvider>
        <MemoryRouter>
          <AppShell>
            <p>content</p>
          </AppShell>
        </MemoryRouter>
      </AuthProvider>
    </ThemeProvider>,
  )
  // Wait for the boot refresh to authenticate and the shell to render.
  await waitFor(() => expect(screen.getByText('content')).toBeInTheDocument())
}

describe('AppShell', () => {
  it.each<Theme>(['dark', 'light'])('renders chrome + content in %s theme', async (theme) => {
    await renderShell(theme)
    expect(document.documentElement).toHaveAttribute('data-theme', theme)
    expect(screen.getAllByRole('navigation', { name: 'Primary' }).length).toBeGreaterThanOrEqual(1)
    expect(screen.getByText('Since 2026')).toBeInTheDocument()
    // The identity block shows the authenticated user.
    expect(screen.getAllByText(TEST_USER.displayName).length).toBeGreaterThan(0)
  })

  it('exposes a skip link that targets the main landmark', async () => {
    await renderShell('dark')
    const skip = screen.getByRole('link', { name: 'Skip to content' })
    expect(skip).toHaveAttribute('href', '#main')
    const main = document.getElementById('main')
    expect(main?.tagName).toBe('MAIN')
  })

  it('marks the active nav item with aria-current (react-router default)', async () => {
    await renderShell('dark') // MemoryRouter starts at '/'
    const current = screen
      .getAllByRole('link')
      .filter((l) => l.getAttribute('aria-current') === 'page')
    expect(current.length).toBeGreaterThan(0)
    expect(current.some((l) => /Dashboard|Goals/.test(l.textContent ?? ''))).toBe(true)
  })
})
