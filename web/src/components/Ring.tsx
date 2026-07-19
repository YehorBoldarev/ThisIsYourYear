import { accentVar, type AccentColor } from './accent'

interface RingProps {
  /** Fill fraction 0–1. */
  value: number
  size?: number
  accent?: AccentColor
  /** Explicit stroke colour (e.g. a topic's stored hex); overrides `accent`. */
  color?: string
  /** Show the centered percent label. */
  showLabel?: boolean
}

/** A progress ring with a centered percent label (topic-detail progress mark). */
export function Ring({ value, size = 84, accent = 'teal', color, showLabel = true }: RingProps) {
  const clamped = Math.max(0, Math.min(1, value))
  const stroke = Math.round(size * 0.09)
  const r = (size - stroke) / 2
  const c = 2 * Math.PI * r
  const center = size / 2
  const pct = Math.round(clamped * 100)

  return (
    <div
      className="relative inline-flex items-center justify-center"
      style={{ width: size, height: size }}
      role="progressbar"
      aria-valuenow={pct}
      aria-valuemin={0}
      aria-valuemax={100}
    >
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
            stroke={color ?? accentVar(accent)}
            strokeWidth={stroke}
            strokeLinecap="round"
            strokeDasharray={`${clamped * c} ${c}`}
          />
        </g>
      </svg>
      {showLabel && (
        <span className="absolute font-serif text-lg font-semibold text-text">{pct}%</span>
      )}
    </div>
  )
}
