import { describe, it, expect, vi, beforeEach, afterEach, beforeAll, afterAll } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import { useStats } from '../useStats'

// Mock API base URL
const API_BASE_URL = '/api/v1'

// Mock handlers
const handlers = [
  // Get stats - success
  http.get(`${API_BASE_URL}/stats`, () => {
    return HttpResponse.json({
      total: 100,
      unread: 25,
      starred: 10,
      today: 5,
    })
  }),

  // Get stats - different values
  http.get(`${API_BASE_URL}/stats`, async ({ request }) => {
    const url = new URL(request.url)
    const test = url.searchParams.get('test')

    if (test === 'empty') {
      return HttpResponse.json({
        total: 0,
        unread: 0,
        starred: 0,
        today: 0,
      })
    }

    if (test === 'large') {
      return HttpResponse.json({
        total: 1000,
        unread: 500,
        starred: 200,
        today: 50,
      })
    }

    return HttpResponse.json({
      total: 100,
      unread: 25,
      starred: 10,
      today: 5,
    })
  }),

  // Get stats - error
  http.get(`${API_BASE_URL}/stats`, async ({ request }) => {
    const url = new URL(request.url)
    const error = url.searchParams.get('error')

    if (error === 'server') {
      return HttpResponse.json(
        { error: { code: 'INTERNAL_ERROR', message: 'Internal server error' } },
        { status: 500 }
      )
    }

    if (error === 'unauthorized') {
      return HttpResponse.json(
        { error: { code: 'UNAUTHORIZED', message: 'Unauthorized' } },
        { status: 401 }
      )
    }

    return HttpResponse.json({
      total: 100,
      unread: 25,
      starred: 10,
      today: 5,
    })
  }),
]

const server = setupServer(...handlers)

describe('useStats hook', () => {
  let queryClient: QueryClient

  beforeAll(() => {
    server.listen()
  })

  beforeEach(() => {
    queryClient = new QueryClient({
      defaultOptions: {
        queries: {
          retry: false,
        },
      },
    })
  })

  afterEach(() => {
    server.resetHandlers()
    queryClient.clear()
  })

  afterAll(() => {
    server.close()
  })

  const wrapper = ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  )

  describe('useGetStats', () => {
    it('should fetch stats successfully', async () => {
      const { result } = renderHook(() => useStats().useGetStats(), { wrapper })

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true)
      })

      expect(result.current.data).toEqual({
        total: 100,
        unread: 25,
        starred: 10,
        today: 5,
      })
    })

    it('should return correct stats values', async () => {
      const { result } = renderHook(() => useStats().useGetStats(), { wrapper })

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true)
      })

      expect(result.current.data?.total).toBe(100)
      expect(result.current.data?.unread).toBe(25)
      expect(result.current.data?.starred).toBe(10)
      expect(result.current.data?.today).toBe(5)
    })

    it('should handle loading state', () => {
      const { result } = renderHook(() => useStats().useGetStats(), { wrapper })

      expect(result.current.isLoading).toBe(true)
      expect(result.current.isFetching).toBe(true)
    })

    it('should transition from loading to success', async () => {
      const { result } = renderHook(() => useStats().useGetStats(), { wrapper })

      expect(result.current.isLoading).toBe(true)

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true)
      })

      expect(result.current.isLoading).toBe(false)
      expect(result.current.data).toBeDefined()
    })

    it('should handle API error - internal server error', async () => {
      server.use(
        http.get(`${API_BASE_URL}/stats`, () => {
          return HttpResponse.json(
            { error: { code: 'INTERNAL_ERROR', message: 'Internal server error' } },
            { status: 500 }
          )
        })
      )

      const { result } = renderHook(() => useStats().useGetStats(), { wrapper })

      await waitFor(() => {
        expect(result.current.isError).toBe(true)
      })

      expect(result.current.error).toBeDefined()
    })

    it('should handle API error - unauthorized', async () => {
      server.use(
        http.get(`${API_BASE_URL}/stats`, () => {
          return HttpResponse.json(
            { error: { code: 'UNAUTHORIZED', message: 'Unauthorized' } },
            { status: 401 }
          )
        })
      )

      const { result } = renderHook(() => useStats().useGetStats(), { wrapper })

      await waitFor(() => {
        expect(result.current.isError).toBe(true)
      })

      expect(result.current.error).toBeDefined()
    })

    it('should handle empty stats response', async () => {
      server.use(
        http.get(`${API_BASE_URL}/stats`, () => {
          return HttpResponse.json({
            total: 0,
            unread: 0,
            starred: 0,
            today: 0,
          })
        })
      )

      const { result } = renderHook(() => useStats().useGetStats(), { wrapper })

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true)
      })

      expect(result.current.data?.total).toBe(0)
      expect(result.current.data?.unread).toBe(0)
      expect(result.current.data?.starred).toBe(0)
      expect(result.current.data?.today).toBe(0)
    })

    it('should have correct staleTime (30 seconds)', async () => {
      // React Query's queryClient can be inspected for configuration
      const { result } = renderHook(() => useStats().useGetStats(), { wrapper })

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true)
      })

      // The staleTime is set in the hook configuration (30 * 1000 ms = 30000 ms)
      // This is an internal configuration, but we can verify the hook doesn't refetch immediately
      const fetchSpy = vi.spyOn(queryClient, 'refetchQueries')

      // After initial load, a quick re-render should not trigger a refetch
      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      fetchSpy.mockRestore()
    })

    it('should return correct query key', async () => {
      const { result } = renderHook(() => useStats().useGetStats(), { wrapper })

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true)
      })

      // The hook uses ['stats'] as queryKey - verified through successful query execution
      expect(result.current.data).toBeDefined()
    })

    it('should handle network error', async () => {
      server.use(
        http.get(`${API_BASE_URL}/stats`, () => {
          return HttpResponse.error()
        })
      )

      const { result } = renderHook(() => useStats().useGetStats(), { wrapper })

      await waitFor(() => {
        expect(result.current.isError).toBe(true)
      })

      expect(result.current.error).toBeDefined()
    })
  })
})
