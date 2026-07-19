import { cn } from '../lib/cn'

/** Small ring spinner in the current text color. */
export function Spinner({ size = 20, className }: { size?: number; className?: string }) {
  return (
    <span
      role="status"
      aria-label="Loading"
      className={cn('inline-block animate-spin rounded-full border-2 border-current', className)}
      style={{ width: size, height: size, borderRightColor: 'transparent' }}
    />
  )
}
