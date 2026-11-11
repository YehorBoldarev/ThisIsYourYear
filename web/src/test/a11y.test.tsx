import { describe, expect, it } from 'vitest'
import { http, HttpResponse } from 'msw'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { axe } from 'vitest-axe'
import { dashboardResponse, goalResponse, server, tokenResponse, topicResponse } from './msw'
import { AuthProvider } from '../auth/AuthProvider'
import { ThemeProvider } from '../theme/ThemeProvider'
import { Login } from '../routes/Login'
import { Register } from '../routes/Register'
import { Dashboard } from '../features/dashboard/Dashboard'
import { ModalHost } from '../features/topics/ModalHost'
import { AchievementsPage } from '../features/achievements/AchievementsPage'
import { TopicFormModal } from '../features/topics/TopicFormModal'

// Component-scoped renders aren't whole pages, so the page-structure rules
// (single main, one h1, everything-in-a-landmark) don't apply — disable them and
// let axe check the element-level a11y that matters here (roles, names, labels,
// aria, duplicate ids). Colour contrast is verified analytically, not in jsdom.
async function expectNoViolations(container: HTMLElement) {
  const results = await axe(container, {
    rules: {
      region: { enabled: false },
      'landmark-one-main': { enabled: false },
      'page-has-heading-one': { enabled: false },
    },
  })
  expect(results).toHaveNoViolations()
}

function client() {
  return new QueryClient({ defaultOptions: { queries: { retry: false } } })
}

describe('accessibility (axe)', () => {
  it('login screen has no violations', async () => {
    const { container } = render(
      <MemoryRouter>
        <QueryClientProvider client={client()}>
          <AuthProvider>
            <ThemeProvider>
              <Login />
            </ThemeProvider>
          </AuthProvider>
        </QueryClientProvider>
      </MemoryRouter>,
    )
    expect(await screen.findByRole('button', { name: /Sign in/i })).toBeInTheDocument()
    await expectNoViolations(container)
  })

  it('invite-only sign-up screen has no violations', async () => {
    server.use(
      http.get('/api/v1/auth/registration', () => HttpResponse.json({ inviteRequired: true })),
    )
    const { container } = render(
      <MemoryRouter initialEntries={['/register?invite=sunflower-2027']}>
        <QueryClientProvider client={client()}>
          <AuthProvider>
            <ThemeProvider>
              <Register />
            </ThemeProvider>
          </AuthProvider>
        </QueryClientProvider>
      </MemoryRouter>,
    )
    expect(await screen.findByLabelText('Invite code')).toHaveValue('sunflower-2027')
    await expectNoViolations(container)
  })

  it('populated dashboard has no violations', async () => {
    server.use(
      http.post('/api/v1/auth/refresh', () => HttpResponse.json(tokenResponse())),
      http.get('/api/v1/dashboard', () =>
        HttpResponse.json(dashboardResponse({ topics: { total: 1, completed: 0 } })),
      ),
      http.get('/api/v1/topics', () => HttpResponse.json([topicResponse()])),
    )
    const { container } = render(
      <MemoryRouter>
        <QueryClientProvider client={client()}>
          <AuthProvider>
            <ThemeProvider>
              <ModalHost>
                <Dashboard />
              </ModalHost>
            </ThemeProvider>
          </AuthProvider>
        </QueryClientProvider>
      </MemoryRouter>,
    )
    expect(await screen.findAllByText('Run a Half-Marathon')).not.toHaveLength(0)
    await expectNoViolations(container)
  })

  it('wins page has no violations', async () => {
    server.use(
      http.get('/api/v1/topics', () => HttpResponse.json([topicResponse({ id: 't1' })])),
      http.get('/api/v1/topics/:id/goals', () =>
        HttpResponse.json([
          goalResponse({ id: 'g1', title: 'First 10k', status: 'done', isAchievement: true }),
        ]),
      ),
    )
    const { container } = render(
      <QueryClientProvider client={client()}>
        <ThemeProvider>
          <AchievementsPage />
        </ThemeProvider>
      </QueryClientProvider>,
    )
    expect(await screen.findByText('First 10k')).toBeInTheDocument()
    await expectNoViolations(container)
  })

  it('an open form modal has no violations', async () => {
    const { baseElement } = render(
      <QueryClientProvider client={client()}>
        <ThemeProvider>
          <TopicFormModal onClose={() => {}} />
        </ThemeProvider>
      </QueryClientProvider>,
    )
    expect(await screen.findByRole('dialog')).toBeInTheDocument()
    // The modal renders through a portal, so check the whole document body.
    await expectNoViolations(baseElement)
  })
})
