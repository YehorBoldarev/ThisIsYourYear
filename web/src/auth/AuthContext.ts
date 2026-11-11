import { createContext } from 'react'
import type { UserResponse } from '../api/client'

export type AuthStatus = 'loading' | 'authenticated' | 'unauthenticated'

export interface AuthContextValue {
  status: AuthStatus
  user: UserResponse | null
  login: (email: string, password: string, rememberMe: boolean) => Promise<void>
  /** `inviteCode` is sent only when non-empty (invite-only servers require it). */
  register: (
    email: string,
    displayName: string,
    password: string,
    inviteCode?: string,
  ) => Promise<void>
  logout: () => Promise<void>
}

export const AuthContext = createContext<AuthContextValue | null>(null)
