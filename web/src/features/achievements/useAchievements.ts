import { useQuery } from '@tanstack/react-query'
import { client } from '../../api/client'
import { ApiError } from '../../api/errors'
import type { Achievement } from '../../lib/achievements'

/** Query key for the Wins list; goal/topic mutations invalidate it (mutations.ts). */
export const ACHIEVEMENTS_KEY = ['achievements'] as const

interface AchievementsResult {
  items: Achievement[]
  isLoading: boolean
  isError: boolean
  refetch: () => void
}

/**
 * The "Wins" list: the caller's finished, starred milestones for the season
 * year, newest first, in one request (`GET /achievements`).
 */
export function useAchievements(): AchievementsResult {
  const query = useQuery({
    queryKey: ACHIEVEMENTS_KEY,
    queryFn: async (): Promise<Achievement[]> => {
      const { data, error, response } = await client.GET('/api/v1/achievements')
      if (error || !data) throw ApiError.fromBody(response.status, error)
      return data
    },
  })
  return {
    items: query.data ?? [],
    isLoading: query.isPending,
    isError: query.isError,
    refetch: () => void query.refetch(),
  }
}
