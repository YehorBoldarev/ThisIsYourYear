import type { InputHTMLAttributes } from 'react'
import { cn } from '../lib/cn'

const field =
  'w-full rounded-field border bg-input px-3.5 py-3 text-sm text-text placeholder:text-text-muted ' +
  'transition-colors focus:outline-none focus-visible:border-teal'

interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  invalid?: boolean
}

export function Input({ className, invalid, ...rest }: InputProps) {
  return (
    <input
      aria-invalid={invalid || undefined}
      className={cn(field, invalid ? 'border-rose' : 'border-border-input', className)}
      {...rest}
    />
  )
}
