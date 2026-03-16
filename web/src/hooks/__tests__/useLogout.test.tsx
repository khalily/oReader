import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import { useAuth } from '../useAuth'

// Mock localStorage
const localStorageMock = {
  getItem: vi.fn(() => 'mock-csrf-token'),
  setItem: vi.fn(),
  removeItem: vi.fn(),
  clear: vi.fn(),
}
Object.defineProperty(window, 'localStorage', {
  value: localStorageMock,
})

const API_BASE_URL = '/api/v1'

const handlers = [
  // Logout
  http.post(`${API_BASE_URL}/auth/logout`, () => {
    return HttpResponse.json({ success: true })
  }),
]

const server = setupServer(...handlers)

describe('useLogout', () => {
  let queryClient: QueryClient

  beforeAll(() => {
    server.listen()
  })

  beforeEach(() => {
    queryClient = new QueryClient({
      defaultOptions: {
        mutations: {
          retry: false,
        },
      },
    })
  })

  afterEach(() => {
    server.resetHandlers()
    queryClient.clear()
    vi.clearAllMocks()
  })

  afterAll(() => {
    server.close()
  })

  const wrapper = ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>
      {children}
    </QueryClientProvider>
  )

  it('should successfully logout', async () => {
    const { result } = renderHook(() => useAuth().useLogout(), { wrapper })

    result.current.mutate()

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(result.current.data).toEqual({ success: true })
  })

  it('should call authStore clearUser on successful logout', async () => {
    const mockClearUser = vi.fn()

    // Mock authStore
    vi.doMock('../../stores/authStore', () => ({
      authStore: {
        getState: vi.fn(() => ({
          clearUser: mockClearUser,
        })),
      },
    }))

    const { result } = renderHook(() => useAuth().useLogout(), { wrapper })

    result.current.mutate()

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })
  })

  it('should handle logout error', async () => {
    server.use(
      http.post(`${API_BASE_URL}/auth/logout`, () => {
        return HttpResponse.json(
          { error: { code: 'INTERNAL_ERROR', message: 'Logout failed' } },
          { status: 500 }
        )
      })
    )

    const { result } = renderHook(() => useAuth().useLogout(), { wrapper })

    result.current.mutate()

    await waitFor(() => {
      expect(result.current.isError).toBe(true)
    })

    expect(result.current.error).toBeDefined()
  })

  it('should have loading state during logout', async () => {
    const { result } = renderHook(() => useAuth().useLogout(), { wrapper })

    // Initially idle
    expect(result.current.isPending).toBe(false)

    result.current.mutate()

    // Loading state
    expect(result.current.isPending).toBe(true)

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })
  })

  it('should remove CSRF token from localStorage on logout', async () => {
    const { result } = renderHook(() => useAuth().useLogout(), { wrapper })

    result.current.mutate()

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    // Verify localStorage was cleared
    expect(localStorageMock.removeItem).toHaveBeenCalledWith('csrf_token')
  })
})
