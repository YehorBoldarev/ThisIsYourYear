import type { HTMLInputTypeAttribute } from 'react'
import { Label } from './Label'
import { Input } from './Input'

interface FieldProps {
  id: string
  label: string
  value: string
  onChange: (value: string) => void
  type?: HTMLInputTypeAttribute
  error?: string
  autoComplete?: string
  placeholder?: string
  required?: boolean
}

/** Label + Input + inline error, wired for accessibility. */
export function Field({
  id,
  label,
  value,
  onChange,
  type = 'text',
  error,
  autoComplete,
  placeholder,
  required,
}: FieldProps) {
  const errorId = `${id}-error`
  return (
    <div className="space-y-2">
      <Label htmlFor={id}>{label}</Label>
      <Input
        id={id}
        type={type}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        invalid={!!error}
        aria-describedby={error ? errorId : undefined}
        autoComplete={autoComplete}
        placeholder={placeholder}
        required={required}
      />
      {error && (
        <p id={errorId} className="text-[12px] text-rose">
          {error}
        </p>
      )}
    </div>
  )
}
