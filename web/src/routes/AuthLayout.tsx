import type { ReactNode } from 'react'
import { Logo } from '../components'
import { ThemeToggle } from '../layout/ThemeToggle'

interface AuthLayoutProps {
  title: string
  subtitle?: string
  children: ReactNode
  footer?: ReactNode
}

/** Centered, on-brand shell for the login / register screens (no app chrome). */
export function AuthLayout({ title, subtitle, children, footer }: AuthLayoutProps) {
  return (
    <div className="relative flex min-h-screen items-center justify-center px-4 py-10">
      <div className="absolute right-5 top-5">
        <ThemeToggle optionSize={24} />
      </div>
      <div className="w-full max-w-sm">
        <div className="mb-8 flex flex-col items-center text-center">
          <Logo progress={0.48} size={44} />
          <h1 className="mt-4 font-serif text-3xl font-medium text-text">{title}</h1>
          {subtitle && <p className="mt-2 text-sm text-text-2">{subtitle}</p>}
        </div>
        <div className="rounded-card border border-border bg-surface p-7 shadow-card">
          {children}
        </div>
        {footer && <p className="mt-6 text-center text-sm text-text-muted">{footer}</p>}
      </div>
    </div>
  )
}
