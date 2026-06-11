export type Stats = {
  total_today: number;
  pending_ops: number;
  pending_mm: number;
  for_docking: number;
  confirmed_trucks: number;
  rejected: number;
};

export type SessionUser = {
  id?: number;
  unique_id?: string;
  name: string;
  role: Role;
  email?: string | null;
  ops_id?: string | null;
};

export type Role = "" | "ops_pic" | "fte_ops" | "fte_mm" | "dock_officer" | "doc_officer";

export type Queue = "all" | "ops" | "mm" | "dock" | "settings";

export type RequestRow = {
  id: number;
  request_timestamp: string;
  request_date: string;
  cluster: string;
  region: string;
  dock_no: string;
  backlogs: number;
  truck_size: string;
  truck_type: string;
  plate_number: string;
  driver_id: string;
  linehaul_trip_no: string;
  docking_time: string;
  status: RequestStatus;
  status_label: string;
  ob_fte: string;
  ob_ops_pic: string;
  midmile_fte: string;
  remarks: string;
};

export type RequestStatus =
  | "PENDING"
  | "APPROVED"
  | "ASSIGNED"
  | "FOR_DOCKING"
  | "DOCKED"
  | "CONFIRMED"
  | "CANCELLED"
  | "REJECTED_BY_MM"
  | "";

export type RequestPayload = {
  cluster_id?: number;
  cluster?: string;
  region?: string;
  dock_no?: string;
  backlogs?: number;
  truck_size?: string;
  truck_type?: string;
  ob_ops_pic?: string;
  ob_fte?: string;
  midmile_fte?: string;
  plate_number?: string;
  driver_id?: string;
  linehaul_trip_no?: string;
  docking_time?: string;
  remarks?: string;
};

export type RequestList = {
  requests: RequestRow[];
  count: number;
  total: number;
  page: number;
  per_page: number;
};

export type Cluster = {
  ID?: number;
  id?: number;
  ClusterName?: string;
  cluster_name?: string;
  HubName?: string;
  hub_name?: string;
  Region?: string;
  region?: string;
  DockNumber?: string;
  dock_number?: string;
  Backlogs?: number;
  backlogs?: number;
};

export type NotificationRow = {
  id: number;
  role: Role;
  request_id: number;
  event_type: string;
  message: string;
  is_read: boolean;
  created_at: string;
};

export type UserPayload = {
  name: string;
  role: Role;
  email?: string;
  ops_id?: string;
  password?: string;
  actor_role?: Role;
};

const apiBase = String(import.meta.env.VITE_API_BASE ?? "").replace(/\/$/, "");

async function requestJSON<T>(path: string, init: RequestInit = {}): Promise<T> {
  const response = await fetch(`${apiBase}${path}`, {
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...(init.headers ?? {})
    },
    ...init
  });
  const text = await response.text();
  const body = text ? JSON.parse(text) : null;
  if (!response.ok) {
    throw new Error(body?.error ?? body?.message ?? "Request failed");
  }
  return body as T;
}

export function fetchStats() {
  return requestJSON<Stats>("/api/stats");
}

export function fetchRequests(params: {
  queue?: Queue;
  search?: string;
  page?: number;
  perPage?: number;
  status?: string;
} = {}) {
  const query = new URLSearchParams();
  if (params.queue && params.queue !== "all" && params.queue !== "settings") query.set("queue", params.queue);
  if (params.search) query.set("search", params.search);
  if (params.status) query.set("status", params.status);
  query.set("page", String(params.page ?? 1));
  query.set("per_page", String(params.perPage ?? 20));
  return requestJSON<RequestList>(`/api/requests?${query.toString()}`);
}

export function createRequest(payload: RequestPayload) {
  return requestJSON<RequestRow>("/api/requests", {
    method: "POST",
    body: JSON.stringify(payload)
  });
}

export function updateRequest(id: number, action: string, payload: RequestPayload = {}) {
  return requestJSON<RequestRow>(`/api/requests/${id}/${action}`, {
    method: "POST",
    body: JSON.stringify(payload)
  });
}

export function login(payload: { login_type: "fte" | "backroom"; email?: string; ops_id?: string; password?: string }) {
  return requestJSON<SessionUser & { redirect?: string }>("/api/login", {
    method: "POST",
    body: JSON.stringify(payload)
  });
}

export function logout() {
  return requestJSON<{ ok: boolean }>("/api/auth/logout", { method: "POST", body: "{}" });
}

export function fetchMe() {
  return requestJSON<SessionUser>("/api/auth/me");
}

export function fetchClusters() {
  return requestJSON<{ clusters?: Cluster[] } | Cluster[]>("/api/clusters").then((body) =>
    Array.isArray(body) ? body : body.clusters ?? []
  );
}

export function createUser(payload: UserPayload) {
  return requestJSON<Record<string, unknown>>("/api/users", {
    method: "POST",
    body: JSON.stringify(payload)
  });
}

export function fetchNotifications(role: Role) {
  const query = role ? `?role=${encodeURIComponent(role)}` : "";
  return requestJSON<{ notifications: NotificationRow[]; count: number }>(`/api/notifications${query}`);
}

export function roleLabel(role: Role | string) {
  switch (role) {
    case "ops_pic":
      return "Ops PIC";
    case "fte_ops":
      return "FTE Ops";
    case "fte_mm":
      return "FTE MM";
    case "dock_officer":
    case "doc_officer":
      return "Dock Officer";
    default:
      return role || "Guest";
  }
}

export function clusterID(cluster: Cluster) {
  return Number(cluster.id ?? cluster.ID ?? 0);
}

export function clusterName(cluster: Cluster) {
  return String(cluster.cluster_name ?? cluster.ClusterName ?? "");
}

export function clusterRegion(cluster: Cluster) {
  return String(cluster.region ?? cluster.Region ?? "");
}

export function clusterDock(cluster: Cluster) {
  return String(cluster.dock_number ?? cluster.DockNumber ?? "");
}

export function clusterBacklogs(cluster: Cluster) {
  return Number(cluster.backlogs ?? cluster.Backlogs ?? 0);
}
