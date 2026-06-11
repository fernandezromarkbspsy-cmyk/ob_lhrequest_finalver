import { FormEvent, useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import deliveryIcon from "../../frontend/static/icons/delivery.png";
import {
  Cluster,
  Queue,
  RequestPayload,
  RequestRow,
  Role,
  clusterBacklogs,
  clusterDock,
  clusterID,
  clusterName,
  clusterRegion,
  createRequest,
  createUser,
  fetchClusters,
  fetchMe,
  fetchNotifications,
  fetchRequests,
  fetchStats,
  login,
  logout,
  roleLabel,
  updateRequest
} from "./api";
import { useAppStore } from "./store";

type Page = "dashboard" | "ops" | "mm" | "dock" | "settings";

const pageQueue: Record<Page, Queue> = {
  dashboard: "all",
  ops: "ops",
  mm: "mm",
  dock: "dock",
  settings: "settings"
};

export default function App() {
  const queryClient = useQueryClient();
  const user = useAppStore((state) => state.user);
  const setUser = useAppStore((state) => state.setUser);
  const setQueue = useAppStore((state) => state.setQueue);
  const [page, setPage] = useState<Page>("dashboard");
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState("ALL");
  const [showLogin, setShowLogin] = useState(false);
  const [showRequest, setShowRequest] = useState(false);
  const [themeDark, setThemeDark] = useState(localStorage.getItem("soc5_theme") === "dark");
  const queue = pageQueue[page];

  useEffect(() => {
    setQueue(queue);
  }, [queue, setQueue]);

  useEffect(() => {
    document.body.classList.add("role-ready");
    document.body.classList.toggle("theme-dark", themeDark);
    document.documentElement.classList.toggle("theme-dark", themeDark);
    localStorage.setItem("soc5_theme", themeDark ? "dark" : "light");
  }, [themeDark]);

  const me = useQuery({ queryKey: ["me"], queryFn: fetchMe, retry: false });
  useEffect(() => {
    if (me.data) setUser(me.data);
  }, [me.data, setUser]);

  const stats = useQuery({ queryKey: ["stats"], queryFn: fetchStats, refetchInterval: 15000 });
  const clusters = useQuery({ queryKey: ["clusters"], queryFn: fetchClusters });
  const requests = useQuery({
    queryKey: ["requests", queue, search, status],
    queryFn: () => fetchRequests({ queue, search, status: status === "ALL" ? "" : status, perPage: 50 }),
    enabled: page !== "settings",
    refetchInterval: 20000
  });
  const notifications = useQuery({
    queryKey: ["notifications", user?.role],
    queryFn: () => fetchNotifications(user?.role ?? ""),
    enabled: Boolean(user?.role),
    refetchInterval: 30000
  });

  useEffect(() => {
    const source = new EventSource("/api/events", { withCredentials: true });
    const refresh = () => void invalidateLive(queryClient);
    source.onmessage = refresh;
    [
      "request.created",
      "request.approved",
      "request.rejected_by_mm",
      "request.cancelled",
      "truck.assigned",
      "truck.for_docking",
      "truck.docked",
      "request.confirmed"
    ].forEach((eventName) => source.addEventListener(eventName, refresh));
    return () => source.close();
  }, [queryClient]);

  const rows = requests.data?.requests ?? [];
  const counts = stats.data;

  async function refresh() {
    await invalidateLive(queryClient);
  }

  function go(nextPage: Page) {
    setPage(nextPage);
    setStatus("ALL");
    setSearch("");
  }

  return (
    <div className="app-shell">
      <aside className="sidebar">
        <button className="brand react-clickable" type="button" onClick={() => go("dashboard")} aria-label="SOC 5 Outbound dashboard">
          <span className="brand-mark"><img src={deliveryIcon} alt="" aria-hidden="true" /></span>
          <span>
            <strong>OUTBOUND</strong>
            <small>SOC 5</small>
          </span>
        </button>

        <nav className="nav" aria-label="Primary navigation">
          <p className="nav-label">Menu</p>
          <NavButton active={page === "dashboard"} icon="icon-dashboard" label="Dashboard" onClick={() => go("dashboard")} />

          <section className={roleVisible(user?.role, ["ops_pic", "fte_ops"]) ? "nav-group" : "nav-group is-hidden"}>
            <button className="nav-group-button" type="button" aria-expanded="true">
              <span className="nav-group-main"><span className="nav-icon"><span className="ui-icon icon-box" /></span><span>Outbound</span></span>
              <span className="chevron"><span className="ui-icon icon-chevron-down" /></span>
            </button>
            <div className="nav-group-items">
              <NavButton child active={page === "ops"} icon="icon-truck" label="LH Request" badge={counts?.pending_ops ?? 0} onClick={() => go("ops")} />
            </div>
          </section>

          <section className={roleVisible(user?.role, ["fte_mm"]) ? "nav-group" : "nav-group is-hidden"}>
            <button className="nav-group-button" type="button" aria-expanded="true">
              <span className="nav-group-main"><span className="nav-icon"><span className="ui-icon icon-box" /></span><span>Midmile</span></span>
              <span className="chevron"><span className="ui-icon icon-chevron-down" /></span>
            </button>
            <div className="nav-group-items">
              <NavButton child active={page === "mm"} icon="icon-clock" label="Truck Request" badge={counts?.pending_mm ?? 0} onClick={() => go("mm")} />
            </div>
          </section>

          <section className={roleVisible(user?.role, ["dock_officer", "doc_officer"]) ? "nav-group" : "nav-group is-hidden"}>
            <button className="nav-group-button" type="button" aria-expanded="true">
              <span className="nav-group-main"><span className="nav-icon"><span className="ui-icon icon-box" /></span><span>Dock</span></span>
              <span className="chevron"><span className="ui-icon icon-chevron-down" /></span>
            </button>
            <div className="nav-group-items">
              <NavButton child active={page === "dock"} icon="icon-clock" label="Dock Officer" badge={counts?.for_docking ?? 0} onClick={() => go("dock")} />
            </div>
          </section>

          <div className={roleVisible(user?.role, ["fte_ops", "fte_mm"]) ? "nav-divider" : "nav-divider is-hidden"} />
          <section className={roleVisible(user?.role, ["fte_ops", "fte_mm"]) ? "nav-group" : "nav-group is-hidden"}>
            <button className="nav-group-button" type="button" aria-expanded="true">
              <span className="nav-group-main"><span className="nav-icon"><span className="ui-icon icon-settings" /></span><span>Settings</span></span>
              <span className="chevron"><span className="ui-icon icon-chevron-down" /></span>
            </button>
            <div className="nav-group-items">
              <NavButton child active={page === "settings"} icon="icon-user-plus" label="Add New Role" onClick={() => go("settings")} />
            </div>
          </section>
        </nav>

        <div className="sidebar-user">
          <span className="avatar user-avatar">{initials(user?.name ?? "Guest")}</span>
          <span>
            <strong>{user?.name ?? "Guest"}</strong>
            <small>{user ? roleLabel(user.role) : "Not signed in"}</small>
          </span>
          {user ? (
            <button className="icon-button" type="button" onClick={() => void logout().then(() => setUser(null))} title="Logout" aria-label="Logout">
              <span className="ui-icon icon-logout" aria-hidden="true" />
            </button>
          ) : null}
        </div>
      </aside>

      <main className="main-panel">
        <header className="topbar">
          <div className="topbar-title">
            <button className="icon-button mobile-only" type="button" title="Toggle sidebar" aria-label="Toggle sidebar">
              <span className="ui-icon icon-menu" aria-hidden="true" />
            </button>
            <div>
              <span className="topbar-kicker">Gentelella Admin</span>
              <h1>{pageTitle(page)}</h1>
            </div>
          </div>

          <div className="topbar-actions">
            <label className="topbar-search">
              <span className="ui-icon icon-search" aria-hidden="true" />
              <input type="search" placeholder="Search" value={search} onChange={(event) => setSearch(event.target.value)} />
            </label>
            <button className="icon-button" type="button" onClick={() => setThemeDark((value) => !value)} title="Switch theme" aria-label="Switch theme">
              <span className="ui-icon icon-theme" aria-hidden="true" />
            </button>
            <button className="notification-button" type="button" title="Notifications" aria-label="Notifications">
              <span className="ui-icon icon-bell" aria-hidden="true" />
              <strong>{notifications.data?.count ?? 0}</strong>
            </button>
            <span className="connection-pill"><span className="ui-icon icon-wifi" aria-hidden="true" />Live Connected</span>
            {user ? (
              <div className="topbar-user user-menu">
                <button className="user-menu-button" type="button">
                  <span className="avatar small user-avatar">{initials(user.name)}</span>
                  <span><strong>{user.name}</strong><small>{roleLabel(user.role)}</small></span>
                  <span className="ui-icon icon-chevron-down" aria-hidden="true" />
                </button>
              </div>
            ) : (
              <button className="primary-button compact" type="button" onClick={() => setShowLogin(true)}>
                <span className="ui-icon icon-login" aria-hidden="true" /><span>Login</span>
              </button>
            )}
          </div>
        </header>

        <div className="page-content">
          {page === "dashboard" ? <Dashboard stats={counts} rows={rows} go={go} /> : null}
          {page === "settings" ? <Settings actorRole={user?.role ?? ""} onDone={refresh} /> : null}
          {page !== "dashboard" && page !== "settings" ? (
            <RequestsPage
              page={page}
              rows={rows}
              loading={requests.isLoading}
              stats={counts}
              status={status}
              setStatus={setStatus}
              userRole={user?.role ?? ""}
              clusters={clusters.data ?? []}
              onNew={() => setShowRequest(true)}
              onDone={refresh}
            />
          ) : null}
        </div>
      </main>

      {showLogin ? <LoginModal onClose={() => setShowLogin(false)} onDone={refresh} /> : null}
      {showRequest ? <RequestModal clusters={clusters.data ?? []} userName={user?.name ?? ""} onClose={() => setShowRequest(false)} onDone={refresh} /> : null}
    </div>
  );
}

function NavButton({ active, child, icon, label, badge, onClick }: { active?: boolean; child?: boolean; icon: string; label: string; badge?: number; onClick: () => void }) {
  return (
    <button className={`nav-link ${child ? "nav-child" : ""} ${active ? "is-active" : ""}`} type="button" onClick={onClick}>
      <span className="nav-icon"><span className={`ui-icon ${icon}`} aria-hidden="true" /></span>
      <span>{label}</span>
      {badge !== undefined ? <span className={badge > 0 ? "badge is-visible" : "badge"}>{badge}</span> : null}
    </button>
  );
}

function Dashboard({ stats, rows, go }: { stats?: Awaited<ReturnType<typeof fetchStats>>; rows: RequestRow[]; go: (page: Page) => void }) {
  const openAlerts = (stats?.pending_ops ?? 0) + (stats?.pending_mm ?? 0) + (stats?.for_docking ?? 0) + (stats?.rejected ?? 0);
  return (
    <>
      <section className="panel command-view">
        <div className="command-copy">
          <p className="eyebrow">Live Operations</p>
          <h2>Linehaul Command View</h2>
          <p>Monitor request volume, handoffs, docking readiness, and exceptions from one working view.</p>
        </div>
        <div className="command-metrics">
          <button type="button" onClick={() => go("ops")}><span>Today</span><strong>{stats?.total_today ?? 0}</strong><small>Requests created</small></button>
          <button type="button" onClick={() => go("mm")}><span>Active Flow</span><strong>{stats?.confirmed_trucks ?? 0}</strong><small>Assigned, docking, or docked</small></button>
          <button type="button" onClick={() => go("ops")}><span>Open Alerts</span><strong>{openAlerts}</strong><small>Needs team action</small></button>
        </div>
      </section>

      <section className="dashboard-grid">
        <article className="panel chart-panel">
          <div className="panel-heading"><div><h2>Request Trend</h2><p><span>Live request activity</span> <span className="legend-dot orange" /> Created requests</p></div></div>
          <div className="line-chart" aria-label="Request trend chart">
            <div className="chart-scale"><span>4</span><span>3</span><span>2</span><span>1</span><span>0</span></div>
            <div className="chart-plot"><span className="grid-line" /><span className="grid-line" /><span className="grid-line" /><span className="grid-line" /><div className="trend-bars" /></div>
          </div>
        </article>
        <aside className="panel priority-panel">
          <div className="panel-heading"><div><h2>Priority Work</h2><p>Queues that need action now</p></div></div>
          <div className="priority-list">
            <Priority label="Ops Approval" note="Pending or returned linehaul requests" value={stats?.pending_ops ?? 0} onClick={() => go("ops")} urgent />
            <Priority label="Midmile Handoff" note="Approved requests awaiting truck action" value={stats?.pending_mm ?? 0} onClick={() => go("mm")} />
            <Priority label="Docking Queue" note="Trucks ready for docking details" value={stats?.for_docking ?? 0} onClick={() => go("dock")} ready />
            <Priority label="Exceptions" note="Rejected or cancelled requests" value={stats?.rejected ?? 0} onClick={() => go("ops")} exception />
          </div>
        </aside>
      </section>

      <section className="dashboard-widget-grid">
        <MetricCard className="requested" label="Today" value={stats?.total_today ?? 0} note="Total requests" />
        <MetricCard className="pending" label="Pending Ops" value={stats?.pending_ops ?? 0} note="Needs Ops action" />
        <MetricCard className="docking" label="For Docking" value={stats?.for_docking ?? 0} note="Dock queue" />
        <MetricCard className="REJECTED_BY_MM" label="Exceptions" value={stats?.rejected ?? 0} note="Rejected or cancelled" />
      </section>

      <section className="panel table-panel">
        <div className="panel-heading"><div><h2>Recent Requests</h2><p>Latest workflow records</p></div></div>
        <RequestTable rows={rows.slice(0, 8)} userRole="" onDone={async () => {}} />
      </section>
    </>
  );
}

function RequestsPage(props: {
  page: Page;
  rows: RequestRow[];
  loading: boolean;
  stats?: Awaited<ReturnType<typeof fetchStats>>;
  status: string;
  setStatus: (status: string) => void;
  userRole: Role;
  clusters: Cluster[];
  onNew: () => void;
  onDone: () => Promise<void>;
}) {
  const title = props.page === "ops" ? "Linehaul Requests" : props.page === "mm" ? "Truck Request" : "Dock Officer";
  return (
    <>
      <section className="queue-header">
        <div><p className="eyebrow">{props.page === "ops" ? "Outbound" : props.page === "mm" ? "Midmile" : "Dock"}</p><h2>{title}</h2></div>
        <div className="queue-actions">
          {roleVisible(props.userRole, ["ops_pic", "fte_ops"]) ? <button className="primary-button" type="button" onClick={props.onNew}><span className="ui-icon icon-plus" /><span>New Request</span></button> : null}
          <button className="ghost-button" type="button"><span className="ui-icon icon-download" /><span>Export CSV</span></button>
        </div>
      </section>
      <section className="panel approval-strip">
        <div><p className="eyebrow">{title}</p><h2>{props.page === "ops" ? "Pending Approval Queue" : props.page === "mm" ? "Midmile Handoff Queue" : "Docking Queue"}</h2></div>
        <strong>{props.page === "ops" ? props.stats?.pending_ops ?? 0 : props.page === "mm" ? props.stats?.pending_mm ?? 0 : props.stats?.for_docking ?? 0}</strong>
      </section>
      <section className="panel table-panel">
        <div className="quick-filter-tabs">
          {["ALL", "PENDING", "APPROVED", "ASSIGNED", "FOR_DOCKING", "DOCKED", "REJECTED_BY_MM"].map((item) => (
            <button key={item} type="button" className={props.status === item ? "is-active" : ""} onClick={() => props.setStatus(item)}>{statusText(item)}</button>
          ))}
        </div>
        {props.loading ? <p className="react-empty">Loading linehaul requests...</p> : <RequestTable rows={props.rows} userRole={props.userRole} onDone={props.onDone} />}
      </section>
    </>
  );
}

function RequestTable({ rows, userRole, onDone }: { rows: RequestRow[]; userRole: Role; onDone: () => Promise<void> }) {
  return (
    <div className="data-table-wrap">
      <table className="data-table">
        <thead>
          <tr><th>Requested</th><th>Cluster</th><th>Region</th><th>Dock</th><th>Backlogs</th><th>Truck</th><th>Plate</th><th>Driver</th><th>Trip No.</th><th>Docking Time</th><th>Status</th><th>Actions</th></tr>
        </thead>
        <tbody>
          {rows.length === 0 ? <tr><td colSpan={12} className="empty-state">No linehaul requests found.</td></tr> : null}
          {rows.map((row) => <RequestRowView key={row.id} row={row} userRole={userRole} onDone={onDone} />)}
        </tbody>
      </table>
    </div>
  );
}

function RequestRowView({ row, userRole, onDone }: { row: RequestRow; userRole: Role; onDone: () => Promise<void> }) {
  const [payload, setPayload] = useState<RequestPayload>({});
  const [error, setError] = useState("");
  const actions = allowedActions(row, userRole);
  const mutation = useMutation({
    mutationFn: ({ action, body }: { action: string; body: RequestPayload }) => updateRequest(row.id, action, body),
    onSuccess: async () => {
      setPayload({});
      setError("");
      await onDone();
    },
    onError: (err: Error) => setError(err.message)
  });
  return (
    <>
      <tr>
        <td>{row.request_date || row.request_timestamp}</td><td>{row.cluster || "-"}</td><td>{row.region || "-"}</td><td>{row.dock_no || "-"}</td><td>{row.backlogs ?? 0}</td><td>{row.truck_size || "-"} {row.truck_type || ""}</td><td>{row.plate_number || "-"}</td><td>{row.driver_id || "-"}</td><td>{row.linehaul_trip_no || "-"}</td><td>{row.docking_time || "-"}</td><td><span className={`status-pill ${row.status}`}>{row.status_label || row.status}</span></td>
        <td><div className="react-inline-actions">{actions.map((action) => <button key={action.key} className="ghost-button compact" type="button" onClick={() => mutation.mutate({ action: action.key, body: payload })}>{action.label}</button>)}</div></td>
      </tr>
      {actions.length ? (
        <tr className="inline-request-row">
          <td colSpan={12}>
            <div className="react-action-fields">
              {needsField(actions, "ob_fte") ? <input placeholder="OB FTE" value={payload.ob_fte ?? ""} onChange={(event) => setPayload({ ...payload, ob_fte: event.target.value })} /> : null}
              {needsField(actions, "midmile_fte") ? <input placeholder="Midmile FTE" value={payload.midmile_fte ?? ""} onChange={(event) => setPayload({ ...payload, midmile_fte: event.target.value })} /> : null}
              {needsField(actions, "plate_number") ? <input placeholder="Plate number" value={payload.plate_number ?? ""} onChange={(event) => setPayload({ ...payload, plate_number: event.target.value })} /> : null}
              {needsField(actions, "dock") ? <><input placeholder="Driver ID" value={payload.driver_id ?? ""} onChange={(event) => setPayload({ ...payload, driver_id: event.target.value })} /><input placeholder="LH trip number" value={payload.linehaul_trip_no ?? ""} onChange={(event) => setPayload({ ...payload, linehaul_trip_no: event.target.value })} /><input type="datetime-local" value={payload.docking_time ?? ""} onChange={(event) => setPayload({ ...payload, docking_time: event.target.value })} /></> : null}
              {needsField(actions, "remarks") ? <input placeholder="Remarks" value={payload.remarks ?? ""} onChange={(event) => setPayload({ ...payload, remarks: event.target.value })} /> : null}
              {error ? <p className="form-error">{error}</p> : null}
            </div>
          </td>
        </tr>
      ) : null}
    </>
  );
}

function LoginModal({ onClose, onDone }: { onClose: () => void; onDone: () => Promise<void> }) {
  const setUser = useAppStore((state) => state.setUser);
  const [loginType, setLoginType] = useState<"fte" | "backroom">("fte");
  const [email, setEmail] = useState("");
  const [opsID, setOpsID] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const mutation = useMutation({
    mutationFn: login,
    onSuccess: async (nextUser) => {
      setUser(nextUser);
      onClose();
      await onDone();
    },
    onError: (err: Error) => setError(err.message)
  });
  return (
    <div className="react-modal-backdrop">
      <section className="login-modal" role="dialog" aria-modal="true">
        <button className="modal-close" type="button" onClick={onClose}>x</button>
        <h2>Login</h2>
        <form className="form-grid" onSubmit={(event) => { event.preventDefault(); mutation.mutate({ login_type: loginType, email, ops_id: opsID, password }); }}>
          <div className="segmented" role="tablist"><button type="button" className={loginType === "fte" ? "is-active" : ""} onClick={() => setLoginType("fte")}>FTE</button><button type="button" className={loginType === "backroom" ? "is-active" : ""} onClick={() => setLoginType("backroom")}>Backroom</button></div>
          {loginType === "fte" ? <label><span>Email</span><input type="email" value={email} onChange={(event) => setEmail(event.target.value)} /></label> : <label><span>Ops ID</span><input value={opsID} onChange={(event) => setOpsID(event.target.value)} /></label>}
          <label><span>Password</span><input type="password" value={password} onChange={(event) => setPassword(event.target.value)} /></label>
          {error ? <p className="form-error">{error}</p> : null}
          <button className="primary-button" type="submit">Sign in</button>
        </form>
      </section>
    </div>
  );
}

function RequestModal({ clusters, userName, onClose, onDone }: { clusters: Cluster[]; userName: string; onClose: () => void; onDone: () => Promise<void> }) {
  const [payload, setPayload] = useState<RequestPayload>({ ob_ops_pic: userName });
  const [error, setError] = useState("");
  const mutation = useMutation({ mutationFn: createRequest, onSuccess: async () => { onClose(); await onDone(); }, onError: (err: Error) => setError(err.message) });
  function selectCluster(id: number) {
    const match = clusters.find((cluster) => clusterID(cluster) === id);
    setPayload({ ...payload, cluster_id: id, cluster: match ? clusterName(match) : "", region: match ? clusterRegion(match) : "", dock_no: match ? clusterDock(match) : "", backlogs: match ? clusterBacklogs(match) : 0 });
  }
  return (
    <div className="react-modal-backdrop">
      <section className="dialog" role="dialog" aria-modal="true">
        <button className="modal-close" type="button" onClick={onClose}>x</button>
        <h2>New Linehaul Request</h2>
        <form className="form-grid" onSubmit={(event) => { event.preventDefault(); mutation.mutate(payload); }}>
          <label><span>Cluster</span><select value={payload.cluster_id ?? 0} onChange={(event) => selectCluster(Number(event.target.value))}><option value="0">Select cluster</option>{clusters.map((cluster) => <option key={clusterID(cluster)} value={clusterID(cluster)}>{clusterName(cluster)} / {clusterRegion(cluster)}</option>)}</select></label>
          <label><span>Truck size</span><input value={payload.truck_size ?? ""} onChange={(event) => setPayload({ ...payload, truck_size: event.target.value })} /></label>
          <label><span>Truck type</span><input value={payload.truck_type ?? ""} onChange={(event) => setPayload({ ...payload, truck_type: event.target.value })} /></label>
          <label><span>Ops PIC</span><input value={payload.ob_ops_pic ?? ""} onChange={(event) => setPayload({ ...payload, ob_ops_pic: event.target.value })} /></label>
          {error ? <p className="form-error">{error}</p> : null}
          <div className="form-actions"><button className="ghost-button" type="button" onClick={onClose}>Cancel</button><button className="primary-button" type="submit">Create Request</button></div>
        </form>
      </section>
    </div>
  );
}

function Settings({ actorRole, onDone }: { actorRole: Role; onDone: () => Promise<void> }) {
  const [payload, setPayload] = useState({ name: "", role: "" as Role, email: "", ops_id: "", password: "" });
  const [error, setError] = useState("");
  const isFTE = payload.role === "fte_ops" || payload.role === "fte_mm";
  const mutation = useMutation({ mutationFn: () => createUser({ ...payload, actor_role: actorRole }), onSuccess: async () => { setPayload({ name: "", role: "" as Role, email: "", ops_id: "", password: "" }); await onDone(); }, onError: (err: Error) => setError(err.message) });
  return (
    <section className="panel settings-panel" id="add-new-role">
      <div className="panel-heading"><div><p className="eyebrow">Settings</p><h2>Add New Role</h2></div></div>
      <form className="form-grid settings-form" onSubmit={(event: FormEvent) => { event.preventDefault(); mutation.mutate(); }}>
        <label><span>Name</span><input value={payload.name} onChange={(event) => setPayload({ ...payload, name: event.target.value })} /></label>
        <label><span>Role</span><select value={payload.role} onChange={(event) => setPayload({ ...payload, role: event.target.value as Role })}><option value="">Select role</option><option value="fte_ops">FTE Ops</option><option value="fte_mm">FTE MM</option><option value="ops_pic">Ops PIC</option><option value="dock_officer">Dock Officer</option></select></label>
        {isFTE ? <label><span>Email</span><input type="email" value={payload.email} onChange={(event) => setPayload({ ...payload, email: event.target.value })} /></label> : <label><span>Ops ID</span><input value={payload.ops_id} onChange={(event) => setPayload({ ...payload, ops_id: event.target.value })} /></label>}
        <label><span>Temporary password</span><input type="password" value={payload.password} onChange={(event) => setPayload({ ...payload, password: event.target.value })} /></label>
        {error ? <p className="form-error">{error}</p> : null}
        <button className="primary-button" type="submit">Add Role</button>
      </form>
    </section>
  );
}

function Priority({ label, note, value, onClick, urgent, ready, exception }: { label: string; note: string; value: number; onClick: () => void; urgent?: boolean; ready?: boolean; exception?: boolean }) {
  return <button className={`priority-row ${urgent ? "urgent" : ""} ${ready ? "ready" : ""} ${exception ? "exception" : ""}`} type="button" onClick={onClick}><span className="priority-icon"><span className="ui-icon icon-clock" /></span><span><strong>{label}</strong><small>{note}</small></span><b>{value}</b></button>;
}

function MetricCard({ className, label, value, note }: { className: string; label: string; value: number; note: string }) {
  return <article className={`metric-card ${className}`}><span>{label}</span><strong>{value}</strong><small>{note}</small></article>;
}

function allowedActions(row: RequestRow, role: Role) {
  const actions: { key: string; label: string }[] = [];
  if ((role === "fte_ops" || role === "ops_pic") && (row.status === "PENDING" || row.status === "REJECTED_BY_MM" || row.status === "")) actions.push({ key: "approve", label: "Approve" }, { key: "cancel", label: "Cancel" });
  if (role === "fte_mm" && row.status === "APPROVED") actions.push({ key: "reject", label: "Reject" }, { key: "assign", label: "Assign" });
  if (role === "fte_mm" && row.status === "ASSIGNED") actions.push({ key: "for-docking", label: "For Docking" });
  if ((role === "dock_officer" || role === "doc_officer") && row.status === "FOR_DOCKING") actions.push({ key: "dock", label: "Dock" });
  if ((role === "dock_officer" || role === "doc_officer") && row.status === "DOCKED") actions.push({ key: "confirm", label: "Confirm" });
  return actions;
}

function needsField(actions: { key: string }[], field: "ob_fte" | "midmile_fte" | "plate_number" | "dock" | "remarks") {
  if (field === "ob_fte") return actions.some((action) => action.key === "approve");
  if (field === "midmile_fte") return actions.some((action) => action.key === "assign" || action.key === "for-docking");
  if (field === "plate_number") return actions.some((action) => action.key === "for-docking");
  if (field === "dock") return actions.some((action) => action.key === "dock");
  return actions.some((action) => action.key === "reject" || action.key === "cancel");
}

function roleVisible(role: Role | undefined, allowed: Role[]) {
  return Boolean(role && allowed.includes(role));
}

function pageTitle(page: Page) {
  if (page === "ops") return "LH Request";
  if (page === "mm") return "Truck Request";
  if (page === "dock") return "Dock Officer";
  if (page === "settings") return "Settings";
  return "Dashboard";
}

function statusText(status: string) {
  return status === "ALL" ? "All" : status.replaceAll("_", " ");
}

function initials(name: string) {
  return name.split(/\s+/).filter(Boolean).slice(0, 2).map((part) => part[0]?.toUpperCase()).join("") || "G";
}

async function invalidateLive(queryClient: ReturnType<typeof useQueryClient>) {
  await Promise.all([
    queryClient.invalidateQueries({ queryKey: ["stats"] }),
    queryClient.invalidateQueries({ queryKey: ["requests"] }),
    queryClient.invalidateQueries({ queryKey: ["notifications"] })
  ]);
}
