import type { ReactNode } from 'react'
import { Header } from './Header'
import { Footer } from './Footer'
import { BottomTabBar } from './BottomTabBar'
import { yearElapsedFraction } from '../lib/year'
import { useAuth } from '../auth/useAuth'

/**
 * App chrome: header + routed content + footer, plus the mobile bottom tab bar.
 * Rendered only inside RequireAuth, so a user is always present. `onNew` is the
 * tab bar's "New ＋" (create a big goal). While `preparing` (the year being
 * planned hasn't begun) the logo's year ring stays empty.
 */
export function AppShell({
  children,
  onNew,
  preparing = false,
}: {
  children: ReactNode
  onNew: () => void
  preparing?: boolean
}) {
  const { user, logout } = useAuth()
  const yearProgress = preparing ? 0 : yearElapsedFraction()

  if (!user) return null

  return (
    // On mobile the fixed bottom tab bar (~61px) would cover whatever is last on
    // the page, which is the footer. Reserve its height (plus the home-indicator
    // inset on phones that have one) below everything, footer included.
    <div className="flex min-h-screen flex-col pb-[calc(4rem+env(safe-area-inset-bottom))] sm:pb-0">
      <a
        href="#main"
        className="sr-only focus:not-sr-only focus:absolute focus:left-4 focus:top-4 focus:z-50 focus:rounded-full focus:bg-elevated focus:px-4 focus:py-2 focus:text-sm focus:text-text focus:shadow-popup"
      >
        Skip to content
      </a>
      <Header user={user} yearProgress={yearProgress} onLogout={() => void logout()} />
      <main id="main" tabIndex={-1} className="flex-1 outline-none">
        {children}
      </main>
      <Footer />
      <BottomTabBar onNew={onNew} />
    </div>
  )
}
