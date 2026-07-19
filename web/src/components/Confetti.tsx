import { useSyncExternalStore } from 'react'
import { ACCENT_HEX } from './accent'

/** Subscribe to `prefers-reduced-motion` so confetti is skipped when requested. */
function usePrefersReducedMotion(): boolean {
  return useSyncExternalStore(
    (cb) => {
      const mq = window.matchMedia('(prefers-reduced-motion: reduce)')
      mq.addEventListener('change', cb)
      return () => mq.removeEventListener('change', cb)
    },
    () => window.matchMedia('(prefers-reduced-motion: reduce)').matches,
    () => false,
  )
}

const COLORS = [ACCENT_HEX.teal, ACCENT_HEX.blue, ACCENT_HEX.gold, ACCENT_HEX.rose]

// Deterministic scatter (matches the design's hand-placed dots/squares): each
// piece has a horizontal position, colour, shape, rotation, size, and stagger.
const PIECES = Array.from({ length: 22 }, (_, i) => ({
  left: (i * 37 + 11) % 100,
  color: COLORS[i % COLORS.length],
  round: i % 3 === 0,
  rotate: (i * 53) % 360,
  size: 5 + (i % 3),
  delay: (i % 7) * 0.14,
}))

/**
 * A confetti burst for the celebration modal. Purely decorative (aria-hidden) and
 * skipped entirely under `prefers-reduced-motion`.
 */
export function Confetti() {
  const reduced = usePrefersReducedMotion()
  if (reduced) return null

  return (
    <div className="pointer-events-none absolute inset-0 overflow-hidden" aria-hidden="true">
      {PIECES.map((p, i) => (
        <span
          key={i}
          data-testid="confetti-piece"
          className="animate-confetti absolute top-0"
          style={{
            left: `${p.left}%`,
            width: p.size,
            height: p.size,
            background: p.color,
            borderRadius: p.round ? '999px' : '1px',
            transform: `rotate(${p.rotate}deg)`,
            animationDelay: `${p.delay}s`,
          }}
        />
      ))}
    </div>
  )
}
