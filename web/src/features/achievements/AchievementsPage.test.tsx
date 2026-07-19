import { describe, expect, it } from 'vitest'
import { http, HttpResponse } from 'msw'
import { render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { goalResponse, server, topicResponse } from '../../test/msw'
import { ThemeProvider } from '../../theme/ThemeProvider'
import { AchievementsPage } from './AchievementsPage'

function renderPage() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <ThemeProvider>
      <QueryClientProvider client={qc}>
        <AchievementsPage />
      </QueryClientProvider>
    </ThemeProvider>,
  )
}

describe('AchievementsPage', () => {
  it('lists completed, starred milestones and hides the rest', async () => {
    server.use(
      http.get('/api/v1/topics', () =>
        HttpResponse.json([topicResponse({ id: 't1', title: 'Run a Half-Marathon' })]),
      ),
      http.get('/api/v1/topics/:id/goals', () =>
        HttpResponse.json([
          goalResponse({
            id: 'g1',
            title: 'First 10k run',
            status: 'done',
            isAchievement: true,
            completedAt: '2026-05-01T00:00:00Z',
          }),
          goalResponse({ id: 'g2', title: 'Buy running shoes', status: 'done' }),
          goalResponse({ id: 'g3', title: 'Register', status: 'todo', isAchievement: true }),
        ]),
      ),
    )
    renderPage()

    expect(await screen.findByText('First 10k run')).toBeInTheDocument()
    expect(screen.getByText(/1 moment worth keeping/i)).toBeInTheDocument()
    expect(screen.queryByText('Buy running shoes')).not.toBeInTheDocument()
    expect(screen.queryByText('Register')).not.toBeInTheDocument()
  })

  it('shows the empty state when there are no wins', async () => {
    // Default handlers: no topics.
    renderPage()
    expect(await screen.findByText(/Mark a milestone as an achievement/i)).toBeInTheDocument()
  })
})
