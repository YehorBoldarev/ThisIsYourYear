import { Button, Chip, Icon, Logo } from '../../components'
import { useTopicModals } from './topicModalsContext'

const EXAMPLES = ['Run a marathon', 'Read 24 books', 'Learn a language']

/** First-run / zero-topics state (handoff screen 8). */
export function EmptyState() {
  const { openCreate } = useTopicModals()
  return (
    <div className="mx-auto flex w-full max-w-[520px] flex-col items-center px-6 py-16 text-center">
      <Logo progress={0.06} size={64} />
      <h1 className="mt-6 font-serif text-3xl font-medium text-text">Your year is a blank page.</h1>
      <p className="mt-3 font-serif text-lg italic text-text-2">
        Every remarkable year starts with a single intention. Name the first goal that matters to
        you.
      </p>
      <Button onClick={openCreate} className="mt-8">
        <Icon name="plus" size={16} />
        Create your first big goal
      </Button>
      <div className="mt-8 flex flex-wrap items-center justify-center gap-2">
        <span className="label-mono text-[11px] text-text-faint">Try:</span>
        {EXAMPLES.map((ex) => (
          <Chip key={ex} onClick={openCreate}>
            {ex}
          </Chip>
        ))}
      </div>
    </div>
  )
}
