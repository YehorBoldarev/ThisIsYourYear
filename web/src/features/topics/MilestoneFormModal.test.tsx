import { describe, expect, it, vi } from 'vitest'
import { http, HttpResponse } from 'msw'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { server, goalResponse, topicResponse } from '../../test/msw'
import { ThemeProvider } from '../../theme/ThemeProvider'
import { MilestoneFormModal } from './MilestoneFormModal'

function renderModal(ui: React.ReactElement) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <ThemeProvider>
      <QueryClientProvider client={qc}>{ui}</QueryClientProvider>
    </ThemeProvider>,
  )
}

describe('MilestoneFormModal', () => {
  it('adds a milestone with the achievement flag and closes', async () => {
    const onClose = vi.fn()
    let created: { title?: string; isAchievement?: boolean } = {}
    server.use(
      http.post('/api/v1/topics/:id/goals', async ({ request }) => {
        created = (await request.json()) as typeof created
        return HttpResponse.json(goalResponse(), { status: 201 })
      }),
    )
    renderModal(<MilestoneFormModal topic={topicResponse()} onClose={onClose} />)

    await userEvent.type(screen.getByLabelText('Milestone'), 'Register for the race')
    await userEvent.click(screen.getByRole('switch', { name: 'Celebrate as an achievement' }))
    await userEvent.click(screen.getByRole('button', { name: 'Add milestone' }))

    await waitFor(() => expect(onClose).toHaveBeenCalled())
    expect(created).toMatchObject({ title: 'Register for the race', isAchievement: true })
  })

  it('requires a title', async () => {
    const onClose = vi.fn()
    renderModal(<MilestoneFormModal topic={topicResponse()} onClose={onClose} />)

    await userEvent.click(screen.getByRole('button', { name: 'Add milestone' }))

    expect(await screen.findByText('A title is required.')).toBeInTheDocument()
    expect(onClose).not.toHaveBeenCalled()
  })
})
