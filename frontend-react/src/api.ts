export type Stats = {
  total_today: number;
  pending_ops: number;
  pending_mm: number;
  for_docking: number;
  confirmed_trucks: number;
  rejected: number;
};

export type RequestRow = {
  id: number;
  request_timestamp: string;
  cluster: string;
  region: string;
  dock_no: string;
  truck_size: string;
  truck_type: string;
  plate_number: string;
  status: string;
  status_label: string;
};

const apiBase = import.meta.env.VITE_API_BASE ?? "";

async function getJSON<T>(path: string): Promise<T> {
  const response = await fetch(`${apiBase}${path}`, { credentials: "include" });
  if (!response.ok) {
    const body = await response.json().catch(() => ({}));
    throw new Error(body.error ?? "Request failed");
  }
  return response.json() as Promise<T>;
}

export function fetchStats() {
  return getJSON<Stats>("/api/stats");
}

export async function fetchRequests() {
  const body = await getJSON<{ requests: RequestRow[] }>("/api/requests");
  return body.requests;
}
