import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, render, screen } from '@testing-library/react'
import { NewYearCountdown } from './NewYearCountdown'

// Fixed "now": 2 days, 3 hours, 4 minutes and 5 seconds before midnight UTC.
const TARGET = '2027-01-01T00:00:00Z'
const NOW = Date.parse(TARGET) - ((2 * 24 + 3) * 3600 + 4 * 60 + 5) * 1000

describe('NewYearCountdown', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(NOW)
  })
  afterEach(() => {
    vi.useRealTimers()
  })

  it('shows days, hours, minutes and seconds, and a minute-precision sentence for screen readers', () => {
    render(<NewYearCountdown target={TARGET} year={2027} />)
    expect(screen.getByText('2027 begins in')).toBeInTheDocument()
    expect(screen.getByText('02')).toBeInTheDocument() // days
    expect(screen.getByText('03')).toBeInTheDocument() // hours
    expect(screen.getByText('04')).toBeInTheDocument() // minutes
    expect(screen.getByText('05')).toBeInTheDocument() // seconds
    expect(
      screen.getByText('2027 begins in 2 days, 3 hours and 4 minutes.', { selector: '.sr-only' }),
    ).toBeInTheDocument()
    expect(screen.getByRole('region', { name: 'Countdown to 2027' })).toBeInTheDocument()
  })

  it('ticks every second', () => {
    const { container } = render(<NewYearCountdown target={TARGET} year={2027} />)
    const digits = () => [...container.querySelectorAll('.tabular-nums')].map((d) => d.textContent)
    expect(digits()).toEqual(['02', '03', '04', '05'])
    act(() => {
      vi.advanceTimersByTime(3000)
    })
    expect(digits()).toEqual(['02', '03', '04', '02'])
  })

  it('celebrates at midnight and keeps asking the page to refresh until it switches', () => {
    vi.setSystemTime(Date.parse(TARGET) - 2000)
    const onReached = vi.fn()
    render(<NewYearCountdown target={TARGET} year={2027} onReached={onReached} />)

    act(() => {
      vi.advanceTimersByTime(2000)
    })
    expect(screen.getByRole('status')).toHaveTextContent('Happy New Year — your 2027 starts now.')
    expect(onReached).not.toHaveBeenCalled()

    act(() => {
      vi.advanceTimersByTime(3000)
    })
    expect(onReached).toHaveBeenCalledTimes(1)

    act(() => {
      vi.advanceTimersByTime(10_000)
    })
    expect(onReached).toHaveBeenCalledTimes(2)
  })

  it('shows the celebration right away when opened after midnight', () => {
    vi.setSystemTime(Date.parse(TARGET) + 60_000)
    render(<NewYearCountdown target={TARGET} year={2027} />)
    expect(screen.getByRole('status')).toHaveTextContent('Happy New Year')
  })
})
