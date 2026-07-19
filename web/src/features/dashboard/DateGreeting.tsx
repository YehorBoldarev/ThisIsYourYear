import { timeOfDayGreeting } from '../../lib/greeting'

/** Date eyebrow + greeting (name + time of day) + italic sub-message. */
export function DateGreeting({ name, message }: { name: string; message: string }) {
  const today = new Date().toLocaleDateString(undefined, {
    weekday: 'long',
    month: 'long',
    day: 'numeric',
  })
  const firstName = name.trim().split(/\s+/)[0] || name

  return (
    <div className="text-center">
      <div className="label-mono text-[11px] text-text-muted">{today}</div>
      <hr className="mx-auto my-5 w-10 border-border" />
      <h1 className="font-serif text-[25px] font-medium leading-tight text-text sm:text-[30px] lg:text-4xl">
        {timeOfDayGreeting()}, {firstName}.
      </h1>
      {message && <p className="mt-3 font-serif text-lg italic text-text-2">{message}</p>}
    </div>
  )
}
