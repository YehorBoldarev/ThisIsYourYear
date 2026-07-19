/*
 * Placeholder data for the B1 shell and component gallery only. No screen wires
 * to this once the real API lands (B2+). Kept isolated so it is trivial to delete.
 */
import type { AccentColor } from '../components'

export interface MockChapter {
  id: string
  title: string
  category: string
  accent: AccentColor
  done: number
  total: number
}

export const mockChapters: MockChapter[] = [
  {
    id: '1',
    title: 'Run a Half-Marathon',
    category: 'Endurance',
    accent: 'teal',
    done: 6,
    total: 8,
  },
  { id: '2', title: 'Read 24 Books', category: 'Mind', accent: 'blue', done: 11, total: 24 },
  { id: '3', title: 'Learn to Cook Well', category: 'Craft', accent: 'gold', done: 8, total: 8 },
  {
    id: '4',
    title: 'Save for a Home',
    category: 'Foundations',
    accent: 'rose',
    done: 2,
    total: 12,
  },
]
