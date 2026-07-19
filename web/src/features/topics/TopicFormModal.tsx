import { useState } from 'react'
import { Button, Field, Label, Modal, SwatchGroup, Textarea } from '../../components'
import { ACCENT_HEX, accentFromHex, type AccentColor } from '../../components/accent'
import { isApiError, type FieldErrors } from '../../api/errors'
import { toApiDate, toInputDate } from '../../lib/date'
import type { Topic } from './useTopics'
import { useCreateTopic, useUpdateTopic } from './mutations'

interface TopicFormModalProps {
  /** Present → edit that chapter; absent → create a new one. */
  topic?: Topic
  onClose: () => void
}

export function TopicFormModal({ topic, onClose }: TopicFormModalProps) {
  const editing = !!topic
  const [title, setTitle] = useState(topic?.title ?? '')
  const [description, setDescription] = useState(topic?.description ?? '')
  const [targetDate, setTargetDate] = useState(toInputDate(topic?.targetDate ?? ''))
  const [category, setCategory] = useState(topic?.category ?? '')
  const [accent, setAccent] = useState<AccentColor>(topic ? accentFromHex(topic.color) : 'teal')
  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({})
  const [formError, setFormError] = useState('')

  const create = useCreateTopic()
  const update = useUpdateTopic()
  const pending = create.isPending || update.isPending

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
    const color = ACCENT_HEX[accent]
    if (editing) {
      update.mutate(
        {
          id: topic.id,
          body: {
            title: title.trim(),
            description: description.trim(),
            category: category.trim(),
            color,
            targetDate: toApiDate(targetDate),
          },
        },
        { onSuccess: onClose, onError: fail },
      )
    } else {
      const dueDate = toApiDate(targetDate)
      create.mutate(
        {
          title: title.trim(),
          color,
          ...(description.trim() ? { description: description.trim() } : {}),
          ...(category.trim() ? { category: category.trim() } : {}),
          ...(dueDate ? { targetDate: dueDate } : {}),
        },
        { onSuccess: onClose, onError: fail },
      )
    }
  }

  return (
    <Modal
      open
      onClose={onClose}
      eyebrow={editing ? 'Edit chapter' : `New for ${new Date().getFullYear()}`}
      title={editing ? 'Edit big goal' : 'Create a big goal'}
      width={560}
    >
      <form onSubmit={submit} noValidate className="space-y-5">
        <Field
          id="topic-title"
          label="What is the goal?"
          value={title}
          onChange={setTitle}
          error={fieldErrors.title}
          placeholder="Run a half-marathon"
          required
        />

        <div className="space-y-2">
          <Label htmlFor="topic-description">Why does it matter?</Label>
          <Textarea
            id="topic-description"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            invalid={!!fieldErrors.description}
            placeholder="A sentence to remind future-you what this is for."
          />
          {fieldErrors.description && (
            <p className="text-[12px] text-rose">{fieldErrors.description}</p>
          )}
        </div>

        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <Field
            id="topic-target"
            label="Target date"
            value={targetDate}
            onChange={setTargetDate}
            type="date"
            error={fieldErrors.targetDate}
          />
          <Field
            id="topic-category"
            label="Category"
            value={category}
            onChange={setCategory}
            error={fieldErrors.category}
            placeholder="Endurance"
          />
        </div>

        <div className="space-y-2">
          {/* SwatchGroup is a self-labelled radiogroup; this is a visual caption. */}
          <span className="label-mono block text-[11px] text-text-muted">Accent colour</span>
          <SwatchGroup value={accent} onChange={setAccent} />
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
            {pending ? 'Saving…' : editing ? 'Save changes' : 'Create goal'}
          </Button>
        </div>
      </form>
    </Modal>
  )
}
