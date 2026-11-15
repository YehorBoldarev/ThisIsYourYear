/**
 * Invite links: `?invite=<code>` on any URL. The code must survive however a
 * signed-out visitor arrives (front page, deep link, switching between sign-in
 * and sign-up), so these helpers read it and carry it along.
 */

/** The invite code in a URL query string (`location.search`), or ''. */
export function inviteFrom(search: string): string {
  return new URLSearchParams(search).get('invite')?.trim() ?? ''
}

/** `path` with the invite code attached when there is one. */
export function withInvite(path: string, code: string | null | undefined): string {
  const c = code?.trim()
  return c ? `${path}?invite=${encodeURIComponent(c)}` : path
}
