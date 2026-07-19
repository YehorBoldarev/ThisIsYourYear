import { Button, Modal } from '../../components'

interface ConfirmDeleteProps {
  title: string
  message: string
  confirmLabel: string
  pending: boolean
  error?: string
  onConfirm: () => void
  onCancel: () => void
}

/** A small confirm step in a Modal for destructive actions (no native confirm). */
export function ConfirmDelete({
  title,
  message,
  confirmLabel,
  pending,
  error,
  onConfirm,
  onCancel,
}: ConfirmDeleteProps) {
  return (
    <Modal open onClose={onCancel} title={title} width={440}>
      <p className="text-sm text-text-2">{message}</p>
      {error && (
        <p role="alert" className="mt-3 text-[13px] text-rose">
          {error}
        </p>
      )}
      <div className="mt-6 flex justify-end gap-3">
        <Button type="button" variant="outline" onClick={onCancel} disabled={pending}>
          Cancel
        </Button>
        <Button type="button" variant="danger" onClick={onConfirm} disabled={pending}>
          {pending ? 'Deleting…' : confirmLabel}
        </Button>
      </div>
    </Modal>
  )
}
