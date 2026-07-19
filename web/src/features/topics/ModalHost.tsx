import { useCallback, useMemo, useState, type ReactNode } from 'react'
import { useNavigate } from 'react-router-dom'
import { isApiError } from '../../api/errors'
import { useTopics, type Topic } from './useTopics'
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

/**
 * Owns which topic popup is open and its target. Renders one modal at a time so
 * focus/scroll management stays clean; child popups opened from the detail view
 * return to it on close. Exposes `openDetail`/`openCreate` to the subtree.
 */
export function ModalHost({ children }: { children: ReactNode }) {
  const [state, setState] = useState<State>({ kind: 'none' })
  const [deleteError, setDeleteError] = useState('')
  const navigate = useNavigate()
  const topicsQuery = useTopics()
  const deleteTopic = useDeleteTopic()
  const deleteGoal = useDeleteGoal()

  const close = useCallback(() => {
    setDeleteError('')
    setState({ kind: 'none' })
  }, [])

  const backToDetail = useCallback((topicId: string) => {
    setDeleteError('')
    setState({ kind: 'detail', topicId })
  }, [])

  const api = useMemo<TopicModalsApi>(
    () => ({
      openDetail: (topic) => setState({ kind: 'detail', topicId: topic.id }),
      openCreate: () => setState({ kind: 'create-topic' }),
    }),
    [],
  )

  // Modals bound to a topic always render the live copy from cache so optimistic
  // progress (and the celebration's 100% stats) reflect the latest state.
  const boundTopicId =
    state.kind === 'detail' || state.kind === 'celebrate' ? state.topicId : undefined
  const liveTopic = boundTopicId ? topicsQuery.data?.find((t) => t.id === boundTopicId) : undefined

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
          onReflect={() => {
            close()
            navigate('/reflections')
          }}
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
