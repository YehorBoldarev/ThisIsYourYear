import type { LabelHTMLAttributes } from 'react'
import { cn } from '../lib/cn'

/**
 * The handoff's 11px mono, uppercase, letter-spaced field label. Association
 * with a control is the caller's responsibility (via `htmlFor` or nesting), so
 * the a11y rule can't verify it at this reusable primitive's definition.
 */
export function Label({ className, ...rest }: LabelHTMLAttributes<HTMLLabelElement>) {
  return (
    // eslint-disable-next-line jsx-a11y/label-has-associated-control
    <label className={cn('label-mono block text-[11px] text-text-muted', className)} {...rest} />
  )
}
