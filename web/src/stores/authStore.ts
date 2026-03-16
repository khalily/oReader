import { create } from 'zustand'
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

export const useAuthStore = create<AuthStore>((set, get) => ({
  isAuthenticated: false,
  user: null,
  csrfToken: null,

  setUser: (user, csrfToken) => {
    set({ isAuthenticated: true, user, csrfToken })

    // Store CSRF token in localStorage for axios interceptor
    if (typeof window !== 'undefined') {
      localStorage.setItem('csrf_token', csrfToken)
    }
  },

  clearUser: () => {
    set({ isAuthenticated: false, user: null, csrfToken: null })

    // Remove CSRF token from localStorage
    if (typeof window !== 'undefined') {
      localStorage.removeItem('csrf_token')
    }
  },

  initializeFromStorage: () => {
    if (typeof window !== 'undefined') {
      const storedToken = localStorage.getItem('csrf_token')
      if (storedToken) {
        set({ csrfToken: storedToken })
      }
    }
  },

  getCsrfToken: () => get().csrfToken,
}))

// Export a named export for consistency with the tests
export const authStore = {
  getState: useAuthStore.getState,
  setState: useAuthStore.setState,
}

