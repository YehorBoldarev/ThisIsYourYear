import { Button, Logo } from '../../components'
import { useAchievements } from './useAchievements'
import { AchievementCard } from './AchievementCard'

export function AchievementsPage() {
  const { items, isLoading, isError, refetch } = useAchievements()

  if (isLoading) return <WinsSkeleton />
  if (isError) return <WinsError onRetry={refetch} />

  return (
    <div className="mx-auto w-full max-w-[680px] px-6 py-12">
      <div className="text-center">
        <div className="label-mono text-[11px] text-gold">Wins</div>
        <h1 className="mt-3 font-serif text-3xl font-medium text-text">Your achievements</h1>
        <p className="mt-3 font-serif text-lg italic text-text-2">
          {items.length === 0
            ? 'The moments worth keeping will gather here.'
            : `${items.length} moment${items.length === 1 ? '' : 's'} worth keeping.`}
        </p>
      </div>

      {items.length === 0 ? (
        <WinsEmptyState />
      ) : (
        <ul className="mt-8 space-y-3">
          {items.map(({ goal, topic }) => (
            <AchievementCard key={goal.id} goal={goal} topic={topic} />
          ))}
        </ul>
      )}
    </div>
  )
}

/** Zero derived wins — encourages starring a milestone. */
function WinsEmptyState() {
  return (
    <div className="mt-10 flex flex-col items-center rounded-card border border-border-subtle bg-surface px-6 py-12 text-center">
      <Logo progress={0.06} size={56} />
      <p className="mt-6 max-w-[400px] text-sm text-text-2">
        Mark a milestone as an achievement <span className="text-gold">★</span> when you create it —
        finishing it lands the win here.
      </p>
    </div>
  )
}

function WinsSkeleton() {
  return (
    <div className="mx-auto w-full max-w-[680px] px-6 py-12" aria-hidden="true">
      <div className="mx-auto h-3 w-16 animate-pulse rounded bg-fill-muted" />
      <div className="mx-auto mt-4 h-8 w-64 animate-pulse rounded bg-fill-muted" />
      <ul className="mt-10 space-y-3">
        {[0, 1, 2].map((i) => (
          <li key={i} className="h-16 animate-pulse rounded-field bg-fill-muted" />
        ))}
      </ul>
    </div>
  )
}

function WinsError({ onRetry }: { onRetry: () => void }) {
  return (
    <div className="mx-auto flex w-full max-w-[520px] flex-col items-center px-6 py-16 text-center">
      <h1 className="font-serif text-2xl font-medium text-text">
        We couldn&rsquo;t load your wins
      </h1>
      <p className="mt-2 text-sm text-text-2">Something went wrong reaching the server.</p>
      <Button onClick={onRetry} className="mt-6">
        Try again
      </Button>
    </div>
  )
}
