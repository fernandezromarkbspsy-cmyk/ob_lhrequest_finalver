export interface AuthUser {
  id: number
  name: string
  role: UserRole
  email?: string
  ops_id?: string
  is_fte: boolean
  redirect?: string
}

export type UserRole =
  | 'fte_ops' | 'fte_mm' | 'ops_pic' | 'dock_officer'
  | 'doc_officer' | 'data_team' | 'admin'

export interface AppStats {
  total_today: number
  pending_ops: number
  pending_mm: number
  for_docking: number
  confirmed_trucks: number
  rejected: number
}

export interface Request {
  id: number
  request_timestamp: string
  request_date: string
  cluster: string
  region: string
  dock_no: string
  backlogs: number
  truck_size: string
  truck_type: string
  plate_number: string
  driver_id: string
  linehaul_trip_no: string
  docking_time: string
  status: RequestStatus
  status_label: string
  ob_fte: string
  ob_ops_pic: string
  midmile_fte: string
  remarks: string
}

export type RequestStatus =
  | 'PENDING_OPS' | 'PENDING_MM' | 'ASSIGNED'
  | 'FOR_DOCKING' | 'DOCKED' | 'CONFIRMED'
  | 'CANCELED' | 'REJECTED'

export type RequestAction =
  | 'approve' | 'reject' | 'assign' | 'for-docking'
  | 'dock' | 'confirm' | 'cancel' | 'edit'

export interface Cluster {
  id: number
  cluster: string
  region: string
  dock_no: string
  backlogs: number
}

export interface TrendPoint {
  label: string
  count: number
}

export interface TrendData {
  start: string
  end: string
  period_label: string
  points: TrendPoint[]
}

export interface RequestsResponse {
  requests: Request[]
  count: number
  total: number
  page: number
  page_size: number
  has_more: boolean
}

export interface RequestsFilter {
  queue?: string
  status?: string
  search?: string
  date_from?: string
  date_to?: string
  page?: number
  page_size?: number
}

export interface RequestPayload {
  cluster_id?: number
  cluster?: string
  region?: string
  dock_no?: string
  backlogs?: number
  truck_size?: string
  truck_type?: string
  ob_ops_pic?: string
  ob_fte?: string
  midmile_fte?: string
  plate_number?: string
  driver_id?: string
  linehaul_trip_no?: string
  docking_time?: string
  remarks?: string
}
