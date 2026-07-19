import { cn } from '../lib/cn'
import { Icon } from '../components'
import { useTheme } from '../theme/useTheme'
import type { Theme } from '../theme/theme'

interface OptionProps {
  option: Theme
  active: boolean
  size: number
  onSelect: (t: Theme) => void
}

function Option({ option, active, size, onSelect }: OptionProps) {
  const isLight = option === 'light'
  return (
    <button
      type="button"
      aria-pressed={active}
      aria-label={isLight ? 'Light theme' : 'Dark theme'}
      onClick={() => onSelect(option)}
      className={cn(
        'grid place-items-center rounded-full transition-colors cursor-pointer',
        active ? 'bg-fill-raised text-gold shadow-sm' : 'text-text-muted hover:text-text',
      )}
      style={{ width: size, height: size }}
    >
      <Icon name={isLight ? 'sun' : 'moon'} size={size * 0.55} />
    </button>
  )
}

/** Segmented sun/moon control that flips the whole app theme. */
export function ThemeToggle({ optionSize = 26 }: { optionSize?: number }) {
  const { theme, setTheme } = useTheme()
  return (
    <div className="inline-flex items-center gap-1 rounded-full bg-fill-muted p-1">
      <Option option="light" active={theme === 'light'} size={optionSize} onSelect={setTheme} />
      <Option option="dark" active={theme === 'dark'} size={optionSize} onSelect={setTheme} />
    </div>
  )
}
