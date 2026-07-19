import { describe, expect, it, vi } from 'vitest'
import { http, HttpResponse } from 'msw'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { server, errorBody, topicResponse } from '../../test/msw'
import { ThemeProvider } from '../../theme/ThemeProvider'
import { TopicFormModal } from './TopicFormModal'

function renderModal(ui: React.ReactElement) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <ThemeProvider>
      <QueryClientProvider client={qc}>{ui}</QueryClientProvider>
    </ThemeProvider>,
  )
}

describe('TopicFormModal', () => {
  it('creates a chapter and closes on success', async () => {
    const onClose = vi.fn()
    let created: unknown
    server.use(
      http.post('/api/v1/topics', async ({ request }) => {
        created = await request.json()
        return HttpResponse.json(topicResponse(), { status: 201 })
      }),
    )
    renderModal(<TopicFormModal onClose={onClose} />)

    await userEvent.type(screen.getByLabelText('What is the goal?'), 'Learn piano')
    await userEvent.click(screen.getByRole('button', { name: 'Create goal' }))

    await waitFor(() => expect(onClose).toHaveBeenCalled())
    expect(created).toMatchObject({ title: 'Learn piano', color: '#3ec5a0' })
  })

  it('blocks submit and shows an error when the title is empty', async () => {
    const onClose = vi.fn()
    renderModal(<TopicFormModal onClose={onClose} />)

    await userEvent.click(screen.getByRole('button', { name: 'Create goal' }))

    expect(await screen.findByText('A title is required.')).toBeInTheDocument()
    expect(onClose).not.toHaveBeenCalled()
  })

  it('maps a 422 field error onto the input', async () => {
    server.use(
      http.post('/api/v1/topics', () =>
        HttpResponse.json(errorBody('validation_error', 'invalid', { title: 'Too long.' }), {
          status: 422,
        }),
      ),
    )
    renderModal(<TopicFormModal onClose={vi.fn()} />)

    await userEvent.type(screen.getByLabelText('What is the goal?'), 'x')
    await userEvent.click(screen.getByRole('button', { name: 'Create goal' }))

    expect(await screen.findByText('Too long.')).toBeInTheDocument()
  })

  it('pre-fills when editing an existing chapter', () => {
    renderModal(
      <TopicFormModal topic={topicResponse({ title: 'Run a Half-Marathon' })} onClose={vi.fn()} />,
    )
    expect(screen.getByLabelText('What is the goal?')).toHaveValue('Run a Half-Marathon')
    expect(screen.getByRole('button', { name: 'Save changes' })).toBeInTheDocument()
  })
})
