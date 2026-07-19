import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { Modal } from './Modal'
import { Button } from './Button'

function Harness({ onClose }: { onClose?: () => void }) {
  const [open, setOpen] = useState(false)
  return (
    <>
      <Button onClick={() => setOpen(true)}>Open</Button>
      <Modal
        open={open}
        onClose={() => {
          setOpen(false)
          onClose?.()
        }}
        title="Create a big goal"
      >
        <input aria-label="field" />
      </Modal>
    </>
  )
}

describe('Modal', () => {
  it('is a labelled dialog when open', async () => {
    const user = userEvent.setup()
    render(<Harness />)
    await user.click(screen.getByRole('button', { name: 'Open' }))
    const dialog = screen.getByRole('dialog')
    expect(dialog).toHaveAttribute('aria-modal', 'true')
    expect(dialog).toHaveAccessibleName('Create a big goal')
  })

  it('closes on Escape and returns focus to the opener', async () => {
    const user = userEvent.setup()
    render(<Harness />)
    const opener = screen.getByRole('button', { name: 'Open' })
    await user.click(opener)
    expect(screen.getByRole('dialog')).toBeInTheDocument()
    await user.keyboard('{Escape}')
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    expect(opener).toHaveFocus()
  })

  it('closes on backdrop click but not on content click', async () => {
    const onClose = vi.fn()
    const user = userEvent.setup()
    render(<Harness onClose={onClose} />)
    await user.click(screen.getByRole('button', { name: 'Open' }))
    await user.click(screen.getByLabelText('field'))
    expect(onClose).not.toHaveBeenCalled()
    // The backdrop is the dialog's parent element.
    const backdrop = screen.getByRole('dialog').parentElement as HTMLElement
    await user.click(backdrop)
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('traps Tab within the dialog', async () => {
    const user = userEvent.setup()
    render(<Harness />)
    await user.click(screen.getByRole('button', { name: 'Open' }))
    // Focusables: the field and the close button. Tabbing past the last wraps.
    const field = screen.getByLabelText('field')
    const close = screen.getByRole('button', { name: 'Close' })
    field.focus()
    await user.tab()
    expect(close).toHaveFocus()
    await user.tab()
    expect(field).toHaveFocus()
  })
})
