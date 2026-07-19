import { Icon, ProgressBar } from '../../components'
import { motivationBand } from '../../lib/motivation'
import { useTopicModals } from './topicModalsContext'
import type { Topic } from './useTopics'

/** An "Achieved" gold tag shown when a chapter reaches 100%. */
function AchievedTag() {
  return (
    <span className="label-mono inline-flex items-center gap-1 text-[10px] text-gold">
      <Icon name="star" size={11} />
      Achieved
    </span>
  )
}

export function ChapterRow({ topic }: { topic: Topic }) {
  const { openDetail } = useTopicModals()
  const { total, completed, ratio } = topic.progress
  const percent = Math.round(ratio * 100)
  const achieved = total > 0 && ratio >= 1
  const accent = topic.color || 'var(--accent-teal)'
  const { shortLabel } = motivationBand(ratio, total)
  const statusLine = `${shortLabel} · ${completed}/${total} milestone${total === 1 ? '' : 's'}`
  const open = () => openDetail(topic)

  return (
    <li className="sm:border-b sm:border-border-subtle sm:last:border-b-0">
      {/* Desktop / tablet: a single clickable row. */}
      <button
        type="button"
        onClick={open}
        className="hidden w-full items-center gap-4 py-4 text-left transition-colors hover:bg-fill-muted/40 sm:flex"
      >
        <span className="h-10 w-[3px] shrink-0 rounded-full" style={{ background: accent }} />
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2.5">
            <span className="truncate font-serif text-[19px] font-semibold text-text">
              {topic.title}
            </span>
            {achieved && <AchievedTag />}
          </div>
          <div className="label-mono mt-1 text-[11px] text-text-muted">{statusLine}</div>
        </div>
        <div className="flex shrink-0 items-center gap-3">
          <span className="font-serif text-lg text-text-2">{percent}%</span>
          <ProgressBar
            value={ratio}
            height={6}
            color={accent}
            className="w-24"
            label={topic.title}
          />
        </div>
      </button>

      {/* Mobile: a stacked clickable card with a full-width bar. */}
      <button
        type="button"
        onClick={open}
        className="w-full rounded-field border border-border bg-surface p-4 text-left sm:hidden"
        style={{ borderLeft: `3px solid ${accent}` }}
      >
        <div className="flex items-center justify-between gap-2">
          <span className="truncate font-serif text-[17px] font-semibold text-text">
            {topic.title}
          </span>
          <span className="shrink-0 font-serif text-text-2">{percent}%</span>
        </div>
        <div className="label-mono mt-1 flex items-center gap-2 text-[11px] text-text-muted">
          {achieved && <AchievedTag />}
          <span>{statusLine}</span>
        </div>
        <ProgressBar value={ratio} height={6} color={accent} className="mt-3" label={topic.title} />
      </button>
    </li>
  )
}
