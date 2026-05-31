import { useState, FormEvent } from 'react'
import toast from 'react-hot-toast'
import { createUser } from '@/api/client'
import { UserRole } from '@/types'

const ROLES: { value: UserRole; label: string }[] = [
  { value: 'fte_ops',       label: 'FTE — Outbound' },
  { value: 'fte_mm',        label: 'FTE — Midmile' },
  { value: 'ops_pic',       label: 'OB OPS PIC' },
  { value: 'dock_officer',  label: 'Dock Officer' },
  { value: 'doc_officer',   label: 'Doc Officer' },
  { value: 'data_team',     label: 'Data Team' },
  { value: 'admin',         label: 'Administrator' },
]

type LoginType = 'fte' | 'backroom'

export default function Settings() {
  const [loginType, setLoginType] = useState<LoginType>('fte')
  const [loading, setLoading] = useState(false)
  const [form, setForm] = useState({
    name:    '',
    role:    'fte_ops' as UserRole,
    email:   '',
    ops_id:  '',
  })

  const set = (k: keyof typeof form) =>
    (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) =>
      setForm((f) => ({ ...f, [k]: e.target.value }))

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setLoading(true)
    try {
      await createUser({
        name:    form.name,
        role:    form.role,
        email:   loginType === 'fte' ? form.email : undefined,
        ops_id:  loginType === 'backroom' ? form.ops_id : undefined,
      })
      toast.success(`User "${form.name}" added successfully`)
      setForm({ name: '', role: 'fte_ops', email: '', ops_id: '' })
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to add user')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div style={{ maxWidth: 600 }}>
      <div className="page-header">
        <div>
          <h1 className="page-title">Settings</h1>
          <p className="page-subtitle">Manage team members</p>
        </div>
      </div>

      <div className="card" style={{ padding: '28px 32px' }}>
        <h2 style={{ fontSize: 16, fontWeight: 700, marginBottom: 22 }}>Add New User</h2>

        <div className="login-tabs" style={{ marginBottom: 22, maxWidth: 280 }}>
          <button
            type="button"
            className={`login-tab${loginType === 'fte' ? ' login-tab--active' : ''}`}
            onClick={() => setLoginType('fte')}
          >FTE (Email)</button>
          <button
            type="button"
            className={`login-tab${loginType === 'backroom' ? ' login-tab--active' : ''}`}
            onClick={() => setLoginType('backroom')}
          >Backroom (Ops ID)</button>
        </div>

        <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
          <div className="form-row">
            <div className="form-group">
              <label className="form-label">Full Name</label>
              <input className="form-input" value={form.name} onChange={set('name')} required placeholder="Juan Dela Cruz" />
            </div>
            <div className="form-group">
              <label className="form-label">Role</label>
              <select className="form-select" value={form.role} onChange={set('role')} required>
                {ROLES.map((r) => (
                  <option key={r.value} value={r.value}>{r.label}</option>
                ))}
              </select>
            </div>
          </div>

          {loginType === 'fte' ? (
            <div className="form-group">
              <label className="form-label">Email</label>
              <input className="form-input" type="email" value={form.email} onChange={set('email')} required placeholder="name@shopee.com" />
            </div>
          ) : (
            <div className="form-group">
              <label className="form-label">Ops ID</label>
              <input className="form-input" value={form.ops_id} onChange={set('ops_id')} required placeholder="Enter Ops ID" />
            </div>
          )}

          <div style={{ marginTop: 4 }}>
            <button type="submit" className="btn btn--primary" disabled={loading}>
              {loading ? 'Adding…' : 'Add User'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
