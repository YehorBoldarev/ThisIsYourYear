import { Icon } from '../../components'
import { ACCENT_HEX } from '../../components/accent'
import { formatIsoDate } from '../../lib/date'
import type { Achievement } from '../../lib/achievements'

/** One win: gold star medallion, milestone title, its chapter, and the date earned. */
export function AchievementCard({ goal, topic }: Achievement) {
  const accent = topic.color || ACCENT_HEX.teal
  const date = formatIsoDate(goal.completedAt)

  return (
    <li className="flex items-center gap-4 rounded-field border border-border-subtle bg-surface p-4">
      <span
        className="grid size-9 shrink-0 place-items-center rounded-full text-gold"
        style={{ background: 'rgba(232,183,101,0.14)' }}
        aria-hidden="true"
      >
        <Icon name="star" size={16} />
      </span>
      <div className="min-w-0 flex-1">
        <div className="truncate font-serif text-[17px] font-medium text-text">{goal.title}</div>
        <div className="label-mono mt-1 flex items-center gap-1.5 text-[10px] text-text-muted">
          <span className="size-2 rounded-full" style={{ background: accent }} aria-hidden="true" />
          <span className="truncate">{topic.title}</span>
        </div>
      </div>
      {date && <span className="label-mono shrink-0 text-[10px] text-text-faint">{date}</span>}
    </li>
  )
}
