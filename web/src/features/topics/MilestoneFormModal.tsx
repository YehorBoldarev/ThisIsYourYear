import { useState } from 'react'
import { Button, Field, Label, Modal, Textarea, Toggle } from '../../components'
import { ACCENT_HEX } from '../../components/accent'
import { isApiError, type FieldErrors } from '../../api/errors'
import { toApiDate, toInputDate } from '../../lib/date'
import type { Topic } from './useTopics'
import type { Goal } from './useTopicGoals'
import { useCreateGoal, useUpdateGoal } from './mutations'

interface MilestoneFormModalProps {
  topic: Topic
  /** Present → edit that milestone; absent → add a new one. */
  goal?: Goal
  onClose: () => void
}

export function MilestoneFormModal({ topic, goal, onClose }: MilestoneFormModalProps) {
  const editing = !!goal
  const [title, setTitle] = useState(goal?.title ?? '')
  const [dueDate, setDueDate] = useState(toInputDate(goal?.dueDate ?? ''))
  const [notes, setNotes] = useState(goal?.notes ?? '')
  const [isAchievement, setIsAchievement] = useState(goal?.isAchievement ?? false)
  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({})
  const [formError, setFormError] = useState('')

  const create = useCreateGoal(topic.id)
  const update = useUpdateGoal()
  const pending = create.isPending || update.isPending
  const accent = topic.color || ACCENT_HEX.teal

  function fail(err: unknown) {
    if (isApiError(err) && err.fields) setFieldErrors(err.fields)
    else setFormError(isApiError(err) ? err.message : 'Something went wrong. Please try again.')
  }

  function submit(e: React.FormEvent) {
    e.preventDefault()
    setFieldErrors({})
    setFormError('')
    if (!title.trim()) {
      setFieldErrors({ title: 'A title is required.' })
      return
    }
    if (editing) {
      update.mutate(
        {
          goal,
          body: {
            title: title.trim(),
            notes: notes.trim(),
            dueDate: toApiDate(dueDate),
            isAchievement,
          },
        },
        { onSuccess: onClose, onError: fail },
      )
    } else {
      const due = toApiDate(dueDate)
      create.mutate(
        {
          title: title.trim(),
          isAchievement,
          ...(notes.trim() ? { notes: notes.trim() } : {}),
          ...(due ? { dueDate: due } : {}),
        },
        { onSuccess: onClose, onError: fail },
      )
    }
  }

  const eyebrow = (
    <span className="inline-flex items-center gap-2">
      <span className="size-2 rounded-full" style={{ background: accent }} aria-hidden="true" />
      {topic.title}
    </span>
  )

  return (
    <Modal
      open
      onClose={onClose}
      eyebrow={eyebrow}
      title={editing ? 'Edit milestone' : 'Add a milestone'}
      width={520}
    >
      <form onSubmit={submit} noValidate className="space-y-5">
        <Field
          id="milestone-title"
          label="Milestone"
          value={title}
          onChange={setTitle}
          error={fieldErrors.title}
          placeholder="Register for the race"
          required
        />

        <Field
          id="milestone-due"
          label="Target date — optional"
          value={dueDate}
          onChange={setDueDate}
          type="date"
          error={fieldErrors.dueDate}
        />

        <div className="space-y-2">
          <Label htmlFor="milestone-notes">Notes — optional</Label>
          <Textarea
            id="milestone-notes"
            value={notes}
            onChange={(e) => setNotes(e.target.value)}
            invalid={!!fieldErrors.notes}
            placeholder="Anything worth remembering about this step."
          />
          {fieldErrors.notes && <p className="text-[12px] text-rose">{fieldErrors.notes}</p>}
        </div>

        <div className="flex items-center justify-between gap-4 rounded-field border border-border bg-surface p-4">
          <div>
            <div className="text-sm font-semibold text-text">Celebrate as an achievement ★</div>
            <div className="mt-0.5 text-[12px] text-text-2">
              Shows a gold star and adds it to your Wins once it&rsquo;s done.
            </div>
          </div>
          <Toggle
            checked={isAchievement}
            onChange={setIsAchievement}
            label="Celebrate as an achievement"
          />
        </div>

        {formError && (
          <p role="alert" className="text-[13px] text-rose">
            {formError}
          </p>
        )}

        <div className="flex justify-end gap-3 pt-1">
          <Button type="button" variant="outline" onClick={onClose} disabled={pending}>
            Cancel
          </Button>
          <Button type="submit" variant="teal" disabled={pending}>
            {pending ? 'Saving…' : editing ? 'Save changes' : 'Add milestone'}
          </Button>
        </div>
      </form>
    </Modal>
  )
}
