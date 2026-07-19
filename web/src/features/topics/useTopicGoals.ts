import { useQuery } from '@tanstack/react-query'
import { client } from '../../api/client'
import { ApiError } from '../../api/errors'
import type { components } from '../../api/schema'

export type Goal = components['schemas']['Goal']

/** Cache key for a topic's milestone list. Kept off the `['topics', …]` prefix
 *  so topic-list optimistic updates never collide with goal lists. */
export function topicGoalsKey(topicId: string) {
  return ['goals', topicId] as const
}

export function useTopicGoals(topicId: string) {
  return useQuery({
    queryKey: topicGoalsKey(topicId),
    queryFn: async (): Promise<Goal[]> => {
      const { data, error, response } = await client.GET('/api/v1/topics/{id}/goals', {
        params: { path: { id: topicId } },
      })
      if (error || !data) throw ApiError.fromBody(response.status, error)
      return data
    },
  })
}
