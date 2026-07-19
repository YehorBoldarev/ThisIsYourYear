import { Outlet, Route, Routes } from 'react-router-dom'
import { AppShell } from './layout/AppShell'
import { RequireAuth } from './auth/RequireAuth'
import { Dashboard } from './routes/Dashboard'
import { Achievements } from './routes/Achievements'
import { Reflections } from './routes/Reflections'
import { Gallery } from './routes/Gallery'
import { Login } from './routes/Login'
import { Register } from './routes/Register'

/** Protected shell: gate on auth, then render the app chrome around the route. */
function ProtectedLayout() {
  return (
    <RequireAuth>
      <AppShell>
        <Outlet />
      </AppShell>
    </RequireAuth>
  )
}

export default function App() {
  return (
    <Routes>
      {/* Public, chrome-less auth screens. */}
      <Route path="/login" element={<Login />} />
      <Route path="/register" element={<Register />} />

      {/* Everything else requires auth and renders inside the app shell. */}
      <Route element={<ProtectedLayout />}>
        <Route path="/" element={<Dashboard />} />
        <Route path="/achievements" element={<Achievements />} />
        <Route path="/reflections" element={<Reflections />} />
        {/* Dev-only component gallery; excluded from production builds. */}
        {import.meta.env.DEV && <Route path="/gallery" element={<Gallery />} />}
        <Route path="*" element={<Dashboard />} />
      </Route>
    </Routes>
  )
}
