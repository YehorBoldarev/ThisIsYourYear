import { describe, expect, it } from 'vitest'
import { http, HttpResponse } from 'msw'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { server, dashboardResponse, topicResponse, tokenResponse } from '../../test/msw'
import { AuthProvider } from '../../auth/AuthProvider'
import { ModalHost } from '../topics/ModalHost'
import { Dashboard } from './Dashboard'

function renderDashboard() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  // Authenticate so the greeting has a name.
  server.use(http.post('/api/v1/auth/refresh', () => HttpResponse.json(tokenResponse())))
  return render(
    <MemoryRouter>
      <QueryClientProvider client={qc}>
        <AuthProvider>
          <ModalHost>
            <Dashboard />
          </ModalHost>
        </AuthProvider>
      </QueryClientProvider>
    </MemoryRouter>,
  )
}

describe('Dashboard', () => {
  it('shows the skeleton while pending', () => {
    renderDashboard()
    expect(screen.getByTestId('dashboard-skeleton')).toBeInTheDocument()
  })

  it('renders the greeting, year bar, and chapters from real data', async () => {
    server.use(
      http.get('/api/v1/dashboard', () =>
        HttpResponse.json(
          dashboardResponse({ topics: { total: 1, completed: 0 }, message: 'Keep showing up.' }),
        ),
      ),
      http.get('/api/v1/topics', () => HttpResponse.json([topicResponse()])),
    )
    renderDashboard()
    expect(await screen.findByText(/Nostradamus/)).toBeInTheDocument()
    expect(screen.getByText('Keep showing up.')).toBeInTheDocument()
    expect(screen.getByText(/You are here ·/)).toBeInTheDocument()
    // Both desktop row and mobile card render the topic (one hidden via CSS).
    expect(screen.getAllByText('Run a Half-Marathon').length).toBeGreaterThan(0)
    expect(screen.getAllByText('75%').length).toBeGreaterThan(0)
  })

  it('marks a 100% chapter as achieved', async () => {
    server.use(
      http.get('/api/v1/dashboard', () =>
        HttpResponse.json(dashboardResponse({ topics: { total: 1, completed: 1 } })),
      ),
      http.get('/api/v1/topics', () =>
        HttpResponse.json([topicResponse({ progress: { total: 8, completed: 8, ratio: 1 } })]),
      ),
    )
    renderDashboard()
    await waitFor(() => expect(screen.getAllByText('Complete').length).toBeGreaterThan(0))
  })

  it('shows the empty state when there are no chapters', async () => {
    server.use(
      http.get('/api/v1/dashboard', () => HttpResponse.json(dashboardResponse())),
      http.get('/api/v1/topics', () => HttpResponse.json([])),
    )
    renderDashboard()
    expect(await screen.findByText('Your year is a blank page.')).toBeInTheDocument()
  })

  it('counts down to the year being planned during the preparation stage', async () => {
    server.use(
      http.get('/api/v1/dashboard', () =>
        HttpResponse.json(
          dashboardResponse({
            year: 2027,
            phase: 'preparation',
            topics: { total: 1, completed: 0 },
            message: 'Every goal you plan now is a head start. 2027 is almost here.',
            countdown: { target: '2027-01-01T00:00:00Z', secondsRemaining: 4_000_000 },
          }),
        ),
      ),
      http.get('/api/v1/topics', () => HttpResponse.json([topicResponse({ year: 2027 })])),
    )
    renderDashboard()
    expect(await screen.findByRole('region', { name: 'Countdown to 2027' })).toBeInTheDocument()
    expect(screen.getByText('2027 begins in')).toBeInTheDocument()
    expect(screen.getByText(/2027 is almost here/)).toBeInTheDocument()
    expect(screen.queryByText(/You are here ·/)).not.toBeInTheDocument()
    expect(screen.getAllByText('Run a Half-Marathon').length).toBeGreaterThan(0)
  })

  it('shows the countdown in the empty state while preparing', async () => {
    server.use(
      http.get('/api/v1/dashboard', () =>
        HttpResponse.json(
          dashboardResponse({
            year: 2027,
            phase: 'preparation',
            countdown: { target: '2027-01-01T00:00:00Z', secondsRemaining: 4_000_000 },
          }),
        ),
      ),
      http.get('/api/v1/topics', () => HttpResponse.json([])),
    )
    renderDashboard()
    expect(await screen.findByText('Your 2027 is a blank page.')).toBeInTheDocument()
    expect(screen.getByRole('region', { name: 'Countdown to 2027' })).toBeInTheDocument()
  })

  it('shows an error state with retry when a query fails', async () => {
    server.use(
      http.get('/api/v1/dashboard', () =>
        HttpResponse.json({ error: { code: 'internal', message: 'boom' } }, { status: 500 }),
      ),
      http.get('/api/v1/topics', () => HttpResponse.json([])),
    )
    renderDashboard()
    expect(await screen.findByText(/load your year/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Try again' })).toBeInTheDocument()
  })
})
