import type { ReactNode } from 'react'
import { Header } from './Header'
import { Footer } from './Footer'
import { BottomTabBar } from './BottomTabBar'
import { yearElapsedFraction } from '../lib/year'
import { useAuth } from '../auth/useAuth'

/**
 * App chrome: header + routed content + footer, plus the mobile bottom tab bar.
 * Rendered only inside RequireAuth, so a user is always present.
 */
export function AppShell({ children }: { children: ReactNode }) {
  const { user, logout } = useAuth()
  const yearProgress = yearElapsedFraction()

  if (!user) return null

  return (
    <div className="flex min-h-screen flex-col">
      <a
        href="#main"
        className="sr-only focus:not-sr-only focus:absolute focus:left-4 focus:top-4 focus:z-50 focus:rounded-full focus:bg-elevated focus:px-4 focus:py-2 focus:text-sm focus:text-text focus:shadow-popup"
      >
        Skip to content
      </a>
      <Header user={user} yearProgress={yearProgress} onLogout={() => void logout()} />
      {/* pb-24 on mobile keeps content clear of the fixed bottom tab bar. */}
      <main id="main" tabIndex={-1} className="flex-1 pb-24 outline-none sm:pb-0">
        {children}
      </main>
      <Footer />
      <BottomTabBar onNew={() => {}} />
    </div>
  )
}
