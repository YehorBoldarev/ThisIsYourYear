import { describe, expect, it } from 'vitest'
import { http, HttpResponse } from 'msw'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom'
import { server, tokenResponse } from '../test/msw'
import { AuthProvider } from './AuthProvider'
import { RequireAuth } from './RequireAuth'

function renderProtected(path = '/secret') {
  return render(
    <AuthProvider>
      <MemoryRouter initialEntries={[path]}>
        <Routes>
          <Route path="/login" element={<div>login screen</div>} />
          <Route path="/register" element={<RegisterProbe />} />
          <Route
            path="/secret"
            element={
              <RequireAuth>
                <div>secret content</div>
              </RequireAuth>
            }
          />
        </Routes>
      </MemoryRouter>
    </AuthProvider>,
  )
}

function RegisterProbe() {
  const { search } = useLocation()
  return <div>register screen {search}</div>
}

describe('RequireAuth', () => {
  it('redirects to /login when unauthenticated', async () => {
    // default refresh handler → 401
    renderProtected()
    await waitFor(() => expect(screen.getByText('login screen')).toBeInTheDocument())
    expect(screen.queryByText('secret content')).not.toBeInTheDocument()
  })

  it('sends a signed-out visitor with an invite code straight to sign-up, code kept', async () => {
    renderProtected('/secret?invite=sunflower-2027')
    await waitFor(() =>
      expect(screen.getByText('register screen ?invite=sunflower-2027')).toBeInTheDocument(),
    )
  })

  it('renders children when authenticated', async () => {
    server.use(http.post('/api/v1/auth/refresh', () => HttpResponse.json(tokenResponse())))
    renderProtected()
    await waitFor(() => expect(screen.getByText('secret content')).toBeInTheDocument())
  })
})
