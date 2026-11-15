import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { isApiError } from '../../api/errors'
import { useTopic, useTopics, type Topic } from './useTopics'
import type { Goal } from './useTopicGoals'
import { useDeleteGoal, useDeleteTopic } from './mutations'
import { TopicModalsContext, type TopicModalsApi } from './topicModalsContext'
import { TopicDetailModal } from './TopicDetailModal'
import { TopicFormModal } from './TopicFormModal'
import { MilestoneFormModal } from './MilestoneFormModal'
import { ConfirmDelete } from './ConfirmDelete'
import { CelebrationModal } from '../celebration/CelebrationModal'

type State =
  | { kind: 'none' }
  | { kind: 'detail'; topicId: string }
  | { kind: 'create-topic' }
  | { kind: 'edit-topic'; topic: Topic }
  | { kind: 'add-milestone'; topic: Topic }
  | { kind: 'edit-milestone'; topic: Topic; goal: Goal }
  | { kind: 'delete-topic'; topic: Topic }
  | { kind: 'delete-goal'; topic: Topic; goal: Goal }
  | { kind: 'celebrate'; topicId: string }

/** The URL parameter that names the open goal popup (plan D3, Part D). */
const TOPIC_PARAM = 'topic'

/** History-state flag on the entry openDetail pushed, so closing can pop it. */
interface PopupHistoryState {
  topicPopup?: boolean
}

function searchWithTopic(search: string, topicId: string | null): string {
  const params = new URLSearchParams(search)
  if (topicId) params.set(TOPIC_PARAM, topicId)
  else params.delete(TOPIC_PARAM)
  const qs = params.toString()
  return qs ? `?${qs}` : ''
}

/** The topic a popup is about, for every popup that belongs to one. */
function boundTopic(s: State): string | undefined {
  switch (s.kind) {
    case 'detail':
    case 'celebrate':
      return s.topicId
    case 'edit-topic':
    case 'add-milestone':
    case 'edit-milestone':
    case 'delete-topic':
    case 'delete-goal':
      return s.topic.id
    default:
      return undefined
  }
}

/**
 * Owns which topic popup is open and its target. Renders one modal at a time so
 * focus/scroll management stays clean; child popups opened from the detail view
 * return to it on close. Exposes `openDetail`/`openCreate` to the subtree. Mounted
 * once, above the routes (App.tsx), so any page or the app chrome can open them.
 *
 * The open goal lives in the address (`?topic=<id>`): opening pushes a history
 * entry, so Back closes the popup, and a reload or a pasted link reopens it.
 * Only the goal popup is in the URL; forms and confirmations opened from it are
 * transient and come back as the goal popup.
 */
export function ModalHost({ children }: { children: ReactNode }) {
  const location = useLocation()
  const navigate = useNavigate()
  const urlTopicId = new URLSearchParams(location.search).get(TOPIC_PARAM)

  const [state, setState] = useState<State>(() =>
    urlTopicId ? { kind: 'detail', topicId: urlTopicId } : { kind: 'none' },
  )
  const [deleteError, setDeleteError] = useState('')
  const topicsQuery = useTopics()
  const deleteTopic = useDeleteTopic()
  const deleteGoal = useDeleteGoal()

  // Follow the address (Back/Forward, a link): adjust the popup during render
  // when ?topic= changes, rather than in an effect.
  const [syncedTopicId, setSyncedTopicId] = useState(urlTopicId)
  if (urlTopicId !== syncedTopicId) {
    setSyncedTopicId(urlTopicId)
    const bound = boundTopic(state)
    if (urlTopicId && bound !== urlTopicId) setState({ kind: 'detail', topicId: urlTopicId })
    else if (!urlTopicId && bound) setState({ kind: 'none' })
  }

  // Remove ?topic=: step back over the entry openDetail pushed, or, when the
  // page was opened with the parameter (reload, pasted link), replace it.
  const dropTopicParam = useCallback(() => {
    if ((location.state as PopupHistoryState | null)?.topicPopup) navigate(-1)
    else
      navigate(
        { pathname: location.pathname, search: searchWithTopic(location.search, null) },
        { replace: true },
      )
  }, [location, navigate])

  const close = useCallback(() => {
    setDeleteError('')
    setState({ kind: 'none' })
    if (urlTopicId) dropTopicParam()
  }, [urlTopicId, dropTopicParam])

  const backToDetail = useCallback((topicId: string) => {
    setDeleteError('')
    setState({ kind: 'detail', topicId })
  }, [])

  const api = useMemo<TopicModalsApi>(
    () => ({
      openDetail: (topic) => {
        setState({ kind: 'detail', topicId: topic.id })
        if (urlTopicId === topic.id) return
        const to = {
          pathname: location.pathname,
          search: searchWithTopic(location.search, topic.id),
        }
        const historyState: PopupHistoryState = { topicPopup: true }
        navigate(to, { state: historyState })
      },
      openCreate: () => setState({ kind: 'create-topic' }),
    }),
    [location, navigate, urlTopicId],
  )

  // Modals bound to a topic always render the live copy from cache so optimistic
  // progress (and the celebration's 100% stats) reflect the latest state. A
  // linked topic outside this year's list (another year, archived) is fetched
  // on its own.
  const boundTopicId =
    state.kind === 'detail' || state.kind === 'celebrate' ? state.topicId : undefined
  const listTopic = boundTopicId ? topicsQuery.data?.find((t) => t.id === boundTopicId) : undefined
  const fetchSingle = !!boundTopicId && topicsQuery.isSuccess && !listTopic
  const singleQuery = useTopic(boundTopicId, fetchSingle)
  const liveTopic = listTopic ?? (fetchSingle ? singleQuery.data : undefined)

  // A link to a goal that doesn't exist (or isn't yours): forget it quietly.
  const missing = fetchSingle && singleQuery.isError
  useEffect(() => {
    if (missing) close()
  }, [missing, close])

  return (
    <TopicModalsContext value={api}>
      {children}

      {state.kind === 'detail' && liveTopic && (
        <TopicDetailModal
          topic={liveTopic}
          onClose={close}
          onAddMilestone={() => setState({ kind: 'add-milestone', topic: liveTopic })}
          onEditTopic={() => setState({ kind: 'edit-topic', topic: liveTopic })}
          onDeleteTopic={() => setState({ kind: 'delete-topic', topic: liveTopic })}
          onEditGoal={(goal) => setState({ kind: 'edit-milestone', topic: liveTopic, goal })}
          onDeleteGoal={(goal) => setState({ kind: 'delete-goal', topic: liveTopic, goal })}
          onCelebrate={() => setState({ kind: 'celebrate', topicId: liveTopic.id })}
        />
      )}

      {state.kind === 'celebrate' && liveTopic && (
        <CelebrationModal
          topic={liveTopic}
          allTopics={topicsQuery.data ?? []}
          onClose={() => backToDetail(liveTopic.id)}
        />
      )}

      {state.kind === 'create-topic' && <TopicFormModal onClose={close} />}

      {state.kind === 'edit-topic' && (
        <TopicFormModal topic={state.topic} onClose={() => backToDetail(state.topic.id)} />
      )}

      {state.kind === 'add-milestone' && (
        <MilestoneFormModal topic={state.topic} onClose={() => backToDetail(state.topic.id)} />
      )}

      {state.kind === 'edit-milestone' && (
        <MilestoneFormModal
          topic={state.topic}
          goal={state.goal}
          onClose={() => backToDetail(state.topic.id)}
        />
      )}

      {state.kind === 'delete-topic' && (
        <ConfirmDelete
          title="Delete this chapter?"
          message={`"${state.topic.title}" and all its milestones will be permanently deleted.`}
          confirmLabel="Delete chapter"
          pending={deleteTopic.isPending}
          error={deleteError}
          onCancel={() => backToDetail(state.topic.id)}
          onConfirm={() =>
            deleteTopic.mutate(
              { id: state.topic.id },
              {
                onSuccess: close,
                onError: (err) =>
                  setDeleteError(isApiError(err) ? err.message : 'Could not delete. Try again.'),
              },
            )
          }
        />
      )}

      {state.kind === 'delete-goal' && (
        <ConfirmDelete
          title="Delete this milestone?"
          message={`"${state.goal.title}" will be permanently deleted.`}
          confirmLabel="Delete milestone"
          pending={deleteGoal.isPending}
          error={deleteError}
          onCancel={() => backToDetail(state.topic.id)}
          onConfirm={() =>
            deleteGoal.mutate(
              { goal: state.goal },
              {
                onSuccess: () => backToDetail(state.topic.id),
                onError: (err) =>
                  setDeleteError(isApiError(err) ? err.message : 'Could not delete. Try again.'),
              },
            )
          }
        />
      )}
    </TopicModalsContext>
  )
}
