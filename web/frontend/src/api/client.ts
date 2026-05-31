import { AppStats, AuthUser, Cluster, Request, RequestPayload, RequestsFilter, RequestsResponse, TrendData } from '@/types'

type OnAuthError = () => void
let onAuthError: OnAuthError = () => {}

export function setOnAuthError(cb: OnAuthError) {
  onAuthError = cb
}

async function apiFetch<T>(url: string, options?: RequestInit): Promise<T> {
  const res = await fetch(url, {
    ...options,
    headers: { 'Content-Type': 'application/json', ...options?.headers },
  })

  if (res.status === 401) {
    onAuthError()
    throw new Error('Session expired. Please sign in again.')
  }

  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    throw new Error(body.message || `Request failed (${res.status})`)
  }

  return res.json()
}

// ── Auth ─────────────────────────────────────────────────────

export async function login(loginType: string, email: string, opsId: string, password: string): Promise<AuthUser> {
  const res = await fetch('/api/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ login_type: loginType, email, ops_id: opsId, password }),
  })
  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    throw new Error(body.message || 'Invalid credentials')
  }
  return res.json()
}

export async function logout(): Promise<void> {
  await fetch('/api/logout', { method: 'POST' }).catch(() => {})
}

// ── Stats ─────────────────────────────────────────────────────

export function fetchStats(): Promise<AppStats> {
  return apiFetch('/api/stats')
}

// ── Trend ────────────────────────────────────────────────────

export function fetchTrend(): Promise<TrendData> {
  return apiFetch('/api/request-trend')
}

// ── Requests ─────────────────────────────────────────────────

export function fetchRequests(filter: RequestsFilter = {}): Promise<RequestsResponse> {
  const params = new URLSearchParams()
  if (filter.queue) params.set('queue', filter.queue)
  if (filter.status && filter.status !== 'ALL') params.set('status', filter.status)
  if (filter.search) params.set('search', filter.search)
  if (filter.date_from) params.set('date_from', filter.date_from)
  if (filter.date_to) params.set('date_to', filter.date_to)
  if (filter.page) params.set('page', String(filter.page))
  if (filter.page_size) params.set('page_size', String(filter.page_size))
  const qs = params.toString()
  return apiFetch(`/api/requests${qs ? '?' + qs : ''}`)
}

export function createRequest(payload: RequestPayload): Promise<Request> {
  return apiFetch('/api/requests', { method: 'POST', body: JSON.stringify(payload) })
}

export function editRequest(id: number, payload: RequestPayload): Promise<Request> {
  return apiFetch(`/api/requests/${id}/edit`, { method: 'POST', body: JSON.stringify(payload) })
}

export function actionRequest(id: number, action: string, payload: RequestPayload = {}): Promise<Request> {
  return apiFetch(`/api/requests/${id}/${action}`, { method: 'POST', body: JSON.stringify(payload) })
}

// ── Clusters ─────────────────────────────────────────────────

export function fetchClusters(): Promise<Cluster[]> {
  return apiFetch('/api/clusters')
}

// ── Users ────────────────────────────────────────────────────

export function createUser(payload: {
  name: string; role: string; email?: string; ops_id?: string; password?: string
}): Promise<unknown> {
  return apiFetch('/api/users', { method: 'POST', body: JSON.stringify(payload) })
}
