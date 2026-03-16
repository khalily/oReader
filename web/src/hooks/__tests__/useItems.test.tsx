import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import { useItems } from '../useItems'

// Mock API base URL
const API_BASE_URL = '/api/v1'

// Mock handlers
const handlers = [
  // List items
  http.get(`${API_BASE_URL}/items`, ({ request }) => {
    const url = new URL(request.url)
    const limit = url.searchParams.get('limit') || '20'
    const cursor = url.searchParams.get('cursor')
    const feedId = url.searchParams.get('feed_id')
    const starred = url.searchParams.get('starred')
    const read = url.searchParams.get('read')

    return HttpResponse.json({
      items: [
        {
          id: 'item-1',
          feed_id: 'feed-1',
          guid: 'guid-1',
          title: 'First Article',
          link: 'https://example.com/article1',
          description: 'Description 1',
          content: 'Content 1',
          pub_date: '2024-01-01T00:00:00Z',
          creator: 'Author 1',
          created_at: '2024-01-01T00:00:00Z',
          feed: {
            id: 'feed-1',
            title: 'Example Feed',
            feed_url: 'https://example.com/feed.xml',
            description: 'An example feed',
            image_url: null,
            last_fetched_at: '2024-01-01T00:00:00Z',
            created_at: '2024-01-01T00:00:00Z',
          },
          user_state: {
            item_id: 'item-1',
            is_read: false,
            is_starred: false,
            read_at: null,
          },
        },
        {
          id: 'item-2',
          feed_id: 'feed-1',
          guid: 'guid-2',
          title: 'Second Article',
          link: 'https://example.com/article2',
          description: 'Description 2',
          content: 'Content 2',
          pub_date: '2024-01-02T00:00:00Z',
          creator: 'Author 2',
          created_at: '2024-01-02T00:00:00Z',
          feed: {
            id: 'feed-1',
            title: 'Example Feed',
            feed_url: 'https://example.com/feed.xml',
            description: 'An example feed',
            image_url: null,
            last_fetched_at: '2024-01-01T00:00:00Z',
            created_at: '2024-01-01T00:00:00Z',
          },
          user_state: {
            item_id: 'item-2',
            is_read: true,
            is_starred: true,
            read_at: '2024-01-02T01:00:00Z',
          },
        },
      ],
      total: 2,
      has_more: false,
      next_cursor: null,
    })
  }),

  // List items with pagination
  http.get(`${API_BASE_URL}/items`, async ({ request }) => {
    const url = new URL(request.url)
    const cursor = url.searchParams.get('cursor')

    if (cursor === 'page1') {
      return HttpResponse.json({
        items: [
          {
            id: 'item-3',
            feed_id: 'feed-1',
            guid: 'guid-3',
            title: 'Third Article',
            link: 'https://example.com/article3',
            description: 'Description 3',
            content: null,
            pub_date: '2024-01-03T00:00:00Z',
            creator: null,
            created_at: '2024-01-03T00:00:00Z',
            feed: {
              id: 'feed-1',
              title: 'Example Feed',
              feed_url: 'https://example.com/feed.xml',
              description: 'An example feed',
              image_url: null,
              last_fetched_at: '2024-01-01T00:00:00Z',
              created_at: '2024-01-01T00:00:00Z',
            },
            user_state: null,
          },
        ],
        total: 3,
        has_more: false,
        next_cursor: null,
      })
    }

    return HttpResponse.json({
      items: [
        {
          id: 'item-1',
          feed_id: 'feed-1',
          guid: 'guid-1',
          title: 'First Article',
          link: 'https://example.com/article1',
          description: 'Description 1',
          content: 'Content 1',
          pub_date: '2024-01-01T00:00:00Z',
          creator: 'Author 1',
          created_at: '2024-01-01T00:00:00Z',
          feed: {
            id: 'feed-1',
            title: 'Example Feed',
            feed_url: 'https://example.com/feed.xml',
            description: 'An example feed',
            image_url: null,
            last_fetched_at: '2024-01-01T00:00:00Z',
            created_at: '2024-01-01T00:00:00Z',
          },
          user_state: {
            item_id: 'item-1',
            is_read: false,
            is_starred: false,
            read_at: null,
          },
        },
      ],
      total: 3,
      has_more: true,
      next_cursor: 'page1',
    })
  }),

  // Get item
  http.get(`${API_BASE_URL}/items/:id`, ({ params }) => {
    if (params.id === 'item-1') {
      return HttpResponse.json({
        item: {
          id: 'item-1',
          feed_id: 'feed-1',
          guid: 'guid-1',
          title: 'First Article',
          link: 'https://example.com/article1',
          description: 'Description 1',
          content: 'Content 1',
          pub_date: '2024-01-01T00:00:00Z',
          creator: 'Author 1',
          created_at: '2024-01-01T00:00:00Z',
          feed: {
            id: 'feed-1',
            title: 'Example Feed',
            feed_url: 'https://example.com/feed.xml',
            description: 'An example feed',
            image_url: null,
            last_fetched_at: '2024-01-01T00:00:00Z',
            created_at: '2024-01-01T00:00:00Z',
          },
          user_state: {
            item_id: 'item-1',
            is_read: false,
            is_starred: false,
            read_at: null,
          },
        },
      })
    }

    return HttpResponse.json(
      { error: { code: 'NOT_FOUND', message: 'Item not found' } },
      { status: 404 }
    )
  }),

  // Toggle star
  http.put(`${API_BASE_URL}/items/:id/star`, async ({ params, request }) => {
    if (params.id === 'item-1') {
      const body = await request.json() as { starred: boolean }

      return HttpResponse.json({
        item: {
          id: 'item-1',
          feed_id: 'feed-1',
          guid: 'guid-1',
          title: 'First Article',
          link: 'https://example.com/article1',
          description: 'Description 1',
          content: 'Content 1',
          pub_date: '2024-01-01T00:00:00Z',
          creator: 'Author 1',
          created_at: '2024-01-01T00:00:00Z',
          feed: {
            id: 'feed-1',
            title: 'Example Feed',
            feed_url: 'https://example.com/feed.xml',
            description: 'An example feed',
            image_url: null,
            last_fetched_at: '2024-01-01T00:00:00Z',
            created_at: '2024-01-01T00:00:00Z',
          },
          user_state: {
            item_id: 'item-1',
            is_read: false,
            is_starred: body.starred,
            read_at: null,
          },
        },
      })
    }

    return HttpResponse.json(
      { error: { code: 'NOT_FOUND', message: 'Item not found' } },
      { status: 404 }
    )
  }),

  // Toggle read
  http.put(`${API_BASE_URL}/items/:id/read`, async ({ params, request }) => {
    if (params.id === 'item-1') {
      const body = await request.json() as { read: boolean }

      return HttpResponse.json({
        item: {
          id: 'item-1',
          feed_id: 'feed-1',
          guid: 'guid-1',
          title: 'First Article',
          link: 'https://example.com/article1',
          description: 'Description 1',
          content: 'Content 1',
          pub_date: '2024-01-01T00:00:00Z',
          creator: 'Author 1',
          created_at: '2024-01-01T00:00:00Z',
          feed: {
            id: 'feed-1',
            title: 'Example Feed',
            feed_url: 'https://example.com/feed.xml',
            description: 'An example feed',
            image_url: null,
            last_fetched_at: '2024-01-01T00:00:00Z',
            created_at: '2024-01-01T00:00:00Z',
          },
          user_state: {
            item_id: 'item-1',
            is_read: body.read,
            is_starred: false,
            read_at: body.read ? '2024-01-01T01:00:00Z' : null,
          },
        },
      })
    }

    return HttpResponse.json(
      { error: { code: 'NOT_FOUND', message: 'Item not found' } },
      { status: 404 }
    )
  }),

  // Mark all read
  http.post(`${API_BASE_URL}/feeds/:id/read-all`, ({ params }) => {
    if (params.id === 'feed-1') {
      return HttpResponse.json({
        count: 5,
      })
    }

    return HttpResponse.json(
      { error: { code: 'NOT_FOUND', message: 'Feed not found' } },
      { status: 404 }
    )
  }),
]

const server = setupServer(...handlers)

describe('useItems hook', () => {
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

  describe('useListItems', () => {
    it('should fetch items successfully', async () => {
      const { result } = renderHook(() => useItems().useListItems(), { wrapper })

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true)
      })

      expect(result.current.data?.items).toHaveLength(2)
      expect(result.current.data?.total).toBe(2)
      expect(result.current.data?.has_more).toBe(false)
      expect(result.current.data?.items[0].title).toBe('First Article')
    })

    it('should filter by feed_id', async () => {
      const { result } = renderHook(
        () => useItems().useListItems({ feed_id: 'feed-1' }),
        { wrapper }
      )

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true)
      })

      expect(result.current.data?.items).toBeDefined()
    })

    it('should filter by starred status', async () => {
      const { result } = renderHook(
        () => useItems().useListItems({ starred: true }),
        { wrapper }
      )

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true)
      })

      expect(result.current.data?.items).toBeDefined()
    })

    it('should filter by read status', async () => {
      const { result } = renderHook(
        () => useItems().useListItems({ read: false }),
        { wrapper }
      )

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true)
      })

      expect(result.current.data?.items).toBeDefined()
    })

    it('should handle cursor-based pagination', async () => {
      const { result } = renderHook(
        () => useItems().useListItems({ cursor: 'page1' }),
        { wrapper }
      )

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true)
      })

      expect(result.current.data?.has_more).toBe(false)
    })

    it('should have loading state initially', () => {
      const { result } = renderHook(() => useItems().useListItems(), { wrapper })

      expect(result.current.isLoading).toBe(true)
    })
  })

  describe('useGetItem', () => {
    it('should fetch a single item successfully', async () => {
      const { result } = renderHook(() => useItems().useGetItem('item-1'), { wrapper })

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true)
      })

      expect(result.current.data?.item.id).toBe('item-1')
      expect(result.current.data?.item.title).toBe('First Article')
      expect(result.current.data?.item.user_state?.is_read).toBe(false)
    })

    it('should return error for non-existent item', async () => {
      const { result } = renderHook(() => useItems().useGetItem('non-existent'), { wrapper })

      await waitFor(() => {
        expect(result.current.isError).toBe(true)
      })

      expect(result.current.error).toHaveProperty('response.data.error.code', 'NOT_FOUND')
    })

    it('should not fetch when itemId is null', () => {
      const { result } = renderHook(() => useItems().useGetItem(null as unknown as string), { wrapper })

      expect(result.current.isLoading).toBe(false)
      expect(result.current.data).toBeUndefined()
    })
  })

  describe('useToggleStar', () => {
    it('should toggle star status to true', async () => {
      const { result } = renderHook(() => useItems().useToggleStar(), { wrapper })

      result.current.mutate({
        itemId: 'item-1',
        starred: true,
      })

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true)
      })

      expect(result.current.data?.item.user_state?.is_starred).toBe(true)
    })

    it('should toggle star status to false', async () => {
      const { result } = renderHook(() => useItems().useToggleStar(), { wrapper })

      result.current.mutate({
        itemId: 'item-1',
        starred: false,
      })

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true)
      })

      expect(result.current.data?.item.user_state?.is_starred).toBe(false)
    })

    it('should fail for non-existent item', async () => {
      const { result } = renderHook(() => useItems().useToggleStar(), { wrapper })

      result.current.mutate({
        itemId: 'non-existent',
        starred: true,
      })

      await waitFor(() => {
        expect(result.current.isError).toBe(true)
      })

      expect(result.current.error).toHaveProperty('response.data.error.code', 'NOT_FOUND')
    })
  })

  describe('useToggleRead', () => {
    it('should toggle read status to true', async () => {
      const { result } = renderHook(() => useItems().useToggleRead(), { wrapper })

      result.current.mutate({
        itemId: 'item-1',
        read: true,
      })

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true)
      })

      expect(result.current.data?.item.user_state?.is_read).toBe(true)
      expect(result.current.data?.item.user_state?.read_at).toBe('2024-01-01T01:00:00Z')
    })

    it('should toggle read status to false', async () => {
      const { result } = renderHook(() => useItems().useToggleRead(), { wrapper })

      result.current.mutate({
        itemId: 'item-1',
        read: false,
      })

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true)
      })

      expect(result.current.data?.item.user_state?.is_read).toBe(false)
      expect(result.current.data?.item.user_state?.read_at).toBeNull()
    })

    it('should fail for non-existent item', async () => {
      const { result } = renderHook(() => useItems().useToggleRead(), { wrapper })

      result.current.mutate({
        itemId: 'non-existent',
        read: true,
      })

      await waitFor(() => {
        expect(result.current.isError).toBe(true)
      })

      expect(result.current.error).toHaveProperty('response.data.error.code', 'NOT_FOUND')
    })
  })

  describe('useMarkAllRead', () => {
    it('should mark all items in feed as read', async () => {
      const { result } = renderHook(() => useItems().useMarkAllRead(), { wrapper })

      result.current.mutate('feed-1')

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true)
      })

      expect(result.current.data?.count).toBe(5)
    })

    it('should fail for non-existent feed', async () => {
      const { result } = renderHook(() => useItems().useMarkAllRead(), { wrapper })

      result.current.mutate('non-existent')

      await waitFor(() => {
        expect(result.current.isError).toBe(true)
      })

      expect(result.current.error).toHaveProperty('response.data.error.code', 'NOT_FOUND')
    })
  })
})
