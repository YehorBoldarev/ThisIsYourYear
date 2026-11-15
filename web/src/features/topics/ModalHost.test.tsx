import { describe, expect, it } from 'vitest'
import { http, HttpResponse } from 'msw'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, useLocation, useNavigate } from 'react-router-dom'
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

/** Shows the current query string and offers the browser's Back. */
function AddressBar() {
  const { search } = useLocation()
  const navigate = useNavigate()
  return (
    <>
      <output aria-label="address">{search}</output>
      <button type="button" onClick={() => navigate(-1)}>
        browser back
      </button>
    </>
  )
}

function renderAt(entries: string[]) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <ThemeProvider>
      <MemoryRouter initialEntries={entries} initialIndex={entries.length - 1}>
        <QueryClientProvider client={qc}>
          <ModalHost>
            <OpenDetail />
            <AddressBar />
          </ModalHost>
        </QueryClientProvider>
      </MemoryRouter>
    </ThemeProvider>,
  )
}

const address = () => screen.getByRole('status', { name: 'address' }).textContent

describe('ModalHost — goal popup in the address (?topic=)', () => {
  const listed = () =>
    server.use(
      http.get('/api/v1/topics', () =>
        HttpResponse.json([topicResponse({ id: 't1', title: 'Run a Half-Marathon' })]),
      ),
    )

  it('opening adds ?topic= and Back closes the popup', async () => {
    listed()
    const user = userEvent.setup()
    renderAt(['/'])
    await user.click(screen.getByRole('button', { name: 'open detail' }))
    expect(await screen.findByRole('dialog', { name: 'Run a Half-Marathon' })).toBeInTheDocument()
    expect(address()).toBe('?topic=t1')

    await user.click(screen.getByRole('button', { name: 'browser back' }))
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    expect(address()).toBe('')
  })

  it('closing the popup removes ?topic= (stepping back over its history entry)', async () => {
    listed()
    const user = userEvent.setup()
    renderAt(['/'])
    await user.click(screen.getByRole('button', { name: 'open detail' }))
    await screen.findByRole('dialog', { name: 'Run a Half-Marathon' })
    await user.click(screen.getByRole('button', { name: 'Close' }))
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    expect(address()).toBe('')
  })

  it('a reload (or a pasted link) reopens the popup, and closing then clears the address', async () => {
    listed()
    const user = userEvent.setup()
    renderAt(['/?topic=t1'])
    expect(await screen.findByRole('dialog', { name: 'Run a Half-Marathon' })).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Close' }))
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    expect(address()).toBe('')
  })

  it('keeps other parameters in the address', async () => {
    listed()
    const user = userEvent.setup()
    renderAt(['/?year=2026'])
    await user.click(screen.getByRole('button', { name: 'open detail' }))
    await screen.findByRole('dialog', { name: 'Run a Half-Marathon' })
    expect(address()).toBe('?year=2026&topic=t1')
  })

  it('opens a linked goal that is not in this year’s list by fetching it', async () => {
    server.use(
      http.get('/api/v1/topics', () => HttpResponse.json([])),
      http.get('/api/v1/topics/:id', ({ params }) =>
        HttpResponse.json(
          topicResponse({ id: String(params.id), title: 'Last year’s goal', year: 2026 }),
        ),
      ),
    )
    renderAt(['/?topic=t9'])
    expect(await screen.findByRole('dialog', { name: 'Last year’s goal' })).toBeInTheDocument()
  })

  it('quietly forgets a link to a goal that doesn’t exist (or isn’t yours)', async () => {
    server.use(
      http.get('/api/v1/topics', () => HttpResponse.json([])),
      http.get('/api/v1/topics/:id', () =>
        HttpResponse.json({ error: { code: 'not_found', message: 'nope' } }, { status: 404 }),
      ),
    )
    renderAt(['/?topic=missing'])
    await waitFor(() => expect(address()).toBe(''))
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })
})
