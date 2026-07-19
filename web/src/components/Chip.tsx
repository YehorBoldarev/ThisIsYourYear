import type { ButtonHTMLAttributes } from 'react'
import { cn } from '../lib/cn'
import { accentVar, type AccentColor } from './accent'

interface ChipProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  /** Render an accent dot before the label. */
  dot?: AccentColor
}

/** A small pill used for example goals ("TRY:") and tags. */
export function Chip({ dot, className, children, ...rest }: ChipProps) {
  return (
    <button
      type="button"
      className={cn(
        'inline-flex items-center gap-2 rounded-full border border-border bg-fill-muted px-3.5 py-1.5',
        'text-[13px] text-text-2 transition-colors hover:text-text hover:border-text-muted cursor-pointer',
        className,
      )}
      {...rest}
    >
      {dot && (
        <span
          className="size-2 rounded-full"
          style={{ background: accentVar(dot) }}
          aria-hidden="true"
        />
      )}
      {children}
    </button>
  )
}
