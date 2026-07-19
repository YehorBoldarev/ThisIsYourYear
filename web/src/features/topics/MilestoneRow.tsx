import { Icon } from '../../components'
import { cn } from '../../lib/cn'
import { formatDueShort } from '../../lib/date'
import type { Goal } from './useTopicGoals'

interface MilestoneRowProps {
  goal: Goal
  accent: string
  onToggle: (goal: Goal) => void
  onEdit: (goal: Goal) => void
  onDelete: (goal: Goal) => void
}

/** One checklist row: status token + label + trailing tag/date + edit/delete. */
export function MilestoneRow({ goal, accent, onToggle, onEdit, onDelete }: MilestoneRowProps) {
  const done = goal.status === 'done'
  const achievement = goal.isAchievement
  const due = formatDueShort(goal.dueDate ?? '')

  return (
    <li className="group flex items-center gap-3 py-2.5">
      <button
        type="button"
        onClick={() => onToggle(goal)}
        aria-pressed={done}
        aria-label={done ? `Reopen ${goal.title}` : `Complete ${goal.title}`}
        className={cn(
          'grid size-[22px] shrink-0 place-items-center rounded-full border-2 transition-colors cursor-pointer',
          done && achievement && 'border-gold text-gold',
          done && !achievement && 'border-transparent text-white',
          !done && 'border-border-input text-transparent hover:border-text-muted',
        )}
        style={done && !achievement ? { background: accent } : undefined}
      >
        {done && <Icon name={achievement ? 'star' : 'check'} size={achievement ? 12 : 13} />}
      </button>

      <span
        className={cn(
          'min-w-0 flex-1 truncate text-sm',
          done ? 'text-text-muted line-through' : 'text-text',
        )}
      >
        {goal.title}
      </span>

      {done && achievement && (
        <span className="label-mono shrink-0 text-[10px] text-gold">Achieved</span>
      )}
      {!done && due && (
        <span className="label-mono shrink-0 text-[10px] text-text-muted">Due {due}</span>
      )}

      <span className="flex shrink-0 items-center gap-1 opacity-0 transition-opacity group-hover:opacity-100 focus-within:opacity-100">
        <button
          type="button"
          onClick={() => onEdit(goal)}
          aria-label={`Edit ${goal.title}`}
          className="grid size-7 place-items-center rounded-full text-text-faint transition-colors hover:text-text cursor-pointer"
        >
          <Icon name="pencil" size={14} />
        </button>
        <button
          type="button"
          onClick={() => onDelete(goal)}
          aria-label={`Delete ${goal.title}`}
          className="grid size-7 place-items-center rounded-full text-text-faint transition-colors hover:text-rose cursor-pointer"
        >
          <Icon name="trash" size={14} />
        </button>
      </span>
    </li>
  )
}
