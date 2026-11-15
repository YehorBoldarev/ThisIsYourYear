import type { QueryClient } from '@tanstack/react-query'
import { useDashboard, type DashboardResponse } from './useDashboard'

/**
 * The year the app is about, as the server decided it for this user's timezone
 * (see GET /dashboard): during the preparation stage this is the year being
 * planned, otherwise the current year. Until the dashboard has loaded it falls
 * back to the browser's year, outside the preparation stage.
 */
export interface Season {
  year: number
  preparing: boolean
}

export function seasonFrom(dashboard: DashboardResponse | undefined): Season {
  return {
    year: dashboard?.year ?? new Date().getFullYear(),
    preparing: dashboard?.phase === 'preparation',
  }
}

export function useSeason(): Season {
  return seasonFrom(useDashboard().data)
}

/** The season year from the query cache, for non-component code (mutations). */
export function seasonYear(qc: QueryClient): number {
  return seasonFrom(qc.getQueryData<DashboardResponse>(['dashboard'])).year
}
