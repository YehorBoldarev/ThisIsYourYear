import { Logo } from '../components'

/** "Coming soon" placeholder — Reflections is deferred (locked decision). */
export function Reflections() {
  return (
    <div className="mx-auto flex w-full max-w-[680px] flex-col items-center px-6 py-16 text-center">
      <Logo progress={0.15} size={56} />
      <div className="label-mono mt-6 text-[11px] text-text-muted">Reflections</div>
      <h1 className="mt-3 font-serif text-3xl font-medium text-text">Coming soon</h1>
      <p className="mt-3 max-w-md font-serif text-lg italic text-text-2">
        A quiet place to write about your year is on the way. For now, keep making marks on the
        chapters that matter.
      </p>
    </div>
  )
}
