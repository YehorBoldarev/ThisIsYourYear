import { useCallback, useEffect, useId, useRef } from 'react'
import { createPortal } from 'react-dom'
import { Button, Confetti } from '../../components'
import { celebrationStats } from '../../lib/achievements'
import type { Topic } from '../topics/useTopics'

interface CelebrationModalProps {
  topic: Topic
  /** The full topic list, for the "#N of {year}" rank. */
  allTopics: Topic[]
  /** "Reflect on it" — go write a reflection. */
  onReflect: () => void
  /** "Keep the momentum" / Esc / backdrop — dismiss and return. */
  onClose: () => void
}

const FOCUSABLE =
  'a[href],button:not([disabled]),textarea,input,select,[tabindex]:not([tabindex="-1"])'

/** The 100% celebration (design screen 7): medallion, quote, stat strip, actions. */
export function CelebrationModal({ topic, allTopics, onReflect, onClose }: CelebrationModalProps) {
  const cardRef = useRef<HTMLDivElement>(null)
  const openerRef = useRef<Element | null>(null)
  const titleId = useId()
  const stats = celebrationStats(topic, allTopics)

  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault()
        onClose()
        return
      }
      if (e.key !== 'Tab' || !cardRef.current) return
      const nodes = cardRef.current.querySelectorAll<HTMLElement>(FOCUSABLE)
      if (nodes.length === 0) return
      const first = nodes[0]
      const last = nodes[nodes.length - 1]
      if (e.shiftKey && document.activeElement === first) {
        e.preventDefault()
        last.focus()
      } else if (!e.shiftKey && document.activeElement === last) {
        e.preventDefault()
        first.focus()
      }
    },
    [onClose],
  )

  useEffect(() => {
    openerRef.current = document.activeElement
    document.addEventListener('keydown', handleKeyDown)
    const { overflow } = document.body.style
    document.body.style.overflow = 'hidden'
    cardRef.current?.querySelector<HTMLElement>(FOCUSABLE)?.focus()
    return () => {
      document.removeEventListener('keydown', handleKeyDown)
      document.body.style.overflow = overflow
      if (openerRef.current instanceof HTMLElement) openerRef.current.focus()
    }
  }, [handleKeyDown])

  return createPortal(
    // eslint-disable-next-line jsx-a11y/no-static-element-interactions
    <div
      className="fixed inset-0 z-50 flex items-center justify-center overflow-hidden p-4"
      style={{
        background:
          'radial-gradient(circle at 50% 28%, rgba(232,183,101,0.18), rgba(0,0,0,0.78) 62%)',
      }}
      onMouseDown={(e) => {
        if (e.target === e.currentTarget) onClose()
      }}
    >
      <Confetti />
      <div
        ref={cardRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        className="relative w-full max-w-[480px] rounded-card border border-border-popup bg-elevated p-9 text-center shadow-popup"
      >
        {/* Star-in-glow medallion. */}
        <div
          className="mx-auto grid size-[78px] place-items-center rounded-full"
          style={{
            background:
              'radial-gradient(circle at 50% 40%, rgba(232,183,101,0.25), transparent 70%)',
          }}
          aria-hidden="true"
        >
          <svg width="58" height="58" viewBox="0 0 34 34" fill="none">
            <circle cx="17" cy="17" r="14" stroke="var(--accent-gold)" strokeWidth="2.5" />
            <path
              d="M17 10 L18.7 14.65 L23.66 14.84 L19.76 17.9 L21.12 22.66 L17 19.9 L12.88 22.66 L14.24 17.9 L10.34 14.84 L15.3 14.65 Z"
              fill="var(--accent-gold)"
            />
          </svg>
        </div>

        <div className="label-mono mt-[18px] text-[11px] tracking-[0.2em] text-gold">
          Achievement unlocked
        </div>
        <h2 id={titleId} className="mt-2 font-serif text-3xl font-semibold text-text">
          {topic.title}
        </h2>
        <p className="mx-auto mt-3.5 max-w-[360px] font-serif text-[16px] italic leading-relaxed text-text-2">
          “You kept a promise to yourself. This is exactly what that feels like. Take a moment to
          savour it.”
        </p>

        {/* Stat strip. */}
        <div className="mt-5 flex justify-center gap-6 border-t border-border-subtle pt-5">
          <Stat value={stats.milestones} label="Milestones" />
          <Stat value={stats.duration} label="To complete" />
          <Stat value={`#${stats.rank}`} label={`Of ${topic.year}`} gold />
        </div>

        <div className="mt-6 flex gap-3">
          <Button variant="outline" className="flex-1" onClick={onReflect}>
            Reflect on it
          </Button>
          <Button variant="gold" className="flex-1" onClick={onClose}>
            Keep the momentum →
          </Button>
        </div>
      </div>
    </div>,
    document.body,
  )
}

function Stat({ value, label, gold }: { value: string; label: string; gold?: boolean }) {
  return (
    <div>
      <div className={`font-serif text-[22px] font-semibold ${gold ? 'text-gold' : 'text-text'}`}>
        {value}
      </div>
      <div className="label-mono mt-0.5 text-[10px] text-text-muted">{label}</div>
    </div>
  )
}
