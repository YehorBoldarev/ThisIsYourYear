import { describe, expect, it } from 'vitest'
import type { ReactNode } from 'react'
import { http, HttpResponse } from 'msw'
import { act, renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { server, goalResponse, topicResponse, errorBody } from '../../test/msw'
import type { Topic } from './useTopics'
import type { Goal } from './useTopicGoals'
import { useToggleGoalDone } from './mutations'

// No component observes these queries, so invalidate-on-settle won't refetch —
// the optimistic writes (and rollbacks) stay put and are directly assertable.
function seed() {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  qc.setQueryData(['goals', 't1'], [goalResponse({ id: 'g1', topicId: 't1', status: 'todo' })])
  qc.setQueryData(
    ['topics', {}],
    [topicResponse({ id: 't1', progress: { total: 1, completed: 0, ratio: 0 } })],
  )
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={qc}>{children}</QueryClientProvider>
  )
  return { qc, wrapper }
}

const goal = (): Goal => goalResponse({ id: 'g1', topicId: 't1', status: 'todo' })

describe('useToggleGoalDone', () => {
  it('optimistically completes a milestone and bumps topic progress', async () => {
    server.use(
      http.patch('/api/v1/goals/:id', () =>
        HttpResponse.json(goalResponse({ id: 'g1', status: 'done' })),
      ),
    )
    const { qc, wrapper } = seed()
    const { result } = renderHook(() => useToggleGoalDone(), { wrapper })

    act(() => result.current.toggle(goal()))

    await waitFor(() => {
      expect(qc.getQueryData<Goal[]>(['goals', 't1'])?.[0].status).toBe('done')
      expect(qc.getQueryData<Topic[]>(['topics', {}])?.[0].progress).toEqual({
        total: 1,
        completed: 1,
        ratio: 1,
      })
    })
  })

  it('rolls back when the request fails', async () => {
    server.use(
      http.patch('/api/v1/goals/:id', () =>
        HttpResponse.json(errorBody('internal', 'boom'), { status: 500 }),
      ),
    )
    const { qc, wrapper } = seed()
    const { result } = renderHook(() => useToggleGoalDone(), { wrapper })

    act(() => result.current.toggle(goal()))

    // Settles back to the pre-mutation state after the server rejects.
    await waitFor(() => {
      expect(qc.getQueryData<Goal[]>(['goals', 't1'])?.[0].status).toBe('todo')
      expect(qc.getQueryData<Topic[]>(['topics', {}])?.[0].progress.completed).toBe(0)
    })
  })
})
