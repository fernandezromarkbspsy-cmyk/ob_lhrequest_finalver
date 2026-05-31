import { useEffect } from 'react'
import { Routes, Route, Navigate } from 'react-router-dom'
import { setOnAuthError } from '@/api/client'
import { useAuthStore } from '@/store/authStore'
import { useUIStore } from '@/store/uiStore'
import { useSSE } from '@/hooks/useSSE'
import Layout from '@/components/Layout/Layout'
import LoginModal from '@/components/LoginModal/LoginModal'
import Dashboard from '@/pages/Dashboard'
import LHRequests from '@/pages/LHRequests'
import TruckRequests from '@/pages/TruckRequests'
import DockOfficer from '@/pages/DockOfficer'
import Settings from '@/pages/Settings'

function AppShell() {
  const user = useAuthStore((s) => s.user)
  const logout = useAuthStore((s) => s.logout)

  useSSE()

  useEffect(() => {
    setOnAuthError(() => logout())
  }, [logout])

  return (
    <>
      {!user && <LoginModal />}
      {user && (
        <Layout>
          <Routes>
            <Route path="/" element={<Navigate to="/dashboard" replace />} />
            <Route path="/dashboard" element={<Dashboard />} />
            <Route path="/outbound/lh-request" element={<LHRequests />} />
            <Route path="/midmile/truck-request" element={<TruckRequests />} />
            <Route path="/dock/officer" element={<DockOfficer />} />
            <Route path="/settings" element={<Settings />} />
            <Route path="*" element={<Navigate to="/dashboard" replace />} />
          </Routes>
        </Layout>
      )}
    </>
  )
}

export default function App() {
  const theme = useUIStore((s) => s.theme)

  useEffect(() => {
    document.body.dataset.theme = theme
  }, [theme])

  return (
    <Routes>
      <Route path="/*" element={<AppShell />} />
    </Routes>
  )
}
