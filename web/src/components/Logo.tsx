import { cn } from '../lib/cn'

interface LogoProps {
  /** Overall year progress 0–1; drives how far the teal timer ring sweeps. */
  progress?: number
  /** Mark diameter in px. */
  size?: number
  /** Show the "This is Your Year" wordmark beside the mark. */
  withWordmark?: boolean
  className?: string
}

/**
 * The brand mark: a faint full ring, a teal timer ring whose sweep encodes
 * year progress, and a gold five-point star centered inside — "achievement
 * earned through the year".
 */
export function Logo({ progress = 0.48, size = 30, withWordmark = false, className }: LogoProps) {
  const stroke = size * 0.09
  const r = (size - stroke) / 2
  const c = 2 * Math.PI * r
  const clamped = Math.max(0, Math.min(1, progress))
  const center = size / 2
  const star = starPoints(center, center, size * 0.26, size * 0.11)

  return (
    <span className={cn('inline-flex items-center gap-2.5', className)}>
      <svg width={size} height={size} viewBox={`0 0 ${size} ${size}`} aria-hidden="true">
        <g transform={`rotate(-90 ${center} ${center})`}>
          <circle
            cx={center}
            cy={center}
            r={r}
            fill="none"
            stroke="var(--track)"
            strokeWidth={stroke}
          />
          <circle
            cx={center}
            cy={center}
            r={r}
            fill="none"
            stroke="var(--accent-teal)"
            strokeWidth={stroke}
            strokeLinecap="round"
            strokeDasharray={`${clamped * c} ${c}`}
          />
        </g>
        <polygon points={star} fill="var(--accent-gold)" />
      </svg>
      {withWordmark && (
        <span className="font-serif text-[19px] font-semibold leading-none text-text">
          This is Your Year
        </span>
      )}
    </span>
  )
}

/** Build the point list for a five-point star centered at (cx, cy). */
function starPoints(cx: number, cy: number, outer: number, inner: number): string {
  const pts: string[] = []
  for (let i = 0; i < 10; i++) {
    const radius = i % 2 === 0 ? outer : inner
    const angle = (Math.PI / 5) * i - Math.PI / 2
    pts.push(`${cx + radius * Math.cos(angle)},${cy + radius * Math.sin(angle)}`)
  }
  return pts.join(' ')
}
