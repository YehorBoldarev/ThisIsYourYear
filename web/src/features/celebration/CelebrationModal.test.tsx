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
    render(
      <CelebrationModal topic={topic} allTopics={[topic]} onReflect={vi.fn()} onClose={vi.fn()} />,
    )
    expect(screen.getByRole('heading', { name: 'Launch a Side Project' })).toBeInTheDocument()
    expect(screen.getByText(/Achievement unlocked/i)).toBeInTheDocument()
    expect(screen.getByText(/kept a promise/i)).toBeInTheDocument()
    expect(screen.getByText('6/6')).toBeInTheDocument()
    expect(screen.getByText('#1')).toBeInTheDocument()
    expect(screen.getByText(/Of 2026/i)).toBeInTheDocument()
  })

  it('wires the two actions', async () => {
    const onReflect = vi.fn()
    const onClose = vi.fn()
    render(
      <CelebrationModal
        topic={topic}
        allTopics={[topic]}
        onReflect={onReflect}
        onClose={onClose}
      />,
    )
    await userEvent.click(screen.getByRole('button', { name: 'Reflect on it' }))
    expect(onReflect).toHaveBeenCalledOnce()
    await userEvent.click(screen.getByRole('button', { name: /Keep the momentum/ }))
    expect(onClose).toHaveBeenCalledOnce()
  })
})
