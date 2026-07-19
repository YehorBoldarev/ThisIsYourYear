import { useState, type FormEvent } from 'react'
import { Link, Navigate, useNavigate } from 'react-router-dom'
import { AuthLayout } from './AuthLayout'
import { Button, Field, Spinner } from '../components'
import { useAuth } from '../auth/useAuth'
import { isApiError, type FieldErrors } from '../api/errors'

export function Register() {
  const { status, register } = useAuth()
  const navigate = useNavigate()

  const [email, setEmail] = useState('')
  const [displayName, setDisplayName] = useState('')
  const [password, setPassword] = useState('')
  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({})
  const [formError, setFormError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  if (status === 'authenticated') return <Navigate to="/" replace />

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setFieldErrors({})
    setFormError(null)
    setSubmitting(true)
    try {
      await register(email, displayName, password)
      navigate('/', { replace: true })
    } catch (err) {
      if (isApiError(err)) {
        if (err.status === 409) setFieldErrors({ email: 'That email is already registered.' })
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
      title="Begin your year"
      subtitle="Set the goals that will define it."
      footer={
        <>
          Already have an account?{' '}
          <Link to="/login" className="text-teal hover:underline">
            Sign in
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
          id="displayName"
          label="Display name"
          value={displayName}
          onChange={setDisplayName}
          error={fieldErrors.displayName}
          autoComplete="name"
          required
        />
        <Field
          id="password"
          label="Password"
          type="password"
          value={password}
          onChange={setPassword}
          error={fieldErrors.password}
          autoComplete="new-password"
          placeholder="At least 8 characters"
          required
        />
        <Button type="submit" disabled={submitting} className="w-full">
          {submitting ? <Spinner size={18} /> : 'Create account'}
        </Button>
      </form>
    </AuthLayout>
  )
}
