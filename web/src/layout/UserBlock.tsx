import { Avatar } from '../components'
import { initialsFrom } from '../lib/user'
import type { UserResponse } from '../api/client'

// The API has no role field; show a fixed on-brand eyebrow for now.
const ROLE_EYEBROW = 'Dreamer-in-Chief'

/** Header identity block: name + role eyebrow, with the gradient avatar. */
export function UserBlock({ user, compact = false }: { user: UserResponse; compact?: boolean }) {
  return (
    <div className="flex items-center gap-3">
      {!compact && (
        <div className="text-right leading-tight">
          <div className="text-[13px] font-semibold text-text">{user.displayName}</div>
          <div className="text-[11px] text-text-muted">{ROLE_EYEBROW}</div>
        </div>
      )}
      <Avatar initials={initialsFrom(user.displayName)} />
    </div>
  )
}
