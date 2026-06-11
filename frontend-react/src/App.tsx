import { useQuery } from "@tanstack/react-query";
import { fetchRequests, fetchStats } from "./api";
import { useAppStore } from "./store";

const queues = [
  { key: "all", label: "All" },
  { key: "ops", label: "Ops" },
  { key: "mm", label: "Midmile" },
  { key: "dock", label: "Dock" }
] as const;

export default function App() {
  const queue = useAppStore((state) => state.queue);
  const setQueue = useAppStore((state) => state.setQueue);
  const stats = useQuery({ queryKey: ["stats"], queryFn: fetchStats });
  const requests = useQuery({ queryKey: ["requests"], queryFn: fetchRequests });

  const filteredRequests = (requests.data ?? []).filter((row) => {
    if (queue === "ops") return row.status === "PENDING" || row.status === "REJECTED_BY_MM";
    if (queue === "mm") return row.status === "APPROVED" || row.status === "ASSIGNED";
    if (queue === "dock") return row.status === "FOR_DOCKING" || row.status === "DOCKED";
    return true;
  });

  return (
    <main className="app">
      <aside className="sidebar">
        <strong>OUTBOUND</strong>
        <span>SOC 5</span>
      </aside>

      <section className="workspace">
        <header className="topbar">
          <div>
            <p>Live Operations</p>
            <h1>Truck Request Portal</h1>
          </div>
          <nav aria-label="Queue filters">
            {queues.map((item) => (
              <button
                className={queue === item.key ? "active" : ""}
                key={item.key}
                type="button"
                onClick={() => setQueue(item.key)}
              >
                {item.label}
              </button>
            ))}
          </nav>
        </header>

        <section className="metrics" aria-label="Operational metrics">
          <Metric label="Today" value={stats.data?.total_today ?? 0} />
          <Metric label="Ops" value={stats.data?.pending_ops ?? 0} />
          <Metric label="Midmile" value={stats.data?.pending_mm ?? 0} />
          <Metric label="Dock" value={stats.data?.for_docking ?? 0} />
        </section>

        <section className="panel">
          <div className="panel-heading">
            <h2>Requests</h2>
            <span>{filteredRequests.length} rows</span>
          </div>
          <div className="table">
            {filteredRequests.map((row) => (
              <article key={row.id} className="row">
                <div>
                  <strong>{row.cluster}</strong>
                  <span>{row.region} / Dock {row.dock_no}</span>
                </div>
                <span>{row.truck_size} {row.truck_type}</span>
                <span>{row.plate_number || "-"}</span>
                <b>{row.status_label}</b>
              </article>
            ))}
            {!requests.isLoading && filteredRequests.length === 0 ? <p className="empty">No requests found.</p> : null}
            {requests.isLoading ? <p className="empty">Loading requests...</p> : null}
          </div>
        </section>
      </section>
    </main>
  );
}

function Metric({ label, value }: { label: string; value: number }) {
  return (
    <article className="metric">
      <span>{label}</span>
      <strong>{value}</strong>
    </article>
  );
}
