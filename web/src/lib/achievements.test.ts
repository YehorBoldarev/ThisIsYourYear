import { describe, expect, it } from 'vitest'
import { celebrationStats, deriveAchievements } from './achievements'
import type { Topic } from '../features/topics/useTopics'
import type { Goal } from '../features/topics/useTopicGoals'

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
    motivation: '',
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-06-01T00:00:00Z',
    ...over,
  }
}

function goal(over: Partial<Goal> = {}): Goal {
  return {
    id: 'g1',
    topicId: 't1',
    title: 'First 10k',
    notes: '',
    status: 'done',
    order: 0,
    isAchievement: true,
    dueDate: '',
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
    ...over,
  }
}

describe('deriveAchievements', () => {
  it('keeps only completed, starred milestones', () => {
    const t = topic()
    const goals = [
      goal({ id: 'a', status: 'done', isAchievement: true }),
      goal({ id: 'b', status: 'done', isAchievement: false }), // done but not starred
      goal({ id: 'c', status: 'todo', isAchievement: true }), // starred but not done
    ]
    const items = deriveAchievements([t], { t1: goals })
    expect(items.map((i) => i.goal.id)).toEqual(['a'])
    expect(items[0].topic.id).toBe('t1')
  })

  it('sorts by completion date, newest first, undated last', () => {
    const t = topic()
    const goals = [
      goal({ id: 'old', completedAt: '2026-02-01T00:00:00Z' }),
      goal({ id: 'undated', completedAt: undefined }),
      goal({ id: 'new', completedAt: '2026-09-01T00:00:00Z' }),
    ]
    const items = deriveAchievements([t], { t1: goals })
    expect(items.map((i) => i.goal.id)).toEqual(['new', 'old', 'undated'])
  })

  it('ignores topics whose goals have not loaded yet', () => {
    expect(deriveAchievements([topic()], {})).toEqual([])
  })
})

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
