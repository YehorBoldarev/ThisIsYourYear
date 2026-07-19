import createClient from 'openapi-fetch'
import type { components, paths } from './schema'
import { clearToken, getToken, setToken } from './tokenStore'
import { ApiError } from './errors'

export type TokenResponse = components['schemas']['TokenResponse']
export type UserResponse = components['schemas']['User']

// Same-origin in the browser and under the dev proxy; jsdom provides an origin
// in tests. Requests are same-origin, so the HttpOnly refresh cookie is sent by
// default (no `credentials: 'include'` needed).
const ORIGIN = typeof window !== 'undefined' ? window.location.origin : ''

const AUTH_PREFIX = '/api/v1/auth/'
function isAuthPath(url: string): boolean {
  return new URL(url).pathname.startsWith(AUTH_PREFIX)
}

/**
 * Single-flight access-token refresh: concurrent callers (parallel 401s, plus
 * the boot refresh) share one in-flight POST /auth/refresh.
 */
let refreshInFlight: Promise<string> | null = null
export function refreshAccessToken(): Promise<string> {
  refreshInFlight ??= doRefresh().finally(() => {
    refreshInFlight = null
  })
  return refreshInFlight
}

async function doRefresh(): Promise<string> {
  // Raw fetch (not the typed client) so this never re-enters the 401 middleware.
  const res = await fetch(`${ORIGIN}/api/v1/auth/refresh`, { method: 'POST' })
  if (!res.ok) {
    clearToken()
    throw await ApiError.fromResponse(res)
  }
  const data = (await res.json()) as TokenResponse
  setToken(data.accessToken)
  return data.accessToken
}

export const client = createClient<paths>({
  baseUrl: ORIGIN,
  // Resolve the global fetch at call time (not capture-at-construction), so a
  // test-time MSW patch is honored. A no-op indirection in production.
  fetch: (...args: Parameters<typeof fetch>) => fetch(...args),
})

client.use({
  onRequest({ request }) {
    const token = getToken()
    // Auth endpoints authenticate via the cookie, not the bearer token.
    if (token && !isAuthPath(request.url)) {
      request.headers.set('Authorization', `Bearer ${token}`)
    }
    return request
  },
  async onResponse({ request, response }) {
    if (response.status !== 401 || isAuthPath(request.url)) return response
    let newToken: string | null
    try {
      newToken = await refreshAccessToken()
    } catch {
      newToken = null
    }
    if (!newToken) return response // let the 401 propagate → unauthenticated
    // Replay the original request once with the fresh token.
    const retried = new Request(request, { headers: new Headers(request.headers) })
    retried.headers.set('Authorization', `Bearer ${newToken}`)
    return fetch(retried)
  },
})
