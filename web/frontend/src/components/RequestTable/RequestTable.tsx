import { useState } from 'react'
import { ChevronUp, ChevronDown, Search, RefreshCw } from 'lucide-react'
import { Request, RequestAction, RequestsFilter, UserRole } from '@/types'
import { useAuthStore } from '@/store/authStore'

interface Props {
  requests: Request[]
  loading?: boolean
  filter: RequestsFilter
  onFilterChange: (f: Partial<RequestsFilter>) => void
  onAction: (req: Request, action: RequestAction) => void
  onRefresh: () => void
  showCreate?: boolean
  onCreateClick?: () => void
  statusOptions?: { value: string; label: string }[]
}

function statusClass(s: string) {
  return 'status-pill status-pill--' + s.toLowerCase().replace(/_/g, '-')
}

type SortDir = 'asc' | 'desc'
const PAGE_SIZE = 20

function getActions(req: Request, role: UserRole): RequestAction[] {
  const actions: RequestAction[] = []
  const st = req.status

  if (role === 'fte_ops' || role === 'admin') {
    if (st === 'PENDING_OPS') actions.push('approve', 'reject')
    if (st === 'PENDING_OPS') actions.push('edit')
    if (['PENDING_OPS', 'PENDING_MM', 'ASSIGNED'].includes(st)) actions.push('cancel')
  }
  if (role === 'ops_pic' || role === 'admin') {
    if (st === 'PENDING_OPS') actions.push('approve', 'reject')
  }
  if (role === 'fte_mm' || role === 'admin') {
    if (st === 'PENDING_MM') actions.push('assign')
    if (st === 'ASSIGNED')   actions.push('for-docking')
  }
  if (role === 'dock_officer' || role === 'doc_officer' || role === 'admin') {
    if (st === 'FOR_DOCKING') actions.push('dock')
    if (st === 'DOCKED')      actions.push('confirm')
  }
  return [...new Set(actions)]
}

function actionLabel(a: RequestAction): string {
  const m: Record<RequestAction, string> = {
    approve: 'Approve',
    reject: 'Reject',
    assign: 'Assign',
    'for-docking': 'For Docking',
    dock: 'Dock',
    confirm: 'Confirm',
    cancel: 'Cancel',
    edit: 'Edit',
  }
  return m[a]
}

function actionVariant(a: RequestAction): string {
  if (a === 'reject' || a === 'cancel') return 'btn btn--sm btn--danger'
  if (a === 'approve' || a === 'confirm') return 'btn btn--sm btn--primary'
  return 'btn btn--sm btn--secondary'
}

export default function RequestTable({
  requests,
  loading,
  filter,
  onFilterChange,
  onAction,
  onRefresh,
  showCreate,
  onCreateClick,
  statusOptions,
}: Props) {
  const user = useAuthStore((s) => s.user)
  const [sortKey, setSortKey] = useState<keyof Request>('request_timestamp')
  const [sortDir, setSortDir] = useState<SortDir>('desc')
  const [page, setPage] = useState(1)

  const handleSort = (key: keyof Request) => {
    if (sortKey === key) setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'))
    else { setSortKey(key); setSortDir('asc') }
    setPage(1)
  }

  const sorted = [...requests].sort((a, b) => {
    const av = a[sortKey] ?? ''
    const bv = b[sortKey] ?? ''
    return sortDir === 'asc'
      ? String(av).localeCompare(String(bv))
      : String(bv).localeCompare(String(av))
  })

  const totalPages = Math.max(1, Math.ceil(sorted.length / PAGE_SIZE))
  const pageData = sorted.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE)

  const SortIcon = ({ k }: { k: keyof Request }) => {
    if (sortKey !== k) return <ChevronDown size={12} style={{ opacity: 0.3 }} />
    return sortDir === 'asc' ? <ChevronUp size={12} /> : <ChevronDown size={12} />
  }

  function Th({ label, k }: { label: string; k: keyof Request }) {
    return (
      <th onClick={() => handleSort(k)}>
        {label} <SortIcon k={k} />
      </th>
    )
  }

  return (
    <div className="table-wrapper">
      {/* Filter bar */}
      <div className="filter-bar">
        <div style={{ position: 'relative', flex: '1 1 220px', maxWidth: 340 }}>
          <Search size={14} style={{ position: 'absolute', left: 10, top: '50%', transform: 'translateY(-50%)', color: 'var(--muted)' }} />
          <input
            className="form-input"
            style={{ paddingLeft: 32 }}
            placeholder="Search plate, cluster, driver…"
            value={filter.search ?? ''}
            onChange={(e) => { onFilterChange({ search: e.target.value }); setPage(1) }}
          />
        </div>

        <input
          type="date"
          className="form-input"
          style={{ width: 148 }}
          value={filter.date_from ?? ''}
          onChange={(e) => { onFilterChange({ date_from: e.target.value }); setPage(1) }}
        />
        <input
          type="date"
          className="form-input"
          style={{ width: 148 }}
          value={filter.date_to ?? ''}
          onChange={(e) => { onFilterChange({ date_to: e.target.value }); setPage(1) }}
        />

        {statusOptions && (
          <select
            className="form-select"
            style={{ width: 160 }}
            value={filter.status ?? 'ALL'}
            onChange={(e) => { onFilterChange({ status: e.target.value }); setPage(1) }}
          >
            <option value="ALL">All statuses</option>
            {statusOptions.map((o) => (
              <option key={o.value} value={o.value}>{o.label}</option>
            ))}
          </select>
        )}

        <div style={{ marginLeft: 'auto', display: 'flex', gap: 8 }}>
          <button className="btn-icon btn-icon--ghost" onClick={onRefresh} title="Refresh">
            <RefreshCw size={16} />
          </button>
          {showCreate && (
            <button className="btn btn--primary btn--sm" onClick={onCreateClick}>
              + New Request
            </button>
          )}
        </div>
      </div>

      {/* Table */}
      <table>
        <thead>
          <tr>
            <Th label="Time" k="request_timestamp" />
            <Th label="Cluster" k="cluster" />
            <Th label="Region" k="region" />
            <Th label="Dock" k="dock_no" />
            <th>Backlogs</th>
            <Th label="Truck" k="truck_size" />
            <Th label="Plate" k="plate_number" />
            <Th label="Driver" k="driver_id" />
            <Th label="Trip No" k="linehaul_trip_no" />
            <Th label="Docking Time" k="docking_time" />
            <Th label="Status" k="status" />
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          {loading ? (
            <tr>
              <td colSpan={12} className="table-empty">
                <div className="loading-spinner" />
              </td>
            </tr>
          ) : pageData.length === 0 ? (
            <tr>
              <td colSpan={12} className="table-empty">No requests found</td>
            </tr>
          ) : (
            pageData.map((req) => {
              const actions = user ? getActions(req, user.role) : []
              const ts = req.request_timestamp
                ? new Date(req.request_timestamp).toLocaleTimeString('en-PH', { hour: '2-digit', minute: '2-digit' })
                : '—'
              return (
                <tr key={req.id}>
                  <td style={{ whiteSpace: 'nowrap', color: 'var(--muted)', fontSize: 12 }}>{ts}</td>
                  <td style={{ fontWeight: 700 }}>{req.cluster || '—'}</td>
                  <td>{req.region || '—'}</td>
                  <td>{req.dock_no || '—'}</td>
                  <td>{req.backlogs ?? '—'}</td>
                  <td>{[req.truck_size, req.truck_type].filter(Boolean).join(' · ') || '—'}</td>
                  <td style={{ fontFamily: 'monospace' }}>{req.plate_number || '—'}</td>
                  <td>{req.driver_id || '—'}</td>
                  <td style={{ fontFamily: 'monospace', fontSize: 12 }}>{req.linehaul_trip_no || '—'}</td>
                  <td>{req.docking_time || '—'}</td>
                  <td>
                    <span className={statusClass(req.status)}>
                      {req.status_label || req.status.replace(/_/g, ' ')}
                    </span>
                  </td>
                  <td>
                    <div className="table-actions">
                      {actions.map((a) => (
                        <button
                          key={a}
                          className={actionVariant(a)}
                          onClick={() => onAction(req, a)}
                        >
                          {actionLabel(a)}
                        </button>
                      ))}
                    </div>
                  </td>
                </tr>
              )
            })
          )}
        </tbody>
      </table>

      {/* Pagination */}
      <div className="pagination">
        <span>{requests.length} record{requests.length !== 1 ? 's' : ''}</span>
        <div className="pagination-controls">
          <button
            className="pagination-btn"
            disabled={page <= 1}
            onClick={() => setPage((p) => p - 1)}
          >‹</button>
          {(() => {
            const pages: number[] = []
            for (let i = 1; i <= totalPages; i++) {
              if (i === 1 || i === totalPages || (i >= page - 2 && i <= page + 2)) {
                pages.push(i)
              }
            }
            const result = []
            let prev = 0
            for (const p of pages) {
              if (prev > 0 && p - prev > 1) {
                result.push(<span key={`ell-${p}`} style={{ padding: '0 4px' }}>…</span>)
              }
              result.push(
                <button
                  key={p}
                  className={`pagination-btn${p === page ? ' pagination-btn--active' : ''}`}
                  onClick={() => setPage(p)}
                >{p}</button>
              )
              prev = p
            }
            return result
          })()}
          <button
            className="pagination-btn"
            disabled={page >= totalPages}
            onClick={() => setPage((p) => p + 1)}
          >›</button>
        </div>
      </div>
    </div>
  )
}
