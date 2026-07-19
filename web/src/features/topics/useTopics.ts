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
