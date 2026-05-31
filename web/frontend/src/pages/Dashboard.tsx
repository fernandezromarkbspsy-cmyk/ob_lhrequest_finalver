import { useStats } from '@/hooks/useStats'
import { useTrend } from '@/hooks/useTrend'
import { useRequests } from '@/hooks/useRequests'
import StatsCards from '@/components/StatsCards/StatsCards'
import TrendChart from '@/components/TrendChart/TrendChart'

function statusClass(s: string) {
  return 'status-pill status-pill--' + s.toLowerCase().replace(/_/g, '-')
}

export default function Dashboard() {
  const { data: stats, isLoading: statsLoading } = useStats()
  const { data: trend, isLoading: trendLoading } = useTrend()
  const { data: reqs } = useRequests({ queue: 'all' })

  const recent = reqs?.requests?.slice(0, 8) ?? []

  return (
    <div>
      <div className="page-header">
        <div>
          <h1 className="page-title">Dashboard</h1>
          <p className="page-subtitle">SOC 5 Outbound Linehaul — Real-time overview</p>
        </div>
      </div>

      <StatsCards stats={stats} loading={statsLoading} />

      <div className="dash-grid">
        <TrendChart data={trend} loading={trendLoading} />

        <div className="activity-card">
          <h3 className="activity-title">Recent Requests</h3>
          {recent.length === 0 ? (
            <div className="table-empty" style={{ minHeight: 140 }}>No requests today</div>
          ) : (
            <div className="activity-list">
              {recent.map((req) => (
                <div key={req.id} className="activity-item">
                  <span className="activity-cluster">{req.cluster}</span>
                  <div className="activity-meta">
                    <div>{req.region} · Dock {req.dock_no}</div>
                    <div>{req.truck_size} {req.truck_type}</div>
                  </div>
                  <span className={statusClass(req.status)}>
                    {req.status_label || req.status.replace(/_/g, ' ')}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
