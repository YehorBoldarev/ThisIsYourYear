import { Icon } from '../../components'
import { ChapterRow } from './ChapterRow'
import { useTopicModals } from './topicModalsContext'
import type { Topic } from './useTopics'

export function ChaptersSection({ topics }: { topics: Topic[] }) {
  const { openCreate } = useTopicModals()
  return (
    <section className="mt-10 text-left">
      <div className="flex items-center justify-between border-b border-border pb-3">
        <h2 className="font-serif text-xl font-semibold text-text">The chapters</h2>
        <button
          type="button"
          onClick={openCreate}
          className="label-mono flex items-center gap-1 text-[11px] text-teal transition-colors hover:text-teal-hover cursor-pointer"
        >
          <Icon name="plus" size={12} />
          Begin a new chapter
        </button>
      </div>
      <ul className="space-y-3 sm:space-y-0">
        {topics.map((topic) => (
          <ChapterRow key={topic.id} topic={topic} />
        ))}
      </ul>
    </section>
  )
}
