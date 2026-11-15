import type { ReactNode } from 'react'
import { Navigate, useLocation } from 'react-router-dom'
import { useAuth } from './useAuth'
import { inviteFrom, withInvite } from './invite'
import { Spinner } from '../components/Spinner'

/**
 * Gate for protected routes. While the boot refresh runs we show a spinner (so
 * the login screen doesn't flash); once resolved, unauthenticated users are sent
 * to /login with the intended location preserved for post-login return — except
 * that a URL carrying an invite code (`?invite=…`) goes straight to sign-up with
 * the code, since that visitor has been invited to create an account.
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
    const invite = inviteFrom(location.search)
    if (invite) return <Navigate to={withInvite('/register', invite)} replace />
    return <Navigate to="/login" state={{ from: location }} replace />
  }

  return <>{children}</>
}
