import type { ReactElement, SVGProps } from 'react'

export type IconName =
  | 'check'
  | 'star'
  | 'x'
  | 'plus'
  | 'grid'
  | 'pencil'
  | 'sun'
  | 'moon'
  | 'dot'
  | 'arrow-right'
  | 'trash'

interface IconProps extends Omit<SVGProps<SVGSVGElement>, 'name'> {
  name: IconName
  size?: number
  /** Accessible label. When omitted the icon is decorative (aria-hidden). */
  title?: string
}

// Each glyph draws on a 24x24 grid using currentColor, so `text-*` sets color.
const paths: Record<IconName, ReactElement> = {
  check: <path d="M4 12.5 9 17.5 20 6.5" />,
  star: (
    <path d="M12 3.5 14.6 9.2 20.8 9.9 16.2 14.1 17.5 20.2 12 17 6.5 20.2 7.8 14.1 3.2 9.9 9.4 9.2Z" />
  ),
  x: <path d="M6 6 18 18M18 6 6 18" />,
  plus: <path d="M12 5V19M5 12H19" />,
  grid: (
    <>
      <rect x="4" y="4" width="7" height="7" rx="1.5" />
      <rect x="13" y="4" width="7" height="7" rx="1.5" />
      <rect x="4" y="13" width="7" height="7" rx="1.5" />
      <rect x="13" y="13" width="7" height="7" rx="1.5" />
    </>
  ),
  pencil: <path d="M4 20 4 16 15 5 19 9 8 20ZM13 7 17 11" />,
  sun: (
    <>
      <circle cx="12" cy="12" r="4" />
      <path d="M12 2V4M12 20V22M2 12H4M20 12H22M4.9 4.9 6.3 6.3M17.7 17.7 19.1 19.1M19.1 4.9 17.7 6.3M6.3 17.7 4.9 19.1" />
    </>
  ),
  moon: <path d="M20 14.5A8 8 0 1 1 9.5 4 6.5 6.5 0 0 0 20 14.5Z" />,
  dot: <circle cx="12" cy="12" r="5" />,
  'arrow-right': <path d="M4 12H20M14 6 20 12 14 18" />,
  trash: <path d="M4 7H20M9 7V5H15V7M6 7 7 20H17L18 7M10 11V16M14 11V16" />,
}

// Glyphs that read as filled shapes rather than strokes.
const filled: Partial<Record<IconName, boolean>> = { star: true, dot: true }

export function Icon({ name, size = 20, title, ...rest }: IconProps) {
  const isFilled = filled[name]
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill={isFilled ? 'currentColor' : 'none'}
      stroke={isFilled ? 'none' : 'currentColor'}
      strokeWidth={2}
      strokeLinecap="round"
      strokeLinejoin="round"
      role={title ? 'img' : undefined}
      aria-hidden={title ? undefined : true}
      aria-label={title}
      {...rest}
    >
      {title ? <title>{title}</title> : null}
      {paths[name]}
    </svg>
  )
}
