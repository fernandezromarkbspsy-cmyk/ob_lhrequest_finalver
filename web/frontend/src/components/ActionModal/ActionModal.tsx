import { useState, FormEvent } from 'react'
import { X } from 'lucide-react'
import { Request, RequestAction, RequestPayload } from '@/types'
import { useAuthStore } from '@/store/authStore'

interface Props {
  req: Request
  action: RequestAction
  onClose: () => void
  onSubmit: (id: number, action: string, payload: RequestPayload) => Promise<void>
}

const titles: Record<RequestAction, string> = {
  approve:      'Approve Request',
  reject:       'Reject Request',
  assign:       'Assign Midmile FTE',
  'for-docking': 'Set For Docking',
  dock:         'Confirm Docking',
  confirm:      'Confirm Truck',
  cancel:       'Cancel Request',
  edit:         'Edit Request',
}

export default function ActionModal({ req, action, onClose, onSubmit }: Props) {
  const user = useAuthStore((s) => s.user)
  const [loading, setLoading] = useState(false)
  const [form, setForm] = useState({
    ob_fte:           req.ob_fte || user?.name || '',
    midmile_fte:      req.midmile_fte || '',
    plate_number:     req.plate_number || '',
    driver_id:        req.driver_id || '',
    linehaul_trip_no: req.linehaul_trip_no || '',
    docking_time:     req.docking_time || '',
    remarks:          req.remarks || '',
  })

  const set = (k: keyof typeof form) => (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) =>
    setForm((f) => ({ ...f, [k]: e.target.value }))

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setLoading(true)
    try {
      const payload: RequestPayload = {}
      if (action === 'approve')      payload.ob_fte = form.ob_fte
      if (action === 'reject')       payload.remarks = form.remarks
      if (action === 'assign')       payload.midmile_fte = form.midmile_fte
      if (action === 'for-docking')  { payload.midmile_fte = form.midmile_fte; payload.plate_number = form.plate_number }
      if (action === 'dock')         { payload.driver_id = form.driver_id; payload.linehaul_trip_no = form.linehaul_trip_no; payload.docking_time = form.docking_time }
      if (action === 'cancel')       payload.remarks = form.remarks
      await onSubmit(req.id, action, payload)
      onClose()
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="modal-overlay" onClick={(e) => e.target === e.currentTarget && onClose()}>
      <div className="modal">
        <div className="modal-header">
          <h2 className="modal-title">{titles[action]}</h2>
          <button className="modal-close" onClick={onClose}><X size={18} /></button>
        </div>

        <form onSubmit={handleSubmit}>
          <div className="modal-body">
            <div style={{ padding: '10px 14px', background: 'var(--soft)', borderRadius: 'var(--radius-sm)', fontSize: 13 }}>
              <strong>#{req.id}</strong> · {req.cluster} · {req.status.replace(/_/g, ' ')}
            </div>

            {action === 'approve' && (
              <div className="form-group">
                <label className="form-label">OB FTE Name</label>
                <input className="form-input" value={form.ob_fte} onChange={set('ob_fte')} required autoFocus />
              </div>
            )}

            {(action === 'reject' || action === 'cancel') && (
              <div className="form-group">
                <label className="form-label">Remarks</label>
                <textarea className="form-textarea" value={form.remarks} onChange={set('remarks')} required autoFocus />
              </div>
            )}

            {action === 'assign' && (
              <div className="form-group">
                <label className="form-label">Midmile FTE</label>
                <input className="form-input" value={form.midmile_fte} onChange={set('midmile_fte')} required autoFocus />
              </div>
            )}

            {action === 'for-docking' && (
              <>
                <div className="form-group">
                  <label className="form-label">Midmile FTE</label>
                  <input className="form-input" value={form.midmile_fte} onChange={set('midmile_fte')} required autoFocus />
                </div>
                <div className="form-group">
                  <label className="form-label">Plate Number</label>
                  <input className="form-input" value={form.plate_number} onChange={set('plate_number')} required />
                </div>
              </>
            )}

            {action === 'dock' && (
              <>
                <div className="form-group">
                  <label className="form-label">Driver ID</label>
                  <input className="form-input" value={form.driver_id} onChange={set('driver_id')} required autoFocus />
                </div>
                <div className="form-group">
                  <label className="form-label">Linehaul Trip No</label>
                  <input className="form-input" value={form.linehaul_trip_no} onChange={set('linehaul_trip_no')} required />
                </div>
                <div className="form-group">
                  <label className="form-label">Docking Time</label>
                  <input className="form-input" type="time" value={form.docking_time} onChange={set('docking_time')} required />
                </div>
              </>
            )}

            {action === 'confirm' && (
              <p style={{ fontSize: 14, color: 'var(--muted)' }}>
                Confirm that truck <strong>{req.plate_number || 'unknown'}</strong> has been verified and is ready?
              </p>
            )}
          </div>

          <div className="modal-footer">
            <button type="button" className="btn btn--secondary" onClick={onClose} disabled={loading}>
              Cancel
            </button>
            <button
              type="submit"
              className={`btn ${action === 'reject' || action === 'cancel' ? 'btn--danger' : 'btn--primary'}`}
              disabled={loading}
            >
              {loading ? 'Submitting…' : titles[action]}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
