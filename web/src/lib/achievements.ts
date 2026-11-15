/*
 * Wins and celebration statistics. A win is a completed milestone flagged as an
 * achievement (★); the server lists them (`GET /achievements`). This module
 * holds the Wins item type and computes the celebration stat strip (screen 7)
 * for a topic that just reached 100%.
 */
import type { components } from '../api/schema'
import type { Topic } from '../features/topics/useTopics'

/** One win: the finished, starred milestone and a summary of its big goal. */
export type Achievement = components['schemas']['Achievement']

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
