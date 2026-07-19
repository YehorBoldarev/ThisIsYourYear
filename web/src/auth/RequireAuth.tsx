import type { ReactNode } from 'react'
import { Navigate, useLocation } from 'react-router-dom'
import { useAuth } from './useAuth'
import { Spinner } from '../components/Spinner'

/**
 * Gate for protected routes. While the boot refresh runs we show a spinner (so
 * the login screen doesn't flash); once resolved, unauthenticated users are sent
 * to /login with the intended location preserved for post-login return.
 */
export function RequireAuth({ children }: { children: ReactNode }) {
  const { status } = useAuth()
  const location = useLocation()

  if (status === 'loading') {
    return (
      <div className="flex min-h-screen items-center justify-center text-text-muted">
        <Spinner size={28} />
      </div>
    )
  }

  if (status === 'unauthenticated') {
    return <Navigate to="/login" state={{ from: location }} replace />
  }

  return <>{children}</>
}
