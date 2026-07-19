import { useMutation, useQueryClient, type QueryClient } from '@tanstack/react-query'
import { client } from '../../api/client'
import { ApiError } from '../../api/errors'
import type { components } from '../../api/schema'
import type { DashboardResponse } from '../dashboard/useDashboard'
import type { Topic } from './useTopics'
import { type Goal, topicGoalsKey } from './useTopicGoals'

type Progress = components['schemas']['Progress']
type CreateTopicBody = components['schemas']['CreateTopicRequest']
type UpdateTopicBody = components['schemas']['UpdateTopicRequest']
type CreateGoalBody = components['schemas']['CreateGoalRequest']
type UpdateGoalBody = components['schemas']['UpdateGoalRequest']

// All topic lists share the `['topics', …]` prefix; goal lists live under
// `['goals', …]` so these optimistic writes never touch each other.
const TOPICS_LISTS = { queryKey: ['topics'] as const }
const DEFAULT_LIST_KEY = ['topics', {}] as const
const DASHBOARD_KEY = ['dashboard'] as const

function progressOf(goals: Goal[]): Progress {
  const total = goals.length
  const completed = goals.filter((g) => g.status === 'done').length
  return { total, completed, ratio: total ? completed / total : 0 }
}

/** Merge a patch into a topic across every cached topic-list query. */
function patchTopicInLists(qc: QueryClient, topicId: string, patch: Partial<Topic>) {
  qc.setQueriesData<Topic[]>(TOPICS_LISTS, (old) =>
    old ? old.map((t) => (t.id === topicId ? { ...t, ...patch } : t)) : old,
  )
}

/** Rederive the dashboard aggregate from the (already-updated) default topic list. */
function recomputeDashboard(qc: QueryClient) {
  const topics = qc.getQueryData<Topic[]>(DEFAULT_LIST_KEY)
  const dash = qc.getQueryData<DashboardResponse>(DASHBOARD_KEY)
  if (!topics || !dash) return
  const total = topics.reduce((sum, t) => sum + t.progress.total, 0)
  const completed = topics.reduce((sum, t) => sum + t.progress.completed, 0)
  const topicsDone = topics.filter((t) => t.progress.total > 0 && t.progress.ratio >= 1).length
  qc.setQueryData<DashboardResponse>(DASHBOARD_KEY, {
    ...dash,
    progress: { total, completed, ratio: total ? completed / total : 0 },
    topics: { total: topics.length, completed: topicsDone },
  })
}

interface Snapshot {
  topics: [readonly unknown[], Topic[] | undefined][]
  goals?: { key: readonly unknown[]; data: Goal[] | undefined }
  dashboard: DashboardResponse | undefined
}

/** Cancel in-flight fetches and snapshot the caches an optimistic write touches. */
async function beginOptimistic(qc: QueryClient, topicId?: string): Promise<Snapshot> {
  await qc.cancelQueries(TOPICS_LISTS)
  await qc.cancelQueries({ queryKey: DASHBOARD_KEY })
  if (topicId) await qc.cancelQueries({ queryKey: topicGoalsKey(topicId) })
  return {
    topics: qc.getQueriesData<Topic[]>(TOPICS_LISTS),
    goals: topicId
      ? { key: topicGoalsKey(topicId), data: qc.getQueryData<Goal[]>(topicGoalsKey(topicId)) }
      : undefined,
    dashboard: qc.getQueryData<DashboardResponse>(DASHBOARD_KEY),
  }
}

function rollback(qc: QueryClient, snap: Snapshot) {
  snap.topics.forEach(([key, data]) => qc.setQueryData(key, data))
  if (snap.goals) qc.setQueryData(snap.goals.key, snap.goals.data)
  qc.setQueryData(DASHBOARD_KEY, snap.dashboard)
}

/** Refetch the authoritative state so server-computed progress/motivation win. */
function settle(qc: QueryClient, topicId?: string) {
  void qc.invalidateQueries(TOPICS_LISTS)
  void qc.invalidateQueries({ queryKey: DASHBOARD_KEY })
  if (topicId) void qc.invalidateQueries({ queryKey: topicGoalsKey(topicId) })
}

async function del(res: { error?: unknown; response: Response }) {
  if (res.error) throw ApiError.fromBody(res.response.status, res.error)
}

function unwrap<T>(res: { data?: T; error?: unknown; response: Response }): T {
  if (res.error || !res.data) throw ApiError.fromBody(res.response.status, res.error)
  return res.data
}

const now = () => new Date().toISOString()

// ---------------------------------------------------------------- topics ----

export function useCreateTopic() {
  const qc = useQueryClient()
  return useMutation<Topic, ApiError, CreateTopicBody, Snapshot>({
    mutationFn: (body) => client.POST('/api/v1/topics', { body }).then(unwrap),
    onMutate: async (body) => {
      const snap = await beginOptimistic(qc)
      const temp: Topic = {
        id: `tmp-${Date.now()}`,
        title: body.title,
        description: body.description ?? '',
        year: body.year ?? new Date().getFullYear(),
        color: body.color ?? '',
        icon: body.icon ?? '',
        category: body.category ?? '',
        targetDate: body.targetDate ?? '',
        archived: false,
        progress: { total: 0, completed: 0, ratio: 0 },
        motivation: '',
        createdAt: now(),
        updatedAt: now(),
      }
      qc.setQueriesData<Topic[]>(TOPICS_LISTS, (old) => (old ? [...old, temp] : old))
      recomputeDashboard(qc)
      return snap
    },
    onError: (_e, _v, snap) => snap && rollback(qc, snap),
    onSettled: () => settle(qc),
  })
}

export function useUpdateTopic() {
  const qc = useQueryClient()
  return useMutation<Topic, ApiError, { id: string; body: UpdateTopicBody }, Snapshot>({
    mutationFn: ({ id, body }) =>
      client.PATCH('/api/v1/topics/{id}', { params: { path: { id } }, body }).then(unwrap),
    onMutate: async ({ id, body }) => {
      const snap = await beginOptimistic(qc)
      patchTopicInLists(qc, id, body as Partial<Topic>)
      recomputeDashboard(qc)
      return snap
    },
    onError: (_e, _v, snap) => snap && rollback(qc, snap),
    onSettled: () => settle(qc),
  })
}

export function useDeleteTopic() {
  const qc = useQueryClient()
  return useMutation<void, ApiError, { id: string }, Snapshot>({
    mutationFn: ({ id }) =>
      client.DELETE('/api/v1/topics/{id}', { params: { path: { id } } }).then(del),
    onMutate: async ({ id }) => {
      const snap = await beginOptimistic(qc)
      qc.setQueriesData<Topic[]>(TOPICS_LISTS, (old) => old?.filter((t) => t.id !== id))
      recomputeDashboard(qc)
      return snap
    },
    onError: (_e, _v, snap) => snap && rollback(qc, snap),
    onSettled: () => settle(qc),
  })
}

// ----------------------------------------------------------------- goals ----

/** Recompute the parent topic's progress from its (mutated) goal list. */
function syncTopicProgress(qc: QueryClient, topicId: string, goals: Goal[]) {
  qc.setQueryData(topicGoalsKey(topicId), goals)
  patchTopicInLists(qc, topicId, { progress: progressOf(goals) })
  recomputeDashboard(qc)
}

export function useCreateGoal(topicId: string) {
  const qc = useQueryClient()
  return useMutation<Goal, ApiError, CreateGoalBody, Snapshot>({
    mutationFn: (body) =>
      client
        .POST('/api/v1/topics/{id}/goals', { params: { path: { id: topicId } }, body })
        .then(unwrap),
    onMutate: async (body) => {
      const snap = await beginOptimistic(qc, topicId)
      const goals = qc.getQueryData<Goal[]>(topicGoalsKey(topicId)) ?? []
      const temp: Goal = {
        id: `tmp-${Date.now()}`,
        topicId,
        title: body.title,
        notes: body.notes ?? '',
        status: 'todo',
        order: goals.length,
        dueDate: body.dueDate ?? '',
        isAchievement: body.isAchievement ?? false,
        createdAt: now(),
        updatedAt: now(),
      }
      syncTopicProgress(qc, topicId, [...goals, temp])
      return snap
    },
    onError: (_e, _v, snap) => snap && rollback(qc, snap),
    onSettled: () => settle(qc, topicId),
  })
}

export function useUpdateGoal() {
  const qc = useQueryClient()
  return useMutation<Goal, ApiError, { goal: Goal; body: UpdateGoalBody }, Snapshot>({
    mutationFn: ({ goal, body }) =>
      client.PATCH('/api/v1/goals/{id}', { params: { path: { id: goal.id } }, body }).then(unwrap),
    onMutate: async ({ goal, body }) => {
      const topicId = goal.topicId
      const snap = await beginOptimistic(qc, topicId)
      const goals = qc.getQueryData<Goal[]>(topicGoalsKey(topicId)) ?? []
      const patched: Goal = {
        ...goal,
        ...(body.title !== undefined ? { title: body.title } : {}),
        ...(body.notes !== undefined ? { notes: body.notes } : {}),
        ...(body.dueDate !== undefined ? { dueDate: body.dueDate } : {}),
        ...(body.isAchievement !== undefined ? { isAchievement: body.isAchievement } : {}),
        ...(body.done !== undefined
          ? {
              status: body.done ? 'done' : 'todo',
              completedAt: body.done ? now() : undefined,
            }
          : {}),
        updatedAt: now(),
      }
      syncTopicProgress(
        qc,
        topicId,
        goals.map((g) => (g.id === goal.id ? patched : g)),
      )
      return snap
    },
    onError: (_e, _v, snap) => snap && rollback(qc, snap),
    onSettled: (_d, _e, { goal }) => settle(qc, goal.topicId),
  })
}

/** Convenience wrapper over {@link useUpdateGoal} for the checklist toggle. */
export function useToggleGoalDone() {
  const update = useUpdateGoal()
  return {
    ...update,
    toggle: (goal: Goal) => update.mutate({ goal, body: { done: goal.status !== 'done' } }),
  }
}

export function useDeleteGoal() {
  const qc = useQueryClient()
  return useMutation<void, ApiError, { goal: Goal }, Snapshot>({
    mutationFn: ({ goal }) =>
      client.DELETE('/api/v1/goals/{id}', { params: { path: { id: goal.id } } }).then(del),
    onMutate: async ({ goal }) => {
      const topicId = goal.topicId
      const snap = await beginOptimistic(qc, topicId)
      const goals = qc.getQueryData<Goal[]>(topicGoalsKey(topicId)) ?? []
      syncTopicProgress(
        qc,
        topicId,
        goals.filter((g) => g.id !== goal.id),
      )
      return snap
    },
    onError: (_e, _v, snap) => snap && rollback(qc, snap),
    onSettled: (_d, _e, { goal }) => settle(qc, goal.topicId),
  })
}
