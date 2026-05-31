import { NavLink, useNavigate } from 'react-router-dom'
import {
  LayoutDashboard, Truck, PackageOpen, Warehouse,
  Settings, ChevronLeft, ChevronRight, LogOut
} from 'lucide-react'
import { useAuthStore } from '@/store/authStore'
import { useUIStore } from '@/store/uiStore'
import { useStats } from '@/hooks/useStats'
import { UserRole } from '@/types'

interface NavItem {
  label: string
  path: string
  icon: React.ReactNode
  badge?: number
  roles: UserRole[]
}

function roleLabel(role: UserRole): string {
  const map: Record<UserRole, string> = {
    fte_ops: 'FTE — Outbound',
    fte_mm: 'FTE — Midmile',
    ops_pic: 'OB OPS PIC',
    dock_officer: 'Dock Officer',
    doc_officer: 'Doc Officer',
    data_team: 'Data Team',
    admin: 'Administrator',
  }
  return map[role] ?? role
}

export default function Sidebar() {
  const user = useAuthStore((s) => s.user)
  const logout = useAuthStore((s) => s.logout)
  const { sidebarCollapsed, toggleSidebar } = useUIStore()
  const { data: stats } = useStats()
  const navigate = useNavigate()

  const handleLogout = async () => {
    await logout()
    navigate('/', { replace: true })
  }

  const allRoles: UserRole[] = ['fte_ops', 'fte_mm', 'ops_pic', 'dock_officer', 'doc_officer', 'data_team', 'admin']

  const navItems: NavItem[] = [
    {
      label: 'Dashboard',
      path: '/dashboard',
      icon: <LayoutDashboard size={18} />,
      roles: allRoles,
    },
    {
      label: 'LH Request',
      path: '/outbound/lh-request',
      icon: <Truck size={18} />,
      badge: stats?.pending_ops,
      roles: ['fte_ops', 'ops_pic', 'data_team', 'admin'],
    },
    {
      label: 'Truck Request',
      path: '/midmile/truck-request',
      icon: <PackageOpen size={18} />,
      badge: stats?.pending_mm,
      roles: ['fte_mm', 'data_team', 'admin'],
    },
    {
      label: 'Dock Officer',
      path: '/dock/officer',
      icon: <Warehouse size={18} />,
      badge: stats?.for_docking,
      roles: ['dock_officer', 'doc_officer', 'data_team', 'admin'],
    },
    {
      label: 'Settings',
      path: '/settings',
      icon: <Settings size={18} />,
      roles: ['fte_ops', 'fte_mm', 'admin'],
    },
  ]

  const visibleItems = user
    ? navItems.filter((item) => item.roles.includes(user.role))
    : []

  const initials = user?.name
    ? user.name.split(' ').map((n) => n[0]).join('').toUpperCase().slice(0, 2)
    : '?'

  return (
    <aside className={`sidebar${sidebarCollapsed ? ' sidebar--collapsed' : ''}`}>
      <div className="brand">
        <div className="brand-mark">
          <img src="/truck_label/icon.png" alt="SOC5" onError={(e) => {
            (e.target as HTMLImageElement).style.display = 'none'
          }} />
        </div>
        {!sidebarCollapsed && (
          <div>
            <strong>SOC 5</strong>
            <span className="brand-sub">OUTBOUND</span>
          </div>
        )}
      </div>

      <nav className="sidebar-nav">
        {visibleItems.map((item) => (
          <NavLink
            key={item.path}
            to={item.path}
            className={({ isActive }) => `nav-item${isActive ? ' nav-item--active' : ''}`}
            title={sidebarCollapsed ? item.label : undefined}
          >
            <span className="nav-icon">{item.icon}</span>
            {!sidebarCollapsed && <span className="nav-label">{item.label}</span>}
            {!sidebarCollapsed && !!item.badge && (
              <span className="nav-badge">{item.badge > 99 ? '99+' : item.badge}</span>
            )}
            {sidebarCollapsed && !!item.badge && (
              <span className="nav-badge nav-badge--dot" />
            )}
          </NavLink>
        ))}
      </nav>

      <div className="sidebar-footer">
        {user && (
          <div className="sidebar-user">
            <div className="user-avatar">{initials}</div>
            {!sidebarCollapsed && (
              <div className="user-info">
                <strong>{user.name}</strong>
                <small>{roleLabel(user.role)}</small>
              </div>
            )}
            <button
              className="btn-icon btn-icon--ghost logout-btn"
              onClick={handleLogout}
              title="Sign out"
            >
              <LogOut size={16} />
            </button>
          </div>
        )}
        <button
          className="sidebar-collapse-btn"
          onClick={toggleSidebar}
          title={sidebarCollapsed ? 'Expand sidebar' : 'Collapse sidebar'}
        >
          {sidebarCollapsed ? <ChevronRight size={16} /> : <ChevronLeft size={16} />}
          {!sidebarCollapsed && <span>Collapse</span>}
        </button>
      </div>
    </aside>
  )
}
