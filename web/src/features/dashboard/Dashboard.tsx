import { Button } from '../../components'
import { useAuth } from '../../auth/useAuth'
import { useDashboard } from './useDashboard'
import { useTopics } from '../topics/useTopics'
import { DateGreeting } from './DateGreeting'
import { YearProgress } from './YearProgress'
import { DashboardSkeleton } from './DashboardSkeleton'
import { ChaptersSection } from '../topics/ChaptersSection'
import { EmptyState } from '../topics/EmptyState'
import { ModalHost } from '../topics/ModalHost'

function ErrorState({ onRetry }: { onRetry: () => void }) {
  return (
    <div className="mx-auto flex w-full max-w-[520px] flex-col items-center px-6 py-16 text-center">
      <h1 className="font-serif text-2xl font-medium text-text">
        We couldn&rsquo;t load your year
      </h1>
      <p className="mt-2 text-sm text-text-2">Something went wrong reaching the server.</p>
      <Button onClick={onRetry} className="mt-6">
        Try again
      </Button>
    </div>
  )
}

export function Dashboard() {
  const { user } = useAuth()
  const dashboard = useDashboard()
  const topics = useTopics()

  if (dashboard.isPending || topics.isPending) return <DashboardSkeleton />

  if (dashboard.isError || topics.isError) {
    return (
      <ErrorState
        onRetry={() => {
          void dashboard.refetch()
          void topics.refetch()
        }}
      />
    )
  }

  const { data: dash } = dashboard
  const chapters = topics.data

  return (
    <ModalHost>
      {chapters.length === 0 ? (
        <EmptyState />
      ) : (
        <div className="mx-auto w-full max-w-[680px] px-6 py-12">
          <DateGreeting name={user?.displayName ?? ''} message={dash.message} />
          <YearProgress secondsRemaining={dash.countdown.secondsRemaining} />
          <ChaptersSection topics={chapters} />
        </div>
      )}
    </ModalHost>
  )
}
