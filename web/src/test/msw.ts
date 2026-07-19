import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import type { TokenResponse, UserResponse } from '../api/client'
import type { DashboardResponse } from '../features/dashboard/useDashboard'
import type { Topic } from '../features/topics/useTopics'
import type { Goal } from '../features/topics/useTopicGoals'

export const TEST_USER: UserResponse = {
  id: 'u1',
  email: 'nostra@example.com',
  displayName: 'Nostradamus Affterfluchell',
  createdAt: '2026-01-01T00:00:00Z',
}

export function tokenResponse(accessToken = 'access-1'): TokenResponse {
  return {
    accessToken,
    expiresAt: '2026-12-31T00:00:00Z',
    user: TEST_USER,
  }
}

export function errorBody(code: string, message: string, fields?: Record<string, string>) {
  return { error: { code, message, ...(fields ? { fields } : {}) } }
}

export function dashboardResponse(overrides: Partial<DashboardResponse> = {}): DashboardResponse {
  return {
    year: 2026,
    message: 'Small days compound into a remarkable year.',
    progress: { total: 0, completed: 0, ratio: 0 },
    topics: { total: 0, completed: 0 },
    countdown: { target: '2027-01-01T00:00:00Z', secondsRemaining: 15_724_800 },
    ...overrides,
  }
}

export function topicResponse(overrides: Partial<Topic> = {}): Topic {
  return {
    id: 't1',
    title: 'Run a Half-Marathon',
    description: '',
    year: 2026,
    color: '#3ec5a0',
    icon: '',
    category: 'Endurance',
    targetDate: '',
    archived: false,
    progress: { total: 8, completed: 6, ratio: 0.75 },
    motivation: 'You can see it from here.',
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

export function goalResponse(overrides: Partial<Goal> = {}): Goal {
  return {
    id: 'g1',
    topicId: 't1',
    title: 'Register for the race',
    notes: '',
    status: 'todo',
    order: 0,
    dueDate: '',
    isAchievement: false,
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

// Default handlers assume a signed-out visitor (refresh 401). Tests override
// with server.use(...) for the flows they exercise.
export const handlers = [
  http.post('/api/v1/auth/refresh', () =>
    HttpResponse.json(errorBody('unauthorized', 'no session'), { status: 401 }),
  ),
  http.post('/api/v1/auth/logout', () => new HttpResponse(null, { status: 204 })),
  http.get('/api/v1/dashboard', () => HttpResponse.json(dashboardResponse())),
  http.get('/api/v1/topics', () => HttpResponse.json([])),
  http.post('/api/v1/topics', () => HttpResponse.json(topicResponse(), { status: 201 })),
  http.patch('/api/v1/topics/:id', () => HttpResponse.json(topicResponse())),
  http.delete('/api/v1/topics/:id', () => new HttpResponse(null, { status: 204 })),
  http.get('/api/v1/topics/:id/goals', () => HttpResponse.json([])),
  http.post('/api/v1/topics/:id/goals', () => HttpResponse.json(goalResponse(), { status: 201 })),
  http.patch('/api/v1/goals/:id', () => HttpResponse.json(goalResponse())),
  http.delete('/api/v1/goals/:id', () => new HttpResponse(null, { status: 204 })),
]

export const server = setupServer(...handlers)
