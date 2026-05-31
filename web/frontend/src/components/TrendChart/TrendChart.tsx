import { TrendData } from '@/types'

interface Props { data?: TrendData; loading?: boolean }

export default function TrendChart({ data, loading }: Props) {
  const points = data?.points ?? []
  const max = Math.max(...points.map((p) => p.count), 1)

  return (
    <div className="trend-card">
      <div className="trend-header">
        <span className="trend-title">Hourly Request Volume</span>
        {data?.period_label && (
          <span className="trend-period">{data.period_label}</span>
        )}
      </div>

      {loading ? (
        <div className="loading-spinner" style={{ minHeight: 140 }} />
      ) : points.length === 0 ? (
        <div className="table-empty" style={{ minHeight: 140 }}>No data for this period</div>
      ) : (
        <div className="trend-bars">
          {points.map((pt) => {
            const pct = max > 0 ? (pt.count / max) * 100 : 0
            return (
              <div key={pt.label} className="trend-bar-wrap" title={`${pt.label}: ${pt.count}`}>
                <div
                  className="trend-bar"
                  style={{ height: `${Math.max(pct, 3)}%` }}
                />
                <span className="trend-bar-label">{pt.label}</span>
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}
