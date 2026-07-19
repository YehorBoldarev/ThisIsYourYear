import { describe, expect, it, vi } from 'vitest'
import { http, HttpResponse } from 'msw'
import { server } from '../test/msw'
import { TEST_USER, tokenResponse } from '../test/msw'
import { client } from './client'
import { getToken, setToken } from './tokenStore'

describe('api client', () => {
  it('attaches the bearer token to protected requests', async () => {
    let seen: string | null = null
    server.use(
      http.get('/api/v1/me', ({ request }) => {
        seen = request.headers.get('Authorization')
        return HttpResponse.json(TEST_USER)
      }),
    )
    setToken('tok-123')
    const { data } = await client.GET('/api/v1/me')
    expect(seen).toBe('Bearer tok-123')
    expect(data?.email).toBe(TEST_USER.email)
  })

  it('refreshes once and retries after a 401', async () => {
    const refresh = vi.fn()
    server.use(
      http.get('/api/v1/me', ({ request }) => {
        if (request.headers.get('Authorization') === 'Bearer fresh') {
          return HttpResponse.json(TEST_USER)
        }
        return HttpResponse.json(
          { error: { code: 'unauthorized', message: 'nope' } },
          { status: 401 },
        )
      }),
      http.post('/api/v1/auth/refresh', () => {
        refresh()
        return HttpResponse.json(tokenResponse('fresh'))
      }),
    )
    setToken('stale')
    const { data, response } = await client.GET('/api/v1/me')
    expect(response.status).toBe(200)
    expect(data?.email).toBe(TEST_USER.email)
    expect(getToken()).toBe('fresh')
    expect(refresh).toHaveBeenCalledTimes(1)
  })

  it('shares a single refresh across concurrent 401s', async () => {
    const refresh = vi.fn()
    server.use(
      http.get('/api/v1/me', ({ request }) =>
        request.headers.get('Authorization') === 'Bearer fresh'
          ? HttpResponse.json(TEST_USER)
          : HttpResponse.json({ error: { code: 'unauthorized', message: 'no' } }, { status: 401 }),
      ),
      http.post('/api/v1/auth/refresh', () => {
        refresh()
        return HttpResponse.json(tokenResponse('fresh'))
      }),
    )
    setToken('stale')
    const [a, b] = await Promise.all([client.GET('/api/v1/me'), client.GET('/api/v1/me')])
    expect(a.response.status).toBe(200)
    expect(b.response.status).toBe(200)
    expect(refresh).toHaveBeenCalledTimes(1)
  })

  it('clears the token and surfaces the 401 when refresh fails', async () => {
    server.use(
      http.get('/api/v1/me', () =>
        HttpResponse.json({ error: { code: 'unauthorized', message: 'no' } }, { status: 401 }),
      ),
      http.post('/api/v1/auth/refresh', () =>
        HttpResponse.json({ error: { code: 'unauthorized', message: 'expired' } }, { status: 401 }),
      ),
    )
    setToken('stale')
    const { response, error } = await client.GET('/api/v1/me')
    expect(response.status).toBe(401)
    expect(error).toBeDefined()
    expect(getToken()).toBeNull()
  })
})
