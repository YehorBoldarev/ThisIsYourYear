/*
 * In-memory holder for the access token. Deliberately NOT persisted to
 * localStorage — the durable credential is the HttpOnly refresh cookie, and a
 * memory-only access token can't be exfiltrated by XSS. On reload the token is
 * gone and rebuilt by a silent refresh (see AuthProvider boot).
 */
let accessToken: string | null = null

export function getToken(): string | null {
  return accessToken
}

export function setToken(token: string | null): void {
  accessToken = token
}

export function clearToken(): void {
  accessToken = null
}
