const SUPPORT_EMAIL = 'hello@thisisyour.year'
const GITHUB_URL = 'https://github.com/kibou/thisisyouryear'

export function Footer() {
  return (
    <footer className="label-mono flex flex-wrap items-center justify-center gap-x-3 gap-y-1 border-t border-border bg-bar px-7 py-5 text-[11px] text-text-muted">
      <a href={`mailto:${SUPPORT_EMAIL}`} className="hover:text-text">
        {SUPPORT_EMAIL}
      </a>
      <span aria-hidden="true">·</span>
      <a href={GITHUB_URL} target="_blank" rel="noreferrer" className="hover:text-text">
        GitHub
      </a>
      <span aria-hidden="true">·</span>
      <span>Since 2026</span>
    </footer>
  )
}
