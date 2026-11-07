import type { ButtonHTMLAttributes } from 'react'
import { cn } from '../lib/cn'

export type ButtonVariant = 'teal' | 'gold' | 'outline' | 'danger'
export type ButtonSize = 'md' | 'sm'

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant
  size?: ButtonSize
}

const base =
  'inline-flex items-center justify-center gap-2 rounded-full font-sans font-semibold ' +
  'transition-colors cursor-pointer disabled:cursor-not-allowed disabled:opacity-50'

const variants: Record<ButtonVariant, string> = {
  teal: 'bg-teal text-btn-ink hover:bg-teal-hover',
  gold: 'bg-gold text-btn-ink hover:bg-gold-hover',
  outline:
    'border border-border-popup bg-transparent text-text-2 hover:border-text-muted hover:text-text',
  danger: 'bg-rose text-danger-ink hover:opacity-90',
}

const sizes: Record<ButtonSize, string> = {
  md: 'px-5 py-2.5 text-sm',
  sm: 'px-3.5 py-2 text-[13px]',
}

export function Button({
  variant = 'teal',
  size = 'md',
  className,
  type = 'button',
  ...rest
}: ButtonProps) {
  return (
    <button type={type} className={cn(base, variants[variant], sizes[size], className)} {...rest} />
  )
}
