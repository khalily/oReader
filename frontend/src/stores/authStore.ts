import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { User } from '@/types'

interface AuthStore {
  isAuthenticated: boolean
  user: User | null
  csrfToken: string | null

  // Actions
  setUser: (user: User, csrfToken: string) => void
  clearUser: () => void
  initializeFromStorage: () => void
  getCsrfToken: () => string | null
}

export const useAuthStore = create<AuthStore>()(
  persist(
    (set, get) => ({
      isAuthenticated: false,
      user: null,
      csrfToken: null,

      setUser: (user, csrfToken) => {
        set({ isAuthenticated: true, user, csrfToken })
      },

      clearUser: () => {
        set({ isAuthenticated: false, user: null, csrfToken: null })
      },

      // This is now handled automatically by persist middleware
      // but we keep it for backward compatibility
      initializeFromStorage: () => {
        // No-op: persist middleware handles this automatically
        // The state is already hydrated from localStorage before any render
      },

      getCsrfToken: () => get().csrfToken,
    }),
    {
      name: 'auth-storage', // unique name for localStorage key
      partialize: (state) => ({
        // Only persist these fields
        isAuthenticated: state.isAuthenticated,
        user: state.user,
        csrfToken: state.csrfToken,
      }),
    }
  )
)

// Export a named export for consistency with the tests
export const authStore = {
  getState: useAuthStore.getState,
  setState: useAuthStore.setState,
}
