const GITHUB_URL = 'https://github.com/YehorBoldarev/ThisIsYourYear'

// The design handoff also shows a support email here; it was removed because
// there is no real mailbox for it (see plans/backlog.md).
export function Footer() {
  return (
    <footer className="label-mono flex flex-wrap items-center justify-center gap-x-3 gap-y-1 border-t border-border bg-bar px-7 py-5 text-[11px] text-text-muted">
      <a href={GITHUB_URL} target="_blank" rel="noreferrer" className="hover:text-text">
        GitHub
      </a>
      <span aria-hidden="true">·</span>
      <span>Since 2026</span>
    </footer>
  )
}
