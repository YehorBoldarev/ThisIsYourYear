import { describe, expect, it } from 'vitest'
import { celebrationStats } from './achievements'
import type { Topic } from '../features/topics/useTopics'

function topic(over: Partial<Topic> = {}): Topic {
  return {
    id: 't1',
    title: 'Run a Half-Marathon',
    description: '',
    year: 2026,
    color: '#3ec5a0',
    icon: '',
    category: '',
    targetDate: '',
    archived: false,
    progress: { total: 2, completed: 2, ratio: 1 },
    motivation: { band: 'achieved', label: 'Complete', title: 'A promise kept', message: '' },
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-06-01T00:00:00Z',
    ...over,
  }
}

describe('celebrationStats', () => {
  it('computes milestones, duration, and rank', () => {
    const done = topic({
      id: 'done',
      createdAt: '2026-01-01T00:00:00Z',
      updatedAt: '2026-06-01T00:00:00Z',
    })
    const other = topic({ id: 'early', updatedAt: '2026-03-01T00:00:00Z' })
    const now = new Date('2026-06-01T00:00:00Z')
    const stats = celebrationStats(done, [other, done], now)
    expect(stats.milestones).toBe('2/2')
    expect(stats.duration).toBe('5 mo')
    expect(stats.rank).toBe(2) // "early" completed first, then "done"
  })

  it('floors sub-month durations to "<1 mo"', () => {
    const t = topic({ createdAt: '2026-06-20T00:00:00Z' })
    const stats = celebrationStats(t, [t], new Date('2026-06-30T00:00:00Z'))
    expect(stats.duration).toBe('<1 mo')
  })
})
