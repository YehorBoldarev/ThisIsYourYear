/** Time-of-day greeting from the local hour (matches the handoff's "Good evening,"). */
export function timeOfDayGreeting(hour: number = new Date().getHours()): string {
  if (hour < 12) return 'Good morning'
  if (hour < 18) return 'Good afternoon'
  return 'Good evening'
}
