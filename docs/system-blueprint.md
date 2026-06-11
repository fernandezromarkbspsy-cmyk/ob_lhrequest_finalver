# SOC 5 Outbound — System Blueprint

## 1. System Purpose

SOC 5 Outbound is a real-time linehaul request management system for Shopee Sorting Facility Dispatch.

The system replaces the manual Google Sheets workflow with a role-aware dashboard for creating, approving, assigning, docking, confirming, and tracking truck requests through their full lifecycle.

---

Database:
- Supabase PostgreSQL

Realtime:
- Supabase Realtime
- Optional backend event logs

Authentication:
- Backend login
- Password hash
- JWT stored in HTTP-only cookie

---

## 3. User Roles

### Ops PIC

Type:
- Backroom

Main responsibility:
- Create linehaul truck requests.

Modules:
- Dashboard
- LH Request

Permissions:
- Create own linehaul request
- View own requests
- View request status updates
- Cancel own pending requests only

---

### FTE Ops

Type:
- FTE

Main responsibility:
- Review, approve, edit, cancel, and create linehaul requests.

Modules:
- Dashboard
- LH Request
- KPI
- User Management

Permissions:
- Create request
- View pending requests
- View rejected requests from FTE MM
- Approve request
- Edit request
- Cancel request
- Bulk approve requests
- Manage users

---

### FTE MM

Type:
- FTE

Main responsibility:
- Assign trucks and reject requests if needed.

Modules:
- Dashboard
- Truck Request
- User Management

Permissions:
- View approved requests
- Assign truck
- Encode plate number
- Reject request back to FTE Ops
- Manage users

---

### Doc Officer

Type:
- Backroom

Main responsibility:
- input Drivers ID and LHTrip once the truck is already at the dockings

Modules:
- Dashboard
- Truck Request

Permissions:
- View assigned / for docking requests
- Mark truck as docked
- Encode Driver ID
- Encode LH Trip Number
- Confirm request

---

## 4. Request Lifecycle

### Normal Flow

1. Ops PIC creates truck request.
2. Request status becomes PENDING.
3. FTE Ops reviews the request.
4. FTE Ops approves, edits, or cancels the request.
5. Once approved, request is routed to FTE MM.
6. FTE MM assigns truck plate number.
7. Once assigned, request is routed to Doc Officer.
8. Doc Officer encodes Driver ID and LH Trip Number.
9. Doc Officer actions the truck as docked.
10. Request status becomes DOCKED.
11. Printable image becomes available (this is already applied no need to revise)

### Rejection Flow

1. FTE MM rejects the approved request.
2. Request status becomes REJECTED_BY_MM.
3. Request goes back to FTE Ops.
4. FTE Ops may edit, cancel, or approve again.

---

## 5. Request Statuses

| Status | Description |
|---|---|
| PENDING | Created request waiting for FTE Ops checking |
| APPROVED | Approved by FTE Ops and routed to FTE MM |
| CANCELLED | Cancelled request |
| REJECTED_BY_MM | Rejected by FTE MM and returned to FTE Ops |
| ASSIGNED | Truck plate number assigned by FTE MM |
| FOR_DOCKING | Routed to Doc Officer for docking confirmation |
| DOCKED | Truck marked as docked |
| CONFIRMED | Final confirmed request with Driver ID and LH Trip Number |

---

## 6. Event-Driven Actions

Every major action must create an event record.

Events:
- REQUEST_CREATED
- REQUEST_APPROVED
- REQUEST_EDITED
- REQUEST_CANCELLED
- REQUEST_REJECTED_BY_MM
- TRUCK_ASSIGNED
- TRUCK_FOR_DOCKING
- TRUCK_DOCKED
- REQUEST_CONFIRMED
- USER_CREATED
- USER_UPDATED
- USER_DISABLED

Purpose:
- Realtime routing
- Audit trail
- KPI tracking
- Notification sound trigger
- Debugging and traceability

---

## 7. Alert Notification Rules

Sound alert should trigger when:

| Event | Alert Receiver |
|---|---|
| REQUEST_CREATED | FTE Ops |
| REQUEST_APPROVED | FTE MM |
| REQUEST_REJECTED_BY_MM | FTE Ops |
| TRUCK_ASSIGNED | Doc Officer |
| TRUCK_FOR_DOCKING | Doc Officer |
| REQUEST_CONFIRMED | FTE Ops and FTE MM |

Frontend behavior:
1. Listen for realtime changes.
2. Check current user role.
3. Show toast notification.
4. Play alert sound.
5. Refresh affected table.

---

## 8. Main Database Tables

Initial tables:
- users
- requests
- request_events
- notifications

Future tables:
- kpi_daily_summary
- system_settings

---

## 9. User Fields

users table:

- id
- name
- role
- ops_id
- email
- is_fte
- is_active
- created_at
- password_hash

Role values:
- ops_pic
- fte_ops
- fte_mm
- dock_officer
- doc_officer

Lookup rules:
- FTE users log in by email.
- Backroom users log in by ops_id.
- role determines whether the user is FTE or Backroom.

clusters table:

- id
- cluster_name
- hub_name
- region
- dock_number
- backlogs
- backlogs_ts

---

## 10. Request Fields

requests table:

- id
- request_timestamp
- cluster
- region
- dock_no
- backlogs
- backlogs_timestamp
- ob_fte
- ob_ops_pic
- midmile_fte
- truck_size
- truck_type
- plate_number
- provide_time
- linehaul_trip_no
- docked_time
- status
- rejection_remarks
- approved_at
- approved_at
- rejected_at
- confirmed_at
- driver_id
- updated_at

Truck size values:
- 4W
- 6W
- 10W
- 6WF

Truck type values:
- WETLEASE
- DRYLEASE

---

## 11. Role-Based Page Access

| Page | Ops PIC | FTE Ops | FTE MM | Doc Officer |
|---|---|---|---|---|
| Dashboard | Yes | Yes | Yes | Yes |
| LH Request | Yes | Yes | No | No |
| Truck Request | No | No | Yes | No |
| Docking Confirmation | No | No | No | Yes |
| KPI | No | Yes | No | No |
| User Management | No | Yes | Yes | No |

---

## 12. Business Rules

1. Only FTE users can add, update, disable, or remove users.
2. Ops PIC can create requests but cannot approve requests.
3. FTE Ops can approve, edit, cancel, and create requests.
4. FTE Ops can approve multiple pending requests at once.
5. FTE MM can only assign truck details or reject requests.
6. FTE MM rejection must include rejection remarks.
7. Doc Officer can only act on assigned or for docking requests.
8. Driver ID and LH Trip Number are required before final confirmation.
9. Every status change must create a request event.
10. Every routed request should trigger a notification.
11. All users must see updated statuses in realtime.
12. Disabled users cannot log in.
13. Passwords must never be stored as plain text.

---

## 13. API Groups

Auth:
- POST /api/auth/login
- POST /api/auth/logout
- GET /api/auth/me

Users:
- GET /api/users
- POST /api/users
- PUT /api/users/:id
- PATCH /api/users/:id/disable

Linehaul Requests:
- GET /api/requests
- POST /api/requests
- GET /api/requests/:id
- PUT /api/requests/:id
- POST /api/requests/:id/approve
- POST /api/requests/bulk-approve
- POST /api/requests/:id/cancel
- POST /api/requests/:id/reject-mm
- POST /api/requests/:id/assign-truck
- POST /api/requests/:id/mark-docked
- POST /api/requests/:id/confirm

Events:
- GET /api/requests/:id/events

Notifications:
- GET /api/notifications
- PATCH /api/notifications/:id/read

KPI:
- GET /api/kpi/daily
- GET /api/kpi/summary

---

## 14. Frontend Pages

Pages:
- Login
- Dashboard
- LH Request
- Truck Request
- Docking Confirmation
- KPI
- User Management

Reusable Components:
- Sidebar
- Topbar
- StatusBadge
- RequestTable
- ToastNotification
- AlertSound
- RoleGuard
- LoadingState
- EmptyState

---

## 15. Development Rule

This project must be built phase by phase.

No random coding.
No skipping database design.
No exposing Supabase service keys to frontend.
No plain text passwords.
No direct frontend write access to protected database operations.
