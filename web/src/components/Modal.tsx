import { useCallback, useEffect, useId, useRef } from 'react'
import type { ReactNode } from 'react'
import { createPortal } from 'react-dom'
import { cn } from '../lib/cn'
import { Icon } from './Icon'

interface ModalProps {
  open: boolean
  onClose: () => void
  title: ReactNode
  /** Small mono eyebrow above the title. */
  eyebrow?: ReactNode
  children?: ReactNode
  /** Right-aligned footer actions. */
  footer?: ReactNode
  /** Max width in px (desktop). Mobile becomes a near-full-width sheet. */
  width?: number
}

const FOCUSABLE =
  'a[href],button:not([disabled]),textarea,input,select,[tabindex]:not([tabindex="-1"])'

export function Modal({
  open,
  onClose,
  title,
  eyebrow,
  children,
  footer,
  width = 560,
}: ModalProps) {
  const dialogRef = useRef<HTMLDivElement>(null)
  const openerRef = useRef<Element | null>(null)
  const titleId = useId()

  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault()
        onClose()
        return
      }
      if (e.key !== 'Tab' || !dialogRef.current) return
      const nodes = dialogRef.current.querySelectorAll<HTMLElement>(FOCUSABLE)
      if (nodes.length === 0) {
        e.preventDefault()
        dialogRef.current.focus()
        return
      }
      const first = nodes[0]
      const last = nodes[nodes.length - 1]
      const active = document.activeElement
      if (e.shiftKey && active === first) {
        e.preventDefault()
        last.focus()
      } else if (!e.shiftKey && active === last) {
        e.preventDefault()
        first.focus()
      }
    },
    [onClose],
  )

  useEffect(() => {
    if (!open) return
    openerRef.current = document.activeElement
    document.addEventListener('keydown', handleKeyDown)
    const { overflow } = document.body.style
    document.body.style.overflow = 'hidden'

    // Move focus into the dialog (first focusable, else the dialog itself).
    const focusables = dialogRef.current?.querySelectorAll<HTMLElement>(FOCUSABLE)
    if (focusables && focusables.length > 0) focusables[0].focus()
    else dialogRef.current?.focus()

    return () => {
      document.removeEventListener('keydown', handleKeyDown)
      document.body.style.overflow = overflow
      // Return focus to whatever opened the modal.
      if (openerRef.current instanceof HTMLElement) openerRef.current.focus()
    }
  }, [open, handleKeyDown])

  if (!open) return null

  return createPortal(
    // The backdrop is a convenience click target; keyboard users close with
    // Escape (handled above), so it needs no interactive role of its own.
    // eslint-disable-next-line jsx-a11y/no-static-element-interactions
    <div
      className="fixed inset-0 z-50 flex items-center justify-center p-4"
      style={{
        background: 'radial-gradient(circle at 50% 20%, rgba(0,0,0,0.55), rgba(0,0,0,0.72))',
      }}
      onMouseDown={(e) => {
        // Backdrop click (but not clicks that start inside the dialog) closes.
        if (e.target === e.currentTarget) onClose()
      }}
    >
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        tabIndex={-1}
        className={cn(
          'w-full max-h-[90vh] overflow-y-auto rounded-card border border-border-popup',
          'bg-elevated p-7 shadow-popup outline-none',
        )}
        style={{ maxWidth: width }}
      >
        <div className="mb-5 flex items-start justify-between gap-4">
          <div>
            {eyebrow && (
              <div className="label-mono mb-2 text-[11px] text-text-muted">{eyebrow}</div>
            )}
            <h2 id={titleId} className="font-serif text-2xl font-semibold text-text">
              {title}
            </h2>
          </div>
          <button
            type="button"
            onClick={onClose}
            aria-label="Close"
            className="grid size-[30px] shrink-0 place-items-center rounded-full border border-border-popup text-text-muted transition-colors hover:text-text cursor-pointer"
          >
            <Icon name="x" size={16} />
          </button>
        </div>

        {children}

        {footer && <div className="mt-6 flex justify-end gap-3">{footer}</div>}
      </div>
    </div>,
    document.body,
  )
}
