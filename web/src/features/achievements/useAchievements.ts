import { useQueries } from '@tanstack/react-query'
import { client } from '../../api/client'
import { ApiError } from '../../api/errors'
import { deriveAchievements, type Achievement } from '../../lib/achievements'
import { useTopics, type Topic } from '../topics/useTopics'
import { topicGoalsKey, type Goal } from '../topics/useTopicGoals'

interface AchievementsResult {
  items: Achievement[]
  topics: Topic[]
  isLoading: boolean
  isError: boolean
  refetch: () => void
}

/**
 * Derives the "Wins" list. There is no bulk goals endpoint, so we fan out one
 * `['goals', topicId]` query per current-year topic — reusing the cache the topic
 * detail already warms — then flatten to completed, starred milestones.
 */
export function useAchievements(): AchievementsResult {
  const topicsQuery = useTopics()
  const topics = topicsQuery.data ?? []

  const goalQueries = useQueries({
    queries: topics.map((topic) => ({
      queryKey: topicGoalsKey(topic.id),
      queryFn: async (): Promise<Goal[]> => {
        const { data, error, response } = await client.GET('/api/v1/topics/{id}/goals', {
          params: { path: { id: topic.id } },
        })
        if (error || !data) throw ApiError.fromBody(response.status, error)
        return data
      },
    })),
  })

  const goalsByTopic: Record<string, Goal[] | undefined> = {}
  topics.forEach((topic, i) => {
    goalsByTopic[topic.id] = goalQueries[i]?.data
  })

  return {
    items: deriveAchievements(topics, goalsByTopic),
    topics,
    isLoading: topicsQuery.isPending || goalQueries.some((q) => q.isPending),
    isError: topicsQuery.isError || goalQueries.some((q) => q.isError),
    refetch: () => {
      void topicsQuery.refetch()
      goalQueries.forEach((q) => void q.refetch())
    },
  }
}
