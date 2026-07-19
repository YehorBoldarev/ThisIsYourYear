import { ProgressBar } from '../../components'
import { yearElapsedFraction } from '../../lib/year'

/**
 * The year bar: a time-based fill (fraction of the calendar year elapsed) with a
 * gold "YOU ARE HERE" caption. Days-left comes from the API countdown, which is
 * timezone-correct; the smooth % is derived locally.
 */
export function YearProgress({ secondsRemaining }: { secondsRemaining: number }) {
  const fraction = yearElapsedFraction()
  const percent = Math.round(fraction * 100)
  const daysLeft = Math.max(0, Math.ceil(secondsRemaining / 86_400))

  return (
    <div className="mt-10">
      <ProgressBar value={fraction} height={8} gradient label="Year progress" />
      <div className="label-mono mt-2 flex items-center justify-between text-[11px] text-text-muted">
        <span>Jan</span>
        <span className="text-gold">
          You are here · {percent}% · {daysLeft} {daysLeft === 1 ? 'day' : 'days'} left
        </span>
        <span>Dec</span>
      </div>
    </div>
  )
}
