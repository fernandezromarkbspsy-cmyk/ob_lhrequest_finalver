import { AppStats } from '@/types'
import { TrendingUp, Clock, Truck, Warehouse, CheckCircle, XCircle } from 'lucide-react'

interface Props { stats?: AppStats; loading?: boolean }

interface StatDef {
  key: keyof AppStats
  label: string
  icon: React.ReactNode
  variant: string
}

const defs: StatDef[] = [
  { key: 'total_today',       label: 'Total Today',     icon: <TrendingUp size={18} />,  variant: '' },
  { key: 'pending_ops',       label: 'Pending OPS',     icon: <Clock size={18} />,       variant: 'stat-card--orange' },
  { key: 'pending_mm',        label: 'Pending MM',      icon: <Clock size={18} />,       variant: 'stat-card--orange' },
  { key: 'for_docking',       label: 'For Docking',     icon: <Truck size={18} />,       variant: 'stat-card--blue' },
  { key: 'confirmed_trucks',  label: 'Confirmed',       icon: <CheckCircle size={18} />, variant: 'stat-card--green' },
  { key: 'rejected',          label: 'Rejected',        icon: <XCircle size={18} />,     variant: 'stat-card--red' },
]

export default function StatsCards({ stats, loading }: Props) {
  return (
    <div className="stats-grid">
      {defs.map(({ key, label, icon, variant }) => (
        <div key={key} className={`stat-card ${variant}`}>
          <div className="stat-card__icon">{icon}</div>
          <div className="stat-card__label">{label}</div>
          <div className="stat-card__value">
            {loading ? '—' : (stats?.[key] ?? 0)}
          </div>
        </div>
      ))}
    </div>
  )
}
