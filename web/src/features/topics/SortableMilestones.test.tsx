import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { http, HttpResponse } from 'msw'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { axe } from 'vitest-axe'
import { errorBody, goalResponse, server, topicResponse } from '../../test/msw'
import { ThemeProvider } from '../../theme/ThemeProvider'
import { TopicDetailModal } from './TopicDetailModal'
import { moveGoal } from './moveGoal'
import type { Goal } from './useTopicGoals'

describe('moveGoal', () => {
  it('moves an id onto another id’s place', () => {
    expect(moveGoal(['a', 'b', 'c'], 'a', 'c')).toEqual(['b', 'c', 'a'])
    expect(moveGoal(['a', 'b', 'c'], 'c', 'a')).toEqual(['c', 'a', 'b'])
  })
  it('returns null when nothing moves', () => {
    expect(moveGoal(['a', 'b'], 'a', 'a')).toBeNull()
    expect(moveGoal(['a', 'b'], 'x', 'a')).toBeNull()
  })
})

// jsdom has no layout: give every milestone row a 40px-tall slot by its index so
// dnd-kit's keyboard sensor can find the row above/below.
function fakeLayout() {
  return vi.spyOn(Element.prototype, 'getBoundingClientRect').mockImplementation(function (
    this: Element,
  ) {
    const li = this.closest('li')
    const index = li?.parentElement ? Array.from(li.parentElement.children).indexOf(li) : 0
    const top = index * 40
    return {
      x: 0,
      y: top,
      top,
      left: 0,
      width: 300,
      height: 40,
      right: 300,
      bottom: top + 40,
      toJSON: () => ({}),
    } as DOMRect
  })
}

const GOALS: Goal[] = [
  goalResponse({ id: 'g1', title: 'Register', order: 0 }),
  goalResponse({ id: 'g2', title: 'First 10k', order: 1 }),
  goalResponse({ id: 'g3', title: 'Race day', order: 2 }),
]

function renderPopup(onClose = vi.fn()) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const noop = vi.fn()
  const view = render(
    <ThemeProvider>
      <QueryClientProvider client={qc}>
        <TopicDetailModal
          topic={topicResponse({ id: 't1', progress: { total: 3, completed: 0, ratio: 0 } })}
          onClose={onClose}
          onAddMilestone={noop}
          onEditTopic={noop}
          onDeleteTopic={noop}
          onEditGoal={noop}
          onDeleteGoal={noop}
          onCelebrate={noop}
        />
      </QueryClientProvider>
    </ThemeProvider>,
  )
  return { ...view, onClose }
}

function rowTitles() {
  return screen
    .getAllByRole('button', { name: /^Reorder / })
    .map((b) => b.getAttribute('aria-label')?.replace('Reorder ', ''))
}

describe('Milestone drag-and-drop (keyboard)', () => {
  let layout: ReturnType<typeof fakeLayout>
  let saved: unknown
  beforeEach(() => {
    layout = fakeLayout()
    saved = undefined
    // A tiny server: the goal list reflects the last saved order.
    let current = GOALS
    server.use(
      http.get('/api/v1/topics/:id/goals', () => HttpResponse.json(current)),
      http.put('/api/v1/topics/:id/goals/order', async ({ request }) => {
        saved = await request.json()
        const ids = (saved as { goalIds: string[] }).goalIds
        current = ids.flatMap((id, order) => {
          const g = GOALS.find((x) => x.id === id)
          return g ? [{ ...g, order }] : []
        })
        return HttpResponse.json(current)
      }),
    )
  })
  afterEach(() => layout.mockRestore())

  it('moves a milestone with Space, ArrowDown, Space and saves the whole order once', async () => {
    const user = userEvent.setup()
    renderPopup()
    const handle = await screen.findByRole('button', { name: 'Reorder Register' })
    expect(handle).toHaveAttribute('aria-roledescription', 'sortable')

    handle.focus()
    await user.keyboard(' ')
    await user.keyboard('{ArrowDown}')
    await user.keyboard(' ')

    await waitFor(() => expect(saved).toEqual({ goalIds: ['g2', 'g1', 'g3'] }))
    await waitFor(() => expect(rowTitles()).toEqual(['First 10k', 'Register', 'Race day']))
  })

  it('announces moves by title and position inside the popup', async () => {
    const user = userEvent.setup()
    renderPopup()
    const handle = await screen.findByRole('button', { name: 'Reorder Register' })
    handle.focus()
    await user.keyboard(' ')
    const dialog = screen.getByRole('dialog')
    await waitFor(() =>
      expect(
        within(dialog).getByText('Picked up Register, at position 1 of 3.'),
      ).toBeInTheDocument(),
    )
    await user.keyboard('{Escape}')
  })

  it('Escape during a drag cancels the drag, not the popup', async () => {
    const user = userEvent.setup()
    const { onClose } = renderPopup()
    const handle = await screen.findByRole('button', { name: 'Reorder Register' })
    handle.focus()
    await user.keyboard(' ')
    await user.keyboard('{ArrowDown}')
    await user.keyboard('{Escape}')

    await waitFor(() =>
      expect(screen.getByText('Cancelled. Register stays at position 1 of 3.')).toBeInTheDocument(),
    )
    expect(onClose).not.toHaveBeenCalled()
    expect(saved).toBeUndefined()
    expect(rowTitles()).toEqual(['Register', 'First 10k', 'Race day'])

    // With no drag in progress, Escape closes the popup as before.
    await user.keyboard('{Escape}')
    expect(onClose).toHaveBeenCalled()
  })

  it('puts the order back and says so when saving fails', async () => {
    server.use(
      http.put('/api/v1/topics/:id/goals/order', () =>
        HttpResponse.json(errorBody('validation_error', 'stale', { goalIds: 'stale' }), {
          status: 422,
        }),
      ),
    )
    const user = userEvent.setup()
    renderPopup()
    const handle = await screen.findByRole('button', { name: 'Reorder Register' })
    handle.focus()
    await user.keyboard(' ')
    await user.keyboard('{ArrowDown}')
    await user.keyboard(' ')

    expect(await screen.findByRole('alert')).toHaveTextContent('Couldn’t save the new order')
    await waitFor(() => expect(rowTitles()).toEqual(['Register', 'First 10k', 'Race day']))
  })

  it('has no accessibility violations with drag handles', async () => {
    const { baseElement } = renderPopup()
    await screen.findByRole('button', { name: 'Reorder Register' })
    const results = await axe(baseElement, {
      rules: { region: { enabled: false }, 'page-has-heading-one': { enabled: false } },
    })
    expect(results).toHaveNoViolations()
  })
})

describe('Milestone drag handles', () => {
  it('are not shown for a single milestone', async () => {
    server.use(
      http.get('/api/v1/topics/:id/goals', () =>
        HttpResponse.json([goalResponse({ id: 'g1', title: 'Only one' })]),
      ),
    )
    renderPopup()
    expect(await screen.findByText('Only one')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /^Reorder / })).not.toBeInTheDocument()
  })
})
