/** Join truthy class names. A tiny alternative to clsx for our needs. */
export function cn(...parts: Array<string | false | null | undefined>): string {
  return parts.filter(Boolean).join(' ')
}
