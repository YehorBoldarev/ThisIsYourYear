import { describe, expect, it } from 'vitest'
import { http, HttpResponse } from 'msw'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { goalResponse, server, topicResponse } from '../../test/msw'
import { ThemeProvider } from '../../theme/ThemeProvider'
import { ModalHost } from './ModalHost'
import { useTopicModals } from './topicModalsContext'
import type { Topic } from './useTopics'

/** A child that opens the detail popup for topic t1 (id is all ModalHost needs). */
function OpenDetail() {
  const { openDetail } = useTopicModals()
  return (
    <button type="button" onClick={() => openDetail({ id: 't1' } as Topic)}>
      open detail
    </button>
  )
}

function renderHost() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <ThemeProvider>
      <MemoryRouter>
        <QueryClientProvider client={qc}>
          <ModalHost>
            <OpenDetail />
          </ModalHost>
        </QueryClientProvider>
      </MemoryRouter>
    </ThemeProvider>,
  )
}

describe('ModalHost celebration trigger', () => {
  it('celebrates when the final milestone completes the topic', async () => {
    server.use(
      http.get('/api/v1/topics', () =>
        HttpResponse.json([
          topicResponse({ id: 't1', progress: { total: 1, completed: 0, ratio: 0 } }),
        ]),
      ),
      http.get('/api/v1/topics/:id/goals', () =>
        HttpResponse.json([goalResponse({ id: 'g1', title: 'Only step', status: 'todo' })]),
      ),
      http.patch('/api/v1/goals/:id', () =>
        HttpResponse.json(goalResponse({ id: 'g1', title: 'Only step', status: 'done' })),
      ),
    )
    renderHost()

    await userEvent.click(screen.getByRole('button', { name: 'open detail' }))
    const circle = await screen.findByRole('button', { name: 'Complete Only step' })
    await userEvent.click(circle)

    expect(await screen.findByText(/Big goal complete/i)).toBeInTheDocument()
  })

  it('does not celebrate when milestones remain', async () => {
    server.use(
      http.get('/api/v1/topics', () =>
        HttpResponse.json([
          topicResponse({ id: 't1', progress: { total: 2, completed: 0, ratio: 0 } }),
        ]),
      ),
      http.get('/api/v1/topics/:id/goals', () =>
        HttpResponse.json([
          goalResponse({ id: 'g1', title: 'Step one', status: 'todo' }),
          goalResponse({ id: 'g2', title: 'Step two', status: 'todo' }),
        ]),
      ),
      http.patch('/api/v1/goals/:id', () =>
        HttpResponse.json(goalResponse({ id: 'g1', title: 'Step one', status: 'done' })),
      ),
    )
    renderHost()

    await userEvent.click(screen.getByRole('button', { name: 'open detail' }))
    const circle = await screen.findByRole('button', { name: 'Complete Step one' })
    await userEvent.click(circle)

    // onCelebrate fires synchronously in the toggle handler, so a missing
    // celebration right after the click means it was correctly not triggered.
    expect(screen.queryByText(/Big goal complete/i)).not.toBeInTheDocument()
    // The detail popup stays open (its title is still on screen).
    expect(screen.getByRole('heading', { name: 'Run a Half-Marathon' })).toBeInTheDocument()
  })
})
