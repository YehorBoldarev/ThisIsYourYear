import { cn } from '../lib/cn'

interface AvatarProps {
  initials: string
  size?: number
  className?: string
}

/** Gradient circle (135° teal→blue) with initials — placeholder until a photo. */
export function Avatar({ initials, size = 38, className }: AvatarProps) {
  return (
    <span
      className={cn(
        'inline-flex items-center justify-center rounded-full font-bold text-btn-ink',
        className,
      )}
      style={{
        width: size,
        height: size,
        fontSize: size * 0.36,
        background: 'linear-gradient(135deg, var(--accent-teal), var(--accent-blue))',
      }}
      aria-hidden="true"
    >
      {initials}
    </span>
  )
}
