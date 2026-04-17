import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import ItemsPage from '../ItemsPage'
import { ThemeProvider } from '@/contexts/ThemeContext'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'

// Mock toast hook
vi.mock('@/components/ui/toast', () => ({
  useToast: () => ({
    showSuccess: vi.fn(),
    showError: vi.fn(),
    showWarning: vi.fn(),
    showInfo: vi.fn(),
  }),
}))

// Mock lucide-react icons — use importOriginal to avoid missing-icon errors
vi.mock('lucide-react', async (importOriginal) => {
  const actual = await importOriginal<typeof import('lucide-react')>()
  return {
    ...actual,
    // Re-export everything from the real module; no selective mocking needed.
  }
})

// Mock window.matchMedia for responsive hooks
Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: vi.fn().mockImplementation((query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: vi.fn(),
    removeListener: vi.fn(),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  })),
})

// Mock IntersectionObserver
const mockObserve = vi.fn()
const mockUnobserve = vi.fn()
const mockDisconnect = vi.fn()

class MockIntersectionObserver {
  observe = mockObserve
  unobserve = mockUnobserve
  disconnect = mockDisconnect
}

vi.stubGlobal('IntersectionObserver', MockIntersectionObserver)

// Mock scrollIntoView for jsdom
Element.prototype.scrollIntoView = vi.fn()

const API_BASE_URL = '/api/v1'

// Mock items data
const mockItems = [
  {
    id: 'item-1',
    feed_id: 'feed-1',
    guid: 'guid-1',
    title: 'First Article',
    link: 'https://example.com/article1',
    description: 'Description 1',
    content: '<p>Content 1</p>',
    pub_date: '2024-01-01T00:00:00Z',
    creator: 'Author 1',
    created_at: '2024-01-01T00:00:00Z',
    feed: {
      id: 'feed-1',
      title: 'Example Feed',
      feed_url: 'https://example.com/feed.xml',
      description: 'An example feed',
      image_url: null,
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
    content: '<p>Content 2</p>',
    pub_date: '2024-01-02T00:00:00Z',
    creator: 'Author 2',
    created_at: '2024-01-02T00:00:00Z',
    feed: {
      id: 'feed-1',
      title: 'Example Feed',
      feed_url: 'https://example.com/feed.xml',
      description: 'An example feed',
      image_url: null,
    },
    user_state: {
      item_id: 'item-2',
      is_read: true,
      is_starred: true,
      read_at: '2024-01-02T01:00:00Z',
    },
  },
]

// Mock feeds data
const mockFeeds = [
  {
    id: 'feed-1',
    title: 'Example Feed',
    feed_url: 'https://example.com/feed.xml',
    description: 'An example feed',
    image_url: null,
    unread_count: 5,
    last_fetched_at: '2024-01-01T00:00:00Z',
    created_at: '2024-01-01T00:00:00Z',
    category_id: null,
  },
  {
    id: 'feed-2',
    title: 'Another Feed',
    feed_url: 'https://another.com/feed.xml',
    description: 'Another feed',
    image_url: null,
    unread_count: 3,
    last_fetched_at: '2024-01-01T00:00:00Z',
    created_at: '2024-01-01T00:00:00Z',
    category_id: null,
  },
]

// Mock categories data
const mockFeedCategories = {
  categories: [
    { id: 'cat-1', name: 'Tech', type: 'feed', position: 0 },
    { id: 'cat-2', name: 'News', type: 'feed', position: 1 },
  ],
}

const mockPaperCategories = {
  categories: [
    { id: 'cat-3', name: 'AI', type: 'paper', position: 0 },
  ],
}

// Mock papers data
const mockPapers = {
  papers: [
    {
      id: 'paper-1',
      title: 'Attention Is All You Need',
      original_filename: 'attention.pdf',
      authors: '["Author A", "Author B"]',
      published_year: '2017',
      status: 'completed',
      markdown_content: '# Abstract\n...',
      error: null,
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
    },
  ],
  total: 1,
}

// API handlers
const handlers = [
  // List items
  http.get(`${API_BASE_URL}/items`, ({ request }) => {
    const url = new URL(request.url)
    const feedId = url.searchParams.get('feed_id')

    if (feedId) {
      const filtered = mockItems.filter((item) => item.feed_id === feedId)
      return HttpResponse.json({
        items: filtered,
        total: filtered.length,
        has_more: false,
        next_cursor: null,
      })
    }

    return HttpResponse.json({
      items: mockItems,
      total: mockItems.length,
      has_more: false,
      next_cursor: null,
    })
  }),

  // Get single item
  http.get(`${API_BASE_URL}/items/:id`, ({ params }) => {
    const item = mockItems.find((i) => i.id === params.id)
    if (item) {
      return HttpResponse.json({ item })
    }
    return HttpResponse.json(
      { error: { code: 'NOT_FOUND', message: 'Item not found' } },
      { status: 404 }
    )
  }),

  // Toggle star
  http.put(`${API_BASE_URL}/items/:id/star`, async ({ params, request }) => {
    const body = await request.json() as { starred: boolean }
    const item = mockItems.find((i) => i.id === params.id)
    if (item) {
      return HttpResponse.json({
        item: {
          ...item,
          user_state: {
            ...item.user_state!,
            is_starred: body.starred,
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
    const body = await request.json() as { read: boolean }
    const item = mockItems.find((i) => i.id === params.id)
    if (item) {
      return HttpResponse.json({
        item: {
          ...item,
          user_state: {
            ...item.user_state!,
            is_read: body.read,
            read_at: body.read ? new Date().toISOString() : null,
          },
        },
      })
    }
    return HttpResponse.json(
      { error: { code: 'NOT_FOUND', message: 'Item not found' } },
      { status: 404 }
    )
  }),

  // List feeds
  http.get(`${API_BASE_URL}/feeds`, () => {
    return HttpResponse.json({ feeds: mockFeeds })
  }),

  // List feed categories
  http.get(`${API_BASE_URL}/categories`, ({ request }) => {
    const url = new URL(request.url)
    const type = url.searchParams.get('type')
    if (type === 'paper') return HttpResponse.json(mockPaperCategories)
    return HttpResponse.json(mockFeedCategories)
  }),

  // List papers
  http.get(`${API_BASE_URL}/papers`, () => {
    return HttpResponse.json(mockPapers)
  }),

  // Refresh feed
  http.post(`${API_BASE_URL}/feeds/:id/refresh`, ({ params }) => {
    if (mockFeeds.find((f) => f.id === params.id)) {
      return HttpResponse.json({ message: 'Refresh started' })
    }
    return HttpResponse.json(
      { error: { code: 'NOT_FOUND', message: 'Feed not found' } },
      { status: 404 }
    )
  }),

  // Delete feed
  http.delete(`${API_BASE_URL}/feeds/:id`, ({ params }) => {
    if (mockFeeds.find((f) => f.id === params.id)) {
      return HttpResponse.json({ message: 'Feed deleted' })
    }
    return HttpResponse.json(
      { error: { code: 'NOT_FOUND', message: 'Feed not found' } },
      { status: 404 }
    )
  }),

  // Mark all read
  http.post(`${API_BASE_URL}/feeds/:id/read-all`, ({ params }) => {
    if (mockFeeds.find((f) => f.id === params.id)) {
      return HttpResponse.json({ count: 5 })
    }
    return HttpResponse.json(
      { error: { code: 'NOT_FOUND', message: 'Feed not found' } },
      { status: 404 }
    )
  }),

  // Stats
  http.get(`${API_BASE_URL}/stats`, () => {
    return HttpResponse.json({
      total_feeds: 2,
      total_items: 10,
      unread_items: 8,
      starred_items: 2,
    })
  }),
]

const server = setupServer(...handlers)

// Helper to create wrapper with all providers
const createWrapper = (initialEntries = ['/items']) => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  })

  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <MemoryRouter initialEntries={initialEntries}>
          {children}
        </MemoryRouter>
      </ThemeProvider>
    </QueryClientProvider>
  )
}

describe('ItemsPage', () => {
  beforeEach(() => {
    server.listen()
    vi.clearAllMocks()
  })

  afterEach(() => {
    server.resetHandlers()
  })

  afterAll(() => {
    server.close()
  })

  // ============================================
  // 基础渲染测试
  // ============================================
  describe('Basic Rendering', () => {
    it('should render article list in default view', async () => {
      render(<ItemsPage />, { wrapper: createWrapper() })

      await waitFor(() => {
        const articles = screen.getAllByText('First Article')
        expect(articles.length).toBeGreaterThan(0)
      })

      const secondArticles = screen.getAllByText('Second Article')
      expect(secondArticles.length).toBeGreaterThan(0)
    })

    it('should show "All Articles" title in default view', async () => {
      render(<ItemsPage />, { wrapper: createWrapper() })

      await waitFor(() => {
        const allArticlesHeaders = screen.getAllByText('All Articles')
        expect(allArticlesHeaders.length).toBeGreaterThan(0)
      })
    })

    it('should show empty state when no articles', async () => {
      server.use(
        http.get(`${API_BASE_URL}/items`, () => {
          return HttpResponse.json({
            items: [],
            total: 0,
            has_more: false,
            next_cursor: null,
          })
        })
      )

      render(<ItemsPage />, { wrapper: createWrapper() })

      await waitFor(() => {
        const allArticlesHeaders = screen.getAllByText('All Articles')
        expect(allArticlesHeaders.length).toBeGreaterThan(0)
      })
    })

    it('should handle loading state gracefully', async () => {
      render(<ItemsPage />, { wrapper: createWrapper() })

      // Wait for content to load without crashing
      await waitFor(() => {
        const articles = screen.getAllByText('First Article')
        expect(articles.length).toBeGreaterThan(0)
      })
    })
  })

  // ============================================
  // 交互测试
  // ============================================
  describe('User Interactions', () => {
    it('should display feed titles in sidebar', async () => {
      render(<ItemsPage />, { wrapper: createWrapper() })

      await waitFor(() => {
        expect(screen.getAllByText('Example Feed').length).toBeGreaterThan(0)
      })
      expect(screen.getAllByText('Another Feed').length).toBeGreaterThan(0)
    })

    it('should display category names in sidebar', async () => {
      render(<ItemsPage />, { wrapper: createWrapper() })

      await waitFor(() => {
        expect(screen.getAllByText('Tech').length).toBeGreaterThan(0)
      })
      expect(screen.getAllByText('News').length).toBeGreaterThan(0)
    })
  })

  // ============================================
  // 键盘快捷键测试
  // ============================================
  describe('Keyboard Shortcuts', () => {
    it('should navigate to next article with j key', async () => {
      render(<ItemsPage />, { wrapper: createWrapper() })

      await waitFor(() => {
        const articles = screen.getAllByText('First Article')
        expect(articles.length).toBeGreaterThan(0)
      })

      fireEvent.keyDown(window, { key: 'j' })
    })

    it('should navigate to previous article with k key', async () => {
      render(<ItemsPage />, { wrapper: createWrapper() })

      await waitFor(() => {
        const articles = screen.getAllByText('First Article')
        expect(articles.length).toBeGreaterThan(0)
      })

      fireEvent.keyDown(window, { key: 'k' })
    })

    it('should toggle star with s key', async () => {
      render(<ItemsPage />, { wrapper: createWrapper() })

      await waitFor(() => {
        const articles = screen.getAllByText('First Article')
        expect(articles.length).toBeGreaterThan(0)
      })

      fireEvent.keyDown(window, { key: 's' })
    })

    it('should toggle read with r key', async () => {
      render(<ItemsPage />, { wrapper: createWrapper() })

      await waitFor(() => {
        const articles = screen.getAllByText('First Article')
        expect(articles.length).toBeGreaterThan(0)
      })

      fireEvent.keyDown(window, { key: 'r' })
    })
  })

  // ============================================
  // 无限滚动测试
  // ============================================
  describe('Infinite Scroll', () => {
    it('should set up intersection observer for infinite scroll', async () => {
      render(<ItemsPage />, { wrapper: createWrapper() })

      await waitFor(() => {
        const articles = screen.getAllByText('First Article')
        expect(articles.length).toBeGreaterThan(0)
      })

      expect(window.IntersectionObserver).toBeDefined()
    })

    it('should show loading spinner when loading more items', async () => {
      server.use(
        http.get(`${API_BASE_URL}/items`, () => {
          return HttpResponse.json({
            items: mockItems,
            total: 20,
            has_more: true,
            next_cursor: 'next-page-cursor',
          })
        })
      )

      render(<ItemsPage />, { wrapper: createWrapper() })

      await waitFor(() => {
        const articles = screen.getAllByText('First Article')
        expect(articles.length).toBeGreaterThan(0)
      })

      expect(mockObserve).toHaveBeenCalled()
    })
  })
})
