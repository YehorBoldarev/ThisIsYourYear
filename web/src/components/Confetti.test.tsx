import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { Confetti } from './Confetti'

function mockReducedMotion(reduced: boolean) {
  window.matchMedia = vi.fn().mockImplementation((query: string) => ({
    matches: reduced && query.includes('reduce'),
    media: query,
    onchange: null,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    addListener: vi.fn(),
    removeListener: vi.fn(),
    dispatchEvent: vi.fn(),
  }))
}

describe('Confetti', () => {
  it('renders confetti pieces when motion is allowed', () => {
    mockReducedMotion(false)
    render(<Confetti />)
    expect(screen.getAllByTestId('confetti-piece').length).toBeGreaterThan(0)
  })

  it('renders nothing under prefers-reduced-motion', () => {
    mockReducedMotion(true)
    render(<Confetti />)
    expect(screen.queryAllByTestId('confetti-piece')).toHaveLength(0)
  })
})
