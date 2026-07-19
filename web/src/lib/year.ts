/** Fraction of the current calendar year elapsed (0–1), from the given date. */
export function yearElapsedFraction(now: Date = new Date()): number {
  const year = now.getFullYear()
  const start = new Date(year, 0, 1).getTime()
  const end = new Date(year + 1, 0, 1).getTime()
  return (now.getTime() - start) / (end - start)
}

/** Whole days remaining until next New Year's Day, from the given date. */
export function daysToNewYear(now: Date = new Date()): number {
  const nextYear = new Date(now.getFullYear() + 1, 0, 1).getTime()
  return Math.ceil((nextYear - now.getTime()) / 86_400_000)
}
