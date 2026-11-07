import type { TextareaHTMLAttributes } from 'react'
import { cn } from '../lib/cn'

const field =
  'w-full rounded-field border bg-input px-3.5 py-3 text-sm text-text placeholder:text-text-muted ' +
  'transition-colors resize-y min-h-24 focus:outline-none focus-visible:border-teal'

interface TextareaProps extends TextareaHTMLAttributes<HTMLTextAreaElement> {
  invalid?: boolean
}

export function Textarea({ className, invalid, ...rest }: TextareaProps) {
  return (
    <textarea
      aria-invalid={invalid || undefined}
      className={cn(field, invalid ? 'border-rose' : 'border-border-input', className)}
      {...rest}
    />
  )
}
