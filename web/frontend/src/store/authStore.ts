import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { AuthUser } from '@/types'
import { logout as apiLogout } from '@/api/client'

interface AuthState {
  user: AuthUser | null
  setUser: (user: AuthUser | null) => void
  logout: () => Promise<void>
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      user: null,
      setUser: (user) => set({ user }),
      logout: async () => {
        await apiLogout()
        set({ user: null })
      },
    }),
    {
      name: 'soc5_user',
      partialize: (state) => ({
        user: state.user
          ? { name: state.user.name, role: state.user.role, is_fte: state.user.is_fte } as AuthUser
          : null,
      }),
    }
  )
)
