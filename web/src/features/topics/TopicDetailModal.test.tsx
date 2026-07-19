import { describe, expect, it, vi } from 'vitest'
import { http, HttpResponse } from 'msw'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { server, goalResponse, topicResponse } from '../../test/msw'
import { ThemeProvider } from '../../theme/ThemeProvider'
import type { Goal } from './useTopicGoals'
import { TopicDetailModal } from './TopicDetailModal'

function renderModal(ui: React.ReactElement) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <ThemeProvider>
      <QueryClientProvider client={qc}>{ui}</QueryClientProvider>
    </ThemeProvider>,
  )
}

const noop = vi.fn()

function baseProps() {
  return {
    onClose: noop,
    onAddMilestone: noop,
    onEditTopic: noop,
    onDeleteTopic: noop,
    onEditGoal: noop,
    onDeleteGoal: noop,
    onCelebrate: noop,
  }
}

describe('TopicDetailModal', () => {
  it('renders the header, callout, and milestone list', async () => {
    server.use(
      http.get('/api/v1/topics/:id/goals', () =>
        HttpResponse.json([
          goalResponse({ id: 'g1', title: 'Register for the race' }),
          goalResponse({ id: 'g2', title: 'First 10k run', status: 'done' }),
        ]),
      ),
    )
    renderModal(
      <TopicDetailModal
        topic={topicResponse({ progress: { total: 2, completed: 1, ratio: 0.5 } })}
        {...baseProps()}
      />,
    )

    expect(screen.getByRole('heading', { name: 'Run a Half-Marathon' })).toBeInTheDocument()
    expect(screen.getByText(/Big goal · Endurance/)).toBeInTheDocument()
    expect(await screen.findByText('Register for the race')).toBeInTheDocument()
    expect(screen.getByText(/1 of 2 milestones complete/)).toBeInTheDocument()
  })

  it('optimistically completes a milestone when its circle is clicked', async () => {
    const goals: Goal[] = [goalResponse({ id: 'g1', title: 'Register for the race' })]
    server.use(
      http.get('/api/v1/topics/:id/goals', () => HttpResponse.json(goals)),
      http.patch('/api/v1/goals/:id', async ({ request }) => {
        const body = (await request.json()) as { done?: boolean }
        goals[0] = { ...goals[0], status: body.done ? 'done' : 'todo' }
        return HttpResponse.json(goals[0])
      }),
    )
    renderModal(
      <TopicDetailModal
        topic={topicResponse({ progress: { total: 1, completed: 0, ratio: 0 } })}
        {...baseProps()}
      />,
    )

    const circle = await screen.findByRole('button', { name: 'Complete Register for the race' })
    await userEvent.click(circle)

    expect(
      await screen.findByRole('button', { name: 'Reopen Register for the race' }),
    ).toBeInTheDocument()
  })
})
