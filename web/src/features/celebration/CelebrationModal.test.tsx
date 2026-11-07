import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { topicResponse } from '../../test/msw'
import { CelebrationModal } from './CelebrationModal'

const topic = topicResponse({
  title: 'Launch a Side Project',
  year: 2026,
  progress: { total: 6, completed: 6, ratio: 1 },
  createdAt: '2026-01-01T00:00:00Z',
  updatedAt: '2026-06-01T00:00:00Z',
})

describe('CelebrationModal', () => {
  it('shows the goal, the quote, and the computed stat strip', () => {
    render(<CelebrationModal topic={topic} allTopics={[topic]} onClose={vi.fn()} />)
    expect(screen.getByRole('heading', { name: 'Launch a Side Project' })).toBeInTheDocument()
    expect(screen.getByText(/Big goal complete/i)).toBeInTheDocument()
    // Completing a goal isn't a ★ win, so the popup must not call it one.
    expect(screen.queryByText(/achievement/i)).not.toBeInTheDocument()
    expect(screen.getByText(/kept a promise/i)).toBeInTheDocument()
    expect(screen.getByText('6/6')).toBeInTheDocument()
    expect(screen.getByText('#1')).toBeInTheDocument()
    expect(screen.getByText(/Of 2026/i)).toBeInTheDocument()
  })

  it('offers a single dismiss action (no dead-end Reflections link yet)', async () => {
    const onClose = vi.fn()
    render(<CelebrationModal topic={topic} allTopics={[topic]} onClose={onClose} />)
    expect(screen.queryByRole('button', { name: /Reflect/ })).not.toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: /Keep the momentum/ }))
    expect(onClose).toHaveBeenCalledOnce()
  })
})
