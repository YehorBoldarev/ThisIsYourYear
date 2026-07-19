import { cn } from '../../lib/cn'

function Block({ className }: { className?: string }) {
  return <div className={cn('animate-pulse bg-fill-muted', className)} />
}

/** Loading placeholder mirroring the dashboard's shape. */
export function DashboardSkeleton() {
  return (
    <div
      className="mx-auto w-full max-w-[680px] px-6 py-12"
      aria-hidden="true"
      data-testid="dashboard-skeleton"
    >
      <div className="flex flex-col items-center gap-4">
        <Block className="h-3 w-40 rounded-full" />
        <Block className="h-8 w-72 max-w-full rounded-full" />
        <Block className="h-4 w-96 max-w-full rounded-full" />
        <Block className="mt-6 h-2 w-full rounded-full" />
      </div>
      <div className="mt-12 space-y-4">
        {[0, 1, 2].map((i) => (
          <Block key={i} className="h-14 w-full rounded-field" />
        ))}
      </div>
    </div>
  )
}
