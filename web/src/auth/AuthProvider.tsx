import { useCallback, useEffect, useMemo, useState } from 'react'
import type { ReactNode } from 'react'
import { client, type UserResponse } from '../api/client'
import { clearToken, setToken } from '../api/tokenStore'
import { ApiError } from '../api/errors'
import { AuthContext, type AuthStatus } from './AuthContext'

export function AuthProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<AuthStatus>('loading')
  const [user, setUser] = useState<UserResponse | null>(null)

  const login = useCallback(async (email: string, password: string, rememberMe: boolean) => {
    const { data, error, response } = await client.POST('/api/v1/auth/login', {
      body: { email, password, rememberMe },
    })
    if (error || !data) throw ApiError.fromBody(response.status, error)
    setToken(data.accessToken)
    setUser(data.user)
    setStatus('authenticated')
  }, [])

  const register = useCallback(
    async (email: string, displayName: string, password: string) => {
      const { data, error, response } = await client.POST('/api/v1/auth/register', {
        body: { email, displayName, password },
      })
      if (error || !data) throw ApiError.fromBody(response.status, error)
      // Register returns the user but no token; log in to obtain one.
      await login(email, password, false)
    },
    [login],
  )

  const logout = useCallback(async () => {
    try {
      await client.POST('/api/v1/auth/logout')
    } finally {
      clearToken()
      setUser(null)
      setStatus('unauthenticated')
    }
  }, [])

  // Boot: try to restore the session from the refresh cookie.
  useEffect(() => {
    let cancelled = false
    void (async () => {
      const { data, error } = await client.POST('/api/v1/auth/refresh')
      if (cancelled) return
      if (error || !data) {
        clearToken()
        setStatus('unauthenticated')
        return
      }
      setToken(data.accessToken)
      setUser(data.user)
      setStatus('authenticated')
    })()
    return () => {
      cancelled = true
    }
  }, [])

  const value = useMemo(
    () => ({ status, user, login, register, logout }),
    [status, user, login, register, logout],
  )

  return <AuthContext value={value}>{children}</AuthContext>
}
