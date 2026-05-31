import { useState, FormEvent } from 'react'
import toast from 'react-hot-toast'
import { login } from '@/api/client'
import { useAuthStore } from '@/store/authStore'
import { useNavigate } from 'react-router-dom'

type Tab = 'fte' | 'backroom'

export default function LoginModal() {
  const [tab, setTab] = useState<Tab>('fte')
  const [email, setEmail] = useState('')
  const [opsId, setOpsId] = useState('')
  const [loading, setLoading] = useState(false)
  const setUser = useAuthStore((s) => s.setUser)
  const navigate = useNavigate()

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    if (loading) return

    setLoading(true)
    try {
      const user = await login(tab, tab === 'fte' ? email : '', tab === 'backroom' ? opsId : '')
      setUser(user)
      const redirect = user.redirect || '/dashboard'
      navigate(redirect, { replace: true })
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Login failed')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="login-overlay">
      <div className="login-card">
        <div className="login-header">
          <div className="login-logo">
            <img src="/truck_label/icon.png" alt="SOC5" onError={(e) => {
              (e.target as HTMLImageElement).style.display = 'none'
            }} />
          </div>
          <div>
            <p className="login-brand">SOC 5 OUTBOUND</p>
            <h1 className="login-title">Sign in</h1>
          </div>
        </div>

        <div className="login-tabs">
          <button
            className={`login-tab${tab === 'fte' ? ' login-tab--active' : ''}`}
            onClick={() => setTab('fte')}
            type="button"
          >
            FTE
          </button>
          <button
            className={`login-tab${tab === 'backroom' ? ' login-tab--active' : ''}`}
            onClick={() => setTab('backroom')}
            type="button"
          >
            Backroom
          </button>
        </div>

        <form onSubmit={handleSubmit} className="login-form">
          {tab === 'fte' ? (
            <div className="form-group">
              <label className="form-label">Email</label>
              <input
                className="form-input"
                type="email"
                placeholder="name@shopee.com"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                required
                autoFocus
              />
            </div>
          ) : (
            <div className="form-group">
              <label className="form-label">Ops ID</label>
              <input
                className="form-input"
                type="text"
                placeholder="Enter your Ops ID"
                value={opsId}
                onChange={(e) => setOpsId(e.target.value)}
                required
                autoFocus
              />
            </div>
          )}

          <button
            type="submit"
            className="btn btn--primary btn--full"
            disabled={loading}
          >
            {loading ? 'Signing in…' : '→ Continue'}
          </button>
        </form>
      </div>
    </div>
  )
}
