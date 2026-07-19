/*
 * Client-side derivation of "wins" — there is no backend achievements endpoint.
 * A win is a completed milestone flagged as an achievement (★). We flatten the
 * caller's topics + their goal lists into achievements, and compute the celebration
 * stat strip (screen 7) for a topic that just reached 100%.
 */
import type { Topic } from '../features/topics/useTopics'
import type { Goal } from '../features/topics/useTopicGoals'

export interface Achievement {
  goal: Goal
  topic: Topic
}

function completedTime(goal: Goal): number {
  const t = goal.completedAt ? Date.parse(goal.completedAt) : NaN
  return Number.isNaN(t) ? 0 : t
}

/**
 * Completed, starred milestones across the given topics, newest completion first.
 * Milestones with no `completedAt` sort last.
 */
export function deriveAchievements(
  topics: Topic[],
  goalsByTopic: Record<string, Goal[] | undefined>,
): Achievement[] {
  const items: Achievement[] = []
  for (const topic of topics) {
    const goals = goalsByTopic[topic.id]
    if (!goals) continue
    for (const goal of goals) {
      if (goal.status === 'done' && goal.isAchievement) items.push({ goal, topic })
    }
  }
  return items.sort((a, b) => completedTime(b.goal) - completedTime(a.goal))
}

export interface CelebrationStats {
  /** "6/6" — completed over total milestones. */
  milestones: string
  /** "5 mo" / "<1 mo" — time from the topic's creation to now. */
  duration: string
  /** Ordinal of this topic among the year's completed topics, e.g. 1. */
  rank: number
}

function monthsSince(createdAt: string, now: Date): string {
  const start = Date.parse(createdAt)
  if (Number.isNaN(start)) return '—'
  const months = Math.round((now.getTime() - start) / (1000 * 60 * 60 * 24 * 30.44))
  return months < 1 ? '<1 mo' : `${months} mo`
}

/** How many of `allTopics` in this topic's year are 100% complete. */
function completedRank(topic: Topic, allTopics: Topic[]): number {
  const completed = allTopics.filter(
    (t) => t.year === topic.year && t.progress.total > 0 && t.progress.ratio >= 1,
  )
  const sorted = [...completed].sort((a, b) => Date.parse(a.updatedAt) - Date.parse(b.updatedAt))
  const idx = sorted.findIndex((t) => t.id === topic.id)
  return idx >= 0 ? idx + 1 : sorted.length + 1
}

export function celebrationStats(
  topic: Topic,
  allTopics: Topic[],
  now: Date = new Date(),
): CelebrationStats {
  const { completed, total } = topic.progress
  return {
    milestones: `${completed}/${total}`,
    duration: monthsSince(topic.createdAt, now),
    rank: completedRank(topic, allTopics),
  }
}
