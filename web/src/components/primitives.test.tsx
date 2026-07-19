import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { Button } from './Button'
import { ProgressBar } from './ProgressBar'
import { Ring } from './Ring'
import { Toggle } from './Toggle'
import { SwatchGroup } from './Swatch'
import type { AccentColor } from './accent'

describe('Button', () => {
  it('defaults to type=button and can be disabled', () => {
    render(<Button disabled>Save</Button>)
    const btn = screen.getByRole('button', { name: 'Save' })
    expect(btn).toHaveAttribute('type', 'button')
    expect(btn).toBeDisabled()
  })
})

describe('ProgressBar', () => {
  it('exposes clamped aria values', () => {
    render(<ProgressBar value={1.5} label="Year" />)
    const bar = screen.getByRole('progressbar', { name: 'Year' })
    expect(bar).toHaveAttribute('aria-valuenow', '100')
    expect(bar).toHaveAttribute('aria-valuemax', '100')
  })
})

describe('Ring', () => {
  it('renders the rounded percent label', () => {
    render(<Ring value={0.6} />)
    expect(screen.getByRole('progressbar')).toHaveAttribute('aria-valuenow', '60')
    expect(screen.getByText('60%')).toBeInTheDocument()
  })
})

describe('Toggle', () => {
  it('reflects and flips checked state', async () => {
    const onChange = vi.fn()
    const user = userEvent.setup()
    render(<Toggle checked={false} onChange={onChange} label="Celebrate" />)
    const sw = screen.getByRole('switch', { name: 'Celebrate' })
    expect(sw).toHaveAttribute('aria-checked', 'false')
    await user.click(sw)
    expect(onChange).toHaveBeenCalledWith(true)
  })
})

describe('SwatchGroup', () => {
  it('marks the selected accent and changes selection', async () => {
    const user = userEvent.setup()
    function Wrap() {
      const [v, setV] = useState<AccentColor>('teal')
      return <SwatchGroup value={v} onChange={setV} />
    }
    render(<Wrap />)
    expect(screen.getByRole('radio', { name: 'teal' })).toHaveAttribute('aria-checked', 'true')
    await user.click(screen.getByRole('radio', { name: 'gold' }))
    expect(screen.getByRole('radio', { name: 'gold' })).toHaveAttribute('aria-checked', 'true')
    expect(screen.getByRole('radio', { name: 'teal' })).toHaveAttribute('aria-checked', 'false')
  })
})
