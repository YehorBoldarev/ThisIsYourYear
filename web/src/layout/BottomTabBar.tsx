import { NavLink } from 'react-router-dom'
import { cn } from '../lib/cn'
import { Icon, type IconName } from '../components'

interface Tab {
  to: string
  label: string
  icon: IconName
  end: boolean
}

const TABS: Tab[] = [
  { to: '/', label: 'Goals', icon: 'grid', end: true },
  { to: '/achievements', label: 'Wins', icon: 'star', end: false },
  { to: '/reflections', label: 'Notes', icon: 'pencil', end: false },
]

/** Mobile-only fixed bottom navigation, replacing the header's top nav. */
export function BottomTabBar({ onNew }: { onNew: () => void }) {
  return (
    <nav
      aria-label="Primary"
      className="fixed inset-x-0 bottom-0 z-40 flex items-stretch border-t border-border bg-bar sm:hidden"
    >
      {TABS.map((tab) => (
        <NavLink
          key={tab.to}
          to={tab.to}
          end={tab.end}
          className={({ isActive }) =>
            cn(
              'flex flex-1 flex-col items-center gap-1 py-2.5 text-[11px] transition-colors',
              isActive ? 'text-teal-ink' : 'text-text-muted',
            )
          }
        >
          <Icon name={tab.icon} size={20} />
          {tab.label}
        </NavLink>
      ))}
      <button
        type="button"
        onClick={onNew}
        className="flex flex-1 flex-col items-center gap-1 py-2.5 text-[11px] text-text-muted transition-colors hover:text-text cursor-pointer"
      >
        <Icon name="plus" size={20} />
        New
      </button>
    </nav>
  )
}
