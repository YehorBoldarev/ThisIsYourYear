import { Outlet, Route, Routes, useLocation, useNavigate } from 'react-router-dom'
import { AppShell } from './layout/AppShell'
import { RequireAuth } from './auth/RequireAuth'
import { ModalHost } from './features/topics/ModalHost'
import { useTopicModals } from './features/topics/topicModalsContext'
import { Dashboard } from './routes/Dashboard'
import { Achievements } from './routes/Achievements'
import { Reflections } from './routes/Reflections'
import { Gallery } from './routes/Gallery'
import { Login } from './routes/Login'
import { Register } from './routes/Register'

/**
 * Protected shell: gate on auth, then render the app chrome around the route.
 * The topic popups are app-level (ModalHost sits above the routes), so chrome
 * outside any page — the mobile tab bar's "New ＋" — can open them too.
 */
function ProtectedLayout() {
  return (
    <RequireAuth>
      <ModalHost>
        <ShellWithModals />
      </ModalHost>
    </RequireAuth>
  )
}

function ShellWithModals() {
  const { openCreate } = useTopicModals()
  const navigate = useNavigate()
  const { pathname } = useLocation()
  // Create over the dashboard, wherever "New" was tapped, so the user sees where
  // the new chapter lands once it's saved.
  const onNew = () => {
    if (pathname !== '/') navigate('/')
    openCreate()
  }
  return (
    <AppShell onNew={onNew}>
      <Outlet />
    </AppShell>
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
