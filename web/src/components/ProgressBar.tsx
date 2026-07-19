import { cn } from '../lib/cn'
import { accentVar, type AccentColor } from './accent'

interface ProgressBarProps {
  /** Fill fraction 0–1. */
  value: number
  /** Track height in px (8 = dashboard year bar, 6 = per-goal). */
  height?: 8 | 6
  /** Solid accent fill; ignored when `gradient` or `color` is set. */
  accent?: AccentColor
  /** Arbitrary CSS color fill (e.g. a topic's stored hex); wins over `accent`. */
  color?: string
  /** Teal→gold gradient fill (the dashboard year bar); wins over all. */
  gradient?: boolean
  className?: string
  label?: string
}

export function ProgressBar({
  value,
  height = 6,
  accent = 'teal',
  color,
  gradient = false,
  className,
  label,
}: ProgressBarProps) {
  const pct = Math.max(0, Math.min(1, value)) * 100
  const fill = gradient
    ? 'linear-gradient(90deg, var(--accent-teal), var(--accent-gold))'
    : (color ?? accentVar(accent))

  return (
    <div
      role="progressbar"
      aria-valuenow={Math.round(pct)}
      aria-valuemin={0}
      aria-valuemax={100}
      aria-label={label}
      className={cn('w-full overflow-hidden rounded-full bg-track', className)}
      style={{ height }}
    >
      <div className="h-full rounded-full" style={{ width: `${pct}%`, background: fill }} />
    </div>
  )
}
