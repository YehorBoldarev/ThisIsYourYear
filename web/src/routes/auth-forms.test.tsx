import { describe, expect, it } from 'vitest'
import { http, HttpResponse } from 'msw'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { server, TEST_USER, tokenResponse, errorBody } from '../test/msw'
import { AuthProvider } from '../auth/AuthProvider'
import { ThemeProvider } from '../theme/ThemeProvider'
import { Login } from './Login'
import { Register } from './Register'

function renderAt(path: string) {
  return render(
    <ThemeProvider>
      <AuthProvider>
        <MemoryRouter initialEntries={[path]}>
          <Routes>
            <Route path="/login" element={<Login />} />
            <Route path="/register" element={<Register />} />
            <Route path="/" element={<div>home screen</div>} />
          </Routes>
        </MemoryRouter>
      </AuthProvider>
    </ThemeProvider>,
  )
}

describe('Login', () => {
  it('signs in and navigates home', async () => {
    server.use(http.post('/api/v1/auth/login', () => HttpResponse.json(tokenResponse())))
    const user = userEvent.setup()
    renderAt('/login')
    await user.type(screen.getByLabelText('Email'), 'a@b.co')
    await user.type(screen.getByLabelText('Password'), 'password1')
    await user.click(screen.getByRole('button', { name: 'Sign in' }))
    await waitFor(() => expect(screen.getByText('home screen')).toBeInTheDocument())
  })

  it('shows a friendly message on bad credentials (401)', async () => {
    server.use(
      http.post('/api/v1/auth/login', () =>
        HttpResponse.json(errorBody('unauthorized', 'bad'), { status: 401 }),
      ),
    )
    const user = userEvent.setup()
    renderAt('/login')
    await user.type(screen.getByLabelText('Email'), 'a@b.co')
    await user.type(screen.getByLabelText('Password'), 'wrong')
    await user.click(screen.getByRole('button', { name: 'Sign in' }))
    expect(await screen.findByText('Email or password is incorrect.')).toBeInTheDocument()
  })

  it('maps 422 field errors inline', async () => {
    server.use(
      http.post('/api/v1/auth/login', () =>
        HttpResponse.json(errorBody('validation_error', 'invalid', { email: 'must be an email' }), {
          status: 422,
        }),
      ),
    )
    const user = userEvent.setup()
    renderAt('/login')
    await user.type(screen.getByLabelText('Email'), 'nope')
    await user.type(screen.getByLabelText('Password'), 'password1')
    await user.click(screen.getByRole('button', { name: 'Sign in' }))
    expect(await screen.findByText('must be an email')).toBeInTheDocument()
  })
})

describe('Register', () => {
  it('maps a duplicate email (409) to the email field', async () => {
    server.use(
      http.post('/api/v1/auth/register', () =>
        HttpResponse.json(errorBody('conflict', 'exists'), { status: 409 }),
      ),
    )
    const user = userEvent.setup()
    renderAt('/register')
    await user.type(screen.getByLabelText('Email'), 'taken@b.co')
    await user.type(screen.getByLabelText('Display name'), 'Amy')
    await user.type(screen.getByLabelText('Password'), 'password1')
    await user.click(screen.getByRole('button', { name: 'Create account' }))
    expect(await screen.findByText('That email is already registered.')).toBeInTheDocument()
  })

  it('creates an account and lands home', async () => {
    server.use(
      http.post('/api/v1/auth/register', () => HttpResponse.json(TEST_USER, { status: 201 })),
      http.post('/api/v1/auth/login', () => HttpResponse.json(tokenResponse())),
    )
    const user = userEvent.setup()
    renderAt('/register')
    await user.type(screen.getByLabelText('Email'), 'new@b.co')
    await user.type(screen.getByLabelText('Display name'), 'Newbie')
    await user.type(screen.getByLabelText('Password'), 'password1')
    await user.click(screen.getByRole('button', { name: 'Create account' }))
    await waitFor(() => expect(screen.getByText('home screen')).toBeInTheDocument())
  })
})
