import { Button, Icon, Modal, Ring } from '../../components'
import { ACCENT_HEX } from '../../components/accent'
import { bandToneClass } from '../../lib/motivation'
import type { Topic } from './useTopics'
import { useTopicGoals, type Goal } from './useTopicGoals'
import { useToggleGoalDone } from './mutations'
import { MilestoneRow } from './MilestoneRow'

interface TopicDetailModalProps {
  topic: Topic
  onClose: () => void
  onAddMilestone: () => void
  onEditTopic: () => void
  onDeleteTopic: () => void
  onEditGoal: (goal: Goal) => void
  onDeleteGoal: (goal: Goal) => void
  /** Called when completing a milestone brings this topic to 100%. */
  onCelebrate: () => void
}

export function TopicDetailModal({
  topic,
  onClose,
  onAddMilestone,
  onEditTopic,
  onDeleteTopic,
  onEditGoal,
  onDeleteGoal,
  onCelebrate,
}: TopicDetailModalProps) {
  const goalsQuery = useTopicGoals(topic.id)
  const toggle = useToggleGoalDone()

  // Toggle a milestone; if completing it clears the last open one, celebrate 100%.
  const handleToggle = (goal: Goal) => {
    const goals = goalsQuery.data ?? []
    const willComplete = goal.status !== 'done'
    const otherOpen = goals.filter((g) => g.id !== goal.id && g.status !== 'done').length
    toggle.toggle(goal)
    if (willComplete && goals.length > 0 && otherOpen === 0) onCelebrate()
  }

  const accent = topic.color || ACCENT_HEX.teal
  const { total, completed, ratio } = topic.progress
  const motivation = topic.motivation
  const remaining = total - completed
  const bandLine =
    total === 0
      ? 'No milestones yet'
      : remaining === 0
        ? 'Complete · a promise kept'
        : `${motivation.label} · ${remaining} milestone${remaining === 1 ? '' : 's'} to go`

  const eyebrow = topic.category ? `Big goal · ${topic.category}` : 'Big goal'

  return (
    <Modal open onClose={onClose} eyebrow={eyebrow} title={topic.title} width={720}>
      {topic.description && <p className="-mt-2 mb-4 text-sm text-text-2">{topic.description}</p>}

      {/* Motivational callout — the server's band copy (accent border + tint). */}
      {motivation.message && (
        <blockquote
          className="mb-6 rounded-field border-l-2 py-3 pl-4 pr-3"
          style={{ borderColor: accent, background: `${accent}14` }}
        >
          <div className={`label-mono text-[10px] ${bandToneClass(motivation.band)}`}>
            {motivation.title}
          </div>
          <p className="mt-1 font-serif text-[15px] italic text-text-2">{motivation.message}</p>
        </blockquote>
      )}

      {/* Progress block. */}
      <div className="flex items-center gap-5 border-y border-border py-5">
        <Ring value={ratio} color={accent} size={84} />
        <div>
          <div className="font-serif text-lg text-text">
            {completed} of {total} milestone{total === 1 ? '' : 's'} complete
          </div>
          <div className="label-mono mt-1 text-[11px] text-text-muted">{bandLine}</div>
        </div>
      </div>

      {/* Milestones. */}
      <div className="mt-5">
        <div className="flex items-center justify-between">
          <h3 className="label-mono text-[11px] text-text-muted">Milestones</h3>
          <button
            type="button"
            onClick={onAddMilestone}
            className="label-mono flex items-center gap-1 text-[11px] text-teal-ink transition-colors hover:text-teal-ink-hover cursor-pointer"
          >
            <Icon name="plus" size={12} />
            Add milestone
          </button>
        </div>

        {goalsQuery.isPending ? (
          <ul className="mt-2 space-y-2" aria-hidden="true">
            {[0, 1, 2].map((i) => (
              <li key={i} className="h-9 animate-pulse rounded-field bg-fill-muted" />
            ))}
          </ul>
        ) : goalsQuery.isError ? (
          <p className="mt-3 text-sm text-rose">Couldn&rsquo;t load milestones.</p>
        ) : goalsQuery.data.length === 0 ? (
          <p className="mt-3 text-sm text-text-2">
            No milestones yet — add the first step toward this goal.
          </p>
        ) : (
          <ul className="mt-1 divide-y divide-border-subtle">
            {goalsQuery.data.map((goal) => (
              <MilestoneRow
                key={goal.id}
                goal={goal}
                accent={accent}
                onToggle={handleToggle}
                onEdit={onEditGoal}
                onDelete={onDeleteGoal}
              />
            ))}
          </ul>
        )}
      </div>

      {/* Actions. */}
      <div className="mt-7 flex items-center justify-between border-t border-border pt-5">
        <button
          type="button"
          onClick={onDeleteTopic}
          className="label-mono flex items-center gap-1 text-[11px] text-text-muted transition-colors hover:text-rose cursor-pointer"
        >
          <Icon name="trash" size={13} />
          Delete chapter
        </button>
        <div className="flex gap-3">
          <Button variant="outline" onClick={onEditTopic}>
            Edit chapter
          </Button>
          <Button variant="teal" onClick={onAddMilestone}>
            <Icon name="plus" size={16} />
            Add milestone
          </Button>
        </div>
      </div>
    </Modal>
  )
}
