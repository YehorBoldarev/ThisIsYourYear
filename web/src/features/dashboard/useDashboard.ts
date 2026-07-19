import { useQuery } from '@tanstack/react-query'
import { client } from '../../api/client'
import { ApiError } from '../../api/errors'
import type { components } from '../../api/schema'

export type DashboardResponse = components['schemas']['Dashboard']

export function useDashboard() {
  return useQuery({
    queryKey: ['dashboard'],
    queryFn: async (): Promise<DashboardResponse> => {
      const { data, error, response } = await client.GET('/api/v1/dashboard')
      if (error || !data) throw ApiError.fromBody(response.status, error)
      return data
    },
  })
}
