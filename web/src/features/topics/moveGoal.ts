import { arrayMove } from '@dnd-kit/sortable'

/** The ids after moving `activeId` onto `overId`'s place, or null if nothing moves. */
export function moveGoal(ids: string[], activeId: string, overId: string): string[] | null {
  const from = ids.indexOf(activeId)
  const to = ids.indexOf(overId)
  if (from < 0 || to < 0 || from === to) return null
  return arrayMove(ids, from, to)
}
