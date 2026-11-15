import { useEffect, useMemo, useRef, useState } from 'react'

const SECOND = 1000
const MINUTE = 60 * SECOND
const HOUR = 60 * MINUTE
const DAY = 24 * HOUR

/** After the countdown ends, ask the server again at this pace until it agrees. */
const FIRST_REFRESH_MS = 3 * SECOND
const RETRY_REFRESH_MS = 10 * SECOND

function plural(n: number, unit: string) {
  return `${n} ${unit}${n === 1 ? '' : 's'}`
}

function split(ms: number) {
  return {
    days: Math.floor(ms / DAY),
    hours: Math.floor(ms / HOUR) % 24,
    minutes: Math.floor(ms / MINUTE) % 60,
    seconds: Math.floor(ms / SECOND) % 60,
  }
}

/**
 * The preparation stage's centerpiece: a live countdown to the first moment of
 * the year being planned (local midnight, from the API's `countdown.target`).
 *
 * It ticks every second locally, with no polling. The digits are hidden from screen
 * readers; a visually hidden sentence with minute precision stands in, so
 * assistive tech isn't flooded every second. When it reaches zero it shows a
 * short New-Year message and calls `onReached` (shortly after, then every few
 * seconds) so the page can refetch and switch to the normal year view.
 */
export function NewYearCountdown({
  target,
  year,
  onReached,
}: {
  target: string
  year: number
  onReached?: () => void
}) {
  const targetMs = useMemo(() => Date.parse(target), [target])
  const [now, setNow] = useState(() => Date.now())

  useEffect(() => {
    const id = setInterval(() => setNow(Date.now()), SECOND)
    return () => clearInterval(id)
  }, [])

  const remaining = Math.max(0, targetMs - now)
  const reached = remaining === 0

  // Keep the latest callback without restarting the timers below.
  const onReachedRef = useRef(onReached)
  useEffect(() => {
    onReachedRef.current = onReached
  }, [onReached])

  useEffect(() => {
    if (!reached) return
    // The server decides when the year has begun; ask until it agrees (this
    // component unmounts once the dashboard switches out of preparation).
    const first = setTimeout(() => onReachedRef.current?.(), FIRST_REFRESH_MS)
    const retry = setInterval(() => onReachedRef.current?.(), RETRY_REFRESH_MS)
    return () => {
      clearTimeout(first)
      clearInterval(retry)
    }
  }, [reached])

  if (reached) {
    return (
      <section aria-label={`Countdown to ${year}`} className="mt-10 text-center">
        <p role="status" className="font-serif text-2xl font-medium text-gold-ink">
          Happy New Year — your {year} starts now.
        </p>
      </section>
    )
  }

  const { days, hours, minutes, seconds } = split(remaining)
  const units = [
    { value: days, label: days === 1 ? 'day' : 'days' },
    { value: hours, label: hours === 1 ? 'hour' : 'hours' },
    { value: minutes, label: 'min' },
    { value: seconds, label: 'sec' },
  ]

  return (
    <section aria-label={`Countdown to ${year}`} className="mt-10 text-center">
      <div aria-hidden="true">
        <div className="label-mono text-[11px] text-text-muted">{year} begins in</div>
        <div className="mx-auto mt-4 grid max-w-[440px] grid-cols-4 gap-2 sm:gap-3">
          {units.map((u) => (
            <div key={u.label} className="rounded-field border border-border bg-elevated px-2 py-3">
              <div className="font-serif text-3xl font-medium tabular-nums text-text sm:text-4xl">
                {String(u.value).padStart(2, '0')}
              </div>
              <div className="label-mono mt-1 text-[10px] text-text-muted">{u.label}</div>
            </div>
          ))}
        </div>
      </div>
      <p className="sr-only">
        {year} begins in {plural(days, 'day')}, {plural(hours, 'hour')} and{' '}
        {plural(minutes, 'minute')}.
      </p>
      <p className="label-mono mt-3 text-[11px] text-gold-ink">
        Preparation stage · plan your {year}
      </p>
    </section>
  )
}
