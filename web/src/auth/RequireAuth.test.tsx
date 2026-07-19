import { describe, expect, it } from 'vitest'
import { http, HttpResponse } from 'msw'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { server, tokenResponse } from '../test/msw'
import { AuthProvider } from './AuthProvider'
import { RequireAuth } from './RequireAuth'

function renderProtected() {
  return render(
    <AuthProvider>
      <MemoryRouter initialEntries={['/secret']}>
        <Routes>
          <Route path="/login" element={<div>login screen</div>} />
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

describe('RequireAuth', () => {
  it('redirects to /login when unauthenticated', async () => {
    // default refresh handler → 401
    renderProtected()
    await waitFor(() => expect(screen.getByText('login screen')).toBeInTheDocument())
    expect(screen.queryByText('secret content')).not.toBeInTheDocument()
  })

  it('renders children when authenticated', async () => {
    server.use(http.post('/api/v1/auth/refresh', () => HttpResponse.json(tokenResponse())))
    renderProtected()
    await waitFor(() => expect(screen.getByText('secret content')).toBeInTheDocument())
  })
})
