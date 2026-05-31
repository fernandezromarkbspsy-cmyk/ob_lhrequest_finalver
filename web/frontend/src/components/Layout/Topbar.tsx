import { Sun, Moon } from 'lucide-react'
import { useUIStore } from '@/store/uiStore'
import { useAuthStore } from '@/store/authStore'

export default function Topbar() {
  const { theme, toggleTheme } = useUIStore()
  const user = useAuthStore((s) => s.user)

  return (
    <header className="topbar">
      <div className="topbar-left">
        <span className="topbar-greeting">
          {user ? `Hello, ${user.name.split(' ')[0]}` : ''}
        </span>
      </div>
      <div className="topbar-right">
        <button
          className="btn-icon btn-icon--ghost"
          onClick={toggleTheme}
          title={theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}
        >
          {theme === 'dark' ? <Sun size={18} /> : <Moon size={18} />}
        </button>
      </div>
    </header>
  )
}
