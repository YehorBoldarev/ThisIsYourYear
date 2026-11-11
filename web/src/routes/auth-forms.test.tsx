import { describe, expect, it } from 'vitest'
import { http, HttpResponse } from 'msw'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { server, TEST_USER, tokenResponse, errorBody } from '../test/msw'
import { AuthProvider } from '../auth/AuthProvider'
import { ThemeProvider } from '../theme/ThemeProvider'
import { Login } from './Login'
import { Register } from './Register'

function renderAt(path: string) {
  // A fresh client per render: the sign-up policy is cached indefinitely.
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <ThemeProvider>
      <QueryClientProvider client={qc}>
        <AuthProvider>
          <MemoryRouter initialEntries={[path]}>
            <Routes>
              <Route path="/login" element={<Login />} />
              <Route path="/register" element={<Register />} />
              <Route path="/" element={<div>home screen</div>} />
            </Routes>
          </MemoryRouter>
        </AuthProvider>
      </QueryClientProvider>
    </ThemeProvider>,
  )
}

const inviteOnly = () =>
  http.get('/api/v1/auth/registration', () => HttpResponse.json({ inviteRequired: true }))

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

  it('hides the invite field when sign-up is open', async () => {
    renderAt('/register')
    expect(await screen.findByLabelText('Email')).toBeInTheDocument()
    await waitFor(() => expect(screen.queryByLabelText('Invite code')).not.toBeInTheDocument())
  })

  it('asks for an invite code when sign-up is invite-only and sends it', async () => {
    let sent: unknown
    server.use(
      inviteOnly(),
      http.post('/api/v1/auth/register', async ({ request }) => {
        sent = await request.json()
        return HttpResponse.json(TEST_USER, { status: 201 })
      }),
      http.post('/api/v1/auth/login', () => HttpResponse.json(tokenResponse())),
    )
    const user = userEvent.setup()
    renderAt('/register')
    await user.type(await screen.findByLabelText('Invite code'), 'sunflower-2027')
    await user.type(screen.getByLabelText('Email'), 'fam@b.co')
    await user.type(screen.getByLabelText('Display name'), 'Fam')
    await user.type(screen.getByLabelText('Password'), 'password1')
    await user.click(screen.getByRole('button', { name: 'Create account' }))
    await waitFor(() => expect(screen.getByText('home screen')).toBeInTheDocument())
    expect(sent).toMatchObject({ email: 'fam@b.co', inviteCode: 'sunflower-2027' })
  })

  it('prefills the code from an invite link and shows a rejected code inline', async () => {
    server.use(
      inviteOnly(),
      http.post('/api/v1/auth/register', () =>
        HttpResponse.json(
          errorBody('validation_error', 'invalid', { inviteCode: 'is not valid' }),
          { status: 422 },
        ),
      ),
    )
    const user = userEvent.setup()
    renderAt('/register?invite=old-code-2026')
    expect(await screen.findByLabelText('Invite code')).toHaveValue('old-code-2026')
    await user.type(screen.getByLabelText('Email'), 'fam@b.co')
    await user.type(screen.getByLabelText('Display name'), 'Fam')
    await user.type(screen.getByLabelText('Password'), 'password1')
    await user.click(screen.getByRole('button', { name: 'Create account' }))
    expect(await screen.findByText('is not valid')).toBeInTheDocument()
  })
})
