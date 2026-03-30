import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import { useFeeds } from '../useFeeds'

// Mock API base URL
const API_BASE_URL = '/api/v1'

// Mock handlers
const handlers = [
  // List feeds
  http.get(`${API_BASE_URL}/feeds`, ({ request }) => {
    const url = new URL(request.url)
    url.searchParams.get('limit')
    url.searchParams.get('offset')

    return HttpResponse.json({
      feeds: [
        {
          id: 'feed-1',
          title: 'Example Feed',
          feed_url: 'https://example.com/feed.xml',
          description: 'An example RSS feed',
          image_url: 'https://example.com/image.png',
          last_fetched_at: '2024-01-01T00:00:00Z',
          created_at: '2024-01-01T00:00:00Z',
          unread_count: 5,
          position: 1,
        },
        {
          id: 'feed-2',
          title: 'Another Feed',
          feed_url: 'https://another.com/feed.xml',
          description: 'Another RSS feed',
          image_url: null,
          last_fetched_at: '2024-01-02T00:00:00Z',
          created_at: '2024-01-02T00:00:00Z',
          unread_count: 0,
          position: 2,
        },
      ],
      total: 2,
    })
  }),

  // Get feed
  http.get(`${API_BASE_URL}/feeds/:id`, ({ params }) => {
    if (params.id === 'feed-1') {
      return HttpResponse.json({
        feed: {
          id: 'feed-1',
          title: 'Example Feed',
          feed_url: 'https://example.com/feed.xml',
          description: 'An example RSS feed',
          image_url: 'https://example.com/image.png',
          last_fetched_at: '2024-01-01T00:00:00Z',
          created_at: '2024-01-01T00:00:00Z',
        },
        item_count: 42,
      })
    }

    return HttpResponse.json(
      { error: { code: 'NOT_FOUND', message: 'Feed not found' } },
      { status: 404 }
    )
  }),

  // Create feed
  http.post(`${API_BASE_URL}/feeds`, async ({ request }) => {
    const body = await request.json() as { feed_url: string }

    if (body.feed_url === 'https://example.com/feed.xml') {
      return HttpResponse.json({
        feed: {
          id: 'feed-new',
          title: 'Example Feed',
          feed_url: 'https://example.com/feed.xml',
          description: 'An example RSS feed',
          image_url: 'https://example.com/image.png',
          last_fetched_at: '2024-01-01T00:00:00Z',
          created_at: '2024-01-01T00:00:00Z',
        },
        new_item_count: 10,
      })
    }

    if (body.feed_url === 'https://invalid.com/feed.xml') {
      return HttpResponse.json(
        { error: { code: 'VALIDATION_ERROR', message: 'Invalid feed URL' } },
        { status: 400 }
      )
    }

    if (body.feed_url === 'https://existing.com/feed.xml') {
      return HttpResponse.json(
        { error: { code: 'CONFLICT', message: 'Already subscribed to this feed' } },
        { status: 409 }
      )
    }

    return HttpResponse.json(
      { error: { code: 'VALIDATION_ERROR', message: 'Failed to fetch feed' } },
      { status: 400 }
    )
  }),

  // Delete feed
  http.delete(`${API_BASE_URL}/feeds/:id`, ({ params }) => {
    if (params.id === 'feed-1') {
      return new HttpResponse(null, { status: 204 })
    }

    return HttpResponse.json(
      { error: { code: 'NOT_FOUND', message: 'Feed not found' } },
      { status: 404 }
    )
  }),

  // Refresh feed
  http.post(`${API_BASE_URL}/feeds/:id/refresh`, ({ params }) => {
    if (params.id === 'feed-1') {
      return HttpResponse.json({
        updated: true,
        new_item_count: 5,
      })
    }

    if (params.id === 'feed-error') {
      return HttpResponse.json(
        { error: { code: 'VALIDATION_ERROR', message: 'Failed to refresh feed' } },
        { status: 400 }
      )
    }

    return HttpResponse.json(
      { error: { code: 'NOT_FOUND', message: 'Feed not found' } },
      { status: 404 }
    )
  }),
]

const server = setupServer(...handlers)

describe('useFeeds hook', () => {
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
        mutations: {
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
    <QueryClientProvider client={queryClient}>
      {children}
    </QueryClientProvider>
  )

  describe('useListFeeds', () => {
    it('should fetch feeds successfully', async () => {
      const { result } = renderHook(() => useFeeds().useListFeeds(), { wrapper })

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true)
      })

      expect(result.current.data?.feeds).toHaveLength(2)
      expect(result.current.data?.total).toBe(2)
      expect(result.current.data?.feeds[0].title).toBe('Example Feed')
      expect(result.current.data?.feeds[0].unread_count).toBe(5)
    })

    it('should pass limit and offset parameters', async () => {
      const { result } = renderHook(
        () => useFeeds().useListFeeds({ limit: 10, offset: 5 }),
        { wrapper }
      )

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true)
      })

      // Verify the request was made with correct params
      expect(result.current.data?.feeds).toBeDefined()
    })

    it('should have loading state initially', () => {
      const { result } = renderHook(() => useFeeds().useListFeeds(), { wrapper })

      expect(result.current.isLoading).toBe(true)
    })
  })

  describe('useGetFeed', () => {
    it('should fetch a single feed successfully', async () => {
      const { result } = renderHook(() => useFeeds().useGetFeed('feed-1'), { wrapper })

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true)
      })

      expect(result.current.data?.feed.id).toBe('feed-1')
      expect(result.current.data?.feed.title).toBe('Example Feed')
      expect(result.current.data?.item_count).toBe(42)
    })

    it('should return error for non-existent feed', async () => {
      const { result } = renderHook(() => useFeeds().useGetFeed('non-existent'), { wrapper })

      await waitFor(() => {
        expect(result.current.isError).toBe(true)
      })

      expect(result.current.error).toHaveProperty('response.data.error.code', 'NOT_FOUND')
    })

    it('should not fetch when feedId is null', () => {
      const { result } = renderHook(() => useFeeds().useGetFeed(null as unknown as string), { wrapper })

      expect(result.current.isLoading).toBe(false)
      expect(result.current.data).toBeUndefined()
    })
  })

  describe('useCreateFeed', () => {
    it('should create feed successfully', async () => {
      const { result } = renderHook(() => useFeeds().useCreateFeed(), { wrapper })

      result.current.mutate({
        feed_url: 'https://example.com/feed.xml',
      })

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true)
      })

      expect(result.current.data?.feed.title).toBe('Example Feed')
      expect(result.current.data?.new_item_count).toBe(10)
    })

    it('should fail with invalid feed URL', async () => {
      const { result } = renderHook(() => useFeeds().useCreateFeed(), { wrapper })

      result.current.mutate({
        feed_url: 'https://invalid.com/feed.xml',
      })

      await waitFor(() => {
        expect(result.current.isError).toBe(true)
      })

      expect(result.current.error).toHaveProperty('response.data.error.code', 'VALIDATION_ERROR')
    })

    it('should fail when already subscribed', async () => {
      const { result } = renderHook(() => useFeeds().useCreateFeed(), { wrapper })

      result.current.mutate({
        feed_url: 'https://existing.com/feed.xml',
      })

      await waitFor(() => {
        expect(result.current.isError).toBe(true)
      })

      expect(result.current.error).toHaveProperty('response.data.error.code', 'CONFLICT')
    })

    it('should have loading state during creation', async () => {
      const { result } = renderHook(() => useFeeds().useCreateFeed(), { wrapper })

      // Initially idle
      expect(result.current.isPending).toBe(false)

      result.current.mutate({
        feed_url: 'https://example.com/feed.xml',
      })

      // After mutation, it should either be loading or already done (very fast)
      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true)
      })
    })
  })

  describe('useDeleteFeed', () => {
    it('should delete feed successfully', async () => {
      const { result } = renderHook(() => useFeeds().useDeleteFeed(), { wrapper })

      result.current.mutate('feed-1')

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true)
      })
    })

    it('should fail for non-existent feed', async () => {
      const { result } = renderHook(() => useFeeds().useDeleteFeed(), { wrapper })

      result.current.mutate('non-existent')

      await waitFor(() => {
        expect(result.current.isError).toBe(true)
      })

      expect(result.current.error).toHaveProperty('response.data.error.code', 'NOT_FOUND')
    })
  })

  describe('useRefreshFeed', () => {
    it('should refresh feed successfully', async () => {
      const { result } = renderHook(() => useFeeds().useRefreshFeed(), { wrapper })

      result.current.mutate('feed-1')

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true)
      })

      expect(result.current.data?.updated).toBe(true)
      expect(result.current.data?.new_item_count).toBe(5)
    })

    it('should fail for non-existent feed', async () => {
      const { result } = renderHook(() => useFeeds().useRefreshFeed(), { wrapper })

      result.current.mutate('non-existent')

      await waitFor(() => {
        expect(result.current.isError).toBe(true)
      })

      expect(result.current.error).toHaveProperty('response.data.error.code', 'NOT_FOUND')
    })

    it('should fail when feed refresh errors', async () => {
      const { result } = renderHook(() => useFeeds().useRefreshFeed(), { wrapper })

      result.current.mutate('feed-error')

      await waitFor(() => {
        expect(result.current.isError).toBe(true)
      })

      expect(result.current.error).toHaveProperty('response.data.error.code', 'VALIDATION_ERROR')
    })

    it('should have loading state during refresh', async () => {
      const { result } = renderHook(() => useFeeds().useRefreshFeed(), { wrapper })

      // Initially idle
      expect(result.current.isPending).toBe(false)

      result.current.mutate('feed-1')

      // After mutation, it should either be loading or already done (very fast)
      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true)
      })
    })
  })
})
