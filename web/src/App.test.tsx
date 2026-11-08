import { describe, expect, it } from 'vitest'
import { http, HttpResponse } from 'msw'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { server, tokenResponse } from './test/msw'
import { AuthProvider } from './auth/AuthProvider'
import { ThemeProvider } from './theme/ThemeProvider'
import App from './App'

function renderApp(path: string) {
  server.use(http.post('/api/v1/auth/refresh', () => HttpResponse.json(tokenResponse())))
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <ThemeProvider>
      <QueryClientProvider client={qc}>
        <MemoryRouter initialEntries={[path]}>
          <AuthProvider>
            <App />
          </AuthProvider>
        </MemoryRouter>
      </QueryClientProvider>
    </ThemeProvider>,
  )
}

describe('App', () => {
  it('mobile "New" opens the create popup over the dashboard from any page', async () => {
    renderApp('/achievements')
    expect(await screen.findByRole('heading', { name: 'Your achievements' })).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'New' }))

    expect(await screen.findByRole('dialog', { name: 'Create a big goal' })).toBeInTheDocument()
    // It moved to the dashboard, so the saved chapter appears right behind the popup.
    expect(screen.queryByRole('heading', { name: 'Your achievements' })).not.toBeInTheDocument()
  })
})
