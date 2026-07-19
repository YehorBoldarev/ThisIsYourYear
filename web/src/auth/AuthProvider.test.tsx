import { describe, expect, it } from 'vitest'
import { http, HttpResponse } from 'msw'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { server, TEST_USER, tokenResponse } from '../test/msw'
import { AuthProvider } from './AuthProvider'
import { useAuth } from './useAuth'

function Probe() {
  const { status, user, login, register, logout } = useAuth()
  return (
    <div>
      <span data-testid="status">{status}</span>
      <span data-testid="user">{user?.displayName ?? ''}</span>
      <button onClick={() => void login('a@b.co', 'pw', true)}>login</button>
      <button onClick={() => void register('a@b.co', 'Amy', 'password1')}>register</button>
      <button onClick={() => void logout()}>logout</button>
    </div>
  )
}

function renderAuth() {
  return render(
    <AuthProvider>
      <Probe />
    </AuthProvider>,
  )
}

describe('AuthProvider', () => {
  it('boots unauthenticated when the refresh cookie is absent', async () => {
    renderAuth()
    await waitFor(() => expect(screen.getByTestId('status')).toHaveTextContent('unauthenticated'))
  })

  it('restores the session from a valid refresh cookie', async () => {
    server.use(http.post('/api/v1/auth/refresh', () => HttpResponse.json(tokenResponse())))
    renderAuth()
    await waitFor(() => expect(screen.getByTestId('status')).toHaveTextContent('authenticated'))
    expect(screen.getByTestId('user')).toHaveTextContent(TEST_USER.displayName)
  })

  it('registers then auto-logs-in', async () => {
    server.use(
      http.post('/api/v1/auth/register', () => HttpResponse.json(TEST_USER, { status: 201 })),
      http.post('/api/v1/auth/login', () => HttpResponse.json(tokenResponse())),
    )
    const user = userEvent.setup()
    renderAuth()
    await waitFor(() => expect(screen.getByTestId('status')).toHaveTextContent('unauthenticated'))
    await user.click(screen.getByText('register'))
    await waitFor(() => expect(screen.getByTestId('status')).toHaveTextContent('authenticated'))
  })

  it('logs out and clears the user', async () => {
    server.use(http.post('/api/v1/auth/refresh', () => HttpResponse.json(tokenResponse())))
    const user = userEvent.setup()
    renderAuth()
    await waitFor(() => expect(screen.getByTestId('status')).toHaveTextContent('authenticated'))
    await user.click(screen.getByText('logout'))
    await waitFor(() => expect(screen.getByTestId('status')).toHaveTextContent('unauthenticated'))
    expect(screen.getByTestId('user')).toHaveTextContent('')
  })
})
