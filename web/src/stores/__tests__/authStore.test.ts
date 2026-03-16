import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { authStore, AuthStore } from '../authStore'

// Mock localStorage
const localStorageMock = {
  getItem: vi.fn(),
  setItem: vi.fn(),
  removeItem: vi.fn(),
  clear: vi.fn(),
}
Object.defineProperty(window, 'localStorage', {
  value: localStorageMock,
})

describe('authStore', () => {
  beforeEach(() => {
    // Reset store state before each test
    authStore.setState({
      isAuthenticated: false,
      user: null,
      csrfToken: null,
    })
    vi.clearAllMocks()
  })

  afterEach(() => {
    // Clear localStorage mock
    localStorageMock.clear()
  })

  describe('initial state', () => {
    it('should have initial state with isAuthenticated false', () => {
      const state = authStore.getState()
      expect(state.isAuthenticated).toBe(false)
      expect(state.user).toBeNull()
      expect(state.csrfToken).toBeNull()
    })
  })

  describe('setUser', () => {
    it('should set user and update isAuthenticated', () => {
      const mockUser = {
        id: '550e8400-e29b-41d4-a716-446655440000',
        email: 'test@example.com',
        nickname: 'Test User',
        avatar_url: 'https://example.com/avatar.jpg',
        auth_provider: 'email' as const,
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
      }

      authStore.getState().setUser(mockUser, 'csrf-token-123')

      const state = authStore.getState()
      expect(state.user).toEqual(mockUser)
      expect(state.csrfToken).toBe('csrf-token-123')
      expect(state.isAuthenticated).toBe(true)
    })

    it('should store CSRF token in localStorage', () => {
      const mockUser = {
        id: '550e8400-e29b-41d4-a716-446655440000',
        email: 'test@example.com',
        nickname: null,
        avatar_url: null,
        auth_provider: 'email' as const,
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
      }

      authStore.getState().setUser(mockUser, 'csrf-token-456')

      expect(localStorageMock.setItem).toHaveBeenCalledWith('csrf_token', 'csrf-token-456')
    })
  })

  describe('clearUser', () => {
    it('should clear user and update isAuthenticated', () => {
      // First set a user
      const mockUser = {
        id: '550e8400-e29b-41d4-a716-446655440000',
        email: 'test@example.com',
        nickname: null,
        avatar_url: null,
        auth_provider: 'email' as const,
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
      }

      authStore.getState().setUser(mockUser, 'csrf-token')

      // Verify user is set
      expect(authStore.getState().isAuthenticated).toBe(true)

      // Clear user
      authStore.getState().clearUser()

      const state = authStore.getState()
      expect(state.user).toBeNull()
      expect(state.csrfToken).toBeNull()
      expect(state.isAuthenticated).toBe(false)
    })

    it('should remove CSRF token from localStorage', () => {
      const mockUser = {
        id: '550e8400-e29b-41d4-a716-446655440000',
        email: 'test@example.com',
        nickname: null,
        avatar_url: null,
        auth_provider: 'email' as const,
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
      }

      authStore.getState().setUser(mockUser, 'csrf-token')
      authStore.getState().clearUser()

      expect(localStorageMock.removeItem).toHaveBeenCalledWith('csrf_token')
    })
  })

  describe('initializeFromStorage', () => {
    it('should restore CSRF token from localStorage', () => {
      localStorageMock.getItem.mockReturnValue('stored-csrf-token')

      authStore.getState().initializeFromStorage()

      expect(authStore.getState().csrfToken).toBe('stored-csrf-token')
    })

    it('should handle missing CSRF token in localStorage', () => {
      localStorageMock.getItem.mockReturnValue(null)

      authStore.getState().initializeFromStorage()

      expect(authStore.getState().csrfToken).toBeNull()
      expect(authStore.getState().isAuthenticated).toBe(false)
    })
  })

  describe('getCsrfToken', () => {
    it('should return the current CSRF token', () => {
      authStore.getState().setUser(
        {
          id: '550e8400-e29b-41d4-a716-446655440000',
          email: 'test@example.com',
          nickname: null,
          avatar_url: null,
          auth_provider: 'email' as const,
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z',
        },
        'my-csrf-token'
      )

      expect(authStore.getState().getCsrfToken()).toBe('my-csrf-token')
    })

    it('should return null when no CSRF token is set', () => {
      expect(authStore.getState().getCsrfToken()).toBeNull()
    })
  })

  describe('store integration', () => {
    it('should be a Zustand store with correct structure', () => {
      const state = authStore.getState()

      expect(typeof state.setUser).toBe('function')
      expect(typeof state.clearUser).toBe('function')
      expect(typeof state.initializeFromStorage).toBe('function')
      expect(typeof state.getCsrfToken).toBe('function')
    })
  })
})
