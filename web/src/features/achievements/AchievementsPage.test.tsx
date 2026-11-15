import { describe, expect, it } from 'vitest'
import { http, HttpResponse } from 'msw'
import { render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { goalResponse, server } from '../../test/msw'
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
  it('lists the wins from one request, newest first', async () => {
    let goalListRequests = 0
    server.use(
      http.get('/api/v1/achievements', () =>
        HttpResponse.json([
          {
            goal: goalResponse({
              id: 'g2',
              title: 'Finished book 12',
              status: 'done',
              isAchievement: true,
              completedAt: '2026-06-01T00:00:00Z',
            }),
            topic: {
              id: 't2',
              title: 'Read 24 books',
              year: 2026,
              color: '#5b8def',
              icon: '',
              category: '',
            },
          },
          {
            goal: goalResponse({
              id: 'g1',
              title: 'First 10k run',
              status: 'done',
              isAchievement: true,
              completedAt: '2026-05-01T00:00:00Z',
            }),
            topic: {
              id: 't1',
              title: 'Run a Half-Marathon',
              year: 2026,
              color: '',
              icon: '',
              category: '',
            },
          },
        ]),
      ),
      http.get('/api/v1/topics/:id/goals', () => {
        goalListRequests++
        return HttpResponse.json([])
      }),
    )
    renderPage()

    expect(await screen.findByText('First 10k run')).toBeInTheDocument()
    expect(screen.getByText(/2 moments worth keeping/i)).toBeInTheDocument()
    const titles = screen.getAllByRole('listitem').map((li) => li.textContent)
    expect(titles[0]).toContain('Finished book 12')
    expect(titles[1]).toContain('Run a Half-Marathon')
    expect(goalListRequests).toBe(0)
  })

  it('shows the empty state when there are no wins', async () => {
    // Default handlers: no topics.
    renderPage()
    expect(await screen.findByText(/Mark a milestone as an achievement/i)).toBeInTheDocument()
  })
})
