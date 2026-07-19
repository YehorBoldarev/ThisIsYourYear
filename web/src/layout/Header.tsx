import { NavLink } from 'react-router-dom'
import { cn } from '../lib/cn'
import { Button, Icon, Logo } from '../components'
import { ThemeToggle } from './ThemeToggle'
import { UserBlock } from './UserBlock'
import type { UserResponse } from '../api/client'

const NAV = [
  { to: '/', label: 'Dashboard', end: true },
  { to: '/achievements', label: 'Achievements', end: false },
  { to: '/reflections', label: 'Reflections', end: false },
]

interface HeaderProps {
  user: UserResponse
  yearProgress: number
  onLogout: () => void
}

export function Header({ user, yearProgress, onLogout }: HeaderProps) {
  return (
    <header className="flex items-center justify-between gap-4 border-b border-border bg-bar px-7 py-[15px]">
      <div className="flex items-center gap-6">
        <Logo progress={yearProgress} withWordmark />
        {/* Inline nav on desktop/tablet; hidden on mobile (bottom tab bar takes over). */}
        <nav className="hidden items-center gap-1 sm:flex" aria-label="Primary">
          {NAV.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.end}
              className={({ isActive }) =>
                cn(
                  'rounded-full px-3.5 py-1.5 text-sm transition-colors',
                  isActive ? 'bg-fill-muted text-text' : 'text-text-muted hover:text-text',
                )
              }
            >
              {item.label}
            </NavLink>
          ))}
        </nav>
      </div>

      <div className="flex items-center gap-4">
        <ThemeToggle />
        <div className="hidden sm:block">
          <UserBlock user={user} />
        </div>
        <div className="sm:hidden">
          <UserBlock user={user} compact />
        </div>
        {/* Text on desktop/tablet, icon-only on mobile. */}
        <Button variant="outline" size="sm" onClick={onLogout} aria-label="Sign out">
          <Icon name="arrow-right" size={16} />
          <span className="hidden sm:inline">Sign out</span>
        </Button>
      </div>
    </header>
  )
}
