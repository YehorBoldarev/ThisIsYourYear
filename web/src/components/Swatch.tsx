import { ACCENTS, accentVar, type AccentColor } from './accent'

interface SwatchGroupProps {
  value: AccentColor
  onChange: (accent: AccentColor) => void
  options?: AccentColor[]
  label?: string
}

/** Accent-colour picker: circles where the selected one wears a double-ring. */
export function SwatchGroup({
  value,
  onChange,
  options = ACCENTS,
  label = 'Accent colour',
}: SwatchGroupProps) {
  return (
    <div role="radiogroup" aria-label={label} className="flex items-center gap-3">
      {options.map((accent) => {
        const selected = accent === value
        return (
          <button
            key={accent}
            type="button"
            role="radio"
            aria-checked={selected}
            aria-label={accent}
            onClick={() => onChange(accent)}
            className="size-7 rounded-full cursor-pointer"
            style={{
              background: accentVar(accent),
              // Double-ring when selected: a gap in the surface colour, then the
              // swatch's own colour as the outer ring.
              boxShadow: selected
                ? `0 0 0 2px var(--surface-elevated), 0 0 0 4px ${accentVar(accent)}`
                : undefined,
            }}
          />
        )
      })}
    </div>
  )
}
