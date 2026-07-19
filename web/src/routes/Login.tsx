import { useState, type FormEvent } from 'react'
import { Link, Navigate, useLocation, useNavigate } from 'react-router-dom'
import { AuthLayout } from './AuthLayout'
import { Button, Field, Spinner, Toggle } from '../components'
import { useAuth } from '../auth/useAuth'
import { isApiError, type FieldErrors } from '../api/errors'

export function Login() {
  const { status, login } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const from = (location.state as { from?: { pathname?: string } } | null)?.from?.pathname ?? '/'

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [remember, setRemember] = useState(true)
  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({})
  const [formError, setFormError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  // Already signed in (e.g. navigated here directly) → go home.
  if (status === 'authenticated') return <Navigate to={from} replace />

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setFieldErrors({})
    setFormError(null)
    setSubmitting(true)
    try {
      await login(email, password, remember)
      navigate(from, { replace: true })
    } catch (err) {
      if (isApiError(err)) {
        if (err.status === 401) setFormError('Email or password is incorrect.')
        else if (err.fields) setFieldErrors(err.fields)
        else setFormError(err.message)
      } else {
        setFormError('Something went wrong. Please try again.')
      }
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <AuthLayout
      title="Welcome back"
      subtitle="Pick up where your year left off."
      footer={
        <>
          New here?{' '}
          <Link to="/register" className="text-teal hover:underline">
            Create an account
          </Link>
        </>
      }
    >
      <form onSubmit={onSubmit} noValidate className="space-y-5">
        {formError && (
          <div
            role="alert"
            className="rounded-field border border-rose/50 px-3.5 py-2.5 text-sm text-rose"
          >
            {formError}
          </div>
        )}
        <Field
          id="email"
          label="Email"
          type="email"
          value={email}
          onChange={setEmail}
          error={fieldErrors.email}
          autoComplete="email"
          required
        />
        <Field
          id="password"
          label="Password"
          type="password"
          value={password}
          onChange={setPassword}
          error={fieldErrors.password}
          autoComplete="current-password"
          required
        />
        <div className="flex items-center gap-3">
          <Toggle checked={remember} onChange={setRemember} label="Remember me" />
          <span className="text-sm text-text-2">Remember me</span>
        </div>
        <Button type="submit" disabled={submitting} className="w-full">
          {submitting ? <Spinner size={18} /> : 'Sign in'}
        </Button>
      </form>
    </AuthLayout>
  )
}
