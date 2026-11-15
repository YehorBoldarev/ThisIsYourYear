import { useQuery } from '@tanstack/react-query'
import { client } from '../../api/client'
import { ApiError } from '../../api/errors'
import type { components } from '../../api/schema'

export type Topic = components['schemas']['Topic']

export interface TopicsQuery {
  year?: number
  includeArchived?: boolean
}

export function useTopics(params: TopicsQuery = {}) {
  return useQuery({
    queryKey: ['topics', params],
    queryFn: async (): Promise<Topic[]> => {
      const { data, error, response } = await client.GET('/api/v1/topics', {
        params: { query: params },
      })
      if (error || !data) throw ApiError.fromBody(response.status, error)
      return data
    },
  })
}

/** Query key for one topic fetched on its own (see useTopic). */
export function topicKey(id: string) {
  return ['topic', id] as const
}

/**
 * One topic by id, for a popup opened from a link (`?topic=`) when the topic
 * isn't in the current year's list (another year, or archived). Separate from
 * the `['topics', …]` lists so optimistic list updates never touch it.
 */
export function useTopic(id: string | undefined, enabled: boolean) {
  return useQuery({
    queryKey: topicKey(id ?? ''),
    enabled: enabled && !!id,
    retry: false,
    queryFn: async (): Promise<Topic> => {
      const { data, error, response } = await client.GET('/api/v1/topics/{id}', {
        params: { path: { id: id ?? '' } },
      })
      if (error || !data) throw ApiError.fromBody(response.status, error)
      return data
    },
  })
}
