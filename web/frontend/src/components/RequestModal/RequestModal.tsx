import { useState, FormEvent, useEffect } from 'react'
import { X } from 'lucide-react'
import { Cluster, Request, RequestPayload } from '@/types'
import { useAuthStore } from '@/store/authStore'

interface Props {
  clusters: Cluster[]
  editRequest?: Request
  onClose: () => void
  onSubmit: (payload: RequestPayload) => Promise<void>
}

const TRUCK_SIZES = ['CL', 'CDD', 'LT', '10W', '6W', 'VAN']
const TRUCK_TYPES = ['WING VAN', 'BOX VAN', 'DROPSIDE', 'REEFER', 'TRAILER']

export default function RequestModal({ clusters, editRequest, onClose, onSubmit }: Props) {
  const user = useAuthStore((s) => s.user)
  const isEdit = !!editRequest

  const [loading, setLoading] = useState(false)
  const [clusterId, setClusterId] = useState(0)
  const [form, setForm] = useState({
    cluster:     editRequest?.cluster || '',
    region:      editRequest?.region || '',
    dock_no:     editRequest?.dock_no || '',
    backlogs:    editRequest?.backlogs ?? 0,
    truck_size:  editRequest?.truck_size || '',
    truck_type:  editRequest?.truck_type || '',
    ob_ops_pic:  editRequest?.ob_ops_pic || '',
    ob_fte:      editRequest?.ob_fte || user?.name || '',
  })

  // Auto-fill cluster info when cluster is selected
  useEffect(() => {
    if (!clusterId) return
    const c = clusters.find((cl) => cl.id === clusterId)
    if (c) {
      setForm((f) => ({
        ...f,
        cluster:  c.cluster,
        region:   c.region,
        dock_no:  c.dock_no,
        backlogs: c.backlogs,
      }))
    }
  }, [clusterId, clusters])

  const set = (k: keyof typeof form) =>
    (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) =>
      setForm((f) => ({ ...f, [k]: k === 'backlogs' ? Number(e.target.value) : e.target.value }))

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setLoading(true)
    try {
      const payload: RequestPayload = {
        ...form,
        cluster_id: clusterId || undefined,
      }
      await onSubmit(payload)
      onClose()
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="modal-overlay" onClick={(e) => e.target === e.currentTarget && onClose()}>
      <div className="modal modal--lg">
        <div className="modal-header">
          <h2 className="modal-title">{isEdit ? 'Edit Request' : 'New LH Request'}</h2>
          <button className="modal-close" onClick={onClose}><X size={18} /></button>
        </div>

        <form onSubmit={handleSubmit}>
          <div className="modal-body">
            {/* Cluster */}
            <div className="form-row">
              <div className="form-group">
                <label className="form-label">Cluster</label>
                <select
                  className="form-select"
                  value={clusterId}
                  onChange={(e) => setClusterId(Number(e.target.value))}
                >
                  <option value={0}>— Select cluster —</option>
                  {clusters.map((c) => (
                    <option key={c.id} value={c.id}>{c.cluster}</option>
                  ))}
                </select>
              </div>
              <div className="form-group">
                <label className="form-label">Region</label>
                <input className="form-input" value={form.region} onChange={set('region')} placeholder="Auto-filled" />
              </div>
            </div>

            <div className="form-row">
              <div className="form-group">
                <label className="form-label">Dock No</label>
                <input className="form-input" value={form.dock_no} onChange={set('dock_no')} placeholder="Auto-filled" />
              </div>
              <div className="form-group">
                <label className="form-label">Backlogs</label>
                <input className="form-input" type="number" min={0} value={form.backlogs} onChange={set('backlogs')} />
              </div>
            </div>

            {/* Truck */}
            <div className="form-row">
              <div className="form-group">
                <label className="form-label">Truck Size</label>
                <select className="form-select" value={form.truck_size} onChange={set('truck_size')} required>
                  <option value="">— Select —</option>
                  {TRUCK_SIZES.map((s) => <option key={s} value={s}>{s}</option>)}
                </select>
              </div>
              <div className="form-group">
                <label className="form-label">Truck Type</label>
                <select className="form-select" value={form.truck_type} onChange={set('truck_type')} required>
                  <option value="">— Select —</option>
                  {TRUCK_TYPES.map((t) => <option key={t} value={t}>{t}</option>)}
                </select>
              </div>
            </div>

            {/* Personnel */}
            <div className="form-row">
              <div className="form-group">
                <label className="form-label">OB OPS PIC</label>
                <input className="form-input" value={form.ob_ops_pic} onChange={set('ob_ops_pic')} placeholder="Name" />
              </div>
              <div className="form-group">
                <label className="form-label">OB FTE</label>
                <input className="form-input" value={form.ob_fte} onChange={set('ob_fte')} placeholder="Your name" />
              </div>
            </div>
          </div>

          <div className="modal-footer">
            <button type="button" className="btn btn--secondary" onClick={onClose} disabled={loading}>
              Cancel
            </button>
            <button type="submit" className="btn btn--primary" disabled={loading}>
              {loading ? 'Saving…' : isEdit ? 'Save Changes' : 'Submit Request'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
