import { useRef, useState, type CSSProperties } from 'react'
import {
  DndContext,
  KeyboardSensor,
  PointerSensor,
  closestCenter,
  useSensor,
  useSensors,
  type Announcements,
  type DragEndEvent,
  type UniqueIdentifier,
} from '@dnd-kit/core'
import {
  SortableContext,
  sortableKeyboardCoordinates,
  useSortable,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { Icon } from '../../components'
import { cn } from '../../lib/cn'
import type { Goal } from './useTopicGoals'
import { MilestoneRow, type MilestoneRowProps } from './MilestoneRow'
import { moveGoal } from './moveGoal'

type RowHandlers = Pick<MilestoneRowProps, 'accent' | 'onToggle' | 'onEdit' | 'onDelete'>

interface SortableMilestonesProps extends RowHandlers {
  goals: Goal[]
  /** Called with every goal id in the new order after a successful drop. */
  onReorder: (goalIds: string[]) => void
}

const INSTRUCTIONS =
  'To reorder a milestone, press Space or Enter on its handle, move it with the up and down ' +
  'arrow keys, then press Space or Enter again to drop it. Press Escape to cancel.'

/**
 * The goal popup's milestone list, reorderable by drag-and-drop (plan D3, Part A).
 * Drag by the grip handle with a mouse or finger; or focus the handle and use
 * Space + arrow keys + Space. Moves are announced to screen readers by title
 * and position. The live region renders inside the popup (which is aria-modal),
 * and Escape during a drag cancels the drag rather than closing the popup.
 */
export function SortableMilestones({ goals, onReorder, ...row }: SortableMilestonesProps) {
  const [liveRegion, setLiveRegion] = useState<HTMLDivElement | null>(null)
  const [dragging, setDragging] = useState(false)
  // dnd-kit reports "over itself" right after pick-up; stay quiet until the
  // milestone actually moves, so "Picked up …" isn't immediately overwritten.
  const moved = useRef(false)
  const sensors = useSensors(
    // A few pixels of movement before a drag starts, so taps still work.
    useSensor(PointerSensor, { activationConstraint: { distance: 4 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  )

  const ids = goals.map((g) => g.id)
  const showHandles = goals.length > 1
  // A milestone still being created has a temporary id the server doesn't know.
  const pending = ids.some((id) => id.startsWith('tmp-'))

  const title = (id: UniqueIdentifier) => goals.find((g) => g.id === id)?.title ?? 'milestone'
  const position = (id: UniqueIdentifier) =>
    `position ${ids.indexOf(String(id)) + 1} of ${ids.length}`
  const announcements: Announcements = {
    onDragStart: ({ active }) => {
      moved.current = false
      return `Picked up ${title(active.id)}, at ${position(active.id)}.`
    },
    onDragOver: ({ active, over }) => {
      if (!over) return undefined
      if (over.id === active.id && !moved.current) return undefined
      moved.current = true
      return `${title(active.id)} would move to ${position(over.id)}.`
    },
    onDragEnd: ({ active, over }) =>
      over
        ? `Dropped ${title(active.id)} at ${position(over.id)}.`
        : `Dropped ${title(active.id)}.`,
    onDragCancel: ({ active }) => `Cancelled. ${title(active.id)} stays at ${position(active.id)}.`,
  }

  const onDragEnd = ({ active, over }: DragEndEvent) => {
    setDragging(false)
    if (!over) return
    const next = moveGoal(ids, String(active.id), String(over.id))
    if (next) onReorder(next)
  }

  return (
    <div ref={setLiveRegion} data-captures-escape={dragging ? '' : undefined}>
      <DndContext
        sensors={sensors}
        collisionDetection={closestCenter}
        onDragStart={() => setDragging(true)}
        onDragCancel={() => setDragging(false)}
        onDragEnd={onDragEnd}
        accessibility={{
          container: liveRegion ?? undefined,
          announcements,
          screenReaderInstructions: { draggable: INSTRUCTIONS },
        }}
      >
        <SortableContext items={ids} strategy={verticalListSortingStrategy}>
          <ul className="mt-1 divide-y divide-border-subtle">
            {goals.map((goal) => (
              <SortableRow
                key={goal.id}
                goal={goal}
                showHandle={showHandles}
                disabled={pending}
                {...row}
              />
            ))}
          </ul>
        </SortableContext>
      </DndContext>
    </div>
  )
}

function SortableRow({
  goal,
  showHandle,
  disabled,
  ...row
}: RowHandlers & { goal: Goal; showHandle: boolean; disabled: boolean }) {
  const {
    attributes,
    listeners,
    setNodeRef,
    setActivatorNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id: goal.id, disabled: disabled || !showHandle })
  const style: CSSProperties = { transform: CSS.Transform.toString(transform), transition }

  return (
    <MilestoneRow
      goal={goal}
      {...row}
      rootRef={setNodeRef}
      rootStyle={style}
      dragging={isDragging}
      handle={
        showHandle ? (
          <button
            type="button"
            ref={setActivatorNodeRef}
            {...attributes}
            {...listeners}
            disabled={disabled}
            aria-label={`Reorder ${goal.title}`}
            className={cn(
              // touch-none: dragging the handle must not scroll the popup.
              '-ml-1 grid size-7 shrink-0 touch-none place-items-center rounded-full text-text-muted transition-colors',
              disabled ? 'opacity-50' : 'cursor-grab hover:text-text active:cursor-grabbing',
            )}
          >
            <Icon name="grip" size={16} />
          </button>
        ) : undefined
      }
    />
  )
}
